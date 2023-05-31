// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"fmt"
	"net"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// UDPTransport sends jaeger batches via UDP.
type UDPTransport struct {
	thriftBuffer  *thrift.TMemoryBuffer
	conn          *net.UDPConn
	log           *zap.Logger
	client        *agent.AgentClient
	maxPacketSize int
}

var _ Transport = &UDPTransport{}

// OpenUDPTransport creates new transport to send Jaeger batches via UDP.
func OpenUDPTransport(ctx context.Context, log *zap.Logger, agentAddr string, maxPacketSize int) (*UDPTransport, error) {
	var err error

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", agentAddr)
	if err != nil {
		log.Debug("failed open  UDP connection to Jaeger", zap.Error(err))
		return nil, err
	}

	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		log.Debug("Connection type mismatch", zap.Error(err))
		return nil, err
	}

	if err := udpConn.SetWriteBuffer(maxPacketSize); err != nil {
		log.Debug("failed to set max packet size on Jaeger UDP connection", zap.Error(err), zap.Int("maxPacketSize", maxPacketSize))
		return nil, err
	}

	protocolFactory := thrift.NewTCompactProtocolFactory()
	thriftBuffer := thrift.NewTMemoryBufferLen(maxPacketSize)
	client := agent.NewAgentClientFactory(thriftBuffer, protocolFactory)

	return &UDPTransport{
		client:        client,
		thriftBuffer:  thriftBuffer,
		log:           log,
		conn:          udpConn,
		maxPacketSize: maxPacketSize,
	}, nil

}

// Send sends out the Jaeger spans.
func (u *UDPTransport) Send(ctx context.Context, batch *jaeger.Batch) error {
	// Reset the thriftBuffer so that EmitBatch can write onto an empty buffer
	u.thriftBuffer.Reset()
	if err := u.client.EmitBatch(ctx, batch); err != nil {
		return errs.Wrap(err)
	}

	// Reset the span buffer no matter we succeed or not to prevent getting into an infinite loop
	// it probably is ok if we lose one batch of trace since these are just metrics data
	if u.thriftBuffer.Len() > u.maxPacketSize {
		mon.Counter("jaeger_exceeds_packet_size").Inc(1)
		return fmt.Errorf("data does not fit within one UDP packet; size %d, max %d, spans %d",
			u.thriftBuffer.Len(), u.maxPacketSize, len(batch.Spans))
	}

	_, err := u.conn.Write(u.thriftBuffer.Bytes())
	return err
}

// Close closes the transport.
func (u *UDPTransport) Close() {
	err := u.conn.Close()
	if err != nil {
		u.log.Debug("failed to close Jaeger UDP connection", zap.Error(err))
	}
}
