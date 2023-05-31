package jaeger

import (
	"context"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// Transport defines how the span batches are sent.
type Transport interface {
	// Send sends out the Jaeger spans.
	Send(ctx context.Context, batch *jaeger.Batch) error

	// Close closes the transport.
	Close()
}
