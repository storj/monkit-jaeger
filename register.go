// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"fmt"
	"log"
	"time"

	"github.com/spacemonkeygo/monkit/v3"

	"storj.io/common/rpc/rpctracing"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// Options represents the configuration for the register.
type Options struct {
	Fraction float64 // The Fraction of traces to observe.

	collector TraceCollector
}

// RegisterJaeger configures the given Registry reg to send the Spans from some
// portion of all new Traces to the given TraceCollector.
// it returns the unregister function.
func RegisterJaeger(reg *monkit.Registry, collector TraceCollector,
	opts Options) func() {
	opts.collector = collector

	return reg.ObserveTraces(func(t *monkit.Trace) {
		sampled, exists := t.Get(rpctracing.Sampled).(bool)
		if !exists {
			sampled = rng.Float64() < opts.Fraction
			t.Set(rpctracing.Sampled, sampled)
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

func getParentID(s *monkit.Span) *int64 {
	parent := s.Parent()
	if parent != nil {
		parentID := parent.Id()
		return &parentID
	}
	if remoteParentID, ok := s.Trace().Get(rpctracing.ParentID).(int64); ok {
		return &remoteParentID
	}

	return nil
}

func (opts Options) observeSpan(s *monkit.Span, err error, panicked bool,
	finish time.Time) {
	startTime := s.Start().UnixNano() / 1000
	duration := finish.Sub(s.Start())

	js := &jaeger.Span{
		TraceIdLow:    s.Trace().Id(),
		TraceIdHigh:   0,
		OperationName: s.Func().FullName(),
		SpanId:        s.Id(),
		StartTime:     startTime,
		// this is how jaeger client code calculates duration to send to jaeger agent
		// reference: https://github.com/jaegertracing/jaeger-client-go/blob/master/jaeger_thrift_span.go#L32
		Duration: duration.Nanoseconds() / int64(time.Microsecond),
	}

	parentID := getParentID(s)
	if parentID != nil {
		js.ParentSpanId = *parentID
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
			log.Printf("failed to convert annotation to jaeger format: %v", err)
		}
		tags = append(tags, jaegerTag)
	}

	for idx, arg := range s.Args() {
		arg := arg
		tag := Tag{
			Key:   fmt.Sprintf("arg_%d", idx),
			Value: arg,
		}
		jaegerTag, err := tag.BuildJaegerThrift()
		if err != nil {
			log.Printf("failed to convert args to jaeger format: %v", err)
		}
		tags = append(tags, jaegerTag)
	}

	// only attach trace metadata to the root span
	if s.Parent() == nil {
		for k, v := range s.Trace().GetAll() {
			key, ok := k.(string)
			if !ok {
				continue
			}

			if key == rpctracing.ParentID ||
				key == rpctracing.Sampled ||
				key == rpctracing.TraceID {
				continue
			}
			tag := Tag{
				Key:   key,
				Value: v,
			}

			jaegerTag, err := tag.BuildJaegerThrift()
			if err != nil {
				log.Printf("failed to convert tag to jaeger format: %v", err)
			}
			tags = append(tags, jaegerTag)
		}
	}

	js.Tags = tags

	opts.collector.Collect(js)
}
