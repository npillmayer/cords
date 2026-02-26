package cordext

// CordError is the package-local error type.
type CordError string

func (e CordError) Error() string {
	return string(e)
}

// ErrCordCompleted signals that a builder has been finalized via Cord().
const ErrCordCompleted = CordError("forbidden to add fragments; cord has been completed")

// ErrIndexOutOfBounds is returned for positions outside valid byte ranges.
const ErrIndexOutOfBounds = CordError("index out of bounds")

// ErrIllegalArguments is returned for invalid function arguments.
const ErrIllegalArguments = CordError("illegal arguments")
