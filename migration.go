package cords

import "github.com/npillmayer/cords/chunk"

// chunkLeaf adapts chunk.Chunk to the legacy Leaf interface.
type chunkLeaf struct {
	chunk chunk.Chunk
}

func (leaf chunkLeaf) Weight() uint64 {
	return uint64(leaf.chunk.Len())
}

func (leaf chunkLeaf) String() string {
	return leaf.chunk.String()
}

func (leaf chunkLeaf) Substring(i, j uint64) []byte {
	return leaf.chunk.Bytes()[i:j]
}

func (leaf chunkLeaf) Split(i uint64) (Leaf, Leaf) {
	b := leaf.chunk.Bytes()
	return StringLeaf(string(b[:i])), StringLeaf(string(b[i:]))
}

var _ Leaf = chunkLeaf{}
