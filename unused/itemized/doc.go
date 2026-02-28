/*
Package itemized helps itemizing paragraphs and prepare them for formatting.

_________________________________________________________________________

# BSD 3-Clause License

# Copyright (c) Norbert Pillmayer

All rights reserved.

Please find the license file at the root folder of this package.
*/
package itemized

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global core-tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}
