package cords

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to global the core-tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}
