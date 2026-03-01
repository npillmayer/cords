/*
Package metrics provides focused text-analysis helpers on top of cords/cordext.

The legacy generic metric-combine framework has been removed; this package now
contains direct, purpose-built analyzers that operate on immutable cord ranges
or segment iterators.
*/
package metrics

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'cords.cords'
func tracer() tracing.Trace {
	return tracing.Select("cords.cords")
}

func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

// We will be dealing mainly with cords without extension.
type plainCordType = cordext.CordEx[btree.NO_EXT]
