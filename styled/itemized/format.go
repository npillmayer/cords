package itemized

// A Formatter is able to format a run of text according to a style-format.
/*
type Formatter interface {
	StartRun(Style, io.Writer) error
	Format([]byte, Style, io.Writer) error
	EndRun(Style, io.Writer) error
}
*/

// Format a text. Format reads text from a scanner and applies style-formats to
// runs of text, using a given formatter for output to w.
//
// If any of the arguments is nil, no output is written.
//
/*
func (runs Runs) Format(text *bufio.Scanner, fmtr Formatter, w io.Writer) (err error) {
	if fmtr == nil || text == nil || w == nil {
		return cords.ErrIllegalArguments
	}
	remain := uint64(0) // remaining fragment from text.Bytes to format/output
	err = cords.Cord(runs).EachLeaf(func(l cords.Leaf, pos uint64) (leaferr error) {
		style := l.(*styleLeaf)
		if style.Weight() == 0 {
			return nil
		}
		T().Debugf("formatting leaf %v with length=%d", style, style.Weight())
		leaferr = fmtr.StartRun(style.style, w)
		i := uint64(0) // bytes written for this leaf
		for leaferr == nil && i < style.length {
			if remain > 0 { // do not scan new bytes
				T().Debugf("%d bytes remaining to format", remain)
			} else if !text.Scan() {
				T().Errorf("premature end of input text")
				if leaferr = text.Err(); leaferr == nil {
					leaferr = errors.New("premature end of input text")
				} else {
					leaferr = fmt.Errorf("premature end of input text: %w", leaferr)
				}
				break
			} else {
				remain = uint64(len(text.Bytes()))
				T().Debugf("loaded %d new bytes", remain)
			}
			// now remain holds the (suffix) length of text.Bytes not formatted/output yet
			bstart := uint64(len(text.Bytes())) - remain // start within buffer
			l := style.length - i                        // length of substring which may be output
			if l < remain {                              // we output rest of leaf, but not complete buffer
				fmtr.Format(text.Bytes()[bstart:bstart+l], style.style, w)
				remain -= l
				i += l
			} else { // we output a (sub)string of leaf and complete buffer
				fmtr.Format(text.Bytes()[bstart:], style.style, w)
				i += remain
				remain = 0
			}
		}
		if leaferr == nil {
			leaferr = fmtr.EndRun(style.style, w)
		}
		return
	})
	if err != nil && remain > 0 {
		T().Infof("premature end of formatting runs; cannot format rest of input text")
		err = errors.New("premature end of formatting runs; cannot format rest of input text")
	}
	return
}
*/

// Format a styled text. Format applies the previously set style-formats,
// using a given formatter for output to w.
//
// If any argument is nil, no output is written.
// func (t *Text) Format(fmtr Formatter, w io.Writer) error {
// 	if fmtr == nil || w == nil {
// 		return cords.ErrIllegalArguments
// 	}
// 	scn := bufio.NewScanner(t.text.Reader())
// 	return t.runs.Format(scn, fmtr, w)
// }
