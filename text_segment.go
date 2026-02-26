package cords

import "github.com/npillmayer/cords/cordext"

// TextSegment is a read-only view of one text chunk and its summary.
//
// It is re-exported from sub-package cordext to keep root-package APIs thin.
type TextSegment = cordext.TextSegment
