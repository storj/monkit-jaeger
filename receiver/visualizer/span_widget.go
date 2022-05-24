// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
)

type SpanWidget struct {
	Size        image.Point
	SpanText    string
	TooltipText string
	Color       color.NRGBA

	tooltip    bool
	tooltipPos f32.Point
}

func (sw *SpanWidget) Layout(gtx layout.Context) layout.Dimensions {
	if sw.TooltipText != "" {
		for _, ev := range gtx.Events(sw) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
			case pointer.Enter:
				sw.tooltip = true

			case pointer.Leave, pointer.Cancel:
				sw.tooltip = false

			case pointer.Move:
				sw.tooltipPos = e.Position
			}
		}

		area := clip.Rect(image.Rectangle{Max: sw.Size}).Push(gtx.Ops)
		pointer.InputOp{
			Tag:   sw,
			Types: pointer.Enter | pointer.Leave | pointer.Move,
			Grab:  false,
		}.Add(gtx.Ops)
		area.Pop()
	}

	dims := ColorBoxWidget{
		Size:       sw.Size,
		Background: sw.Color,
		Text:       sw.SpanText,
	}.Layout(gtx)

	if sw.tooltip {
		macro := op.Record(gtx.Ops)
		pos := sw.tooltipPos.Add(f32.Pt(5, 5))

		offset := op.Offset(pos.Round()).Push(gtx.Ops)

		gtx := gtx
		gtx.Constraints.Min = image.Pt(0, 0)

		ColorBoxWidget{
			Background: white,
			Text:       sw.TooltipText,
		}.Layout(gtx)

		offset.Pop()

		op.Defer(gtx.Ops, macro.Stop())
	}

	return dims
}
