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
		jaegerTag.VStr = t.Value.(*string)
		jaegerTag.VType = jaeger.TagType_STRING
	case bool:
		jaegerTag.VBool = t.Value.(*bool)
		jaegerTag.VType = jaeger.TagType_BOOL
	case int, int32, int64:
		jaegerTag.VLong = t.Value.(*int64)
		jaegerTag.VType = jaeger.TagType_LONG
	case float32, float64:
		jaegerTag.VDouble = t.Value.(*float64)
		jaegerTag.VType = jaeger.TagType_DOUBLE
	default:
		return nil, errs.New("illegal type value")
	}

	return jaegerTag, nil
}
