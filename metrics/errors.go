package metrics

import "errors"

var ErrVoidText = errors.New("metrics: cord is void")
var ErrIllegalArguments = errors.New("metrics: illegal arguments")
var ErrIndexOutOfBounds = errors.New("metrics: index out of bounds")
