// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeebo/errs"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func Collect(ctx context.Context, addr string, id int64, cb func(*jaeger.Span) error) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%d", addr, id), nil)
	if err != nil {
		return errs.Wrap(err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Wrap(err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	for {
		var span jaeger.Span
		if err := dec.Decode(&span); err != nil {
			return errs.Wrap(err)
		}
		if err := cb(&span); err != nil {
			return errs.Wrap(err)
		}
		if err := ctx.Err(); err != nil {
			return errs.Wrap(err)
		}
	}
}
