// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/common/context2"
	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

const (
	// max size of UDP packet we can send to jaeger-agent.
	// see: https://github.com/jaegertracing/jaeger-client-go/blob/1db6ae67694d13f4ecb454cd65b40034a687118a/utils/udp_client.go#L30
	maxPacketSize = 65000

	// jaeger-client-go has calculation for how this number is set.
	// see: https://github.com/jaegertracing/jaeger-client-go/blob/e75ea75c424f3127125aad39056a2718a3b5aa1d/transport_udp.go#L33
	emitBatchOverhead = 70

	// defaultQueueSize is the default size of the span queue.
	defaultQueueSize = 1000

	// defaultFlushInterval is the default interval to send data on ticker.
	defaultFlushInterval = 5 * time.Minute

	// estimateSpanSize is the estimation size of a span we pre-allocate for pricise span size calculation.
	estimateSpanSize = 600
)

// UDPCollector matches the TraceCollector interface, but sends serialized
// jaeger.Span objects over UDP, instead of the Scribe protocol. See
// RedirectPackets for the UDP server-side code.
type UDPCollector struct {
	mu               sync.Mutex
	spansToSend      []*jaeger.Span        // the spans waiting to be send to the agent
	thriftBuffer     *thrift.TMemoryBuffer // the buffer where we encode data to send to the agent
	currentSpanBytes int                   // the current bytes used by spans when they are encoded into thrift buffer

	log              *zap.Logger
	ch               chan *jaeger.Span
	flushInterval    time.Duration
	process          *jaeger.Process // the information of which process is sending the spans
	client           *agent.AgentClient
	conn             *net.UDPConn
	maxSpanBytes     int                   // the max bytes spans can take up to make sure we don't exceed maxPacketSize
	maxPacketSize    int                   // the max number of bytes this instance of UDPCollector allows for a single UDP packet
	spanSizeBuffer   *thrift.TMemoryBuffer // spanSizeBuffer helps us calculate the size of the span when thrift-encoded
	thriftProtocol   thrift.TProtocol
	spanSizeProtocol thrift.TProtocol
	batchSeqNo       int64
}

// NewUDPCollector creates a UDPCollector that sends packets to jaeger agent.
func NewUDPCollector(log *zap.Logger, agentAddr string, serviceName string, tags []Tag, packetSize, queueSize int, flushInterval time.Duration) (
	*UDPCollector, error) {

	if packetSize == 0 {
		packetSize = maxPacketSize
	}

	if queueSize == 0 {
		queueSize = defaultQueueSize
	}

	if flushInterval == 0 {
		flushInterval = defaultFlushInterval
	}

	thriftBuffer := thrift.NewTMemoryBufferLen(packetSize)
	spanSizeBuffer := thrift.NewTMemoryBufferLen(estimateSpanSize)
	protocolFactory := thrift.NewTCompactProtocolFactory()
	thriftProtocol := protocolFactory.GetProtocol(thriftBuffer)
	spanSizeProtocol := protocolFactory.GetProtocol(spanSizeBuffer)
	client := agent.NewAgentClientFactory(thriftBuffer, protocolFactory)

	destAddr, err := net.ResolveUDPAddr("udp", agentAddr)
	if err != nil {
		return nil, err
	}

	jaegerTags := make([]*jaeger.Tag, 0, len(tags))
	for _, tag := range tags {
		j, err := tag.BuildJaegerThrift()
		if err != nil {
			log.Info("failed to convert to jaeger tags", zap.Error(err))
			continue
		}
		jaegerTags = append(jaegerTags, j)
	}

	connUDP, err := net.DialUDP(destAddr.Network(), nil, destAddr)
	if err != nil {
		return nil, err
	}
	if err := connUDP.SetWriteBuffer(packetSize); err != nil {
		return nil, errs.Combine(err, connUDP.Close())
	}

	jaegerProcess := &jaeger.Process{
		ServiceName: serviceName,
		Tags:        jaegerTags,
	}

	processByteSize, err := calculateThriftSize(jaegerProcess, spanSizeBuffer, spanSizeProtocol)
	if err != nil {
		return nil, errs.Combine(err, connUDP.Close())
	}

	return &UDPCollector{
		log:              log.Named("tracing collector"),
		ch:               make(chan *jaeger.Span, queueSize),
		client:           client,
		flushInterval:    flushInterval,
		conn:             connUDP,
		maxSpanBytes:     packetSize - emitBatchOverhead - processByteSize,
		spanSizeBuffer:   spanSizeBuffer,
		thriftBuffer:     thriftBuffer,
		thriftProtocol:   thriftProtocol,
		spanSizeProtocol: spanSizeProtocol,
		maxPacketSize:    packetSize,
		process:          jaegerProcess,
	}, nil
}

// Run reads spans off the queue and appends them to the buffer. When the
// buffer fills up, it flushes. It also flushes on a jittered interval.
func (c *UDPCollector) Run(ctx context.Context) {
	c.log.Debug("started")
	defer c.log.Debug("stopped")

	ticker := time.NewTicker(jitter(c.flushInterval))
	defer ticker.Stop()

	for {
		select {
		case s := <-c.ch:
			err := c.handleSpan(ctx, s)
			if err != nil {
				mon.Counter("jaeger-span-handling-failure").Inc(1)
				c.log.Error("failed to handle span", zap.Error(err))
			}
		case <-ticker.C:
			if err := c.Send(ctx); err != nil {
				c.log.Error("failed to send on ticker", zap.Error(err))
			}
			ticker.Reset(jitter(c.flushInterval))
			// clear ticker
			select {
			case <-ticker.C:
			default:
			}
		case <-ctx.Done():
			// drain the channel on shutdown
			left := len(c.ch)
			ctxWithoutCancel := context2.WithoutCancellation(ctx)
			for i := 0; i < left; i++ {
				s := <-c.ch
				err := c.handleSpan(ctxWithoutCancel, s)
				if err != nil {
					mon.Counter("jaeger-span-handling-failure").Inc(1)
					c.log.Error("failed to handle span", zap.Error(err))
				}
			}
			return
		}
	}
}

// Close shutdown the underlying udp connection.
func (c *UDPCollector) Close() error {
	return c.conn.Close()
}

// handleSpan adds a new span into the buffer.
func (c *UDPCollector) handleSpan(ctx context.Context, s *jaeger.Span) (err error) {
	spanSize, err := calculateThriftSize(s, c.spanSizeBuffer, c.spanSizeProtocol)
	if err != nil {
		return errs.Wrap(err)
	}

	if spanSize > c.maxSpanBytes {
		mon.Counter("jaeger-span-too-large").Inc(1)
		return errs.New("span is too large")
	}

	c.mu.Lock()
	currentSpanBytes := c.currentSpanBytes
	c.mu.Unlock()

	if currentSpanBytes+spanSize >= c.maxSpanBytes {
		if err := c.Send(ctx); err != nil {
			return errs.Wrap(err)
		}
	}

	c.mu.Lock()
	c.currentSpanBytes += spanSize
	c.spansToSend = append(c.spansToSend, s)
	c.mu.Unlock()

	return nil
}

// Send sends traces to jaeger agent.
func (c *UDPCollector) Send(ctx context.Context) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.spansToSend) == 0 {
		return nil
	}

	c.batchSeqNo++
	batchSeqNo := c.batchSeqNo
	batch := &jaeger.Batch{
		Process: c.process,
		Spans:   c.spansToSend,
		SeqNo:   &batchSeqNo,
	}

	// Reset the thriftBuffer so that EmitBatch can write onto an empty buffer
	c.thriftBuffer.Reset()
	if err := c.client.EmitBatch(ctx, batch); err != nil {
		return errs.Wrap(err)
	}

	// Reset the span buffer no matter we succeed or not to prevent getting into an infinite loop
	// it probably is ok if we lose one batch of trace since these are just metrics data
	defer c.resetSpanBuffer()
	if c.thriftBuffer.Len() > c.maxPacketSize {
		mon.Counter("jaeger-exceeds-packet-size").Inc(1)
		return fmt.Errorf("data does not fit within one UDP packet; size %d, max %d, spans %d",
			c.thriftBuffer.Len(), c.maxPacketSize, len(batch.Spans))
	}

	_, err = c.conn.Write(c.thriftBuffer.Bytes())
	if err != nil {
		return errs.Wrap(err)
	}

	return nil
}

// Collect takes a jaeger.Span object, serializes it, and sends it to the
// configured collector_addr.
func (c *UDPCollector) Collect(span *jaeger.Span) {
	select {
	case c.ch <- span:
	default:
		mon.Counter("jaeger-buffer-full").Inc(1)
	}
}

// Len returns the amount of spans in the queue currently.
// This is only exposed for testing purpose.
func (c *UDPCollector) Len() int {
	return len(c.ch)
}

func (c *UDPCollector) resetSpanBuffer() {
	for i := range c.spansToSend {
		c.spansToSend[i] = nil
	}
	c.spansToSend = c.spansToSend[:0]
	c.currentSpanBytes = 0
}

func calculateThriftSize(data thrift.TStruct, buffer *thrift.TMemoryBuffer, protocol thrift.TProtocol) (int, error) {
	buffer.Reset()
	err := data.Write(protocol)
	if err != nil {
		return 0, err
	}

	return buffer.Len(), nil
}

func jitter(t time.Duration) time.Duration {
	nanos := rand.NormFloat64()*float64(t/4) + float64(t)
	if nanos <= 0 {
		nanos = 1
	}
	return time.Duration(nanos)
}
