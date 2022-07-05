// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import "math"

type spanLayoutInfo struct {
	Span        SpanInfo
	Parent      int64
	Children    []int64
	LargestTime int64
	Layout      bool
	Rows        [][]int64
	Row         int
}

func computeLayoutInformation(spans []SpanInfo, fs []func(SpanInfo) bool) (map[int64]*spanLayoutInfo, int, int64) {
	spanTree := computeSpanTree(spans, fs)
	for _, si := range spans {
		includeSpanInLayoutInformation(spanTree, spanTree[si.Id])
	}

	usedRows := 1
	for _, spanInfo := range spanTree {
		if _, ok := spanTree[spanInfo.Parent]; !ok {
			usedRows += computeRow(spanTree, spanInfo, usedRows)
		}
	}

	var min, max int64 = math.MaxInt64, math.MinInt64
	for _, span := range spans {
		if span.Start < min {
			min = span.Start
		}
		if span.Finish > max {
			max = span.Finish
		}
	}

	return spanTree, usedRows + 1, max - min
}

func spanInfoAllowed(si SpanInfo, fs []func(SpanInfo) bool) bool {
	for _, f := range fs {
		if !f(si) {
			return false
		}
	}
	return true
}

func computeSpanTree(spans []SpanInfo, fs []func(SpanInfo) bool) map[int64]*spanLayoutInfo {
	out := make(map[int64]*spanLayoutInfo)
	for _, span := range spans {
		id := span.Id
		if out[id] == nil {
			out[id] = new(spanLayoutInfo)
		}
		out[id].Span = span

		if span.Finish > out[id].LargestTime {
			out[id].LargestTime = span.Finish
		}

		if pid := span.Parent; pid != 0 {
			out[id].Parent = pid

			if pedges := out[pid]; pedges == nil {
				out[pid] = new(spanLayoutInfo)
			}
			out[pid].Children = append(out[pid].Children, id)

			if span.Finish > out[pid].LargestTime {
				out[pid].LargestTime = span.Finish
			}
		}
	}

	for _, span := range spans {
		if spanInfoAllowed(span, fs) {
			continue
		}

		sli := out[span.Id]
		delete(out, span.Id)

		for _, cid := range sli.Children {
			out[cid].Parent = sli.Parent
		}

		if psli := out[span.Parent]; psli != nil {
			psli.Children = append(psli.Children, sli.Children...)
			if sli.LargestTime > psli.LargestTime {
				psli.LargestTime = sli.LargestTime
			}
		}
	}

	return out
}

func includeSpanInLayoutInformation(spanTree map[int64]*spanLayoutInfo, sli *spanLayoutInfo) {
	if sli == nil || sli.Layout {
		return
	}
	sli.Layout = true

	pid := sli.Parent
	if pid == 0 || spanTree[pid] == nil {
		return
	}

	psi, ok := spanTree[pid]
	if !ok {
		includeSpanInLayoutInformation(spanTree, spanTree[pid])
		psi = spanTree[pid]
	}

	start := sli.Span.Start
	found := false
	for i, children := range psi.Rows {
		if len(children) == 0 || start > spanTree[children[len(children)-1]].LargestTime {
			psi.Rows[i] = append(psi.Rows[i], sli.Span.Id)
			found = true
			break
		}
	}
	if !found {
		psi.Rows = append(psi.Rows, []int64{sli.Span.Id})
	}
}

func computeRow(spanTree map[int64]*spanLayoutInfo, si *spanLayoutInfo, startingRow int) int {
	si.Row = startingRow
	usedRows := startingRow + 1
	for _, children := range si.Rows {
		maxHeight := 0
		for _, child := range children {
			childHeight := computeRow(spanTree, spanTree[child], usedRows)
			if childHeight > maxHeight {
				maxHeight = childHeight
			}
		}
		usedRows += maxHeight
	}
	return usedRows - startingRow
}
