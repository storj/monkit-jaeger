// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ColorBoxWidget struct {
	Size       image.Point
	Background color.NRGBA
	Text       string
	Alignment  text.Alignment
}

func (cb ColorBoxWidget) Layout(gtx layout.Context) layout.Dimensions {
	label := material.Label(th, unit.Sp(10), cb.Text)
	label.Alignment = cb.Alignment
	label.Color = black

	macro := op.Record(gtx.Ops)
	if cb.Size != (image.Point{}) {
		gtx.Constraints.Max = cb.Size
	}

	dims := layout.UniformInset(unit.Dp(3)).Layout(gtx, label.Layout)
	ops := macro.Stop()

	if cb.Size == (image.Point{}) {
		cb.Size = dims.Size
	}

	return widget.Border{
		Color: black,
		Width: unit.Dp(2),
	}.Layout(gtx, func(gtx C) D {
		defer clip.Rect{Max: cb.Size}.Push(gtx.Ops).Pop()
		paint.ColorOp{Color: cb.Background}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		ops.Add(gtx.Ops)
		return layout.Dimensions{Size: cb.Size}
	})
}
