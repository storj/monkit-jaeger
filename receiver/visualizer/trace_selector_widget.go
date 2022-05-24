// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type TraceSelectorWidget struct {
	Items []string

	id       int
	invalid  bool
	hidden   []bool
	shown    []int
	editor   EditboxWidget
	list     widget.List
	click    gesture.Click
	selected int
}

func NewTraceSelectorWidget(items ...string) TraceSelectorWidget {
	return TraceSelectorWidget{
		Items: items,

		hidden:   make([]bool, len(items)),
		editor:   NewEditboxWidget("Trace ID", false),
		list:     widget.List{List: layout.List{Axis: layout.Vertical}},
		selected: -1,
	}
}

func (t *TraceSelectorWidget) SetHidden(n int, hidden bool) {
	t.hidden[n] = hidden
	t.invalid = true
}

func (t *TraceSelectorWidget) AddItem(label string) int {
	t.Items = append(t.Items, label)
	t.hidden = append(t.hidden, false)
	t.invalid = true
	t.id++
	return t.id - 1
}

func (t *TraceSelectorWidget) filter(text string) {
	for i := range t.Items {
		t.hidden[i] = false
	}
	for _, word := range strings.Fields(text) {
		for i, it := range t.Items {
			t.hidden[i] = t.hidden[i] || !strings.Contains(it, word)
		}
	}
	t.invalid = true
}

func (t *TraceSelectorWidget) Layout(gtx C) D {
	if t.invalid {
		t.shown = t.shown[:0]
		for i := range t.Items {
			if !t.hidden[i] {
				t.shown = append(t.shown, i)
			}
		}
	}

	for _, ev := range t.editor.Events() {
		if _, ok := ev.(widget.ChangeEvent); ok {
			t.filter(t.editor.Text())
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(t.editor.Layout),

		layout.Rigid(func(gtx C) D {
			return layout.Dimensions{Size: image.Pt(0, 10)}
		}),

		layout.Rigid(func(gtx C) D {
			return widget.Border{
				Color: color.NRGBA{A: 0xff},
				Width: unit.Dp(1),
			}.Layout(gtx, func(gtx C) D {
				ls := material.List(th, &t.list)
				ls.AnchorStrategy = material.Overlay

				var height int

				m := op.Record(gtx.Ops)
				dims := ls.Layout(gtx, len(t.shown), func(gtx C, i int) D {
					it := &t.Items[t.shown[i]]

					macro := op.Record(gtx.Ops)
					label := material.Label(th, th.TextSize, *it)
					dims := layout.UniformInset(unit.Dp(6)).Layout(gtx, label.Layout)
					ops := macro.Stop()

					height = dims.Size.Y

					defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()

					bg := [2]color.NRGBA{white, gray}[i%2]
					if t.selected == t.shown[i] {
						bg = selected
					}

					paint.ColorOp{Color: bg}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					ops.Add(gtx.Ops)

					return dims
				})
				c := m.Stop()

				defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()

				for _, cl := range t.click.Events(gtx.Queue) {
					if cl.Type == gesture.TypeClick {
						i := (int(cl.Position.Y)+t.list.Position.Offset)/height + t.list.Position.First
						t.selected = t.shown[i]
					}
				}

				t.click.Add(gtx.Ops)
				c.Add(gtx.Ops)

				return dims
			})
		}),
	)
}
