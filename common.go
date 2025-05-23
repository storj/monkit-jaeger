// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import "github.com/spacemonkeygo/monkit/v3"

var (
	mon = monkit.Package()
)

const (
	// the following values cannot change - they are used across service
	// boundaries in network protocols.

	// TraceID is the key we use to store trace id value into context.
	TraceID = "trace-id"
	// ParentID is the key we use to store parent's span id value into context.
	ParentID = "parent-id"
	// Sampled is the key we use to store sampled flag into context.
	Sampled = "sampled"
	// TraceHost is the host to send the traces to. If unprovided, the default
	// is used.
	TraceHost = "trace-host"
)
