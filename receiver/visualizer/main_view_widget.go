// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"sort"

	"gioui.org/op/paint"
)

type MainViewWidget struct {
	split    SplitWidget
	spans    *SpanViewWidget
	trace    TraceSelectorWidget
	infos    map[int][]SpanInfo
	selected int
}

func NewMainViewWidget() MainViewWidget {
	return MainViewWidget{
		split:    NewSplitWidget(-0.5),
		trace:    NewTraceSelectorWidget(),
		selected: -1,
		infos:    make(map[int][]SpanInfo),
	}
}

func (mv *MainViewWidget) Layout(gtx C) D {
	paint.ColorOp{Color: darkGray}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	dims := mv.split.Layout(gtx)

	mv.split.Left(gtx, mv.trace.Layout)

	if mv.trace.selected != mv.selected {
		mv.selected = mv.trace.selected
		mv.spans = mv.createSpanViewWidget(gtx)
	}

	mv.split.Right(gtx, mv.spans.Layout)

	return dims
}

func (mv *MainViewWidget) AddTrace(label string) int {
	return mv.trace.AddItem(label)
}

func (mv *MainViewWidget) AddSpanInfo(id int, si SpanInfo) {
	// TODO: insertion sort. or something. this is bad.

	mv.infos[id] = append(mv.infos[id], si)
	sort.Slice(mv.infos[id], func(i, j int) bool {
		return mv.infos[id][i].Start < mv.infos[id][j].Start
	})

	// TODO: update the span view widget directly instead of
	// recreating.

	// 	if id == mv.selected {
	// 		mv.spans = NewSpanViewWidget(mv.infos[id]...)
	// 	}
}

func (mv *MainViewWidget) createSpanViewWidget(gtx C) *SpanViewWidget {
	infos := mv.infos[mv.selected]

	// infos = exampleInfos

	return NewSpanViewWidget(gtx, infos...)
}
