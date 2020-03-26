// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/apache/thrift/lib/go/thrift"

	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

const (
	maxPacketSize = 8192
)

// UDPCollector matches the TraceCollector interface, but sends serialized
// jaeger.Span objects over UDP, instead of the Scribe protocol. See
// RedirectPackets for the UDP server-side code.
type UDPCollector struct {
	ch            chan *jaeger.Span
	serviceName   string
	client        *agent.AgentClient
	conn          *net.UDPConn
	thriftBuffer  *thrift.TMemoryBuffer
	maxPacketSize int
	batchSeqNo    int64
}

// NewUDPCollector creates a UDPCollector that sends packets to collector_addr.
// buffer_size is how many outstanding unsent jaeger.Span objects can exist
// before Spans start getting dropped.
func NewUDPCollector(collectorAddr string, bufferSize int, serviceName string) (
	*UDPCollector, error) {

	thriftBuffer := thrift.NewTMemoryBufferLen(bufferSize)
	protocolFactory := thrift.NewTCompactProtocolFactory()
	client := agent.NewAgentClientFactory(thriftBuffer, protocolFactory)

	destAddr, err := net.ResolveUDPAddr("udp", collectorAddr)
	if err != nil {
		return nil, err
	}

	connUDP, err := net.DialUDP(destAddr.Network(), nil, destAddr)
	if err != nil {
		return nil, err
	}
	if err := connUDP.SetWriteBuffer(maxPacketSize); err != nil {
		return nil, err
	}
	c := &UDPCollector{
		ch:            make(chan *jaeger.Span, bufferSize),
		serviceName:   serviceName,
		client:        client,
		conn:          connUDP,
		thriftBuffer:  thriftBuffer,
		maxPacketSize: bufferSize,
	}
	go c.handle()
	return c, nil
}

func (c *UDPCollector) handle() {
	for {
		s, ok := <-c.ch
		if !ok {
			return
		}
		err := c.send(s)
		if err != nil {
			log.Printf("failed write: %v", err)
		}
	}
}

func (c *UDPCollector) send(s *jaeger.Span) error {
	process := &jaeger.Process{ServiceName: c.serviceName}
	c.batchSeqNo++
	batchSeqNo := c.batchSeqNo
	batch := &jaeger.Batch{
		Process: process,
		Spans:   []*jaeger.Span{s},
		SeqNo:   &batchSeqNo,
	}
	c.thriftBuffer.Reset()
	if err := c.client.EmitBatch(context.Background(), batch); err != nil {
		return err
	}
	if c.thriftBuffer.Len() > c.maxPacketSize {
		return fmt.Errorf("data does not fit within one UDP packet; size %d, max %d, spans %d",
			c.thriftBuffer.Len(), c.maxPacketSize, len(batch.Spans))
	}
	_, err := c.conn.Write(c.thriftBuffer.Bytes())
	return err

}

// Collect takes a jaeger.Span object, serializes it, and sends it to the
// configured collector_addr.
func (c *UDPCollector) Collect(span *jaeger.Span) {
	select {
	case c.ch <- span:
	default:
	}
}
