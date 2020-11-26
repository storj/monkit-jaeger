// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"testing"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"

	"storj.io/common/testcontext"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func withCollector(ctx context.Context, t *testing.T, agentAddr string,
	packetSize int, interval time.Duration, f func(*UDPCollector)) {

	// if the interval is 0 (default), make it incredibly so long it's
	// impossible for it to trigger during the test
	if interval == 0 {
		interval = 24 * time.Hour
	}

	collector, err := NewUDPCollector(zaptest.NewLogger(t), agentAddr, "test", nil, packetSize, 0, interval)
	require.NoError(t, err)

	var eg errgroup.Group

	ctx, cancel := context.WithCancel(ctx)
	eg.Go(func() error {
		collector.Run(ctx)
		return nil
	})

	f(collector)
	cancel()
	require.NoError(t, eg.Wait())

	// Ensure the collector queue has been drained on shutdown
	require.Zero(t, collector.Len())
	require.NoError(t, collector.Close())
}

func TestSendIsTriggeredByInterval(t *testing.T) {
	ctx := testcontext.New(t)
	withAgent(t, func(mock *MockAgent) {
		withCollector(ctx, t, mock.Addr(), 99999999, time.Nanosecond, func(collector *UDPCollector) {

			// let's fill it with a one span
			collector.Collect(&jaeger.Span{
				TraceIdLow:    monkit.NewId(),
				SpanId:        monkit.NewId(),
				OperationName: "test-udp-collector",
				StartTime:     time.Now().UnixNano() / 1000,
				Duration:      time.Second.Microseconds(),
			})

			batches := mock.WaitForBatches(time.Second)
			require.Equal(t, len(batches), 1)
		})
	})
}

func TestSendIsTriggeredByManySpans(t *testing.T) {
	ctx := testcontext.New(t)
	withAgent(t, func(mock *MockAgent) {
		withCollector(ctx, t, mock.Addr(), 200, 0, func(collector *UDPCollector) {

			// let's fill it with a number of spans
			for i := 0; i < 100; i++ {
				collector.Collect(&jaeger.Span{
					TraceIdLow:    monkit.NewId(),
					SpanId:        monkit.NewId(),
					OperationName: "test-udp-collector",
					StartTime:     time.Now().UnixNano() / 1000,
					Duration:      time.Second.Microseconds(),
				})
			}

			batches := mock.WaitForBatches(time.Second)
			require.True(t, len(batches) > 0)
		})
	})
}

func TestUDPCollector(t *testing.T) {
	ctx := testcontext.New(t)
	withAgent(t, func(mock *MockAgent) {
		withCollector(ctx, t, mock.Addr(), 0, time.Nanosecond, func(collector *UDPCollector) {
			span := &jaeger.Span{
				TraceIdLow:    monkit.NewId(),
				SpanId:        monkit.NewId(),
				OperationName: "test-udp-collector",
				StartTime:     time.Now().UnixNano() / 1000,
				Duration:      time.Second.Microseconds(),
			}

			collector.Collect(span)

			batches := mock.WaitForBatches(time.Second)
			require.Len(t, batches, 1)
			require.Len(t, batches[0].GetSpans(), 1)
			receivedSpan := batches[0].GetSpans()[0]
			require.Equal(t, span.GetOperationName(), receivedSpan.OperationName)
			require.Equal(t, span.GetTraceIdLow(), receivedSpan.GetTraceIdLow())
			require.Equal(t, span.GetSpanId(), receivedSpan.GetSpanId())
		})
	})
}
