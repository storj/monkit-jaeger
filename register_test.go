package jaeger

import (
	"context"
	"testing"

	"github.com/spacemonkeygo/monkit/v3"
	"github.com/stretchr/testify/require"
	"storj.io/monkit-jaeger/gen-go/jaeger"
)

var mon = monkit.Package()

type expected struct {
	operationName string
	hasParentID   bool
	tags          []*jaeger.Tag
}
type testCollector struct {
	t        *testing.T
	expected expected
}

func (c *testCollector) Collect(s *jaeger.Span) {
	require.Contains(c.t, s.GetOperationName(), c.expected.operationName)
	require.Equal(c.t, c.expected.hasParentID, s.GetParentSpanId() != 0)
	require.Equal(c.t, len(c.expected.tags), len(s.GetTags()))
	for i := range c.expected.tags {
		expectedTag := c.expected.tags[i]
		actualTag, ok := findTag(expectedTag.GetKey(), s)
		require.True(c.t, ok)
		require.Equal(c.t, expectedTag.GetVType(), actualTag.GetVType())
	}
}
func newTestCollector(t *testing.T, testCase expected) *testCollector {
	return &testCollector{
		t:        t,
		expected: testCase,
	}
}

func TestRegisterJaeger(t *testing.T) {

	cases := expected{
		operationName: "test-register",
		hasParentID:   false,
		tags:          make([]*jaeger.Tag, 0),
	}
	collector := newTestCollector(t, cases)
	RegisterJaeger(monkit.Default, collector, Options{
		Fraction: 1,
	})
	newTrace(cases.operationName)
}

func TestRegisterJaeger_WithParent(t *testing.T) {

	cases := expected{
		operationName: "test-register-parent",
		hasParentID:   true,
		tags:          make([]*jaeger.Tag, 0),
	}
	collector := newTestCollector(t, cases)
	RegisterJaeger(monkit.Default, collector, Options{
		Fraction: 1,
	})
	newTraceWithParent(context.Background(), cases.operationName)
}

func TestRegisterJaeger_WithTags(t *testing.T) {

	cases := expected{
		operationName: "test-register-tags",
		hasParentID:   false,
		tags:          make([]*jaeger.Tag, 0),
	}
	tagValue := "test"
	cases.tags = append(cases.tags, &jaeger.Tag{
		Key:   "arg_0",
		VType: jaeger.TagType_STRING,
		VStr:  &tagValue,
	})
	collector := newTestCollector(t, cases)
	RegisterJaeger(monkit.Default, collector, Options{
		Fraction: 1,
	})
	newTraceWithTags(context.Background(), cases.operationName, tagValue)
}

func newTrace(name string) {
	defer mon.TaskNamed(name)(nil)(nil)
}

func newTraceWithParent(ctx context.Context, name string) {
	newTrace := monkit.NewTrace(monkit.NewId())
	newTrace.Set(remoteParentKey, monkit.NewId())
	defer mon.FuncNamed(name).RemoteTrace(&ctx, monkit.NewId(), newTrace)
}

func newTraceWithTags(ctx context.Context, name string, tag string) {
	defer mon.TaskNamed(name)(&ctx, tag)(nil)
}

func findTag(key string, s *jaeger.Span) (*jaeger.Tag, bool) {
	for _, t := range s.GetTags() {
		if t.Key == key {
			return t, true
		}
	}
	return nil, false
}
