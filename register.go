// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/zeebo/errs"

	"storj.io/common/rpc/rpcstatus"
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

func (opts Options) observeSpan(s *monkit.Span, spanErr error, panicked bool,
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

	tags := make([]Tag, 0, len(s.Annotations()))

	for _, annotation := range s.Annotations() {
		annotation := annotation
		tags = append(tags, Tag{
			Key:   annotation.Name,
			Value: annotation.Value,
		})
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
			tags = append(tags, Tag{
				Key:   key,
				Value: v,
			})
		}
	}

	// in order to make sure we don't send error messages that contain private
	// user information to our jaeger instance, we only send errors that we know
	// is privacy clear.
	errMsg := filterErr(spanErr, panicked)
	if errMsg != nil {
		tags = append(tags, NewErrorTag())

		js.Logs = newJaegerLogs(finish, "error", errMsg.Error())
	}
	js.Tags = NewJaegerTags(tags)

	opts.collector.Collect(js)
}

func newJaegerLogs(t time.Time, key, msg string) []*jaeger.Log {
	// converts Go time.Time to a long representing time since epoch in microseconds,
	// which is used expected in the Jaeger spans encoded as Thrift.
	timestamp := t.UnixNano() / 1000

	return []*jaeger.Log{
		{
			Timestamp: timestamp,
			Fields: NewJaegerTags([]Tag{
				{
					Key:   key,
					Value: msg,
				},
			}),
		},
	}
}

// filterErr returns an error that only contains known error messages.
// the known errors are:
// 1. rpc status code, and include it if it exists.
// 2. check for io.EOF
// 3. check for context.Canceled
// 4. check for panicked
// 5. check for net.Error.
func filterErr(spanErr error, panicked bool) error {
	var filteredErr error
	if panicked {
		filteredErr = errs.Combine(filteredErr, errors.New("panicked"))
	}

	if spanErr == nil {
		return filteredErr
	}

	if code := rpcstatus.Code(spanErr); code != rpcstatus.Unknown {
		filteredErr = errs.Combine(filteredErr, rpcstatus.Error(code, ""))
	}

	if errors.Is(spanErr, io.EOF) {
		filteredErr = errs.Combine(filteredErr, io.EOF)
	}

	if errors.Is(spanErr, context.Canceled) {
		filteredErr = errs.Combine(filteredErr, context.Canceled)
	}

	var netErr net.Error
	if errors.As(spanErr, &netErr) {
		var err error
		if netErr.Timeout() {
			err = errs.Combine(err, errors.New("encountered a network timeout issue"))
		}
		if netErr.Temporary() {
			err = errs.Combine(err, errors.New("encountered a temporary network issue"))
		}

		if err == nil {
			err = errors.New("encountered an unknown network issue")
		}

		filteredErr = errs.Combine(filteredErr, err)
	}

	return filteredErr
}
