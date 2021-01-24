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
// in JavaScript (except that html.InnerText cannot respect CSS styling suppressing
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

// TextFromHTML creates a cords.Cord from the textual content of an HTML fragment.
// It does not interpretation of layout and styling, but extracts the pure text.
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
