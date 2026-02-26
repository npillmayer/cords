package cordext

import (
	"unicode/utf8"

	"github.com/npillmayer/cords/chunk"
)

// splitToChunks splits UTF-8 bytes into chunk-sized pieces.
//
// Boundaries are adjusted so no chunk starts or ends in the middle of a UTF-8
// rune.
func splitToChunks(text []byte) ([]chunk.Chunk, error) {
	if len(text) == 0 {
		return nil, nil
	}
	if !utf8.Valid(text) {
		return nil, chunk.ErrInvalidUTF8
	}
	parts := make([]chunk.Chunk, 0, 1+len(text)/chunk.MaxBase)
	for i := 0; i < len(text); {
		end := i + chunk.MaxBase
		if end >= len(text) {
			end = len(text)
		} else {
			for end > i && !utf8.RuneStart(text[end]) {
				end--
			}
			if end == i {
				return nil, chunk.ErrInvalidUTF8
			}
		}
		c, err := chunk.NewBytes(text[i:end])
		if err != nil {
			return nil, err
		}
		parts = append(parts, c)
		i = end
	}
	return parts, nil
}
