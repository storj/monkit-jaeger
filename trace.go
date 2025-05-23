// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"strconv"

	"github.com/spacemonkeygo/monkit/v3"
)

// RemoteTraceHandler returns a new trace and its root span id based on remote trace information.
func RemoteTraceHandler(remoteInfo map[string]string) (trace *monkit.Trace, parentID int64) {
	parentID, err := strconv.ParseInt(remoteInfo[ParentID], 10, 64)
	if err != nil {
		return nil, 0
	}

	traceID, err := strconv.ParseInt(remoteInfo[TraceID], 10, 64)
	if err != nil {
		return nil, 0
	}

	sampled, err := strconv.ParseBool(remoteInfo[Sampled])
	if err != nil {
		return nil, 0
	}

	if traceHost, ok := remoteInfo[TraceHost]; ok {
		trace.Set(TraceHost, traceHost)
	}

	trace = monkit.NewTrace(traceID)
	trace.Set(Sampled, sampled)

	return trace, parentID
}
