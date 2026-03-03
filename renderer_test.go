package htmlc

import (
	"strings"
	"testing"
)

// renderTemplate is a helper that parses a template string and renders it.
func renderTemplate(t *testing.T, tmpl string, scope map[string]any) string {
	t.Helper()
	src := "<template>" + tmpl + "</template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := Render(c, scope)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return out
}

func TestRender_MustacheMemberAccess(t *testing.T) {
	// {{ user.name }} renders as the HTML-escaped value of user.name from scope.
	scope := map[string]any{
		"user": map[string]any{"name": "Alice"},
	}
	out := renderTemplate(t, `<span>{{ user.name }}</span>`, scope)
	if !strings.Contains(out, "<span>Alice</span>") {
		t.Errorf("got %q, want it to contain <span>Alice</span>", out)
	}
}

func TestRender_MustacheMemberAccessHTMLEscaped(t *testing.T) {
	// HTML-special characters in the value must be escaped.
	scope := map[string]any{
		"user": map[string]any{"name": "<b>Alice</b>"},
	}
	out := renderTemplate(t, `<span>{{ user.name }}</span>`, scope)
	if !strings.Contains(out, "<span>&lt;b&gt;Alice&lt;/b&gt;</span>") {
		t.Errorf("got %q, want it to contain escaped HTML", out)
	}
}

func TestRender_MustacheArithmeticExpression(t *testing.T) {
	// {{ price * qty }} evaluates the expression and renders the result.
	scope := map[string]any{
		"price": float64(10),
		"qty":   float64(3),
	}
	out := renderTemplate(t, `<span>{{ price * qty }}</span>`, scope)
	if !strings.Contains(out, "<span>30</span>") {
		t.Errorf("got %q, want it to contain <span>30</span>", out)
	}
}

func TestRender_MustacheWhitespaceTrimmed(t *testing.T) {
	// Whitespace inside {{ }} is trimmed; the expression still evaluates.
	scope := map[string]any{"msg": "hello"}
	out := renderTemplate(t, `<p>{{  msg  }}</p>`, scope)
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("got %q, want it to contain <p>hello</p>", out)
	}
}

func TestRender_VText(t *testing.T) {
	// <p v-text="msg"></p> renders as <p>escaped content</p>.
	scope := map[string]any{"msg": "Hello & World"}
	out := renderTemplate(t, `<p v-text="msg"></p>`, scope)
	if !strings.Contains(out, "<p>Hello &amp; World</p>") {
		t.Errorf("got %q, want it to contain <p>Hello &amp; World</p>", out)
	}
}

func TestRender_VTextHTMLEscaped(t *testing.T) {
	// v-text escapes HTML special characters.
	scope := map[string]any{"msg": "<b>bold</b>"}
	out := renderTemplate(t, `<p v-text="msg"></p>`, scope)
	if !strings.Contains(out, "<p>&lt;b&gt;bold&lt;/b&gt;</p>") {
		t.Errorf("got %q, want HTML-escaped content inside <p>", out)
	}
}

func TestRender_VHtml(t *testing.T) {
	// <div v-html="raw"></div> renders raw HTML without escaping.
	scope := map[string]any{"raw": "<b>bold</b>"}
	out := renderTemplate(t, `<div v-html="raw"></div>`, scope)
	if !strings.Contains(out, "<div><b>bold</b></div>") {
		t.Errorf("got %q, want it to contain <div><b>bold</b></div>", out)
	}
}

func TestRender_VHtmlNotEscaped(t *testing.T) {
	// v-html must NOT escape the angle brackets.
	scope := map[string]any{"raw": "<em>text</em>"}
	out := renderTemplate(t, `<div v-html="raw"></div>`, scope)
	if strings.Contains(out, "&lt;") || strings.Contains(out, "&gt;") {
		t.Errorf("got %q, v-html should not escape HTML", out)
	}
}

func TestRender_VTextReplacesChildren(t *testing.T) {
	// v-text replaces existing child content.
	scope := map[string]any{"msg": "replaced"}
	out := renderTemplate(t, `<p v-text="msg">original</p>`, scope)
	if strings.Contains(out, "original") {
		t.Errorf("got %q, v-text should replace child content", out)
	}
	if !strings.Contains(out, "replaced") {
		t.Errorf("got %q, want 'replaced' in output", out)
	}
}

func TestRender_VHtmlReplacesChildren(t *testing.T) {
	// v-html replaces existing child content.
	scope := map[string]any{"raw": "<i>new</i>"}
	out := renderTemplate(t, `<div v-html="raw">original</div>`, scope)
	if strings.Contains(out, "original") {
		t.Errorf("got %q, v-html should replace child content", out)
	}
}

func TestRender_ElementAttributesPreserved(t *testing.T) {
	// Non-directive attributes should pass through to the output.
	scope := map[string]any{}
	out := renderTemplate(t, `<a href="/home" class="nav">link</a>`, scope)
	if !strings.Contains(out, `href="/home"`) {
		t.Errorf("got %q, want href attribute preserved", out)
	}
	if !strings.Contains(out, `class="nav"`) {
		t.Errorf("got %q, want class attribute preserved", out)
	}
}

func TestRender_NestedElements(t *testing.T) {
	// Renderer correctly recurses into nested elements.
	scope := map[string]any{"title": "World"}
	out := renderTemplate(t, `<div><p><span>{{ title }}</span></p></div>`, scope)
	if !strings.Contains(out, "<span>World</span>") {
		t.Errorf("got %q, want nested interpolation to work", out)
	}
}

func TestRender_MultipleInterpolationsInOneText(t *testing.T) {
	// Multiple {{ }} in a single text node should all be evaluated.
	scope := map[string]any{"a": "foo", "b": "bar"}
	out := renderTemplate(t, `<p>{{ a }} and {{ b }}</p>`, scope)
	if !strings.Contains(out, "foo and bar") {
		t.Errorf("got %q, want 'foo and bar'", out)
	}
}

// --- v-if / v-else-if / v-else tests ---

func TestRender_VIfTrue(t *testing.T) {
	// v-if with a truthy expression renders the element.
	out := renderTemplate(t, `<div v-if="true">yes</div>`, nil)
	if !strings.Contains(out, "<div>yes</div>") {
		t.Errorf("got %q, want <div>yes</div>", out)
	}
}

func TestRender_VIfFalse(t *testing.T) {
	// v-if with a falsy expression produces no output.
	out := renderTemplate(t, `<div v-if="false">yes</div>`, nil)
	if strings.Contains(out, "yes") || strings.Contains(out, "<div>") {
		t.Errorf("got %q, want no output for v-if=false", out)
	}
}

func TestRender_VIfElseChain(t *testing.T) {
	// v-if/v-else-if/v-else: only the first truthy branch renders.
	scope := map[string]any{"a": false, "b": true}
	out := renderTemplate(t, `<span v-if="a">A</span><span v-else-if="b">B</span><span v-else>C</span>`, scope)
	if strings.Contains(out, ">A<") || strings.Contains(out, ">C<") {
		t.Errorf("got %q, want only B branch rendered", out)
	}
	if !strings.Contains(out, ">B<") {
		t.Errorf("got %q, want B branch rendered", out)
	}
}

func TestRender_VElseRendersWhenAllFalsy(t *testing.T) {
	// v-else renders when all preceding conditions are false.
	scope := map[string]any{"a": false, "b": false}
	out := renderTemplate(t, `<span v-if="a">A</span><span v-else-if="b">B</span><span v-else>C</span>`, scope)
	if !strings.Contains(out, ">C<") {
		t.Errorf("got %q, want C branch rendered", out)
	}
	if strings.Contains(out, ">A<") || strings.Contains(out, ">B<") {
		t.Errorf("got %q, want only C rendered", out)
	}
}

func TestRender_VIfTemplateWrapper(t *testing.T) {
	// <template v-if="show"> renders children only, not a <template> element.
	scope := map[string]any{"show": true}
	out := renderTemplate(t, `<template v-if="show"><p>a</p><p>b</p></template>`, scope)
	if strings.Contains(out, "<template") {
		t.Errorf("got %q, <template> element must not appear in output", out)
	}
	if !strings.Contains(out, "<p>a</p>") || !strings.Contains(out, "<p>b</p>") {
		t.Errorf("got %q, want both <p> children rendered", out)
	}
}

func TestRender_VIfTemplateWrapperFalse(t *testing.T) {
	// <template v-if="false"> renders nothing.
	scope := map[string]any{"show": false}
	out := renderTemplate(t, `<template v-if="show"><p>a</p></template>`, scope)
	if strings.Contains(out, "<p>") {
		t.Errorf("got %q, want no output when v-if is false", out)
	}
}

func TestRender_VElseOrphanError(t *testing.T) {
	// v-else without a preceding v-if must return a render error.
	src := "<template><div v-else>oops</div></template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, renderErr := Render(c, nil)
	if renderErr == nil {
		t.Error("expected an error for orphan v-else, got nil")
	}
}

func TestRender_VIfScopeExpression(t *testing.T) {
	// v-if evaluates scope variables.
	scope := map[string]any{"visible": true}
	out := renderTemplate(t, `<p v-if="visible">hello</p>`, scope)
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("got %q, want <p>hello</p>", out)
	}
}

func TestRender_VIfOnlyFirstTruthyBranchRenders(t *testing.T) {
	// When v-if is true, subsequent v-else-if/v-else branches must not render.
	scope := map[string]any{"a": true, "b": true}
	out := renderTemplate(t, `<span v-if="a">A</span><span v-else-if="b">B</span><span v-else>C</span>`, scope)
	if !strings.Contains(out, ">A<") {
		t.Errorf("got %q, want A branch rendered", out)
	}
	if strings.Contains(out, ">B<") || strings.Contains(out, ">C<") {
		t.Errorf("got %q, want only first truthy branch (A) rendered", out)
	}
}

// --- v-for tests ---

func TestRender_VForSimpleArray(t *testing.T) {
	// v-for="item in items" renders one element per array entry with item in scope.
	scope := map[string]any{"items": []any{"a", "b", "c"}}
	out := renderTemplate(t, `<ul><li v-for="item in items">{{ item }}</li></ul>`, scope)
	for _, want := range []string{"<li>a</li>", "<li>b</li>", "<li>c</li>"} {
		if !strings.Contains(out, want) {
			t.Errorf("got %q, want it to contain %s", out, want)
		}
	}
}

func TestRender_VForWithIndex(t *testing.T) {
	// v-for="(item, index) in items" provides both item and zero-based index.
	scope := map[string]any{"items": []any{"x", "y"}}
	out := renderTemplate(t, `<span v-for="(item, index) in items">{{ index }}:{{ item }}</span>`, scope)
	if !strings.Contains(out, "0:x") || !strings.Contains(out, "1:y") {
		t.Errorf("got %q, want index:item pairs 0:x and 1:y", out)
	}
}

func TestRender_VForObject(t *testing.T) {
	// v-for="(value, key) in obj" iterates map entries.
	scope := map[string]any{"obj": map[string]any{"a": "1", "b": "2"}}
	out := renderTemplate(t, `<span v-for="(value, key) in obj">{{ key }}={{ value }}</span>`, scope)
	if !strings.Contains(out, "a=1") || !strings.Contains(out, "b=2") {
		t.Errorf("got %q, want key=value pairs a=1 and b=2", out)
	}
}

func TestRender_VForInteger(t *testing.T) {
	// v-for="n in 5" renders 5 copies with n = 1..5.
	out := renderTemplate(t, `<span v-for="n in 5">{{ n }}</span>`, nil)
	for _, want := range []string{"<span>1</span>", "<span>2</span>", "<span>3</span>", "<span>4</span>", "<span>5</span>"} {
		if !strings.Contains(out, want) {
			t.Errorf("got %q, want it to contain %s", out, want)
		}
	}
}

func TestRender_VForKey(t *testing.T) {
	// :key="item.id" renders as data-key="<value>" on each iteration element.
	scope := map[string]any{
		"items": []any{
			map[string]any{"id": float64(1), "name": "a"},
			map[string]any{"id": float64(2), "name": "b"},
		},
	}
	out := renderTemplate(t, `<li v-for="item in items" :key="item.id">{{ item.name }}</li>`, scope)
	if !strings.Contains(out, `data-key="1"`) || !strings.Contains(out, `data-key="2"`) {
		t.Errorf("got %q, want data-key=\"1\" and data-key=\"2\"", out)
	}
}

func TestRender_VForTemplateWrapper(t *testing.T) {
	// <template v-for="..."> renders only children per iteration, no <template> tag.
	scope := map[string]any{"items": []any{"a", "b"}}
	out := renderTemplate(t, `<template v-for="item in items"><p>{{ item }}</p></template>`, scope)
	if strings.Contains(out, "<template") {
		t.Errorf("got %q, <template> element must not appear in output", out)
	}
	if !strings.Contains(out, "<p>a</p>") || !strings.Contains(out, "<p>b</p>") {
		t.Errorf("got %q, want <p>a</p> and <p>b</p>", out)
	}
}

func TestRender_VForEmptyArray(t *testing.T) {
	// Empty array produces no output.
	scope := map[string]any{"items": []any{}}
	out := renderTemplate(t, `<li v-for="item in items">{{ item }}</li>`, scope)
	if strings.Contains(out, "<li>") {
		t.Errorf("got %q, want no output for empty array", out)
	}
}
