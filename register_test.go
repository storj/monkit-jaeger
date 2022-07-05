// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"testing"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/stretchr/testify/require"

	"storj.io/common/rpc/rpctracing"
	"storj.io/common/testcontext"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

type expected struct {
	operationName string
	hasParentID   bool
	tags          []*jaeger.Tag
}

func TestRegisterJaeger(t *testing.T) {
	ctx := testcontext.New(t)

	tagKey := "test-key"
	tagValue := "test"

	testcases := []struct {
		e expected
		f func(*monkit.Registry, expected)
	}{
		{
			e: expected{
				operationName: "test-register",
				hasParentID:   false,
				tags:          NewJaegerTags(nil),
			},
			f: func(r *monkit.Registry, e expected) {
				newTrace(r.Package(), e.operationName)
			},
		},
		{
			e: expected{
				operationName: "test-register-parent",
				hasParentID:   true,
				tags:          NewJaegerTags(nil),
			},
			f: func(r *monkit.Registry, e expected) {
				newTraceWithParent(ctx, r.Package(), e.operationName)
			},
		},
		{
			e: expected{
				operationName: "test-register-args",
				hasParentID:   false,
				tags:          NewJaegerTags([]Tag{}),
			},
			f: func(r *monkit.Registry, e expected) {
				newTraceWithArgs(r.Package(), e.operationName, tagValue)
			},
		},
		{
			e: expected{
				operationName: "test-register-trace-tags",
				hasParentID:   false,
				tags: NewJaegerTags([]Tag{
					{
						Key:   tagKey,
						Value: tagValue,
					},
				}),
			},
			f: func(r *monkit.Registry, e expected) {
				newTraceWithTags(r.Package(), e.operationName, map[string]string{
					tagKey: tagValue,
				})
			},
		},
	}

	for _, test := range testcases {
		test := test
		t.Run(test.e.operationName, func(t *testing.T) {
			withAgent(t, func(agent *MockAgent) {
				withCollector(ctx, t, agent.Addr(), 0, time.Nanosecond, func(collector *UDPCollector) {
					r := monkit.NewRegistry()
					RegisterJaeger(r, collector, Options{
						Fraction: 1,
					})

					test.f(r, test.e)

					batches := agent.WaitForBatches(time.Second)
					require.True(t, len(batches) > 0)

					spans := batches[0].GetSpans()
					require.True(t, len(spans) > 0)

					span := spans[0]
					require.Contains(t, span.GetOperationName(), test.e.operationName)
					require.Equal(t, test.e.hasParentID, span.GetParentSpanId() != 0)
					require.Equal(t, len(test.e.tags), len(span.GetTags()))
					for _, tag := range test.e.tags {
						actualTag, ok := findTag(tag.GetKey(), span)
						require.True(t, ok)
						require.Equal(t, tag.GetVType(), actualTag.GetVType())
						require.Equal(t, tag.GetVStr(), actualTag.GetVStr())
					}
				})
			})
		})
	}
}

func newTrace(mon *monkit.Scope, name string) {
	defer mon.TaskNamed(name)(nil)(nil)
}

func newTraceWithParent(ctx context.Context, mon *monkit.Scope, name string) {
	newTrace := monkit.NewTrace(monkit.NewId())
	newTrace.Set(rpctracing.ParentID, monkit.NewId())
	defer mon.FuncNamed(name).RemoteTrace(&ctx, monkit.NewId(), newTrace)(nil)
}

func newTraceWithArgs(mon *monkit.Scope, name string, tag string) {
	defer mon.TaskNamed(name)(nil, tag)(nil)
}

func newTraceWithTags(mon *monkit.Scope, name string, tag map[string]string) {
	ctx := context.Background()
	defer mon.TaskNamed(name)(&ctx)(nil)
	s := monkit.SpanFromCtx(ctx)
	for k, v := range tag {
		s.Trace().Set(k, v)
	}
}

func findTag(key string, s *jaeger.Span) (*jaeger.Tag, bool) {
	for _, t := range s.GetTags() {
		if t.Key == key {
			return t, true
		}
	}
	return nil, false
}
