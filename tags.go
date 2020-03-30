// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package jaeger

import (
	"github.com/zeebo/errs"

	"storj.io/monkit-jaeger/gen-go/jaeger"
)

// Tag is a key/value pair that allows us to translate monkit annotations and arguments into jaeger thrift tags.
type Tag struct {
	Key   string
	Value interface{}
}

// BuildJaegerThrift converts tag into jaeger thrift format.
func (t *Tag) BuildJaegerThrift() (*jaeger.Tag, error) {
	jaegerTag := &jaeger.Tag{
		Key: t.Key,
	}

	switch v := t.Value.(type) {
	case string:
		jaegerTag.VStr = &v
		jaegerTag.VType = jaeger.TagType_STRING
	case bool:
		jaegerTag.VBool = &v
		jaegerTag.VType = jaeger.TagType_BOOL
	case int, int32, int64:
		num := t.Value.(int64)
		jaegerTag.VLong = &num
		jaegerTag.VType = jaeger.TagType_LONG
	case float32, float64:
		num := t.Value.(float64)
		jaegerTag.VDouble = &num
		jaegerTag.VType = jaeger.TagType_DOUBLE
	default:
		return nil, errs.New("illegal type value: %T", t.Value)
	}

	return jaegerTag, nil
}
