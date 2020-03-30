// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"testing"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/stretchr/testify/require"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func TestUDPCollector(t *testing.T) {
	agentAdrr := "127.0.0.1:0"
	mock, err := StartMockAgent(agentAdrr)
	require.NoError(t, err)
	defer func() {
		err := mock.Close()
		require.NoError(t, err)
	}()
	collector, err := NewUDPCollector(mock.Addr(), 200, "test", nil)
	require.NoError(t, err)

	span := &jaeger.Span{
		TraceIdLow:    monkit.NewId(),
		SpanId:        monkit.NewId(),
		OperationName: "test-udp-collector",
		StartTime:     time.Now().UnixNano() / 1000,
		Duration:      time.Second.Microseconds(),
	}
	collector.Collect(span)

	data := mock.GetBatches()
	for i := 0; i < 1000; i++ {
		time.Sleep(1 * time.Millisecond)
		data = mock.GetBatches()
		if len(data) > 0 {
			break
		}
	}
	require.Len(t, data, 1)
	require.Len(t, data[0].GetSpans(), 1)
	receivedSpan := data[0].GetSpans()[0]
	require.Equal(t, span.GetOperationName(), receivedSpan.OperationName)
	require.Equal(t, span.GetTraceIdLow(), receivedSpan.GetTraceIdLow())
	require.Equal(t, span.GetSpanId(), receivedSpan.GetSpanId())
}

func TestUDPCollector_HugeSpan(t *testing.T) {
	agentAdrr := "127.0.0.1:0"
	mock, err := StartMockAgent(agentAdrr)
	require.NoError(t, err)
	defer func() {
		err := mock.Close()
		require.NoError(t, err)
	}()
	collector, err := NewUDPCollector(mock.Addr(), 50, "test", nil)
	require.NoError(t, err)

	span := &jaeger.Span{
		TraceIdLow:    monkit.NewId(),
		SpanId:        monkit.NewId(),
		OperationName: "test-udp-collector",
		StartTime:     time.Now().UnixNano() / 1000,
		Duration:      time.Second.Microseconds(),
	}

	err = collector.Send(span)
	require.Error(t, err)

	data := mock.GetBatches()
	require.Empty(t, data)
}
