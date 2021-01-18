package html

import (
	"io"

	"github.com/npillmayer/cords"
	"golang.org/x/net/html"
)

// InnerText creates a text cord for the textual content of an HTML element and all
// its descendents. It resembles the text produced by
//
//      document.getElementById("myNode").innerText
//
// in JavaScript (except that FromHTML cannot respect CSS styling suppressing
// the visibility of the node's descendents).
//
// The fragment organization of the resulting cord will reflect the hierarchy of
// the element node's descendents.
//
func InnerText(n *html.Node) (cords.Cord, error) {
	if n == nil {
		return cords.Cord{}, cords.ErrIllegalArguments
	}
	b := cords.NewBuilder()
	collectText(n, b)
	return b.Cord(), nil
}

func collectText(n *html.Node, b *cords.CordBuilder) {
	if n.Type == html.ElementNode {
		//T().Debugf("<%s>", n.Data)
	} else if n.Type == html.TextNode {
		//T().Debugf("text = %s", n.Data)
		b.Append(cords.StringLeaf(n.Data))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, b)
	}
}

func TextFromHTML(input io.Reader) (cords.Cord, error) {
	nodes, err := html.ParseFragment(input, nil)
	if err != nil {
		return cords.Cord{}, err
	}
	b := cords.NewBuilder()
	for _, n := range nodes {
		collectText(n, b)
	}
	return b.Cord(), nil
}

// ---------------------------------------------------------------------------

/*
// Leaf is the leaf type created for cords from calls to html.InnerText(…).
// It is made public as it may be of use for other implementations of cords.
type Leaf string

// Weight of a leaf is its string length in bytes.
func (l Leaf) Weight() uint64 {
	return uint64(len(l))
}

func (l Leaf) String() string {
	return string(l)
}

// Split splits a leaf at position i, resulting in 2 new leafs.
func (l Leaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	left := l[:i]
	right := l[i:]
	return left, right
}

// Substring returns a string segment of the leaf's text fragment.
func (l Leaf) Substring(i, j uint64) []byte {
	return []byte(l)[i:j]
}

var _ cords.Leaf = Leaf("")
*/
