/*
Package styled makes styled text.

# Status

Work in progress.

_________________________________________________________________________

# BSD 3-Clause License

# Copyright (c) Norbert Pillmayer

For details please refer to the LICENSE file.
*/
package styled

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'styles'
func tracer() tracing.Trace {
	return tracing.Select("styles")
}

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}
