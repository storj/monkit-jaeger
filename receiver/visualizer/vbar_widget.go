// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type VBarWidget struct {
	pos f32.Point
}

func (s *VBarWidget) Add(ops *op.Ops) {
	pointer.InputOp{
		Tag:   s,
		Types: pointer.Move,
	}.Add(ops)
}

func (s *VBarWidget) Move(q event.Queue) {
	for _, e := range q.Events(s) {
		e, ok := e.(pointer.Event)
		if !ok {
			continue
		} else if e.Type != pointer.Move {
			continue
		}
		s.pos = e.Position
	}
}

func (s *VBarWidget) Layout(gtx C) {
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(s.pos.X, 0))
	p.LineTo(f32.Pt(s.pos.X, float32(gtx.Constraints.Max.Y)))

	paint.FillShape(gtx.Ops, black, clip.Stroke{
		Path:  p.End(),
		Width: 1,
	}.Op())
}
