// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"fmt"
	"time"

	"github.com/spacemonkeygo/monkit/v3"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

type traceKey int

const (
	sampleKey       traceKey = 0
	flagsKey        traceKey = 1
	remoteParentKey traceKey = 2
)

// Options represents the configuration for the register.
type Options struct {
	Fraction float64         // The Fraction of traces to observe.
	Debug    bool            // Whether to set the debug flag on new traces.
	Process  *jaeger.Process // What set as the local zipkin.Endpoint

	collector TraceCollector
}

// RegisterJaeger configures the given Registry reg to send the Spans from some
// portion of all new Traces to the given TraceCollector.
func RegisterJaeger(reg *monkit.Registry, collector TraceCollector,
	opts Options) {
	opts.collector = collector
	reg.ObserveTraces(func(t *monkit.Trace) {
		sampled, exists := t.Get(sampleKey).(bool)
		if !exists {
			sampled = rng.Float64() < opts.Fraction
			t.Set(sampleKey, sampled)
		}
		if sampled {
			flags, ok := t.Get(flagsKey).(int64)
			if !ok {
				flags = 0
			}
			if opts.Debug {
				flags = flags | 1
			}
			t.Set(flagsKey, flags)
			t.ObserveSpans(spanFinishObserverFunc(opts.observeSpan))
		}
	})
}

type spanFinishObserverFunc func(s *monkit.Span, err error, panicked bool,
	finish time.Time)

func (f spanFinishObserverFunc) Start(*monkit.Span) {}
func (f spanFinishObserverFunc) Finish(s *monkit.Span, err error,
	panicked bool, finish time.Time) {
	f(s, err, panicked, finish)
}

func getParentId(s *monkit.Span) (parent_id *int64, server bool) {
	parent := s.Parent()
	if parent != nil {
		parent_id := parent.Id()
		return &parent_id, false
	}
	if parent_id, ok := s.Trace().Get(remoteParentKey).(int64); ok {
		return &parent_id, true
	}
	return nil, false
}

func (opts Options) observeSpan(s *monkit.Span, err error, panicked bool,
	finish time.Time) {
	parent_id, _ := getParentId(s)
	startTime := s.Start().UnixNano() / 1000

	js := &jaeger.Span{
		TraceIdLow:    s.Trace().Id(),
		TraceIdHigh:   0,
		OperationName: s.Func().FullName(),
		SpanId:        s.Id(),
		StartTime:     startTime,
		Duration:      s.Duration().Microseconds(),
	}
	if parent_id != nil {
		js.ParentSpanId = *parent_id
	}

	tags := make([]*jaeger.Tag, 0, len(s.Annotations())+len(s.Args()))
	for _, annotation := range s.Annotations() {
		annotation := annotation
		tags = append(tags, &jaeger.Tag{
			Key:   annotation.Name,
			VType: jaeger.TagType_STRING,
			VStr:  &annotation.Value,
		})
	}

	for arg_idx, arg := range s.Args() {
		arg := arg
		tags = append(tags, &jaeger.Tag{
			Key:   fmt.Sprintf("arg_%d", arg_idx),
			VType: jaeger.TagType_STRING,
			VStr:  &arg,
		})
	}

	js.Tags = tags

	opts.collector.Collect(js)
}
