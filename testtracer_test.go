package opentracing

import (
	"strconv"
	"strings"
)

const testHTTPHeaderPrefix = "testprefix-"

// testTracer is a most-noop Tracer implementation that makes it possible for
// unittests to verify whether certain methods were / were not called.
type testTracer struct{}

var fakeIDSource = 1

func nextFakeID() int {
	fakeIDSource++
	return fakeIDSource
}

type testSpanContext struct {
	HasParent bool
	FakeID    int
}

func (n testSpanContext) SetBaggageItem(key, val string) SpanContext { return n }
func (n testSpanContext) BaggageItem(key string) string              { return "" }

type testSpan struct {
	spanContext   testSpanContext
	OperationName string
}

// testSpan:
func (n testSpan) SpanContext() SpanContext                              { return n.spanContext }
func (n testSpan) SetTag(key string, value interface{}) Span             { return n }
func (n testSpan) Finish()                                               {}
func (n testSpan) FinishWithOptions(opts FinishOptions)                  {}
func (n testSpan) LogEvent(event string)                                 {}
func (n testSpan) LogEventWithPayload(event string, payload interface{}) {}
func (n testSpan) Log(data LogData)                                      {}
func (n testSpan) SetOperationName(operationName string) Span            { return n }
func (n testSpan) Tracer() Tracer                                        { return testTracer{} }

// StartSpan belongs to the Tracer interface.
func (n testTracer) StartSpan(operationName string) Span {
	return testSpan{
		OperationName: operationName,
		spanContext: testSpanContext{
			HasParent: false,
			FakeID:    nextFakeID(),
		},
	}
}

// StartSpanWithOptions belongs to the Tracer interface.
func (n testTracer) StartSpanWithOptions(opts StartSpanOptions) Span {
	fakeID := nextFakeID()
	if opts.Parent != nil {
		fakeID = opts.Parent.(testSpanContext).FakeID
	}
	return testSpan{
		OperationName: opts.OperationName,
		spanContext: testSpanContext{
			HasParent: opts.Parent != nil,
			FakeID:    fakeID,
		},
	}
}

// Inject belongs to the Tracer interface.
func (n testTracer) Inject(sp SpanContext, format interface{}, carrier interface{}) error {
	spanContext := sp.(testSpanContext)
	switch format {
	case TextMap:
		carrier.(TextMapWriter).Set(testHTTPHeaderPrefix+"fakeid", strconv.Itoa(spanContext.FakeID))
		return nil
	}
	return ErrUnsupportedFormat
}

// Join belongs to the Tracer interface.
func (n testTracer) Join(operationName string, format interface{}, carrier interface{}) (Span, error) {
	switch format {
	case TextMap:
		// Just for testing purposes... generally not a worthwhile thing to
		// propagate.
		sc := testSpanContext{}
		err := carrier.(TextMapReader).ForeachKey(func(key, val string) error {
			switch strings.ToLower(key) {
			case testHTTPHeaderPrefix + "fakeid":
				i, err := strconv.Atoi(val)
				if err != nil {
					return err
				}
				sc.FakeID = i
			}
			return nil
		})
		return n.StartSpanWithOptions(StartSpanOptions{
			Parent:        sc,
			OperationName: operationName,
		}), err
	}
	return nil, ErrTraceNotFound
}