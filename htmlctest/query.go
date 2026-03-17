package htmlctest

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// Query is an immutable element filter. Build one with [ByTag], [ByClass], or
// [ByAttr] and refine it with [Query.WithClass], [Query.WithAttr], and
// [Query.Descendant]. Combinators return new Query values; the receiver is
// never mutated.
type Query struct {
	tag      string       // required tag name; "" matches any tag
	classes  []string     // all must be present on the element
	attrs    []attrFilter // all must match
	ancestor *Query       // non-nil: element must have an ancestor matching this
}

type attrFilter struct{ attr, value string }

// ByTag creates a Query that matches elements with the given tag name
// (case-insensitive).
func ByTag(name string) Query { return Query{tag: name} }

// ByClass creates a Query that matches elements that have class in their
// class attribute.
func ByClass(class string) Query { return Query{classes: []string{class}} }

// ByAttr creates a Query that matches elements whose attr attribute equals
// value.
func ByAttr(attr, value string) Query { return Query{attrs: []attrFilter{{attr, value}}} }

// WithClass returns a new Query that also requires class to be present.
func (q Query) WithClass(class string) Query {
	newClasses := make([]string, len(q.classes)+1)
	copy(newClasses, q.classes)
	newClasses[len(q.classes)] = class
	return Query{tag: q.tag, classes: newClasses, attrs: q.attrs, ancestor: q.ancestor}
}

// WithAttr returns a new Query that also requires attr to equal value.
func (q Query) WithAttr(attr, value string) Query {
	newAttrs := make([]attrFilter, len(q.attrs)+1)
	copy(newAttrs, q.attrs)
	newAttrs[len(q.attrs)] = attrFilter{attr, value}
	return Query{tag: q.tag, classes: q.classes, attrs: newAttrs, ancestor: q.ancestor}
}

// Descendant returns a new Query that only matches elements that have an
// ancestor satisfying ancestor.
func (q Query) Descendant(ancestor Query) Query {
	anc := ancestor
	return Query{tag: q.tag, classes: q.classes, attrs: q.attrs, ancestor: &anc}
}

// String returns a human-readable description used in failure messages.
func (q Query) String() string {
	var sb strings.Builder
	if q.tag != "" {
		sb.WriteString("tag[")
		sb.WriteString(q.tag)
		sb.WriteByte(']')
	}
	for _, c := range q.classes {
		sb.WriteString(".class[")
		sb.WriteString(c)
		sb.WriteByte(']')
	}
	for _, a := range q.attrs {
		sb.WriteByte('[')
		sb.WriteString(a.attr)
		if a.value != "" {
			sb.WriteByte('=')
			sb.WriteString(a.value)
		}
		sb.WriteByte(']')
	}
	if q.ancestor != nil {
		sb.WriteString(" inside ")
		sb.WriteString(q.ancestor.String())
	}
	return sb.String()
}

// matches reports whether node n satisfies all criteria in q.
func (q Query) matches(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if q.tag != "" && !strings.EqualFold(n.Data, q.tag) {
		return false
	}
	for _, class := range q.classes {
		if !nodeHasClass(n, class) {
			return false
		}
	}
	for _, af := range q.attrs {
		if !nodeHasAttr(n, af.attr, af.value) {
			return false
		}
	}
	if q.ancestor != nil {
		found := false
		for p := n.Parent; p != nil; p = p.Parent {
			if q.ancestor.matches(p) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func nodeHasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
			return false
		}
	}
	return false
}

func nodeHasAttr(n *html.Node, attr, value string) bool {
	for _, a := range n.Attr {
		if a.Key == attr {
			if value == "" {
				return true
			}
			return a.Val == value
		}
	}
	return false
}

// walkNodes does a depth-first walk of root, returning all ElementNodes that
// satisfy q.
func walkNodes(root *html.Node, q Query) []*html.Node {
	var results []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && q.matches(n) {
			results = append(results, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return results
}

// textContent extracts the concatenated text content of n and its descendants,
// normalising whitespace.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return normaliseWS(sb.String())
}

// SelectionChecker is implemented by user-defined assertion types that can be
// passed to [Selection.Check].
type SelectionChecker interface {
	Check(nodes []*html.Node) error
}

// Selection holds the nodes matched by a [Query] and exposes fluent assertion
// methods. Each method calls [testing.TB.Fatalf] on failure and returns the
// receiver to allow chaining.
type Selection struct {
	t     testing.TB
	nodes []*html.Node
}

// AssertExists fails the test if no nodes were matched.
func (s *Selection) AssertExists() *Selection {
	s.t.Helper()
	if len(s.nodes) == 0 {
		f := existenceFailure{wantPresent: true, got: 0}
		s.t.Fatalf("%s", f.format())
	}
	return s
}

// AssertNotExists fails the test if any nodes were matched.
func (s *Selection) AssertNotExists() *Selection {
	s.t.Helper()
	if len(s.nodes) > 0 {
		f := existenceFailure{wantPresent: false, got: len(s.nodes)}
		s.t.Fatalf("%s", f.format())
	}
	return s
}

// AssertCount fails the test if the number of matched nodes is not n.
func (s *Selection) AssertCount(n int) *Selection {
	s.t.Helper()
	if len(s.nodes) != n {
		f := countMismatch{want: n, got: len(s.nodes)}
		s.t.Fatalf("%s", f.format())
	}
	return s
}

// AssertText fails the test if the normalised text content of the first matched
// node is not text.
func (s *Selection) AssertText(text string) *Selection {
	s.t.Helper()
	if len(s.nodes) == 0 {
		s.t.Fatalf("AssertText: no nodes matched")
		return s
	}
	node := s.nodes[0]
	got := textContent(node)
	want := normaliseWS(text)
	if got != want {
		f := textMismatch{want: want, got: got, node: node}
		s.t.Fatalf("%s", f.format())
	}
	return s
}

// AssertAttr fails the test if the first matched node's attr attribute is not
// value.
func (s *Selection) AssertAttr(attr, value string) *Selection {
	s.t.Helper()
	if len(s.nodes) == 0 {
		s.t.Fatalf("AssertAttr: no nodes matched")
		return s
	}
	node := s.nodes[0]
	var got string
	for _, a := range node.Attr {
		if a.Key == attr {
			got = a.Val
			break
		}
	}
	if got != value {
		f := attrMismatch{attr: attr, want: value, got: got, node: node}
		s.t.Fatalf("%s", f.format())
	}
	return s
}

// Nodes returns the raw list of matched nodes.
func (s *Selection) Nodes() []*html.Node {
	return s.nodes
}

// Check passes the matched nodes to checker and fails the test if it returns
// a non-nil error.
func (s *Selection) Check(checker SelectionChecker) *Selection {
	s.t.Helper()
	if err := checker.Check(s.nodes); err != nil {
		s.t.Fatalf("Check: %v", err)
	}
	return s
}
