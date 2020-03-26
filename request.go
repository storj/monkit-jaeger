// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"strconv"

	"github.com/spacemonkeygo/monkit/v3"
)

// Request is a structure representing an incoming RPC request. Every field
// is optional.
type Request struct {
	TraceID  *int64
	SpanID   *int64
	ParentID *int64
	Sampled  *bool
	Flags    *int64
}

// HeaderGetter is an interface that http.Header matches for RequestFromHeader
type HeaderGetter interface {
	Get(string) string
}

// HeaderSetter is an interface that http.Header matches for Request.SetHeader
type HeaderSetter interface {
	Set(string, string)
}

// RequestFromHeader will create a Request object given an http.Header or
// anything that matches the HeaderGetter interface.
func RequestFromHeader(header HeaderGetter) (rv Request) {
	traceID, err := fromHeader(header.Get("X-B3-TraceId"))
	if err == nil {
		rv.TraceID = &traceID
	}
	spanID, err := fromHeader(header.Get("X-B3-SpanId"))
	if err == nil {
		rv.SpanID = &spanID
	}
	parentID, err := fromHeader(header.Get("X-B3-ParentSpanId"))
	if err == nil {
		rv.ParentID = &parentID
	}
	sampled, err := strconv.ParseBool(header.Get("X-B3-Sampled"))
	if err == nil {
		rv.Sampled = &sampled
	}
	flags, err := fromHeader(header.Get("X-B3-Flags"))
	if err == nil {
		rv.Flags = &flags
	}
	return rv
}

func ref(v int64) *int64 {
	return &v
}

// RequestFromSpan consrtructs a new request from an span.
func RequestFromSpan(s *monkit.Span) Request {
	trace := s.Trace()

	sampled, ok := trace.Get(sampleKey).(bool)
	if !ok {
		sampled = false
	}

	if !sampled {
		return Request{Sampled: &sampled}
	}
	flags, ok := trace.Get(flagsKey).(int64)
	if !ok {
		flags = 0
	}
	parentID, _ := getParentID(s)
	return Request{
		TraceID:  ref(trace.Id()),
		SpanID:   ref(s.Id()),
		Sampled:  &sampled,
		Flags:    &flags,
		ParentID: parentID,
	}
}

// SetHeader will take a Request and fill out an http.Header, or anything that
// matches the HeaderSetter interface.
func (r Request) SetHeader(header HeaderSetter) {
	if r.TraceID != nil {
		header.Set("X-B3-TraceId", toHeader(*r.TraceID))
	}
	if r.SpanID != nil {
		header.Set("X-B3-SpanId", toHeader(*r.SpanID))
	}
	if r.ParentID != nil {
		header.Set("X-B3-ParentSpanId", toHeader(*r.ParentID))
	}
	if r.Sampled != nil {
		header.Set("X-B3-Sampled", strconv.FormatBool(*r.Sampled))
	}
	if r.Flags != nil {
		header.Set("X-B3-Flags", toHeader(*r.Flags))
	}
}

// Trace returns a new trace and spanID based on the request.
func (r Request) Trace() (trace *monkit.Trace, spanID int64) {
	if r.TraceID != nil {
		trace = monkit.NewTrace(*r.TraceID)
	} else {
		trace = monkit.NewTrace(monkit.NewId())
	}
	if r.SpanID != nil {
		spanID = *r.SpanID
	} else {
		spanID = monkit.NewId()
	}

	if r.ParentID != nil {
		trace.Set(remoteParentKey, *r.ParentID)
	}
	if r.Sampled != nil {
		trace.Set(sampleKey, *r.Sampled)
	}
	if r.Flags != nil {
		trace.Set(flagsKey, *r.Flags)
	}
	return trace, spanID
}

// fromHeader reads a signed int64 that has been formatted as a hex uint64
func fromHeader(s string) (int64, error) {
	v, err := strconv.ParseUint(s, 16, 64)
	return int64(v), err
}

// toHeader writes a signed int64 as hex uint64
func toHeader(i int64) string {
	return strconv.FormatUint(uint64(i), 16)
}
