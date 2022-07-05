// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/op"
)

type viewportState struct {
	Left  f32.Point
	Scale float32
}

func (v *viewportState) Offset(p f32.Point) {
	p.X *= v.Scale
	v.Left = v.Left.Add(p)
}

func lerpF32Point(a, b f32.Point, t float64) f32.Point {
	return f32.Pt(lerpF32(a.X, b.X, t), lerpF32(a.Y, b.Y, t))
}

func lerpViewportState(a, b viewportState, t float64) viewportState {
	return viewportState{
		Left:  lerpF32Point(a.Left, b.Left, t),
		Scale: lerpF32(a.Scale, b.Scale, t),
	}
}

// type animationViewportState = animation[viewportState]

type ViewportWidget struct {
	state viewportState
	anim  *animationViewportState

	drag   DragWidget
	scroll VScrollWidget
	click  gesture.Click
}

func NewViewportWidget(scale float32) ViewportWidget {
	return ViewportWidget{
		state: viewportState{
			Scale: scale,
		},
	}
}

func (v *ViewportWidget) Add(ops *op.Ops) {
	v.click.Add(ops)
	key.InputOp{Tag: v, Keys: key.Set("(Shift)-W,A,S,D")}.Add(ops)

	v.drag.Add(ops)
	v.scroll.Add(ops)
}

func (v *ViewportWidget) View(gtx C) (image.Rectangle, float32) {
	var animating bool
	if v.anim != nil {
		v.state, animating = v.anim.Update(gtx,
			func(init, final viewportState, t float64) viewportState {
				return lerpViewportState(init, final, t)
			})
		if !animating {
			v.anim = nil
		}
	}

	if len(v.click.Events(gtx.Queue)) > 0 {
		key.FocusOp{Tag: v}.Add(gtx.Ops)
	}

	if dd := v.HandleKeyboard(gtx.Queue); dd != (f32.Point{}) && !animating {
		final := v.state
		final.Offset(dd)
		dur := 100 * time.Millisecond
		if dd.Y != 0 {
			dur /= 5
		}
		v.anim = newAnimationViewportState(gtx, v.state, final, dur)
	}

	v.state.Offset(v.drag.Drag(gtx.Queue))

	if sd, pos := v.scroll.Scroll(gtx.Queue); sd != 0 && !animating {
		abs := pos.Mul(-v.state.Scale).Add(v.state.Left)
		abs.Y = v.state.Left.Y
		center := viewportState{Left: abs, Scale: 0}
		final := lerpViewportState(center, v.state, math.Pow(1.5, float64(sd)))
		v.anim = newAnimationViewportState(gtx, v.state, final, 100*time.Millisecond)
	}

	width := float32(gtx.Constraints.Max.X) * v.state.Scale
	height := float32(gtx.Constraints.Max.Y) * v.state.Scale

	if v.anim != nil {
		op.InvalidateOp{}.Add(gtx.Ops)
	}

	return image.Rect(
		int(v.state.Left.X), int(math.Ceil(float64(v.state.Left.Y))),
		int(v.state.Left.X+width), int(math.Ceil(float64(v.state.Left.Y+height))),
	), v.state.Scale
}

var viewportDirs = [256]f32.Point{
	'w': f32.Pt(0, 20),
	'W': f32.Pt(0, 20).Mul(4),
	'a': f32.Pt(200, 0),
	'A': f32.Pt(200, 0).Mul(4),
	's': f32.Pt(0, -20),
	'S': f32.Pt(0, -20).Mul(4),
	'd': f32.Pt(-200, 0),
	'D': f32.Pt(-200, 0).Mul(4),
}

func (v *ViewportWidget) HandleKeyboard(q event.Queue) (dd f32.Point) {
	for _, e := range q.Events(v) {
		e, ok := e.(key.EditEvent)
		if !ok {
			continue
		}
		dd = dd.Add(viewportDirs[e.Text[0]])
	}
	return dd
}
