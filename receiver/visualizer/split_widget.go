// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

type SplitWidget struct {
	Ratio float32

	drag   bool
	dragID pointer.ID
	dragX  float32

	leftsize    int
	rightsize   int
	rightoffset int
}

func NewSplitWidget(ratio float32) SplitWidget {
	return SplitWidget{
		Ratio: ratio,
	}
}

func (s *SplitWidget) Layout(gtx C) D {
	const smallest = 200
	const bar = 5

	proportion := (s.Ratio + 1) / 2
	s.leftsize = int(proportion*float32(gtx.Constraints.Max.X) - bar)

	if s.leftsize < smallest {
		s.leftsize = smallest
	}
	if s.leftsize+bar+smallest > gtx.Constraints.Max.X {
		s.leftsize = gtx.Constraints.Max.X - bar - smallest
	}

	s.rightoffset = s.leftsize + bar
	s.rightsize = gtx.Constraints.Max.X - s.rightoffset

	{
		for _, ev := range gtx.Events(s) {
			e, ok := ev.(pointer.Event)
			if !ok {
				continue
			}

			switch e.Type {
			case pointer.Press:
				if s.drag {
					break
				}

				s.dragID = e.PointerID
				s.dragX = e.Position.X

			case pointer.Drag:
				if s.dragID != e.PointerID {
					break
				}

				deltaX := e.Position.X - s.dragX
				s.dragX = e.Position.X

				deltaRatio := deltaX * 2 / float32(gtx.Constraints.Max.X)
				s.Ratio += deltaRatio

			case pointer.Release, pointer.Cancel:
				s.drag = false
			}
		}

		barRect := image.Rect(s.leftsize, 0, s.rightoffset, gtx.Constraints.Max.Y)
		area := clip.Rect(barRect).Push(gtx.Ops)
		paint.ColorOp{Color: black}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		pointer.InputOp{
			Tag:   s,
			Types: pointer.Press | pointer.Drag | pointer.Release,
			Grab:  s.drag,
		}.Add(gtx.Ops)
		area.Pop()
	}

	return D{Size: gtx.Constraints.Max}
}

func (s *SplitWidget) Left(gtx C, w layout.Widget) {
	gtx.Constraints = layout.Exact(image.Pt(s.leftsize, gtx.Constraints.Max.Y))
	layout.UniformInset(unit.Dp(5)).Layout(gtx, w)
}

func (s *SplitWidget) Right(gtx C, w layout.Widget) {
	off := op.Offset(image.Pt(s.rightoffset, 0)).Push(gtx.Ops)
	gtx.Constraints.Max = image.Pt(s.rightsize, gtx.Constraints.Max.Y)
	layout.UniformInset(unit.Dp(5)).Layout(gtx, w)
	off.Pop()
}
