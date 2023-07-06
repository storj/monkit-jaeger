// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/common/context2"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

type transportType byte

const (
	// max size of a packet we can send to jaeger-agent in one request.
	// see: https://github.com/jaegertracing/jaeger-client-go/blob/1db6ae67694d13f4ecb454cd65b40034a687118a/utils/udp_client.go#L30
	maxPacketSizeUDP = 1000

	// max size of packet, when we use HTTP.
	maxPacketSizeHTTP = 1000000

	// jaeger-client-go has calculation for how this number is set.
	// see: https://github.com/jaegertracing/jaeger-client-go/blob/e75ea75c424f3127125aad39056a2718a3b5aa1d/transport_udp.go#L33
	emitBatchOverhead = 30

	// defaultQueueSize is the default size of the span queue.
	defaultQueueSize = 1000

	// defaultFlushInterval is the default interval to send data on ticker.
	defaultFlushInterval = 15 * time.Second

	// estimateSpanSize is the estimation size of a span we pre-allocate for pricise span size calculation.
	estimateSpanSize = 600

	udpTransportType  transportType = 1
	httpTransportType transportType = 2
)

// ThriftCollector matches the TraceCollector interface, but sends serialized
// jaeger.Span objects with the help of the registered transport, instead of the Scribe protocol. See
// RedirectPackets for the UDP server-side code.
type ThriftCollector struct {
	mu               sync.Mutex
	spansToSend      []*jaeger.Span // the spans waiting to be send to the agent
	currentSpanBytes int            // the current bytes used by spans when they are encoded into thrift buffer

	log           *zap.Logger
	ch            chan *jaeger.Span
	flushInterval time.Duration
	process       *jaeger.Process // the information of which process is sending the spans

	maxSpanBytes     int                   // the max bytes spans can take up to make sure we don't exceed maxPacketSize
	maxPacketSize    int                   // the max number of bytes this instance of UDPCollector allows for a single UDP packet
	spanSizeBuffer   *thrift.TMemoryBuffer // spanSizeBuffer helps us calculate the size of the span when thrift-encoded
	spanSizeProtocol thrift.TProtocol
	batchSeqNo       int64
	agentAddr        string
	transportType    transportType
}

// NewUDPCollector creates a UDPCollector that sends packets to jaeger agent, unless (!) you use different protocol in agentAddr.
// Deprecated: use NewThriftCollector instead. This exists only for compatibility reasons.
func NewUDPCollector(log *zap.Logger, agentAddr string, serviceName string, tags []Tag, packetSize, queueSize int, flushInterval time.Duration) (*ThriftCollector, error) {
	return NewThriftCollector(log, agentAddr, serviceName, tags, packetSize, queueSize, flushInterval)
}

// NewThriftCollector creates a UDPCollector that sends packets to jaeger agent.
func NewThriftCollector(log *zap.Logger, agentAddr string, serviceName string, tags []Tag, packetSize, queueSize int, flushInterval time.Duration) (
	*ThriftCollector, error) {

	tt := udpTransportType
	parsedURL, err := url.Parse(agentAddr)
	if err == nil && strings.Contains(parsedURL.Scheme, "http") {
		tt = httpTransportType
	}

	if packetSize == 0 {
		if tt == httpTransportType {
			packetSize = maxPacketSizeHTTP
		} else {
			packetSize = maxPacketSizeUDP
		}
	}

	if queueSize == 0 {
		queueSize = defaultQueueSize
	}

	if flushInterval == 0 {
		flushInterval = defaultFlushInterval
	}

	var protocolFactory thrift.TProtocolFactory
	if tt == httpTransportType {
		protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	} else {
		protocolFactory = thrift.NewTCompactProtocolFactory()
	}
	spanSizeBuffer := thrift.NewTMemoryBufferLen(estimateSpanSize)
	spanSizeProtocol := protocolFactory.GetProtocol(spanSizeBuffer)

	jaegerTags := make([]*jaeger.Tag, 0, len(tags))
	for _, tag := range tags {
		j, err := tag.BuildJaegerThrift()
		if err != nil {
			log.Debug("failed to convert to jaeger tags", zap.Error(err))
			continue
		}
		jaegerTags = append(jaegerTags, j)
	}

	jaegerProcess := &jaeger.Process{
		ServiceName: serviceName,
		Tags:        jaegerTags,
	}

	processByteSize, err := calculateThriftSize(jaegerProcess, spanSizeBuffer, spanSizeProtocol)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &ThriftCollector{
		log:              log.Named("tracing collector"),
		ch:               make(chan *jaeger.Span, queueSize),
		flushInterval:    flushInterval,
		maxSpanBytes:     packetSize - emitBatchOverhead - processByteSize,
		spanSizeBuffer:   spanSizeBuffer,
		spanSizeProtocol: spanSizeProtocol,
		maxPacketSize:    packetSize,
		process:          jaegerProcess,
		agentAddr:        agentAddr,
		transportType:    tt,
	}, nil
}

// Run reads spans off the queue and appends them to the buffer. When the
// buffer fills up, it flushes. It also flushes on a jittered interval.
func (c *ThriftCollector) Run(ctx context.Context) {
	c.log.Debug("started")
	defer c.log.Debug("stopped")

	var tp Transport
	var err error
	switch c.transportType {
	case httpTransportType:
		tp, err = OpenHTTPTransport(ctx, c.log, c.agentAddr)
		if err != nil {
			return
		}
	case udpTransportType:
		tp, err = OpenUDPTransport(ctx, c.log, c.agentAddr, c.maxPacketSize)
		if err != nil {
			return
		}
	default:
		panic("Unsupported transport type: ")
	}
	defer tp.Close()

	ticker := time.NewTicker(jitter(c.flushInterval))
	defer ticker.Stop()

	for {
		select {
		case s := <-c.ch:
			err := c.handleSpan(ctx, s, tp)
			if err != nil {
				mon.Counter("jaeger_span_handling_failure").Inc(1)
				c.log.Debug("failed to handle span", zap.Error(err))
			}
		case <-ticker.C:
			if err := c.Send(ctx, tp); err != nil {
				c.log.Debug("failed to send on ticker", zap.Error(err))
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
				err := c.handleSpan(ctxWithoutCancel, s, tp)
				if err != nil {
					mon.Counter("jaeger_span_handling_failure").Inc(1)
					c.log.Debug("failed to handle span", zap.Error(err))
				}
			}
			if err := c.send(ctxWithoutCancel, tp); err != nil {
				c.log.Debug("failed to send on close", zap.Error(err))
			}
			return
		}
	}
}

// Close gracefully shutdown the underlying udp connection, after remaining messages are sent out.
// Deprecated: cancelling the context of run will close the connection.
func (c *ThriftCollector) Close() error {
	return nil
}

// handleSpan adds a new span into the buffer.
func (c *ThriftCollector) handleSpan(ctx context.Context, s *jaeger.Span, transport Transport) (err error) {
	spanSize, err := calculateThriftSize(s, c.spanSizeBuffer, c.spanSizeProtocol)
	if err != nil {
		return errs.Wrap(err)
	}

	if spanSize > c.maxSpanBytes {
		mon.Counter("jaeger_span_too_large").Inc(1)
		return errs.New("span is too large. Expected no bigger than %d, got %d", c.maxSpanBytes, spanSize)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentSpanBytes+spanSize > c.maxSpanBytes {
		if err := c.send(ctx, transport); err != nil {
			return errs.Wrap(err)
		}
	}

	c.currentSpanBytes += spanSize
	c.spansToSend = append(c.spansToSend, s)

	return nil
}

// Send sends traces to jaeger agent.
func (c *ThriftCollector) Send(ctx context.Context, transport Transport) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.send(ctx, transport)
}

func (c *ThriftCollector) send(ctx context.Context, transport Transport) (err error) {
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
	defer c.resetSpanBuffer()
	err = transport.Send(ctx, batch)
	if err != nil {
		return errs.Wrap(err)
	}

	return nil
}

// Collect takes a jaeger.Span object, serializes it, and sends it to the
// configured collector_addr.
func (c *ThriftCollector) Collect(span *jaeger.Span) {
	select {
	case c.ch <- span:
	default:
		mon.Counter("jaeger_buffer_full").Inc(1)
	}
}

// Len returns the amount of spans in the queue currently.
// This is only exposed for testing purpose.
func (c *ThriftCollector) Len() int {
	return len(c.ch)
}

func (c *ThriftCollector) resetSpanBuffer() {
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
