// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/op"
)

type DragWidget struct {
	pressed  bool
	dragging bool
	pid      pointer.ID
	prev     f32.Point
}

func (d *DragWidget) Add(ops *op.Ops) {
	pointer.InputOp{
		Tag:   d,
		Grab:  d.dragging,
		Types: pointer.Press | pointer.Drag | pointer.Release,
	}.Add(ops)
}

func (d *DragWidget) Drag(q event.Queue) f32.Point {
	var delta f32.Point
	for _, e := range q.Events(d) {
		e, ok := e.(pointer.Event)
		if !ok {
			continue
		}

		switch e.Type {
		case pointer.Press:
			if !(e.Buttons == pointer.ButtonPrimary || e.Source == pointer.Touch) {
				continue
			}
			d.pressed = true
			if d.dragging {
				continue
			}
			d.dragging = true
			d.pid = e.PointerID
			d.prev = e.Position
		case pointer.Drag:
			if !d.dragging || e.PointerID != d.pid {
				continue
			}
			dx := e.Position.Sub(d.prev)
			if e.Modifiers.Contain(key.ModShift) {
				dx = dx.Mul(4)
			}
			delta = delta.Add(dx)
			d.prev = e.Position
		case pointer.Release, pointer.Cancel:
			d.pressed = false
			if !d.dragging || e.PointerID != d.pid {
				continue
			}
			d.dragging = false
		}
	}
	return delta
}
