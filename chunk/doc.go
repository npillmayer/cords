package chunk

import "github.com/npillmayer/schuko/tracing"

// tracer traces with key 'cords.cords'.
func tracer() tracing.Trace {
	return tracing.Select("cords.cords")
}

func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
