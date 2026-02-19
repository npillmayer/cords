package cords

// SplitRunes splits a cord into two cords at rune-aware position p.
//
// Position p is validated for this cord before splitting.
func SplitRunes(cord Cord, p Pos) (Cord, Cord, error) {
	b, err := cord.ByteOffset(p)
	if err != nil {
		return Cord{}, Cord{}, err
	}
	return Split(cord, b)
}

// ReportRunes returns n runes starting at rune-aware position start.
//
// The operation validates start for this cord and returns ErrIndexOutOfBounds
// if the requested rune span exceeds the cord length.
func (cord Cord) ReportRunes(start Pos, n uint64) (string, error) {
	if n == 0 {
		return "", nil
	}
	startB, err := cord.ByteOffset(start)
	if err != nil {
		return "", err
	}
	endPos, err := cord.posFromRunes(start.runes + n)
	if err != nil {
		return "", err
	}
	endB := endPos.bytepos
	if endB < startB {
		return "", ErrIllegalPosition
	}
	return cord.Report(startB, endB-startB)
}
