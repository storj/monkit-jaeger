// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"sort"

	"gioui.org/op/paint"
)

type MainViewWidget struct {
	spans *SpanViewWidget
	infos []SpanInfo
}

func NewMainViewWidget() MainViewWidget {
	return MainViewWidget{}
}

func (mv *MainViewWidget) Layout(gtx C) D {
	paint.ColorOp{Color: lightGray}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	return mv.spans.Layout(gtx)
}

func (mv *MainViewWidget) AddSpanInfos(gtx C, infos ...SpanInfo) {
	// TODO: insertion sort. or something? this is bad.
	mv.infos = append(mv.infos, infos...)
	sort.Slice(mv.infos, func(i, j int) bool {
		return mv.infos[i].Start < mv.infos[j].Start
	})

	// TODO: update the span view widget directly instead of
	// recreating.
	mv.spans = NewSpanViewWidget(gtx, mv.infos...)
}
