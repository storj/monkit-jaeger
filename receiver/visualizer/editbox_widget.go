// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type EditboxWidget struct {
	Hint string

	editor widget.Editor
}

func NewEditboxWidget(hint string, submit bool) EditboxWidget {
	return EditboxWidget{
		Hint: hint,
		editor: widget.Editor{
			SingleLine: true,
			Submit:     submit,
		},
	}
}

func (e *EditboxWidget) Events() []widget.EditorEvent {
	return e.editor.Events()
}

func (e *EditboxWidget) Text() string { return e.editor.Text() }

func (e *EditboxWidget) Layout(gtx C) D {
	return widget.Border{
		Color: black,
		Width: unit.Dp(2),
	}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Max.Y = 32

		m := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(6)).Layout(gtx,
			material.Editor(th, &e.editor, e.Hint).Layout)
		c := m.Stop()

		defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
		paint.ColorOp{Color: white}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		c.Add(gtx.Ops)

		return dims
	})
}
