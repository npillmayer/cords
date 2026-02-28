package styled

import (
	"github.com/npillmayer/cords/metrics"
	"github.com/npillmayer/uax/bidi"
)

type ParagraphSpan = metrics.ParagraphSpan
type ParagraphPolicy = metrics.ParagraphPolicy
type ParagraphDelimiterPolicy = metrics.ParagraphDelimiterPolicy

const (
	ParagraphByLineBreak  ParagraphDelimiterPolicy = metrics.ParagraphByLineBreak
	ParagraphByBlankLines ParagraphDelimiterPolicy = metrics.ParagraphByBlankLines
)

// FindParagraphSpans discovers paragraph spans for a styled text.
func FindParagraphSpans(text Text, policy ParagraphPolicy) []ParagraphSpan {
	return metrics.FindParagraphs(text.Raw(), policy)
}

// ParagraphAt returns the paragraph span covering byte position pos.
func ParagraphAt(text Text, pos uint64, policy ParagraphPolicy) (ParagraphSpan, error) {
	return metrics.ParagraphAt(text.Raw(), pos, policy)
}

// ParagraphsInRange returns paragraph spans overlapping [from,to).
func ParagraphsInRange(text Text, from, to uint64, policy ParagraphPolicy) ([]ParagraphSpan, error) {
	return metrics.ParagraphsInRange(text.Raw(), from, to, policy)
}

// ParagraphsFromText discovers paragraph spans and constructs paragraph objects.
func ParagraphsFromText(text *Text, policy ParagraphPolicy, embBidi bidi.Direction,
	m bidi.OutOfLineBidiMarkup) ([]*Paragraph, error) {
	if text == nil {
		return nil, ErrIllegalArguments
	}
	spans := FindParagraphSpans(*text, policy)
	paragraphs := make([]*Paragraph, 0, len(spans))
	for _, sp := range spans {
		para, err := ParagraphFromText(text, sp.From, sp.To, embBidi, m)
		if err != nil {
			return nil, err
		}
		paragraphs = append(paragraphs, para)
	}
	return paragraphs, nil
}
