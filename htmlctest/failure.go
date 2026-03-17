package htmlctest

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// assertionFailure is the common interface for structured test-failure values.
type assertionFailure interface {
	format() string
}

// existenceFailure is produced by [Selection.AssertExists] and
// [Selection.AssertNotExists].
type existenceFailure struct {
	query       Query
	wantPresent bool
	got         int
}

func (f existenceFailure) format() string {
	if f.wantPresent {
		return "AssertExists: no elements found"
	}
	return fmt.Sprintf("AssertNotExists: %d element(s) found", f.got)
}

// countMismatch is produced by [Selection.AssertCount].
type countMismatch struct {
	query     Query
	want, got int
}

func (f countMismatch) format() string {
	return fmt.Sprintf("AssertCount: want %d element(s), got %d", f.want, f.got)
}

// textMismatch is produced by [Selection.AssertText].
type textMismatch struct {
	want, got string
	node      *html.Node
}

func (f textMismatch) format() string {
	path := elemPath(f.node)
	return fmt.Sprintf("AssertText: text mismatch at %s\n  want: %q\n  got:  %q", path, f.want, f.got)
}

// attrMismatch is produced by [Selection.AssertAttr].
type attrMismatch struct {
	attr, want, got string
	node            *html.Node
}

func (f attrMismatch) format() string {
	path := elemPath(f.node)
	return fmt.Sprintf("AssertAttr: attr %q mismatch at %s\n  want: %q\n  got:  %q", f.attr, path, f.want, f.got)
}

// htmlMismatch is produced by [Result.AssertHTML].
type htmlMismatch struct {
	want, got string
}

func (f htmlMismatch) format() string {
	wantNode, errW := html.Parse(strings.NewReader(f.want))
	gotNode, errG := html.Parse(strings.NewReader(f.got))
	if errW != nil || errG != nil {
		return fmt.Sprintf("AssertHTML: trees differ\n  expected: %s\n  got:      %s", f.want, f.got)
	}

	// Navigate into body content for cleaner paths.
	wantBody := findBody(wantNode)
	gotBody := findBody(gotNode)
	if wantBody == nil || gotBody == nil {
		return fmt.Sprintf("AssertHTML: trees differ\n  expected: %s\n  got:      %s", f.want, f.got)
	}

	diffPath, wantVal, gotVal := findFirstDiff(wantBody, gotBody, nil)
	if diffPath == "" {
		return fmt.Sprintf("AssertHTML: trees differ\n  expected: %s\n  got:      %s", f.want, f.got)
	}
	return fmt.Sprintf(
		"AssertHTML: trees differ\n  expected: %s\n  got:      %s\n  first difference at: %s\n    want: %q\n    got:  %q",
		f.want, f.got, diffPath, wantVal, gotVal,
	)
}

// elemPath builds a path like "div > h2" by walking up from n through parent
// element nodes.
func elemPath(n *html.Node) string {
	var parts []string
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.Type == html.ElementNode {
			parts = append([]string{cur.Data}, parts...)
		}
	}
	return strings.Join(parts, " > ")
}

// findBody returns the <body> element in the parsed tree, or nil.
func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findBody(c); found != nil {
			return found
		}
	}
	return nil
}

// findFirstDiff performs a paired depth-first walk of two HTML trees rooted at
// w and g, returning the path string and the want/got values at the first
// point of difference. pathSoFar is the ancestor path accumulated so far.
func findFirstDiff(w, g *html.Node, pathSoFar []string) (path, wantVal, gotVal string) {
	if w == nil && g == nil {
		return "", "", ""
	}
	if w == nil {
		return strings.Join(pathSoFar, " > "), "(none)", nodeDesc(g)
	}
	if g == nil {
		return strings.Join(pathSoFar, " > "), nodeDesc(w), "(none)"
	}

	// Skip identical whitespace-only text nodes in both.
	if w.Type == html.TextNode && g.Type == html.TextNode {
		ww := normaliseWS(w.Data)
		gw := normaliseWS(g.Data)
		if ww == "" && gw == "" {
			return "", "", ""
		}
		if ww != gw {
			p := append(pathSoFar, "#text") //nolint
			return strings.Join(p, " > "), ww, gw
		}
		return "", "", ""
	}

	if w.Type != g.Type {
		return strings.Join(pathSoFar, " > "), nodeDesc(w), nodeDesc(g)
	}

	if w.Type == html.ElementNode {
		if !strings.EqualFold(w.Data, g.Data) {
			return strings.Join(pathSoFar, " > "), "<" + w.Data + ">", "<" + g.Data + ">"
		}
		childPath := append(pathSoFar, w.Data) //nolint
		wc := firstMeaningfulChild(w)
		gc := firstMeaningfulChild(g)
		for wc != nil || gc != nil {
			if p, wv, gv := findFirstDiff(wc, gc, childPath); p != "" {
				return p, wv, gv
			}
			if wc != nil {
				wc = nextMeaningfulSibling(wc)
			}
			if gc != nil {
				gc = nextMeaningfulSibling(gc)
			}
		}
	}

	return "", "", ""
}

// nodeDesc returns a brief human-readable description of a node.
func nodeDesc(n *html.Node) string {
	if n == nil {
		return "(none)"
	}
	switch n.Type {
	case html.TextNode:
		return normaliseWS(n.Data)
	case html.ElementNode:
		var buf bytes.Buffer
		_ = html.Render(&buf, n)
		return buf.String()
	}
	return fmt.Sprintf("(node type %v)", n.Type)
}

// firstMeaningfulChild returns n's first child that is an element or a
// non-whitespace text node.
func firstMeaningfulChild(n *html.Node) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if isMeaningful(c) {
			return c
		}
	}
	return nil
}

// nextMeaningfulSibling returns the next sibling of n that is meaningful.
func nextMeaningfulSibling(n *html.Node) *html.Node {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if isMeaningful(s) {
			return s
		}
	}
	return nil
}

func isMeaningful(n *html.Node) bool {
	if n.Type == html.ElementNode {
		return true
	}
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data) != ""
	}
	return false
}
