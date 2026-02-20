/*
Package textfile provides API helpers to load UTF-8 text files as cords.

The current implementation is aligned with the chunk/sum-tree cord core and
uses a bounded asynchronous prefetch pipeline internally while preserving a
synchronous `Load` API.

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
