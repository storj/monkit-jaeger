// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"testing"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func TestCollect(t *testing.T) {
	ctx := context.Background()
	t.Log(Collect(ctx, "agent.tracing.datasci.storj.io:9911", 12341234, func(sp *jaeger.Span) error {
		fmt.Println(sp)
		return nil
	}))
}
