// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"flag"
	"image/color"
	"log"
	"os"
	"sync"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
	"github.com/zeebo/errs"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

func lerpF32(s, e float32, t float64) float32 {
	return s*float32(1-t) + e*float32(t)
}

var th = material.NewTheme(gofont.Collection())

type (
	D = layout.Dimensions
	C = layout.Context
)

var (
	red       = color.NRGBA{R: 0xC0, G: 0x40, B: 0x40, A: 0xFF}
	black     = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	gray      = color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF}
	white     = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	spanOk    = color.NRGBA{R: 0x80, G: 0x80, B: 0xFF, A: 0xFF}
	selected  = color.NRGBA{R: 0x80, G: 0x80, B: 0xFF, A: 0xFF}
	lightGray = color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xFF}
	darkGray  = color.NRGBA{R: 0xB0, G: 0xB0, B: 0xB0, A: 0xFF}
)

func main() {
	go func() {
		w := app.NewWindow()
		err := run(w)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

var (
	flagCollector = flag.String("collector", "agent.tracing.datasci.storj.io:9911", "the collector address to connect to")
	flagTrace     = flag.Int64("trace-id", 12341234, "the trace id to watch for")
	static        = flag.String("static", "", "load an svg for static traces")
)

func run(w *app.Window) error {
	flag.Parse()
	var ops op.Ops

	mv := NewMainViewWidget()

	var bufMu sync.Mutex
	var buffered []SpanInfo

	if *static != "" {
		infos, err := loadStatic(*static)
		if err != nil {
			return errs.Wrap(err)
		}
		buffered = infos
	} else {
		go func() {
			panic(Collect(context.Background(), *flagCollector, *flagTrace, func(sp *jaeger.Span) error {
				bufMu.Lock()
				defer bufMu.Unlock()

				buffered = append(buffered, SpanInfo{
					Summary: sp.OperationName,
					Id:      sp.SpanId,
					Parent:  sp.ParentSpanId,
					Start:   sp.StartTime,
					Finish:  sp.StartTime + sp.Duration,
					Status:  SpanStatus_OK,
				})
				w.Invalidate()
				return nil
			}))
		}()
	}

	for {
		e := <-w.Events()
		switch e := e.(type) {
		case system.DestroyEvent:
			return e.Err

		case system.FrameEvent:
			gtx := layout.NewContext(&ops, e)

			bufMu.Lock()
			if buffered != nil {
				mv.AddSpanInfos(gtx, buffered...)
				buffered = nil
			}
			bufMu.Unlock()

			mv.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
