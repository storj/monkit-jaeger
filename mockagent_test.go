// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/stretchr/testify/require"

	"storj.io/monkit-jaeger/gen-go/agent"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// withAgent starts a mock agent on a local udp port.
func withAgent(t *testing.T, f func(mock *MockAgent)) {
	mock := NewMockAgent()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := mock.Serve()
		require.NoError(t, err)
		wg.Done()
	}()

	mock.WaitForStart()

	f(mock)

	err := mock.Close()
	require.NoError(t, err)
	wg.Wait()
}

// MockAgent implements jaeger agent interface.
type MockAgent struct {
	conn *net.UDPConn
	addr string

	cond    *sync.Cond
	started chan struct{}
	closed  bool
	batches []*jaeger.Batch
}

func NewMockAgent() *MockAgent {
	return &MockAgent{
		cond:    sync.NewCond(new(sync.Mutex)),
		batches: make([]*jaeger.Batch, 0),
		started: make(chan struct{}),
	}
}

// EmitBatch implements jaeger agent interface.
func (m *MockAgent) EmitBatch(ctx context.Context, batch *jaeger.Batch) (err error) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	m.batches = append(m.batches, batch)
	m.cond.Broadcast()

	return nil
}

func (m *MockAgent) WaitForBatches(dur time.Duration) []*jaeger.Batch {
	done := false
	timer := time.AfterFunc(dur, func() {
		m.cond.L.Lock()
		done = true
		m.cond.L.Unlock()
		m.cond.Broadcast()
	})
	defer timer.Stop()

	m.cond.L.Lock()
	defer m.cond.L.Unlock()

	for len(m.batches) == 0 && !done {
		m.cond.Wait()
	}

	if done {
		return nil
	}

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
	m.cond.L.Lock()
	m.closed = true
	m.cond.L.Unlock()

	return m.conn.Close()
}

// Serve starts the mock agent.
func (m *MockAgent) Serve() error {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	m.conn, err = net.ListenUDP(addr.Network(), addr)
	if err != nil {
		return err
	}

	m.addr = m.conn.LocalAddr().String()

	handler := agent.NewAgentProcessor(m)
	protocolFact := thrift.NewTCompactProtocolFactory()
	trans := thrift.NewTMemoryBufferLen(maxPacketSize)
	buf := make([]byte, maxPacketSize)

	close(m.started)
	for !m.isClosed() {
		n, err := m.conn.Read(buf)
		if err == nil {
			trans.Write(buf[:n])
			protocol := protocolFact.GetProtocol(trans)
			_, _ = handler.Process(context.Background(), protocol, protocol)
		}
	}
	return nil
}

// WaitForStart returns when mock agent server is ready.
func (m *MockAgent) WaitForStart() {
	<-m.started
}

func (m *MockAgent) isClosed() bool {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	return m.closed
}
