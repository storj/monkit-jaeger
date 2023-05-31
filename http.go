// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"context"
	"io"
	"net/http"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// HTTPTransport sends Jaeger spans via HTTP.
type HTTPTransport struct {
	log  *zap.Logger
	addr string
}

var _ Transport = &HTTPTransport{}

// OpenHTTPTransport creates a new HTTP transport.
func OpenHTTPTransport(ctx context.Context, log *zap.Logger, agentAddr string) (*HTTPTransport, error) {
	return &HTTPTransport{
		log:  log,
		addr: agentAddr,
	}, nil

}

// Send sends out the Jaeger spans.
func (u *HTTPTransport) Send(ctx context.Context, batch *jaeger.Batch) error {
	t := thrift.NewTMemoryBuffer()
	p := thrift.NewTBinaryProtocolTransport(t)
	err := batch.Write(p)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.addr, nil)
	if err != nil {
		return errs.Wrap(err)
	}
	req.Header.Add("Content-Type", "application/x-thrift")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return errs.New("Error on posting data to jaeger. HTTP %s: %s", resp.Status, string(raw))
	}
	return nil
}

// Close closes the transport.
func (u *HTTPTransport) Close() {

}
