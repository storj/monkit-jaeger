// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"
)

type SpanViewWidget struct {
	roots   map[int64]struct{}
	ospans  []SpanInfo
	spans   []SpanInfo
	minTime int64
	maxTime int64

	widgets map[int64]*SpanWidget
	layout  map[int64]*spanLayoutInfo
	rows    int

	view   ViewportWidget
	filter EditboxWidget
	vbar   VBarWidget
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
	layout, rows, width := computeLayoutInformation(spans, nil)

	size := gtx.Constraints.Max.X
	scale := float32(width) / float32(size)
	min, max := getTimes(spans)

	return &SpanViewWidget{
		ospans:  spans,
		spans:   spans,
		minTime: min,
		maxTime: max,

		widgets: make(map[int64]*SpanWidget),
		layout:  layout,
		rows:    rows,

		view:   NewViewportWidget(scale),
		filter: NewEditboxWidget("Filter", true),
	}
}

func (s *SpanViewWidget) Layout(gtx C) D {
	if s == nil {
		return D{}
	}

	for _, ev := range s.filter.Events() {
		if ev, ok := ev.(widget.SubmitEvent); ok {
			fs := parseFilterArgs(ev.Text)
			s.spans = applyFilters(s.ospans, fs)

			layout, rows, width := computeLayoutInformation(s.ospans, fs)
			size := gtx.Constraints.Max.X
			scale := float32(width) / float32(size)

			s.widgets = make(map[int64]*SpanWidget)
			s.layout = layout
			s.rows = rows
			s.view = NewViewportWidget(scale)
			s.minTime, s.maxTime = getTimes(s.spans)

			fmt.Println("filtered", len(s.ospans), "into", len(s.spans))
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(s.filter.Layout),
		layout.Rigid(func(gtx C) D {
			defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

			s.view.Add(gtx.Ops)
			box, scale := s.view.View(gtx)
			sbox := box
			sbox.Min.X = int(float32(sbox.Min.X) / scale)
			sbox.Max.X = int(float32(sbox.Min.X) / scale)

			// maintain t = 0 at x = 0
			// we have t = maxTime - minTime at x = (maxTime - minTime) / scale

			// time start is minTime at x = 0
			// time end is maxTime at x =

			s.vbar.Add(gtx.Ops)
			s.vbar.Move(gtx.Queue)
			s.vbar.Layout(gtx)

			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(TimelineWidget{
					GapSize: 100,
					GapSkip: sbox.Min.X % 100,
					Height:  50,

					StartTime: box.Min.X,
					EndTime:   box.Max.X,
				}.Layout),
				layout.Rigid(func(gtx C) D {
					defer clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops).Pop()

					return layout.Inset{Top: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
						defer op.Offset(sbox.Min).Push(gtx.Ops).Pop()

						window := image.Rect(0, 0, gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
						const height = 20

						for _, si := range s.spans {
							sli := s.layout[si.Id]

							width := float32(si.Finish-si.Start) / scale
							startx := float32(si.Start-s.spans[0].Start) / scale
							endx := startx + width
							starty := float32(sli.Row*(height-2) - 2*height + 5)
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

							off.Pop()
						}

						return layout.Dimensions{Size: gtx.Constraints.Max}
					})
				}),
			)
		}),
	)
}

func parseFilterArgs(args string) (fs []func(SpanInfo) bool) {
	for _, arg := range strings.Fields(args) {
		fs = append(fs, parseFilterArg(arg))
	}
	return fs
}

func parseFilterArg(arg string) func(SpanInfo) bool {
	switch {
	case strings.HasPrefix(arg, ">="):
		dur, err := time.ParseDuration(arg[2:])
		if err == nil {
			return func(si SpanInfo) bool { return time.Duration(si.Finish-si.Start) >= dur }
		}
	case strings.HasPrefix(arg, ">"):
		dur, err := time.ParseDuration(arg[1:])
		if err == nil {
			return func(si SpanInfo) bool { return time.Duration(si.Finish-si.Start) > dur }
		}

	case strings.HasPrefix(arg, "<="):
		dur, err := time.ParseDuration(arg[2:])
		if err == nil {
			return func(si SpanInfo) bool { return time.Duration(si.Finish-si.Start) <= dur }
		}

	case strings.HasPrefix(arg, "<"):
		dur, err := time.ParseDuration(arg[1:])
		if err == nil {
			return func(si SpanInfo) bool { return time.Duration(si.Finish-si.Start) < dur }
		}

	default:
		return func(si SpanInfo) bool { return strings.Contains(si.Summary, arg) }
	}
	return func(SpanInfo) bool { return true }
}

func applyFilters(si []SpanInfo, fs []func(SpanInfo) bool) (o []SpanInfo) {
spans:
	for _, s := range si {
		for _, f := range fs {
			if !f(s) {
				continue spans
			}
		}
		o = append(o, s)
	}
	return o
}

func getTimes(si []SpanInfo) (min, max int64) {
	for i, s := range si {
		if i == 0 || s.Start < min {
			min = s.Start
		}
		if i == 0 || s.Finish > max {
			max = s.Finish
		}
	}
	return min, max
}
