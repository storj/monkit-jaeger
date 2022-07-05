// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type TimelineWidget struct {
	GapSize int
	GapSkip int
	Height  int

	StartTime int
	EndTime   int
}

func (t TimelineWidget) Layout(gtx C) D {
	td := float64(t.EndTime - t.StartTime)

	for x := t.GapSkip; x < gtx.Constraints.Max.X; x += t.GapSize {
		var p clip.Path
		p.Begin(gtx.Ops)
		p.MoveTo(f32.Pt(float32(x), 0))
		p.LineTo(f32.Pt(float32(x), float32(t.Height)))
		paint.FillShape(gtx.Ops, black, clip.Stroke{
			Path:  p.End(),
			Width: 2,
		}.Op())

		xd := float64(x) / float64(gtx.Constraints.Max.X)
		d := truncateDuration(time.Duration(td*xd - float64(t.StartTime)))

		label := material.Label(th, unit.Sp(10), d.String())
		label.Color = black

		popper := op.Offset(image.Pt(x+2, t.Height-gtx.Metric.Sp(unit.Sp(10)))).Push(gtx.Ops)
		label.Layout(gtx)
		popper.Pop()
	}
	return layout.Dimensions{
		Size: image.Pt(gtx.Constraints.Max.X, t.Height),
	}
}

func truncateDuration(d time.Duration) time.Duration {
	ad := d
	if ad < 0 {
		ad *= -1
	}
	switch {
	case ad < time.Microsecond:
		return d
	case ad < time.Millisecond:
		return d.Truncate(time.Microsecond)
	default:
		return d.Truncate(time.Millisecond)
	}
}
