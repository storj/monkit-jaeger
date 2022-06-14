// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"
)

type VScrollWidget struct{}

func (s *VScrollWidget) Add(ops *op.Ops) {
	pointer.InputOp{
		Tag:          s,
		Types:        pointer.Scroll,
		ScrollBounds: image.Rect(0, -1, 0, 1),
	}.Add(ops)
}

func (s *VScrollWidget) Scroll(q event.Queue) (d float32, pos f32.Point) {
	for _, e := range q.Events(s) {
		e, ok := e.(pointer.Event)
		if !ok {
			continue
		} else if e.Type != pointer.Scroll {
			continue
		}

		d += e.Scroll.Y
		if e.Modifiers.Contain(key.ModShift) {
			d += 3 * e.Scroll.Y
		}

		pos = e.Position
	}
	return d, pos
}
