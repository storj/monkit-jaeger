// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"github.com/spacemonkeygo/monkit/v3"
)

// remoteTrace is a structure representing an incoming RPC trace.
type remoteTrace struct {
	traceId  *int64
	spanId   *int64
	parentId *int64
	sampled  *bool
}

// RemoteTraceHandler returns a new trace and its root span id based on remote trace information.
func RemoteTraceHandler(traceId *int64, parentId *int64) (trace *monkit.Trace, spanId int64) {
	if traceId == nil || parentId == nil {
		return nil, 0
	}

	rem := remoteTrace{
		traceId:  traceId,
		parentId: parentId,
	}

	return rem.trace()
}

// Trace returns a trace and a span id based on the remote trace.
func (rem remoteTrace) trace() (trace *monkit.Trace, spanId int64) {
	if rem.traceId != nil {
		trace = monkit.NewTrace(*rem.traceId)
	}

	if rem.spanId != nil {
		spanId = *rem.spanId
	} else {
		spanId = monkit.NewId()
	}

	if rem.parentId != nil {
		trace.Set(remoteParentKey, *rem.parentId)
	}

	if rem.sampled != nil {
		trace.Set(sampleKey, *rem.sampled)
	}

	return trace, spanId
}
