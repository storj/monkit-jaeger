// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"strconv"

	"github.com/spacemonkeygo/monkit/v3"

	"storj.io/common/rpc/rpctracing"
)

// RemoteTraceHandler returns a new trace and its root span id based on remote trace information.
func RemoteTraceHandler(remoteInfo map[string]string) (trace *monkit.Trace, parentID int64) {
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

	trace = monkit.NewTrace(traceID)
	trace.Set(rpctracing.Sampled, sampled)

	return trace, parentID
}
