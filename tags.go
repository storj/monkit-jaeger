package jaeger

import (
	"github.com/zeebo/errs"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

type Tag struct {
	Key   string
	Value interface{}
}

func (t *Tag) ToJaeger() (*jaeger.Tag, error) {
	jaegerTag := &jaeger.Tag{
		Key: t.Key,
	}

	switch t.Value.(type) {
	case string:
		v := t.Value.(string)
		jaegerTag.VStr = &v
		jaegerTag.VType = jaeger.TagType_STRING
	case bool:
		v := t.Value.(bool)
		jaegerTag.VBool = &v
		jaegerTag.VType = jaeger.TagType_BOOL
	case int, int32, int64:
		v := t.Value.(int64)
		jaegerTag.VLong = &v
		jaegerTag.VType = jaeger.TagType_LONG
	case float32, float64:
		v := t.Value.(float64)
		jaegerTag.VDouble = &v
		jaegerTag.VType = jaeger.TagType_DOUBLE
	default:
		return nil, errs.New("illegal type value: %T", t.Value)
	}

	return jaegerTag, nil
}
