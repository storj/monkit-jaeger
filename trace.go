// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"strconv"

	"github.com/spacemonkeygo/monkit/v3"

	"storj.io/common/rpc/rpctracing"
)

// remoteTrace is a structure representing an incoming RPC trace.
type remoteTrace struct {
	traceID  *int64
	spanID   *int64
	parentID *int64
	sampled  *bool
}

// RemoteTraceHandler returns a new trace and its root span id based on remote trace information.
func RemoteTraceHandler(remoteInfo map[string]string) (trace *monkit.Trace, spanID int64) {
	if len(remoteInfo) == 0 {
		return nil, 0
	}
	parentID, err := strconv.ParseInt(remoteInfo[rpctracing.ParentID], 10, 64)
	if err != nil {
		return nil, 0
	}

	traceID, err := strconv.ParseInt(remoteInfo[rpctracing.TraceID], 10, 64)
	if err != nil {
		return nil, 0
	}

	sampled, err := strconv.ParseBool(remoteInfo[rpctracing.Sampled])
	if err != nil {
		return nil, 0
	}

	rem := remoteTrace{
		traceID:  &traceID,
		parentID: &parentID,
		sampled:  &sampled,
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
		trace.Set(rpctracing.ParentID, *rem.parentID)
	}

	if rem.sampled != nil {
		trace.Set(rpctracing.Sampled, *rem.sampled)
	}

	return trace, spanID
}
