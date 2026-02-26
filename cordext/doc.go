package cordext

import "github.com/npillmayer/schuko/tracing"

// tracer traces with key 'cordext'.
func tracer() tracing.Trace {
	return tracing.Select("cordext")
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
