// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/zeebo/errs"

	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func main() {
	if err := run(context.Background(), os.Args[1], os.Args[2]); err != nil {
		log.Fatalf("%+v", err)
	}
}

type server struct {
	mu     sync.Mutex
	active map[int64]chan *jaeger.Span

	rbuf  []*jaeger.Span
	rbufn int
}

func run(ctx context.Context, iface, laddr string) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handle, err := pcapgo.NewEthernetHandle(iface)
	if err != nil {
		return err
	}
	src := gopacket.NewPacketSource(handle, layers.LinkTypeEthernet)

	s := &server{
		active: make(map[int64]chan *jaeger.Span),
		rbuf:   make([]*jaeger.Span, 1024*1024),
	}

	errch := make(chan error, 2)
	go func() { errch <- errs.Wrap(s.handlePackets(ctx, src)) }()
	go func() { errch <- errs.Wrap(http.ListenAndServe(laddr, s)) }()

	select {
	case err := <-errch:
		return errs.Wrap(err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *server) handlePackets(ctx context.Context, src *gopacket.PacketSource) error {
	for {
		packet, err := src.NextPacket()
		if err != nil {
			return err
		}
		udp, _ := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
		if udp == nil || udp.DstPort != 5775 {
			continue
		}
		if err := s.handlePacket(ctx, udp.Payload); err != nil {
			log.Printf("%+v", err)
		}
	}
}

func (s *server) handlePacket(ctx context.Context, buf []byte) (err error) {
	var batch agent.AgentEmitBatchArgs
	var mbuf thrift.TMemoryBuffer
	mbuf.Buffer = bytes.NewBuffer(buf)
	proto := thrift.NewTCompactProtocol(&mbuf)

	if _, _, _, err := proto.ReadMessageBegin(); err != nil {
		return errs.Wrap(err)
	}
	if err := batch.Read(proto); err != nil {
		return errs.Wrap(err)
	}
	if err := proto.ReadMessageEnd(); err != nil {
		return errs.Wrap(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, span := range batch.Batch.GetSpans() {
		s.rbuf[s.rbufn] = span
		s.rbufn++
		if s.rbufn >= len(s.rbuf) {
			s.rbufn = 0
		}

		if ch := s.active[span.TraceIdLow]; ch != nil {
			select {
			case ch <- span:
			default:
				log.Printf("dropped trace for %d", span.TraceIdLow)
			}
		}
	}

	return nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fl, _ := w.(interface{ Flush() })

	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/"), 0, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("filtering traces for %d (flush:%v)", id, fl != nil)
	defer log.Printf("done filtering traces for %d", id)

	ch := make(chan *jaeger.Span, 64)

	var mbuf thrift.TMemoryBuffer
	mbuf.Buffer = new(bytes.Buffer)
	proto := thrift.NewTCompactProtocol(&mbuf)

	send := func(span *jaeger.Span) bool {
		mbuf.Reset()

		if err := span.Write(proto); err != nil {
			log.Printf("error writing span: %v", err)
			return true
		}
		if _, err := w.Write(mbuf.Bytes()); err != nil {
			return false
		}
		if fl != nil {
			fl.Flush()
		}
		return true
	}

	s.mu.Lock()
	for _, span := range s.rbuf {
		if span == nil {
			break
		}
		if span.TraceIdLow != id {
			continue
		}
		if !send(span) {
			s.mu.Unlock()
			return
		}
	}

	s.active[id] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.active, id)
		s.mu.Unlock()
	}()

	for s := range ch {
		if !send(s) {
			return
		}
	}
}
