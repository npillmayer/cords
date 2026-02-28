package metrics

import (
	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
)

// ParagraphSpan is a byte range [From,To) describing one paragraph.
type ParagraphSpan struct {
	From uint64
	To   uint64
}

// ParagraphDelimiterPolicy defines how paragraph boundaries are recognized.
type ParagraphDelimiterPolicy uint8

const (
	// ParagraphByLineBreak splits at each recognized line break.
	ParagraphByLineBreak ParagraphDelimiterPolicy = iota
	// ParagraphByBlankLines splits only at runs of 2+ consecutive line breaks.
	ParagraphByBlankLines
)

// ParagraphPolicy configures paragraph discovery.
type ParagraphPolicy struct {
	// Delimiters selects paragraph-separator semantics.
	//
	// Zero-value defaults to ParagraphByLineBreak.
	Delimiters ParagraphDelimiterPolicy
	// KeepEmpty reports empty paragraphs created by separators at boundaries or
	// by consecutive separators.
	KeepEmpty bool
	// TreatCRAsLineBreak recognizes standalone '\r' as a line break.
	//
	// '\r\n' is always recognized as one line break.
	TreatCRAsLineBreak bool
}

// FindParagraphs finds paragraph spans in text according to policy.
//
// Returned ranges are always byte spans [From,To) within text.
func FindParagraphs(text cordext.CordEx[btree.NO_EXT], policy ParagraphPolicy) []ParagraphSpan {
	if text.IsVoid() {
		return nil
	}
	policy = normalizeParagraphPolicy(policy)
	lineBreaks := scanLineBreaks(text, policy.TreatCRAsLineBreak)
	tracer().Debugf("lineBreaks: %v", lineBreaks)
	separators := toParagraphSeparators(lineBreaks, policy.Delimiters)
	return toParagraphSpans(text.Len(), separators, policy.KeepEmpty)
}

// ParagraphAt returns the paragraph span covering byte position pos.
//
// A position that falls into a separator byte (for example '\n' in line-break
// mode) returns ErrIndexOutOfBounds.
func ParagraphAt(text cordext.CordEx[btree.NO_EXT], pos uint64, policy ParagraphPolicy) (ParagraphSpan, error) {
	if text.IsVoid() || pos >= text.Len() {
		return ParagraphSpan{}, cordext.ErrIndexOutOfBounds
	}
	for _, sp := range FindParagraphs(text, policy) {
		if sp.From <= pos && pos < sp.To {
			return sp, nil
		}
	}
	return ParagraphSpan{}, cordext.ErrIndexOutOfBounds
}

// ParagraphsInRange returns paragraph spans overlapping byte range [from,to).
//
// Returned spans are original paragraph bounds and are not clipped to [from,to).
func ParagraphsInRange(text cordext.CordEx[btree.NO_EXT], from, to uint64, policy ParagraphPolicy) ([]ParagraphSpan, error) {
	if from > to {
		return nil, cordext.ErrIllegalArguments
	}
	if to > text.Len() {
		return nil, cordext.ErrIndexOutOfBounds
	}
	if text.IsVoid() || from == to {
		return nil, nil
	}
	spans := FindParagraphs(text, policy)
	out := make([]ParagraphSpan, 0, len(spans))
	for _, sp := range spans {
		if sp.To > from && sp.From < to {
			out = append(out, sp)
		}
	}
	return out, nil
}

func normalizeParagraphPolicy(policy ParagraphPolicy) ParagraphPolicy {
	switch policy.Delimiters {
	case ParagraphByLineBreak, ParagraphByBlankLines:
		return policy
	default:
		policy.Delimiters = ParagraphByLineBreak
		return policy
	}
}

type byteRange struct {
	from uint64
	to   uint64
}

func scanLineBreaks(text cordext.CordEx[btree.NO_EXT], treatCRAsLineBreak bool) []byteRange {
	breaks := make([]byteRange, 0, 8)
	var pendingCR bool
	var crPos uint64
	_ = text.EachTextSegment(func(seg cordext.TextSegment, base uint64) error {
		b := seg.Bytes()
		tracer().Debugf("scanLineBreaks: segment = %v", b)
		for i := range len(b) {
			pos := base + uint64(i)
			c := b[i]
			if pendingCR {
				if c == '\n' {
					breaks = append(breaks, byteRange{from: crPos, to: pos + 1})
					pendingCR = false
					continue
				}
				if treatCRAsLineBreak {
					breaks = append(breaks, byteRange{from: crPos, to: crPos + 1})
				}
				pendingCR = false
			}
			if c == '\r' {
				pendingCR = true
				crPos = pos
				continue
			}
			if c == '\n' {
				breaks = append(breaks, byteRange{from: pos, to: pos + 1})
			}
		}
		return nil
	})
	if pendingCR && treatCRAsLineBreak {
		breaks = append(breaks, byteRange{from: crPos, to: crPos + 1})
	}
	return breaks
}

func toParagraphSeparators(lineBreaks []byteRange, mode ParagraphDelimiterPolicy) []byteRange {
	if len(lineBreaks) == 0 {
		return nil
	}
	if mode == ParagraphByLineBreak {
		return lineBreaks
	}
	separators := make([]byteRange, 0, len(lineBreaks)/2+1)
	runStart := 0
	for i := 1; i <= len(lineBreaks); i++ {
		isRunContinuation := i < len(lineBreaks) && lineBreaks[i].from == lineBreaks[i-1].to
		if isRunContinuation {
			continue
		}
		if i-runStart >= 2 {
			separators = append(separators, byteRange{
				from: lineBreaks[runStart].from,
				to:   lineBreaks[i-1].to,
			})
		}
		runStart = i
	}
	return separators
}

func toParagraphSpans(totalLen uint64, separators []byteRange, keepEmpty bool) []ParagraphSpan {
	spans := make([]ParagraphSpan, 0, len(separators)+1)
	start := uint64(0)
	for _, sep := range separators {
		if keepEmpty || sep.from > start {
			spans = append(spans, ParagraphSpan{From: start, To: sep.from})
		}
		start = sep.to
	}
	if keepEmpty || totalLen > start {
		spans = append(spans, ParagraphSpan{From: start, To: totalLen})
	}
	return spans
}
