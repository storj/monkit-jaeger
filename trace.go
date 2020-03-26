// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"github.com/spacemonkeygo/monkit/v3"
)

// remoteTrace is a structure representing an incoming RPC trace.
type remoteTrace struct {
	traceID  *int64
	spanID   *int64
	parentID *int64
	sampled  *bool
}

// RemoteTraceHandler returns a new trace and its root span id based on remote trace information.
func RemoteTraceHandler(traceID *int64, parentID *int64) (trace *monkit.Trace, spanID int64) {
	if traceID == nil || parentID == nil {
		return nil, 0
	}

	rem := remoteTrace{
		traceID:  traceID,
		parentID: parentID,
	}

	return rem.trace()
}

// Trace returns a trace and a span id based on the remote trace.
func (rem remoteTrace) trace() (trace *monkit.Trace, spanID int64) {
	if rem.traceID != nil {
		trace = monkit.NewTrace(*rem.traceID)
	}

	if rem.spanID != nil {
		spanID = *rem.spanID
	} else {
		spanID = monkit.NewId()
	}

	if rem.parentID != nil {
		trace.Set(remoteParentKey, *rem.parentID)
	}

	if rem.sampled != nil {
		trace.Set(sampleKey, *rem.sampled)
	}

	return trace, spanID
}
