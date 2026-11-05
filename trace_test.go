// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"testing"
)

func TestRemoteTraceHandler(t *testing.T) {
	t.Run("with trace host", func(t *testing.T) {
		trace, parentID := RemoteTraceHandler(map[string]string{
			ParentID:  "2",
			TraceID:   "1",
			Sampled:   "true",
			TraceHost: "collector.storj.tools:5775",
		})
		if trace == nil {
			t.Fatal("expected trace")
		}
		if parentID != 2 || trace.Id() != 1 {
			t.Fatalf("unexpected ids: parent=%d trace=%d", parentID, trace.Id())
		}
		if sampled, _ := trace.Get(Sampled).(bool); !sampled {
			t.Fatal("expected sampled trace")
		}
		if host, _ := trace.Get(TraceHost).(string); host != "collector.storj.tools:5775" {
			t.Fatalf("unexpected trace host: %q", host)
		}
	})

	t.Run("without trace host", func(t *testing.T) {
		trace, _ := RemoteTraceHandler(map[string]string{
			ParentID: "2",
			TraceID:  "1",
			Sampled:  "false",
		})
		if trace == nil {
			t.Fatal("expected trace")
		}
		if trace.Get(TraceHost) != nil {
			t.Fatal("unexpected trace host")
		}
	})

	t.Run("invalid", func(t *testing.T) {
		trace, parentID := RemoteTraceHandler(map[string]string{
			ParentID:  "x",
			TraceHost: "collector.storj.tools:5775",
		})
		if trace != nil || parentID != 0 {
			t.Fatal("expected no trace")
		}
	})
}
