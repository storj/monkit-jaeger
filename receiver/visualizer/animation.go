// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import "time"

// type animation[T any] struct {
// 	init  T
// 	final T
// 	start time.Time
// 	dur   time.Duration
// }

// func newAnimation[T any](gtx C, init, final T, dur time.Duration) *animation[T] {
// 	return &animation[T]{
// 		init:  init,
// 		final: final,
// 		start: gtx.Now,
// 		dur:   dur,
// 	}
// }

// func (a *animation[T]) Update(gtx C, fn func(init, final T, t float64) T) (T, bool) {
// 	t := float64(gtx.Now.Sub(a.start)) / float64(a.dur)
// 	if t >= 1 {
// 		return a.final, false
// 	} else if t <= 0 {
// 		return a.init, true
// 	}
// 	return fn(a.init, a.final, t), true
// }

type animationViewportState struct {
	init  viewportState
	final viewportState
	start time.Time
	dur   time.Duration
}

func newAnimationViewportState(gtx C, init, final viewportState, dur time.Duration) *animationViewportState {
	return &animationViewportState{
		init:  init,
		final: final,
		start: gtx.Now,
		dur:   dur,
	}
}

func (a *animationViewportState) Update(gtx C, fn func(init, final viewportState, t float64) viewportState) (viewportState, bool) {
	t := float64(gtx.Now.Sub(a.start)) / float64(a.dur)
	if t >= 1 {
		return a.final, false
	} else if t <= 0 {
		return a.init, true
	}
	return fn(a.init, a.final, t), true
}
