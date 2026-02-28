/*
Package inline styles inline text such as HTML-spans or console-output.

# Status

Work in progress.

_________________________________________________________________________

# BSD 3-Clause License

# Copyright (c) Norbert Pillmayer

All rights reserved.

Please refer to the LICENSE file for details.
*/
package inline

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'cords.styles'
func tracer() tracing.Trace {
	return tracing.Select("cords.styles")
}

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}
