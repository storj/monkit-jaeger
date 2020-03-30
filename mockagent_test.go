// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"net"
	"sync"

	"github.com/apache/thrift/lib/go/thrift"

	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// StartMockAgent starts a mock agent on a local udp port.
func StartMockAgent(hostPort string) (*MockAgent, error) {
	addr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP(addr.Network(), addr)
	if err != nil {
		return nil, err
	}
	mock := &MockAgent{
		batches: make([]*jaeger.Batch, 0),
		mu:      &sync.Mutex{},
		conn:    conn,
		addr:    conn.LocalAddr().String(),
	}

	mock.serve()
	return mock, err
}

// MockAgent implements jaeger agent interface.
type MockAgent struct {
	conn *net.UDPConn
	addr string

	mu      *sync.Mutex
	closed  bool
	batches []*jaeger.Batch
}

// EmitBatch implements jaeger agent interface.
func (m *MockAgent) EmitBatch(ctx context.Context, batch *jaeger.Batch) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches = append(m.batches, batch)
	return nil
}

// GetBatches returns batches jaeger agent received.
func (m *MockAgent) GetBatches() []*jaeger.Batch {
	m.mu.Lock()
	defer m.mu.Unlock()
	batches := make([]*jaeger.Batch, len(m.batches))
	copy(batches, m.batches)
	return batches
}

// Addr returns the address of the agent.
func (m *MockAgent) Addr() string {
	return m.addr
}

// Close shutdown mock agent server.
func (m *MockAgent) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()

	return m.conn.Close()
}

func (m *MockAgent) serve() {
	handler := agent.NewAgentProcessor(m)
	protocolFact := thrift.NewTCompactProtocolFactory()
	trans := thrift.NewTMemoryBufferLen(maxPacketSize)
	buf := make([]byte, maxPacketSize)
	go func() {
		for !m.isClosed() {
			n, err := m.conn.Read(buf)
			if err == nil {
				trans.Write(buf[:n])
				protocol := protocolFact.GetProtocol(trans)
				_, _ = handler.Process(context.Background(), protocol, protocol)
			}
		}
	}()
}

func (m *MockAgent) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}
