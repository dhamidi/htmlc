package htmlctest

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Result holds the rendered HTML output and supports fluent assertions.
// Obtain one from [Harness.Page] or [Harness.Fragment].
type Result struct {
	t    testing.TB
	html string
	root *html.Node // nil until first Find or Document call
}

// HTML returns the raw rendered HTML string.
func (r *Result) HTML() string {
	return r.html
}

// Document returns the parsed HTML tree, parsing lazily on first call and
// caching the result for subsequent calls.
func (r *Result) Document() *html.Node {
	if r.root == nil {
		node, err := html.Parse(strings.NewReader(r.html))
		if err != nil {
			r.t.Fatalf("htmlctest: failed to parse HTML: %v", err)
		}
		r.root = node
	}
	return r.root
}

// AssertHTML asserts that the rendered HTML equals want after normalising
// whitespace. On mismatch it calls t.Fatalf with a tree-structural diff and
// returns r to allow further assertions.
func (r *Result) AssertHTML(want string) *Result {
	r.t.Helper()
	gotNorm := normaliseWS(r.html)
	wantNorm := normaliseWS(want)
	if gotNorm == wantNorm {
		return r
	}
	f := htmlMismatch{want: wantNorm, got: gotNorm}
	r.t.Fatalf("%s", f.format())
	return r
}

// Find walks the parsed HTML tree and returns a [Selection] of all nodes
// matching q.
func (r *Result) Find(q Query) *Selection {
	root := r.Document()
	nodes := walkNodes(root, q)
	return &Selection{t: r.t, nodes: nodes}
}

// ByTag delegates to the package-level [ByTag] constructor.
func (r *Result) ByTag(name string) Query { return ByTag(name) }

// ByClass delegates to the package-level [ByClass] constructor.
func (r *Result) ByClass(class string) Query { return ByClass(class) }

// ByAttr delegates to the package-level [ByAttr] constructor.
func (r *Result) ByAttr(attr, value string) Query { return ByAttr(attr, value) }

// normaliseWS collapses runs of whitespace (including newlines) into single
// spaces and trims the result. This makes HTML comparisons robust to
// formatting differences.
func normaliseWS(s string) string {
	var sb strings.Builder
	inSpace := false
	for _, ch := range s {
		switch ch {
		case ' ', '\t', '\r', '\n':
			if !inSpace {
				sb.WriteByte(' ')
				inSpace = true
			}
		default:
			inSpace = false
			sb.WriteRune(ch)
		}
	}
	return strings.TrimSpace(sb.String())
}
