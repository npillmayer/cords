package inline

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"golang.org/x/net/html"
)

func TestHTMLFromTree(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	r := strings.NewReader(`
	<!DOCTYPE html>
	<html>
	<body>

	<h1>My First Heading</h1>
	<p>My <b>first</b> paragraph.</p>

	</body>
	</html>
`)
	doc, err := html.Parse(r)
	if err != nil {
		t.Fatal(err.Error())
	}
	text, err := InnerText(doc)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("text = '%s'", text.Raw())
}

func TestHTMLParse(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "cords")
	defer teardown()
	//
	r := strings.NewReader(`<p>My <b>first</b> paragraph.</p>`)
	text, err := TextFromHTML(r)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("text = '%s'", text.Raw())
}
