package htmlctest

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// fakeTB is a minimal testing.TB that panics on Fatalf so tests can recover
// and inspect the message. The embedded nil interface satisfies the full
// testing.TB type; only Helper and Fatalf are overridden.
type fakeTB struct {
	testing.TB
	msg    string
	failed bool
}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
	f.failed = true
	panic("fatalf")
}

// runFailing calls fn and recovers from the panic that fakeTB.Fatalf causes.
func runFailing(fn func()) {
	defer func() { recover() }()
	fn()
}

// ----- 1. NewHarness + Fragment + AssertHTML -------------------------------------

func TestNewHarness_Fragment_AssertHTML(t *testing.T) {
	h := NewHarness(t, map[string]string{
		"Greeting.vue": `<template><p>Hello {{ name }}!</p></template>`,
	})
	h.Fragment("Greeting", map[string]any{"name": "World"}).
		AssertHTML("<p>Hello World!</p>")
}

// ----- 2. Build shorthand --------------------------------------------------------

func TestBuild_Shorthand(t *testing.T) {
	// Without wrapping <template> tag – Build should add one.
	Build(t, `<p>hello</p>`).
		Fragment("Root", nil).
		AssertHTML("<p>hello</p>")
}

func TestBuild_AlreadyHasTemplate(t *testing.T) {
	// If the template already has a <template> wrapper, Build must not double-wrap.
	Build(t, `<template><span>ok</span></template>`).
		Fragment("Root", nil).
		AssertHTML("<span>ok</span>")
}

// ----- 3. Find(ByTag) + AssertExists ---------------------------------------------

func TestFind_ByTag_AssertExists(t *testing.T) {
	Build(t, `<div><p>present</p></div>`).
		Fragment("Root", nil).
		Find(ByTag("p")).AssertExists()
}

// ----- 4. Find(ByTag) + AssertNotExists ------------------------------------------

func TestFind_ByTag_AssertNotExists(t *testing.T) {
	// Use v-if with a false condition so the element is not rendered.
	Build(t, `<p v-if="show">visible</p>`).
		Fragment("Root", map[string]any{"show": false}).
		Find(ByTag("p")).AssertNotExists()
}

// ----- 5. Find(ByTag.WithClass) + AssertText -------------------------------------

func TestFind_ByTag_WithClass_AssertText(t *testing.T) {
	Build(t, `<div class="card"><h2 class="title">Alice</h2></div>`).
		Fragment("Root", nil).
		Find(ByTag("h2").WithClass("title")).AssertText("Alice")
}

// ----- 6. Find(ByAttr) + AssertAttr ----------------------------------------------

func TestFind_ByAttr_AssertAttr(t *testing.T) {
	Build(t, `<img src="/img/alice.png" alt="Alice">`).
		Fragment("Root", nil).
		Find(ByAttr("alt", "Alice")).AssertAttr("src", "/img/alice.png")
}

// ----- 7. AssertCount ------------------------------------------------------------

func TestAssertCount(t *testing.T) {
	Build(t, `<ul><li v-for="item in items">{{ item }}</li></ul>`).
		Fragment("Root", map[string]any{"items": []any{"a", "b", "c"}}).
		Find(ByTag("li")).AssertCount(3)
}

// ----- 8. Descendant query -------------------------------------------------------

func TestDescendant_Query(t *testing.T) {
	h := NewHarness(t, map[string]string{
		"Page.vue": `<template>
<ul id="menu"><li>one</li><li>two</li></ul>
<ol id="steps"><li>step1</li></ol>
</template>`,
	})
	r := h.Fragment("Page", nil)

	// li inside ul matches two nodes.
	r.Find(ByTag("li").Descendant(ByTag("ul"))).AssertCount(2)

	// li inside ol matches one node.
	r.Find(ByTag("li").Descendant(ByTag("ol"))).AssertCount(1)
}

// ----- 9. SelectionChecker -------------------------------------------------------

type atLeastChecker struct{ min int }

func (c atLeastChecker) Check(nodes []*html.Node) error {
	if len(nodes) < c.min {
		return fmt.Errorf("want at least %d nodes, got %d", c.min, len(nodes))
	}
	return nil
}

func TestSelectionChecker_Pass(t *testing.T) {
	Build(t, `<ul><li>a</li><li>b</li></ul>`).
		Fragment("Root", nil).
		Find(ByTag("li")).
		Check(atLeastChecker{min: 2})
}

func TestSelectionChecker_Fail(t *testing.T) {
	tb := &fakeTB{}
	h := Build(t, `<ul><li>a</li></ul>`)
	result := h.Fragment("Root", nil)
	sel := result.Find(ByTag("li"))
	sel.t = tb
	runFailing(func() {
		sel.Check(atLeastChecker{min: 5})
	})
	if !tb.failed {
		t.Fatal("expected Check to fail")
	}
	if !strings.Contains(tb.msg, "want at least 5 nodes") {
		t.Errorf("unexpected message: %s", tb.msg)
	}
}

// ----- 10. AssertHTML failure message --------------------------------------------

func TestAssertHTML_FailureMessage(t *testing.T) {
	h := Build(t, `<div><h2>Bob</h2></div>`)
	result := h.Fragment("Root", nil)

	tb := &fakeTB{}
	r2 := &Result{t: tb, html: result.html}
	runFailing(func() {
		r2.AssertHTML(`<div><h2>Alice</h2></div>`)
	})

	if !tb.failed {
		t.Fatal("expected AssertHTML to fail")
	}
	if !strings.Contains(tb.msg, "AssertHTML: trees differ") {
		t.Errorf("expected 'AssertHTML: trees differ' in message, got: %s", tb.msg)
	}
	if !strings.Contains(tb.msg, "first difference at") {
		t.Errorf("expected 'first difference at' in message, got: %s", tb.msg)
	}
}

// ----- 11. ByTag / ByClass / ByAttr delegation -----------------------------------

func TestHarness_ByTag_Delegation(t *testing.T) {
	h := Build(t, `<p>hello</p>`)
	q := h.ByTag("p")
	if q.tag != "p" {
		t.Errorf("ByTag delegation: want tag %q, got %q", "p", q.tag)
	}
}

func TestHarness_ByClass_Delegation(t *testing.T) {
	h := Build(t, `<p class="card">hello</p>`)
	q := h.ByClass("card")
	if len(q.classes) != 1 || q.classes[0] != "card" {
		t.Errorf("ByClass delegation: unexpected query %+v", q)
	}
}

func TestResult_ByTag_Delegation(t *testing.T) {
	r := Build(t, `<p>hello</p>`).Fragment("Root", nil)
	q := r.ByTag("p")
	if q.tag != "p" {
		t.Errorf("Result.ByTag delegation: want tag %q, got %q", "p", q.tag)
	}
}

func TestResult_ByAttr_Delegation(t *testing.T) {
	r := Build(t, `<a href="/x">link</a>`).Fragment("Root", nil)
	q := r.ByAttr("href", "/x")
	if len(q.attrs) != 1 || q.attrs[0].attr != "href" {
		t.Errorf("Result.ByAttr delegation: unexpected query %+v", q)
	}
}

// ----- 12. With adds component ---------------------------------------------------

func TestWith_AddsComponent(t *testing.T) {
	h := NewHarness(t, map[string]string{
		"Page.vue": `<template><MyWidget/></template>`,
	})
	// Page would fail to render before adding MyWidget – add it now.
	h.With("MyWidget.vue", `<template><span>widget</span></template>`)
	h.Fragment("Page", nil).Find(ByTag("span")).AssertText("widget")
}

// ----- 13. Page ------------------------------------------------------------------

func TestPage(t *testing.T) {
	h := NewHarness(t, map[string]string{
		"Index.vue": `<template><!DOCTYPE html><html><head><title>T</title></head><body><p>hello</p></body></html></template>`,
	})
	r := h.Page("Index", nil)
	r.Find(ByTag("p")).AssertText("hello")
}

// ----- 14. Result.HTML() ---------------------------------------------------------

func TestResult_HTML(t *testing.T) {
	r := Build(t, `<p>raw</p>`).Fragment("Root", nil)
	got := r.HTML()
	if !strings.Contains(got, "raw") {
		t.Errorf("HTML() = %q; want it to contain %q", got, "raw")
	}
}

// ----- 15. Result.Document() -----------------------------------------------------

func TestResult_Document(t *testing.T) {
	r := Build(t, `<p>doc</p>`).Fragment("Root", nil)

	// First call parses.
	root := r.Document()
	if root == nil {
		t.Fatal("Document() returned nil")
	}
	// Second call returns the same cached node.
	root2 := r.Document()
	if root != root2 {
		t.Error("Document() should return the same cached node on repeated calls")
	}
}

// ----- Stub tests ----------------------------------------------------------------

func TestStub_NewEngine(t *testing.T) {
	tb := &fakeTB{}
	runFailing(func() {
		NewEngine(tb, nil)
	})
	if !tb.failed {
		t.Fatal("expected NewEngine stub to call Fatalf")
	}
	if !strings.Contains(tb.msg, "htmlctest.NewEngine is removed") {
		t.Errorf("unexpected stub message: %s", tb.msg)
	}
	if !strings.Contains(tb.msg, "NewHarness") {
		t.Errorf("stub message should mention NewHarness: %s", tb.msg)
	}
}

func TestStub_AssertRendersHTML(t *testing.T) {
	tb := &fakeTB{}
	runFailing(func() {
		AssertRendersHTML(tb, nil, "Index", nil, "")
	})
	if !tb.failed {
		t.Fatal("expected AssertRendersHTML stub to call Fatalf")
	}
	if !strings.Contains(tb.msg, "htmlctest.AssertRendersHTML is removed") {
		t.Errorf("unexpected stub message: %s", tb.msg)
	}
}

func TestStub_AssertFragment(t *testing.T) {
	tb := &fakeTB{}
	runFailing(func() {
		AssertFragment(tb, nil, "Card", nil, "")
	})
	if !tb.failed {
		t.Fatal("expected AssertFragment stub to call Fatalf")
	}
	if !strings.Contains(tb.msg, "htmlctest.AssertFragment is removed") {
		t.Errorf("unexpected stub message: %s", tb.msg)
	}
	if !strings.Contains(tb.msg, "Card") {
		t.Errorf("stub message should include component name: %s", tb.msg)
	}
}

// ----- Extra: Query.String -------------------------------------------------------

func TestQuery_String(t *testing.T) {
	q := ByTag("div").WithClass("card").WithAttr("data-id", "1")
	s := q.String()
	if !strings.Contains(s, "tag[div]") {
		t.Errorf("String() = %q; missing tag", s)
	}
	if !strings.Contains(s, ".class[card]") {
		t.Errorf("String() = %q; missing class", s)
	}
	if !strings.Contains(s, "data-id") {
		t.Errorf("String() = %q; missing attr", s)
	}
}

// ----- Extra: AssertCount failure ------------------------------------------------

func TestAssertCount_Fail(t *testing.T) {
	tb := &fakeTB{}
	h := Build(t, `<ul><li>a</li><li>b</li></ul>`)
	result := h.Fragment("Root", nil)
	sel := result.Find(ByTag("li"))
	sel.t = tb
	runFailing(func() {
		sel.AssertCount(5)
	})
	if !tb.failed {
		t.Fatal("expected AssertCount to fail")
	}
	if !strings.Contains(tb.msg, "AssertCount") {
		t.Errorf("unexpected message: %s", tb.msg)
	}
}

// ----- Extra: AssertText failure -------------------------------------------------

func TestAssertText_Fail(t *testing.T) {
	tb := &fakeTB{}
	h := Build(t, `<p>actual text</p>`)
	result := h.Fragment("Root", nil)
	sel := result.Find(ByTag("p"))
	sel.t = tb
	runFailing(func() {
		sel.AssertText("expected text")
	})
	if !tb.failed {
		t.Fatal("expected AssertText to fail")
	}
	if !strings.Contains(tb.msg, "AssertText") {
		t.Errorf("unexpected message: %s", tb.msg)
	}
}

