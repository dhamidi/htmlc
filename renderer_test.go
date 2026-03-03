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
