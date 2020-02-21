// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package zipkin

import (
	"log"
	"net"

	"github.com/apache/thrift/lib/go/thrift"
	"gopkg.in/spacemonkeygo/monkit-zipkin.v2/gen-go/zipkin"
)

const (
	maxPacketSize = 8192
)

// UDPCollector matches the TraceCollector interface, but sends serialized
// zipkin.Span objects over UDP, instead of the Scribe protocol. See
// RedirectPackets for the UDP server-side code.
type UDPCollector struct {
	ch   chan *zipkin.Span
	conn *net.UDPConn
	addr *net.UDPAddr
}

// NewUDPCollector creates a UDPCollector that sends packets to collector_addr.
// buffer_size is how many outstanding unsent zipkin.Span objects can exist
// before Spans start getting dropped.
func NewUDPCollector(collector_addr string, buffer_size int) (
	*UDPCollector, error) {
	addr, err := net.ResolveUDPAddr("udp", collector_addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}
	c := &UDPCollector{
		ch:   make(chan *zipkin.Span, buffer_size),
		conn: conn,
		addr: addr}
	go c.handle()
	return c, nil
}

func (c *UDPCollector) handle() {
	for {
		select {
		case s, ok := <-c.ch:
			if !ok {
				return
			}
			err := c.send(s)
			if err != nil {
				log.Printf("failed write: %v", err)
			}
		}
	}
}

func (c *UDPCollector) send(s *zipkin.Span) error {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	err := s.Write(p)
	if err != nil {
		return err
	}
	_, err = c.conn.WriteToUDP(t.Buffer.Bytes(), c.addr)
	return err
}

// Collect takes a zipkin.Span object, serializes it, and sends it to the
// configured collector_addr.
func (c *UDPCollector) Collect(span *zipkin.Span) {
	select {
	case c.ch <- span:
	default:
	}
}

// RedirectPackets is a method that handles incoming packets from the
// UDPCollector class. RedirectPackets, when running, will listen for UDP
// packets containing serialized zipkin.Span objects on listen_addr, then will
// resend those packets to the given ScribeCollector. On any error,
// RedirectPackets currently aborts.
func RedirectPackets(listen_addr string, collector *ScribeCollector) error {
	la, err := net.ResolveUDPAddr("udp", listen_addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", la)
	if err != nil {
		return err
	}
	defer conn.Close()
	var buf [maxPacketSize]byte
	for {
		n, _, err := conn.ReadFrom(buf[:])
		if err != nil {
			return err
		}
		err = collector.CollectSerialized(buf[:n])
		if err != nil {
			return err
		}
	}
}
