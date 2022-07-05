// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
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

type subscription struct {
	traceID int64
	ch      chan *jaeger.Span
}

type server struct {
	mu     sync.Mutex
	active map[*http.Request]*subscription

	rbufs map[int64][]*jaeger.Span
	rbuft []int64
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
		active: make(map[*http.Request]*subscription),
		rbufs:  make(map[int64][]*jaeger.Span),
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
		rbuf, ok := s.rbufs[span.TraceIdLow]
		if !ok {
			s.rbuft = append(s.rbuft, span.TraceIdLow)
		}
		s.rbufs[span.TraceIdLow] = append(rbuf, span)
		s.rbufn++

		if s.rbufn >= 1024*1024 {
			evict := s.rbuft[0]
			s.rbuft = s.rbuft[1:]
			s.rbufn -= len(s.rbufs[evict])
			delete(s.rbufs, evict)
		}

		// this _should_ be relatively small compared to the other buffer
		for _, sub := range s.active {
			if sub.traceID == span.TraceIdLow {
				select {
				case sub.ch <- span:
				default:
					log.Printf("dropped trace for %d", span.TraceIdLow)
				}
			}
		}
	}

	return nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	accept := r.Header.Get("Accept")
	switch {
	case accept == "application/json":
		s.Subscribe(w, r)
	default:
		s.Visualize(w, r)
	}
}

func (s *server) Subscribe(w http.ResponseWriter, r *http.Request) {
	fl, _ := w.(interface{ Flush() })

	q := r.URL.Query()

	// wait for additional traces to be returned
	wait := true
	if w := q.Get("wait"); w != "" {
		wait, _ = strconv.ParseBool(w)
	}

	// which trace we're looking at
	traceID, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/"), 0, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("filtering traces for %d (wait=%s)", traceID, strconv.FormatBool(wait))
	defer log.Printf("done filtering traces for %d (wait=%s)", traceID, strconv.FormatBool(wait))

	// stream json spans back to the client
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	out := json.NewEncoder(w)

	sub := &subscription{
		traceID: traceID,
		ch:      make(chan *jaeger.Span, 64),
	}

	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.active[r] != nil {
			delete(s.active, r)
		}

		close(sub.ch)
	}()

	err = func() error {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.active[r] = sub

		for _, span := range s.rbufs[traceID] {
			err = out.Encode(span)
			if err != nil {
				return err
			}
		}

		if fl != nil {
			fl.Flush()
		}

		return nil
	}()

	switch {
	case err != nil:
		// json encoding err, or output closed
		// log err
		return
	case !wait:
		// only return the buffer, don't wait for new data
		return
	}

	for {
		select {
		case <-r.Context().Done():
			// log r.Context().Err()
			return
		case span := <-sub.ch:
			err = out.Encode(span)
			if err != nil {
				// log err
				return
			}
			if fl != nil {
				fl.Flush()
			}
		}
	}
}

func (s *server) Visualize(w http.ResponseWriter, r *http.Request) {
	// render UI
}
