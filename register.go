// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"fmt"
	"log"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

type traceKey int

const (
	sampleKey       traceKey = 0
	remoteParentKey traceKey = 2
)

type Options struct {
	Fraction float64 // The Fraction of traces to observe.

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

func getParentId(s *monkit.Span) *int64 {
	parent := s.Parent()
	if parent != nil {
		parent_id := parent.Id()

		return &parent_id
	}

	if remote_parent_id, ok := s.Trace().Get(remoteParentKey).(int64); ok {
		return &remote_parent_id
	}

	return nil
}

func (opts Options) observeSpan(s *monkit.Span, err error, panicked bool,
	finish time.Time) {
	parent_id := getParentId(s)
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
		tag := Tag{
			Key:   annotation.Name,
			Value: annotation.Value,
		}
		jaegerTag, err := tag.BuildJaegerThrift()
		if err != nil {
			log.Printf("failed to convert tag to jaeger format: %v", err)
		}
		tags = append(tags, jaegerTag)
	}

	for arg_idx, arg := range s.Args() {
		arg := arg
		tag := Tag{
			Key:   fmt.Sprintf("arg_%d", arg_idx),
			Value: arg,
		}
		jaegerTag, err := tag.BuildJaegerThrift()
		if err != nil {
			log.Printf("failed to convert args to jaeger format: %v", err)
		}
		tags = append(tags, jaegerTag)
	}

	js.Tags = tags

	opts.collector.Collect(js)
}
