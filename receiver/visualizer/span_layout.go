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

func computeLayoutInformation(spans []SpanInfo) (map[int64]*spanLayoutInfo, int, int64) {
	spanTree := computeSpanTree(spans)
	for _, span := range spans {
		includeSpanInLayoutInformation(spanTree, span)
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

func computeSpanTree(spans []SpanInfo) map[int64]*spanLayoutInfo {
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
	return out
}

func includeSpanInLayoutInformation(spanTree map[int64]*spanLayoutInfo, span SpanInfo) {
	id := span.Id
	si := spanTree[id]
	if si.Layout {
		return
	}
	si.Layout = true

	pid := span.Parent
	if pid == 0 || spanTree[pid] == nil {
		return
	}

	psi, ok := spanTree[pid]
	if !ok {
		includeSpanInLayoutInformation(spanTree, spanTree[pid].Span)
		psi = spanTree[pid]
	}

	start := span.Start
	found := false
	for i, children := range psi.Rows {
		if len(children) == 0 || start > spanTree[children[len(children)-1]].LargestTime {
			psi.Rows[i] = append(psi.Rows[i], id)
			found = true
			break
		}
	}
	if !found {
		psi.Rows = append(psi.Rows, []int64{id})
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
