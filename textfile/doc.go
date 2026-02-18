/*
Package textfile provides a convenient and efficient API for handling content of large texts as strings.
Texts are loaded into memory as leaf nodes of a cord data structure. The cord allows for concurrent
operations on text fragements and stable performance characterstics for large texts.

_________________________________________________________________________

# BSD 3-Clause License

# Copyright (c) Norbert Pillmayer

All rights reserved.

Please refer to the LICENSE file for details.
*/
package textfile

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'cords'
func tracer() tracing.Trace {
	return tracing.Select("cords")
}
