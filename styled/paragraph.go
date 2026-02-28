package styled

import (
	"io"

	"github.com/npillmayer/cords/btree"
	"github.com/npillmayer/cords/cordext"
	"github.com/npillmayer/uax/bidi"
)

// Paragraph represents a styled paragraph of text. It usually is a substring of
// a styled text, but differs from a text insofar as it may be prepared for output.
// Outputting styled text in general includes identifying runs of bidirectional
// text, which is an operation defined on paragraphs (at least by Unicode Annex #9).
// Moreover, output of styled text may include breaking up paragraphs into lines.
// Linebreaking in turn interacts with the handling of runs of bidirectional text,
// which in a sense restricts line-breaking to paragraphs.
//
// After a styled paragraph has been created, e.g., by copying out a section from
// a styled text, it's textual content is not to change any more. However, it is
// allowed to change styles for spans of a paragraph's text.
//
// Offset is the paragraph's start position in terms of byte positions of the embedding
// text. It is provided when creating a paragraph and held solely for a client's
// bookkeeping purposes. The functions of this package do not in any way depend on it.
type Paragraph struct {
	text     *Text                // a Paragraph is a styled text
	Offset   uint64               // the paragraph's start position in terms of positions of the embedding text.
	cutoff   uint64               // cut off text due to line wrapping
	eBidiDir bidi.Direction       // embedding bidi text direction
	levels   *bidi.ResolvedLevels // levels from UAX#9 algorithm
}

// ParagraphFromText creates a styled paragraph from a segment of a styled text.
// Parameters `from` and `to` denote the segment.
//
// Paragraphs may contain left-to-right text as well as right-to-left text.
// Clients should provide the overall Bidi context to apply, together with an
// optional function providing hints for Bidi runs. ParagraphFromText will apply
// the Unicode Bidi Algorithm to the paragraph's text. Clients may then call
// BidiLevels() to receive the resolved Bidi levels found by the algorithm.
//
// A paragraph remembers the `from` parameter in member `Offset`.
func ParagraphFromText(text *Text, from, to uint64, embBidi bidi.Direction,
	m bidi.OutOfLineBidiMarkup) (*Paragraph, error) {
	//
	if text == nil {
		return nil, ErrIllegalArguments
	}
	para := &Paragraph{
		Offset:   from,
		eBidiDir: embBidi,
	}
	if from == 0 && to == text.Raw().Len() {
		para.text = text
	} else {
		section, err := text.Section(from, to)
		if err != nil {
			return nil, err
		}
		para.text = &section
	}
	para.levels = bidi.ResolveParagraph(para.text.Raw().Reader(), m, bidi.DefaultDirection(embBidi), bidi.IgnoreParagraphSeparators(true))
	return para, nil
}

// Style styles a run of text of a styled paragraph, given the start and end position.
func (para *Paragraph) Style(style Style, from, to uint64) (*Paragraph, error) {
	t, err := para.text.Style(style, from, to)
	if err != nil {
		return nil, err
	}
	para.text = &t
	return para, nil
}

// Raw returns the underlying raw text of the paragraph.
func (para *Paragraph) Raw() cordext.CordEx[btree.NO_EXT] {
	return para.text.Raw()
}

// BidiLevels returns the resolved Bidi levels in a paragraph of text.
func (para *Paragraph) BidiLevels() *bidi.ResolvedLevels {
	return para.levels
}

// StyleAt returns the active style at text position pos, together with an
// index relative to the start of the style run.
//
// Calls [StyleAt] from [styled.Text].
func (para *Paragraph) StyleAt(pos uint64) (Style, uint64, error) {
	sty, i, err := para.text.StyleAt(pos)
	if err != nil {
		return nil, pos, err
	}
	return sty, i, nil
}

// Reader returns an io.Reader for the raw text of the paragraph (without styles).
func (para *Paragraph) Reader() io.Reader {
	return para.text.Raw().Reader()
}

// WrapAt splits off a front segment (usually a “line”) from a paragraph.
func (para *Paragraph) WrapAt(pos uint64) (*Text, *bidi.Ordering, error) {
	if para == nil || para.text == nil {
		return nil, nil, ErrIllegalArguments
	}
	if pos < para.cutoff {
		return nil, nil, ErrIndexOutOfBounds
	}
	pos -= para.cutoff
	if pos >= para.Raw().Len() {
		tracer().Infof("Paragraph.WrapAt(EOT)")
	}
	tracer().Infof("  Levels = %v", para.BidiLevels())
	line, p, err := para.text.Raw().Split(pos)
	if err != nil {
		return nil, nil, err
	}
	para.text.text = p
	text := &Text{
		text: line,
	}
	p_runs := pipeFor(para.text.runs)
	switch p_runs.err {
	case nil:
		lineRuns, paraRuns, splitErr := para.text.runs.SplitAt(pos)
		if splitErr != nil {
			return nil, nil, splitErr
		}
		text.runs = lineRuns
		para.text.runs = paraRuns
	case ErrVoidRuns:
		// Unstyled paragraph: keep both sides unstyled.
	default:
		return nil, nil, p_runs.err
	}
	var lineLev *bidi.ResolvedLevels
	lineLev, para.levels = para.levels.Split(pos, true)
	tracer().Infof("para.levels = %v", para.levels)
	lineRuns := lineLev.Reorder()
	para.cutoff += text.Raw().Len()
	return text, lineRuns, nil
}
