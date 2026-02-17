/*
Package metrics provides some pre-manufactured metrics on texts.

_________________________________________________________________________

# BSD 3-Clause License

# Copyright (c) 2020–21, Norbert Pillmayer

Please refer to the LICENSE file for details.
*/
package metrics

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'cords'
func tracer() tracing.Trace {
	return tracing.Select("cords")
}
