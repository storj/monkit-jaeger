// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

package zipkin

import (
	"gopkg.in/spacemonkeygo/monkit-zipkin.v2/gen-go/zipkin"
)

// TraceCollector is an interface dealing with completed Spans on a
// SpanManager. See RegisterZipkin.
type TraceCollector interface {
	// Collect gets called with a Span whenever a Span is completed on a
	// SpanManager.
	Collect(span *zipkin.Span)
}
