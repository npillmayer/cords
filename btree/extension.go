package btree

type SumExtension[I SummarizedItem[S], S, E any] interface {
	MagicID() string
	Zero() E
	FromItem(I, S) E
	Add(E, E) E
}

type NO_EXT struct{}
