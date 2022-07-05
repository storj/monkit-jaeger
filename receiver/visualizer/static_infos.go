// Copyright (C) 2022 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"html"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"time"
)

var re = regexp.MustCompile(
	`(?ms)<g id="id-([0-9]*)" class="func parent-([0-9]*)" onmouseover="mouseover\('[^']*', '[^']*', '([^']*) Duration:([^ ]*) Started:([^ ]*)'\);" onmouseout="[^"]*" onclick="[^"]*">[^<]*<clipPath[^>]*>[^<]*<rect[^>]*>[^<]*</clipPath[^>]*>[^<]*<rect[^>]* fill="([^"]*)"/>`,
)

var staticData []byte

func loadStatic(file string) ([]SpanInfo, error) {
	var data []byte
	var err error

	if staticData == nil {
		data, err = ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
	} else {
		data = staticData
	}

	colors := map[string]SpanStatus{
		"rgb(128,128,255)": SpanStatus_OK,
		"rgb(255,144,0)":   SpanStatus_Error,
		"rgb(255,0,0)":     SpanStatus_Panic,
		"rgb(255,255,0)":   SpanStatus_Cancel,
	}

	var infos []SpanInfo

	for _, matches := range re.FindAllStringSubmatch(string(data), -1) {
		id, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			panic(err)
		}
		parent, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			panic(err)
		}
		text := html.UnescapeString(matches[3])
		dur, err := time.ParseDuration(matches[4])
		if err != nil {
			panic(err)
		}
		start, err := time.ParseDuration(matches[5])
		if err != nil {
			panic(err)
		}
		color := matches[6]

		infos = append(infos, SpanInfo{
			Summary: text,

			Id:     id,
			Parent: parent,

			Start:  int64(start),
			Finish: int64(start) + int64(dur),

			Status: colors[color],
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Start < infos[j].Start
	})

	return infos, nil
}
