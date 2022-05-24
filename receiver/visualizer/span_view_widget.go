// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
)

type frameInfo struct {
	when   time.Time
	scale  float64
	offset f32.Point
}

type SpanViewWidget struct {
	spans []SpanInfo

	widgets map[int64]*SpanWidget
	layout  map[int64]*spanLayoutInfo
	rows    int

	view      ViewportWidget
	threshold EditboxWidget
}

type SpanInfo struct {
	Summary string

	Id     int64
	Parent int64

	Start  int64
	Finish int64

	Status SpanStatus
}

type SpanStatus int

const (
	SpanStatus_OK SpanStatus = 1 + iota
	SpanStatus_Error
	SpanStatus_Panic
	SpanStatus_Cancel
)

var statusColors = map[SpanStatus]color.NRGBA{
	SpanStatus_OK:     {128, 128, 255, 255},
	SpanStatus_Error:  {255, 144, 0, 255},
	SpanStatus_Panic:  {255, 0, 0, 255},
	SpanStatus_Cancel: {255, 255, 0, 255},
}

func NewSpanViewWidget(gtx C, spans ...SpanInfo) *SpanViewWidget {
	layout, rows, width := computeLayoutInformation(spans)

	size := gtx.Constraints.Max.X
	scale := float32(width) / float32(size)

	return &SpanViewWidget{
		spans: spans,

		widgets: make(map[int64]*SpanWidget),
		layout:  layout,
		rows:    rows,

		view:      NewViewportWidget(scale),
		threshold: NewEditboxWidget("Filter", true),
	}
}

func (s *SpanViewWidget) Layout(gtx C) D {
	if s == nil {
		return D{}
	}

	for _, ev := range s.threshold.Events() {
		if _, ok := ev.(widget.SubmitEvent); ok {
			fmt.Printf("%#v\n", ev)
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(s.threshold.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Dimensions{Size: image.Pt(0, 10)}
		}),

		layout.Rigid(func(gtx C) D {
			gtx.Constraints.Min = gtx.Constraints.Max

			return widget.Border{
				Color: black,
				Width: unit.Dp(1),
			}.Layout(gtx, func(gtx C) D {
				defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

				return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
					s.view.Add(gtx.Ops)

					box, scale := s.view.View(gtx)
					sbox := box
					sbox.Min.X = int(float32(sbox.Min.X) / scale)
					sbox.Max.X = int(float32(sbox.Min.X) / scale)

					defer op.Offset(sbox.Min).Push(gtx.Ops).Pop()

					window := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
					drew := 0
					const height = 20

					for _, si := range s.spans {
						sli := s.layout[si.Id]

						width := float32(si.Finish-si.Start) / scale
						startx := float32(si.Start-s.spans[0].Start) / scale
						endx := startx + width
						starty := float32(sli.Row*height - 60)
						endy := starty + height
						offset := image.Pt(int(startx), int(starty))

						span := image.Rect(
							int(startx), int(starty),
							int(endx), int(endy),
						).Add(image.Pt(
							int(sbox.Min.X),
							int(sbox.Min.Y)),
						)

						if width < 1 || window.Intersect(span).Empty() {
							continue
						}

						w, ok := s.widgets[si.Id]
						if !ok {
							w = &SpanWidget{
								SpanText: si.Summary,
								Color:    statusColors[si.Status],
								TooltipText: fmt.Sprintf("%s\nDuration:%v",
									si.Summary,
									time.Duration(si.Finish-si.Start),
								),
							}
							s.widgets[si.Id] = w
						}
						w.Size = image.Pt(int(width), height)

						off := op.Offset(offset).Push(gtx.Ops)

						w.Layout(gtx)
						drew++

						// ColorBoxWidget{
						// 	Size:       image.Pt(int(width), 32),
						// 	Background: spanOk,
						// 	Text:       si.Summary,
						// }.Layout(gtx)

						off.Pop()
					}

					return layout.Dimensions{Size: gtx.Constraints.Max}
				})
			})
		}),
	)
}
