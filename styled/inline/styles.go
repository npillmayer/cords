package inline

import (
	"fmt"

	"github.com/npillmayer/cords/styled"
)

// Some standard text formats
const (
	PlainStyle Style = 0
	BoldStyle  Style = 1 << iota
	ItalicsStyle
	StrongStyle
	EmStyle
	SmallStyle
	MarkedStyle
)

func styleString(s Style) string {
	switch s {
	case PlainStyle:
		return "plain"
	case BoldStyle:
		return "b"
	case ItalicsStyle:
		return "i"
	case StrongStyle:
		return "strong"
	case EmStyle:
		return "em"
	case SmallStyle:
		return "small"
	case MarkedStyle:
		return "marked"
	}
	return fmt.Sprintf("Style(%d)", s)
}

// Style is a text style, applicable on runs of characters
type Style int

func (s Style) Add(other Style) Style {
	return s | other
}

func (s Style) Minus(other Style) Style {
	return s & ^other
}

func (s Style) String() string {
	if s == 0 {
		return styleString(0)
	}
	str := ""
	for i := 0; i < 7; i++ {
		//T().Debugf("check: %d = %s", 1<<i, styleString(1<<i))
		if s&(1<<i) > 0 {
			str = str + styleString(1<<i)
		}
	}
	if str != "" {
		return str
	}
	return styleString(s)
}

func (s Style) Equals(other styled.Format) bool {
	return false
}
