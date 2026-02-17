package inline

import (
	"io"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/styled"
	"golang.org/x/net/html"
)

// InnerText creates a styled text for the textual content of an HTML element and all
// its descendents. It resembles the text produced by
//
//      document.getElementById("myNode").innerText
//
// in JavaScript, except that `InnerText` cannot respect CSS styling (including
// properties changing the visibility of the node's descendents).
// Therefore the resulting styled text is limited to inline span elements like
//    <strong> … </strong>
//    <i> … </i>
// etc. Clients should provide a paragraph-like element.
//
// The fragment organization of the resulting styled text will reflect the hierarchy of
// the element node's descendents.
//
func InnerText(n *html.Node) (*styled.Text, error) {
	if n == nil {
		return nil, cords.ErrIllegalArguments
	}
	b := styled.NewTextBuilder()
	collectText(n, PlainStyle, b)
	return b.Text(), nil
}

func collectText(n *html.Node, style Style, b *styled.TextBuilder) {
	if n.Type == html.ElementNode {
		T().Debugf("styled inline text: collect text of <%s>", n.Data)
		st := StyleFromHTMLName(n.Data)
		if st != PlainStyle {
			style = st
		}
	} else if n.Type == html.TextNode {
		T().Debugf("styled inline text = %s (%v)", n.Data, style)
		b.Append(cords.StringLeaf(n.Data), style)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, style, b)
	}
}

// TextFromHTML creates a styled.Text from the textual content of an HTML fragment.
// The HTML fragment should reflect the content of a paragraph-like element.
func TextFromHTML(input io.Reader) (*styled.Text, error) {
	nodes, err := html.ParseFragment(input, nil)
	if err != nil {
		return nil, err
	}
	b := styled.NewTextBuilder()
	for _, n := range nodes {
		collectText(n, PlainStyle, b)
	}
	return b.Text(), nil
}
