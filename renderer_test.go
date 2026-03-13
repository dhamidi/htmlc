package htmlc

import (
	"errors"
	"fmt"
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
	out, err := RenderString(c, scope)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
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
	_, renderErr := RenderString(c, nil)
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

// --- v-bind / :attr tests ---

func TestRender_VBindHref(t *testing.T) {
	// :href="url" renders as href="<value>".
	scope := map[string]any{"url": "https://example.com"}
	out := renderTemplate(t, `<a :href="url">link</a>`, scope)
	if !strings.Contains(out, `href="https://example.com"`) {
		t.Errorf("got %q, want href=\"https://example.com\"", out)
	}
}

func TestRender_VBindClassObjectTruthy(t *testing.T) {
	// :class="{ active: true, hidden: false }" renders only the truthy key.
	scope := map[string]any{}
	out := renderTemplate(t, `<div :class="{ active: true, hidden: false }">x</div>`, scope)
	if !strings.Contains(out, "active") {
		t.Errorf("got %q, want class to contain 'active'", out)
	}
	if strings.Contains(out, "hidden") {
		t.Errorf("got %q, 'hidden' should be omitted", out)
	}
}

func TestRender_VBindClassObjectScope(t *testing.T) {
	// :class with scope variable for condition.
	scope := map[string]any{"isActive": true, "isHidden": false}
	out := renderTemplate(t, `<div :class="{ active: isActive, hidden: isHidden }">x</div>`, scope)
	if !strings.Contains(out, "active") {
		t.Errorf("got %q, want 'active' in class", out)
	}
	if strings.Contains(out, "hidden") {
		t.Errorf("got %q, 'hidden' should be omitted", out)
	}
}

func TestRender_VBindClassArrayTrue(t *testing.T) {
	// :class="['a', condition ? 'b' : '']" renders 'a b' when condition is true.
	scope := map[string]any{"condition": true}
	out := renderTemplate(t, `<div :class="['a', condition ? 'b' : '']">x</div>`, scope)
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("got %q, want class to contain both 'a' and 'b'", out)
	}
}

func TestRender_VBindClassArrayFalse(t *testing.T) {
	// :class="['a', condition ? 'b' : '']" renders 'a' only when condition is false.
	scope := map[string]any{"condition": false}
	out := renderTemplate(t, `<div :class="['a', condition ? 'b' : '']">x</div>`, scope)
	if !strings.Contains(out, "a") {
		t.Errorf("got %q, want class to contain 'a'", out)
	}
	if strings.Contains(out, "b") {
		t.Errorf("got %q, 'b' should be omitted when condition is false", out)
	}
}

func TestRender_VBindStyleObject(t *testing.T) {
	// :style="{ color: 'red', fontSize: '14px' }" renders inline style.
	scope := map[string]any{}
	out := renderTemplate(t, `<p :style="{ color: 'red', fontSize: '14px' }">x</p>`, scope)
	if !strings.Contains(out, "color:red") {
		t.Errorf("got %q, want 'color:red' in style", out)
	}
	if !strings.Contains(out, "font-size:14px") {
		t.Errorf("got %q, want 'font-size:14px' in style", out)
	}
}

func TestRender_VBindDisabledFalse(t *testing.T) {
	// :disabled="false" omits the attribute entirely.
	scope := map[string]any{}
	out := renderTemplate(t, `<button :disabled="false">x</button>`, scope)
	if strings.Contains(out, "disabled") {
		t.Errorf("got %q, 'disabled' should be omitted when falsy", out)
	}
}

func TestRender_VBindDisabledTrue(t *testing.T) {
	// :disabled="true" renders the boolean attribute without a value.
	scope := map[string]any{}
	out := renderTemplate(t, `<button :disabled="true">x</button>`, scope)
	if !strings.Contains(out, "disabled") {
		t.Errorf("got %q, want 'disabled' attribute present", out)
	}
	if strings.Contains(out, `disabled="`) {
		t.Errorf("got %q, boolean attr must not have a value", out)
	}
}

func TestRender_VBindStaticAndDynamicClassMerge(t *testing.T) {
	// Static class="foo" and :class="{ bar: true }" merge to class="foo bar".
	scope := map[string]any{}
	out := renderTemplate(t, `<div class="foo" :class="{ bar: true }">x</div>`, scope)
	if !strings.Contains(out, "foo") {
		t.Errorf("got %q, want 'foo' in class", out)
	}
	if !strings.Contains(out, "bar") {
		t.Errorf("got %q, want 'bar' in class", out)
	}
	// Ensure only one class attribute is emitted.
	if strings.Count(out, `class="`) > 1 {
		t.Errorf("got %q, must have only one class attribute", out)
	}
}

func TestRender_VBindChecked(t *testing.T) {
	// :checked="true" renders checked boolean attr; :checked="false" omits it.
	out := renderTemplate(t, `<input :checked="true">`, nil)
	if !strings.Contains(out, "checked") {
		t.Errorf("got %q, want 'checked' attribute", out)
	}
	out2 := renderTemplate(t, `<input :checked="false">`, nil)
	if strings.Contains(out2, "checked") {
		t.Errorf("got %q, 'checked' should be omitted", out2)
	}
}

func TestRender_VBindDynamicValue(t *testing.T) {
	// :href with a scope variable.
	scope := map[string]any{"link": "/page"}
	out := renderTemplate(t, `<a :href="link">go</a>`, scope)
	if !strings.Contains(out, `href="/page"`) {
		t.Errorf("got %q, want href=\"/page\"", out)
	}
}

// --- v-show tests ---

func TestRender_VShowFalse(t *testing.T) {
	// v-show="false" adds style="display:none" and strips the v-show attribute.
	out := renderTemplate(t, `<div v-show="false">x</div>`, nil)
	if !strings.Contains(out, `style="display:none"`) {
		t.Errorf("got %q, want style=\"display:none\"", out)
	}
	if strings.Contains(out, "v-show") {
		t.Errorf("got %q, v-show must not appear in output", out)
	}
	if !strings.Contains(out, ">x<") {
		t.Errorf("got %q, want child content preserved", out)
	}
}

func TestRender_VShowTrue(t *testing.T) {
	// v-show="true" renders normally without adding any style.
	out := renderTemplate(t, `<div v-show="true">x</div>`, nil)
	if strings.Contains(out, "display") {
		t.Errorf("got %q, v-show=true must not add display style", out)
	}
	if strings.Contains(out, "v-show") {
		t.Errorf("got %q, v-show must not appear in output", out)
	}
}

func TestRender_VShowMergesExistingStyle(t *testing.T) {
	// v-show="false" prepends display:none to an existing static style.
	out := renderTemplate(t, `<div v-show="false" style="color:red">x</div>`, nil)
	if !strings.Contains(out, "display:none") {
		t.Errorf("got %q, want display:none in style", out)
	}
	if !strings.Contains(out, "color:red") {
		t.Errorf("got %q, want color:red preserved in style", out)
	}
	// Only one style attribute.
	if strings.Count(out, `style="`) > 1 {
		t.Errorf("got %q, must have only one style attribute", out)
	}
}

func TestRender_VShowScopeExpression(t *testing.T) {
	// v-show evaluates scope variables.
	scope := map[string]any{"visible": false}
	out := renderTemplate(t, `<p v-show="visible">text</p>`, scope)
	if !strings.Contains(out, "display:none") {
		t.Errorf("got %q, want display:none when visible=false", out)
	}
}

// --- v-pre tests ---

func TestRender_VPreLiteral(t *testing.T) {
	// v-pre emits mustache content literally, without interpolation.
	scope := map[string]any{"raw": "evaluated"}
	out := renderTemplate(t, `<div v-pre>{{ raw }}</div>`, scope)
	if !strings.Contains(out, "{{ raw }}") {
		t.Errorf("got %q, want literal {{ raw }} in output", out)
	}
	if strings.Contains(out, "evaluated") {
		t.Errorf("got %q, v-pre must not interpolate expressions", out)
	}
}

func TestRender_VPreStripsDirective(t *testing.T) {
	// The v-pre attribute itself must not appear in the rendered output.
	out := renderTemplate(t, `<div v-pre>text</div>`, nil)
	if strings.Contains(out, "v-pre") {
		t.Errorf("got %q, v-pre attribute must not appear in output", out)
	}
	if !strings.Contains(out, "<div>text</div>") {
		t.Errorf("got %q, want <div>text</div>", out)
	}
}

func TestRender_VPreSkipsDescendants(t *testing.T) {
	// v-pre skips directive processing for all descendant elements.
	scope := map[string]any{"msg": "hello"}
	out := renderTemplate(t, `<div v-pre><span v-text="msg">{{ msg }}</span></div>`, scope)
	// Descendant v-text should NOT be processed.
	if !strings.Contains(out, "v-text") {
		t.Errorf("got %q, v-text on descendant should pass through verbatim", out)
	}
	// Mustache should not be interpolated.
	if strings.Contains(out, "hello") {
		t.Errorf("got %q, v-pre must not interpolate inside descendants", out)
	}
}

// --- v-once tests ---

func TestRender_VOnceRendersNormally(t *testing.T) {
	// v-once renders expressions normally in a server-side context.
	scope := map[string]any{"msg": "hello"}
	out := renderTemplate(t, `<p v-once>{{ msg }}</p>`, scope)
	if !strings.Contains(out, "hello") {
		t.Errorf("got %q, v-once must render expression normally", out)
	}
	if strings.Contains(out, "v-once") {
		t.Errorf("got %q, v-once attribute must not appear in output", out)
	}
}

// --- client-side directive stripping tests ---

func TestRender_VModelPassthrough(t *testing.T) {
	// v-model must be stripped from server-side rendered output.
	scope := map[string]any{"name": "Alice"}
	out := renderTemplate(t, `<input v-model="name">`, scope)
	if strings.Contains(out, `v-model`) {
		t.Errorf("got %q, want v-model stripped from output", out)
	}
}

func TestRender_AtEventPassthrough(t *testing.T) {
	// @click shorthand must be stripped from server-side rendered output.
	out := renderTemplate(t, `<button @click="handler">click</button>`, nil)
	if strings.Contains(out, `@click`) {
		t.Errorf("got %q, want @click stripped from output", out)
	}
}

func TestRender_VOnEventPassthrough(t *testing.T) {
	// v-on:click must be stripped from server-side rendered output.
	out := renderTemplate(t, `<button v-on:click="handler">click</button>`, nil)
	if strings.Contains(out, `v-on:click`) {
		t.Errorf("got %q, want v-on:click stripped from output", out)
	}
}

// --- component composition and slots tests ---

// mustParseComponent is a test helper that parses a .vue source into a *Component.
func mustParseComponent(t *testing.T, path, src string) *Component {
	t.Helper()
	c, err := ParseFile(path, "<template>"+src+"</template>")
	if err != nil {
		t.Fatalf("ParseFile %s: %v", path, err)
	}
	return c
}

func TestRender_ComponentDynamicProp(t *testing.T) {
	// <Card :title="t"> renders Card's template with title in scope.
	card := mustParseComponent(t, "card.vue", `<div class="card"><h1>{{ title }}</h1></div>`)
	main := mustParseComponent(t, "main.vue", `<Card :title="t"></Card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Card": card}).RenderString(map[string]any{"t": "Hello"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<h1>Hello</h1>") {
		t.Errorf("got %q, want <h1>Hello</h1>", out)
	}
}

func TestRender_ComponentSlot(t *testing.T) {
	// <slot /> in the child component emits the caller's inner content.
	card := mustParseComponent(t, "card.vue", `<div class="card"><slot /></div>`)
	main := mustParseComponent(t, "main.vue", `<Card>inner content</Card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "inner content") {
		t.Errorf("got %q, want 'inner content' in output", out)
	}
	if !strings.Contains(out, `class="card"`) {
		t.Errorf("got %q, want Card wrapper rendered", out)
	}
}

func TestRender_ComponentStaticAttr(t *testing.T) {
	// Static attributes like <Card class="x"> pass class as a string prop.
	card := mustParseComponent(t, "card.vue", `<div>{{ class }}</div>`)
	main := mustParseComponent(t, "main.vue", `<Card class="x"></Card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, ">x<") {
		t.Errorf("got %q, want class value 'x' rendered in output", out)
	}
}

func TestRender_ComponentKebabCase(t *testing.T) {
	// <my-card> resolves to a component registered as MyCard.
	card := mustParseComponent(t, "my-card.vue", `<section>{{ label }}</section>`)
	main := mustParseComponent(t, "main.vue", `<my-card :label="lbl"></my-card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"MyCard": card}).RenderString(map[string]any{"lbl": "kebab"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<section>kebab</section>") {
		t.Errorf("got %q, want <section>kebab</section>", out)
	}
}

func TestRender_ComponentKebabCaseDirectMatch(t *testing.T) {
	// <my-card> also resolves to a component registered as "my-card".
	card := mustParseComponent(t, "my-card.vue", `<section>direct</section>`)
	main := mustParseComponent(t, "main.vue", `<my-card></my-card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"my-card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<section>direct</section>") {
		t.Errorf("got %q, want <section>direct</section>", out)
	}
}

func TestRender_ComponentUnknown(t *testing.T) {
	// A kebab-case tag not found in the registry must return an error.
	main := mustParseComponent(t, "main.vue", `<unknown-widget></unknown-widget>`)
	_, err := NewRenderer(main).WithComponents(Registry{}).RenderString(nil)
	if err == nil {
		t.Error("expected an error for unknown component, got nil")
	}
}

func TestRender_ComponentNested(t *testing.T) {
	// A component's template may use other components from the same registry.
	inner := mustParseComponent(t, "inner.vue", `<em>{{ text }}</em>`)
	outer := mustParseComponent(t, "outer.vue", `<div><Inner :text="msg"></Inner></div>`)
	main := mustParseComponent(t, "main.vue", `<Outer :msg="greeting"></Outer>`)
	reg := Registry{"Inner": inner, "Outer": outer}
	out, err := NewRenderer(main).WithComponents(reg).RenderString(map[string]any{"greeting": "hi"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<em>hi</em>") {
		t.Errorf("got %q, want <em>hi</em>", out)
	}
}

func TestRender_ComponentSlotWithExpression(t *testing.T) {
	// Slot content that contains interpolation is evaluated in the caller's scope.
	card := mustParseComponent(t, "card.vue", `<div><slot /></div>`)
	main := mustParseComponent(t, "main.vue", `<Card>{{ val }}</Card>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Card": card}).RenderString(map[string]any{"val": "dynamic"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "dynamic") {
		t.Errorf("got %q, want 'dynamic' in output", out)
	}
}

func TestRender_ComponentSlotVIfVElse(t *testing.T) {
	// v-if/v-else inside slot content must evaluate correctly: only one branch renders.
	layout := mustParseComponent(t, "layout.vue", `<div><slot /></div>`)
	page := mustParseComponent(t, "page.vue", `<Layout><p v-if="items.length === 0">empty</p><ul v-else><li v-for="x in items">{{ x }}</li></ul></Layout>`)
	reg := Registry{"Layout": layout}

	// When items is empty: "empty" renders, list does not.
	outEmpty, err := NewRenderer(page).WithComponents(reg).RenderString(map[string]any{"items": []any{}})
	if err != nil {
		t.Fatalf("Render (empty): %v", err)
	}
	if !strings.Contains(outEmpty, "empty") {
		t.Errorf("empty case: got %q, want 'empty'", outEmpty)
	}
	if strings.Contains(outEmpty, "<ul") {
		t.Errorf("empty case: got %q, unexpected <ul>", outEmpty)
	}

	// When items is non-empty: list renders, "empty" does not.
	outFull, err := NewRenderer(page).WithComponents(reg).RenderString(map[string]any{"items": []any{"a", "b"}})
	if err != nil {
		t.Fatalf("Render (full): %v", err)
	}
	if strings.Contains(outFull, "empty") {
		t.Errorf("full case: got %q, unexpected 'empty'", outFull)
	}
	if !strings.Contains(outFull, "<ul") {
		t.Errorf("full case: got %q, want <ul>", outFull)
	}
}

func TestRender_ComponentPascalCaseMultiWord(t *testing.T) {
	// <PostCard> is lowercased by the HTML parser to "postcard".
	// resolveComponent must find the "PostCard" registry entry via case-insensitive lookup.
	postCard := mustParseComponent(t, "PostCard.vue", `<article><h2>{{ title }}</h2></article>`)
	main := mustParseComponent(t, "main.vue", `<PostCard title="Hello" />`)
	out, err := NewRenderer(main).WithComponents(Registry{"PostCard": postCard}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<article>") {
		t.Errorf("got %q, want PostCard template expanded (contains <article>)", out)
	}
	if !strings.Contains(out, "<h2>Hello</h2>") {
		t.Errorf("got %q, want <h2>Hello</h2>", out)
	}
}

func TestRender_ComponentPascalCaseVFor(t *testing.T) {
	// v-for with <PostCard :title="item.title" :slug="item.slug" /> produces
	// one expanded <article> per item, not raw <postcard> tags.
	postCard := mustParseComponent(t, "PostCard.vue", `<article><h2>{{ title }}</h2></article>`)
	main := mustParseComponent(t, "main.vue",
		`<div><PostCard v-for="item in posts" :title="item.title" :slug="item.slug" /></div>`)
	posts := []any{
		map[string]any{"title": "First", "slug": "first"},
		map[string]any{"title": "Second", "slug": "second"},
	}
	out, err := NewRenderer(main).WithComponents(Registry{"PostCard": postCard}).RenderString(map[string]any{"posts": posts})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "<postcard") {
		t.Errorf("got %q, PostCard must be expanded, not rendered as raw <postcard>", out)
	}
	if strings.Count(out, "<article>") != 2 {
		t.Errorf("got %q, want exactly 2 <article> elements", out)
	}
	if !strings.Contains(out, "First") || !strings.Contains(out, "Second") {
		t.Errorf("got %q, want both post titles in output", out)
	}
}

func TestRender_VIfSliceLengthEmpty(t *testing.T) {
	// v-if="posts.length === 0" renders the element when the slice is empty.
	scope := map[string]any{"posts": []any{}}
	out := renderTemplate(t, `<p v-if="posts.length === 0">No posts yet.</p>`, scope)
	if !strings.Contains(out, "No posts yet.") {
		t.Errorf("got %q, want 'No posts yet.' when posts is empty", out)
	}
}

func TestRender_VIfSliceLengthNonEmpty(t *testing.T) {
	// v-if="posts.length === 0" hides the element when the slice is non-empty.
	scope := map[string]any{"posts": []any{"a", "b"}}
	out := renderTemplate(t, `<p v-if="posts.length === 0">No posts yet.</p>`, scope)
	if strings.Contains(out, "No posts yet.") {
		t.Errorf("got %q, element should be hidden when posts is non-empty", out)
	}
}

// --- missing prop validation tests ---

func TestRender_AllPropsProvided(t *testing.T) {
	// Rendering with all props provided succeeds (no regression).
	c := mustParseComponent(t, "test.vue", `<p>{{ greeting }}, {{ name }}!</p>`)
	out, err := NewRenderer(c).RenderString(map[string]any{"greeting": "Hello", "name": "World"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "Hello, World!") {
		t.Errorf("got %q, want 'Hello, World!'", out)
	}
}

func TestRender_MissingPropError(t *testing.T) {
	// Without any handler, missing prop renders "[missing: name]" placeholder.
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	out, err := NewRenderer(c).RenderString(map[string]any{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "[missing: name]") {
		t.Errorf("got %q, want output to contain '[missing: name]'", out)
	}
}

func TestRender_MissingPropSubstitute(t *testing.T) {
	// SubstituteMissingProp injects "MISSING PROP: <name>" placeholder.
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	out, err := NewRenderer(c).WithMissingPropHandler(SubstituteMissingProp).RenderString(map[string]any{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "MISSING PROP: name") {
		t.Errorf("got %q, want 'MISSING PROP: name'", out)
	}
}

func TestRender_MissingPropNilHandler(t *testing.T) {
	// A handler that returns (nil, nil) succeeds (treats missing props as nil).
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	_, err := NewRenderer(c).WithMissingPropHandler(func(string) (any, error) {
		return nil, nil
	}).RenderString(map[string]any{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestRender_MissingPropHandlerError(t *testing.T) {
	// A handler that returns an error propagates that error.
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	_, err := NewRenderer(c).WithMissingPropHandler(func(string) (any, error) {
		return nil, fmt.Errorf("prop not allowed")
	}).RenderString(map[string]any{})
	if err == nil {
		t.Fatal("expected error from handler, got nil")
	}
	if !strings.Contains(err.Error(), "prop not allowed") {
		t.Errorf("error %q should contain handler error message", err.Error())
	}
}

func TestRender_ChildComponentMissingProp(t *testing.T) {
	// Without any handler, missing prop in a child component renders "[missing: label]" placeholder.
	child := mustParseComponent(t, "child.vue", `<span>{{ label }}</span>`)
	parent := mustParseComponent(t, "parent.vue", `<Child></Child>`)
	out, err := NewRenderer(parent).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "[missing: label]") {
		t.Errorf("got %q, want output to contain '[missing: label]'", out)
	}
}

func TestRender_MissingPropDefaultPlaceholder(t *testing.T) {
	// Without any handler, missing prop renders "[missing: <name>]".
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	out, err := NewRenderer(c).RenderString(map[string]any{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "[missing: name]") {
		t.Errorf("got %q, want output to contain '[missing: name]'", out)
	}
}

func TestRender_ErrorOnMissingPropHandler(t *testing.T) {
	// ErrorOnMissingProp handler restores strict validation.
	c := mustParseComponent(t, "test.vue", `<p>{{ name }}</p>`)
	_, err := NewRenderer(c).WithMissingPropHandler(ErrorOnMissingProp).RenderString(map[string]any{})
	if err == nil {
		t.Fatal("expected error from ErrorOnMissingProp handler, got nil")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error %q should mention prop name", err.Error())
	}
}

// --- parseBindingPattern tests ---

func TestParseBindingPattern_Empty(t *testing.T) {
	bindingVar, bindings, err := parseBindingPattern("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bindingVar != "" {
		t.Errorf("got bindingVar=%q, want empty", bindingVar)
	}
	if len(bindings) != 0 {
		t.Errorf("got bindings=%v, want none", bindings)
	}
}

func TestParseBindingPattern_SingleIdentifier(t *testing.T) {
	bindingVar, bindings, err := parseBindingPattern("props")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bindingVar != "props" {
		t.Errorf("got bindingVar=%q, want %q", bindingVar, "props")
	}
	if len(bindings) != 0 {
		t.Errorf("got bindings=%v, want none", bindings)
	}
}

func TestParseBindingPattern_Destructured(t *testing.T) {
	bindingVar, bindings, err := parseBindingPattern("{ user, index }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bindingVar != "" {
		t.Errorf("got bindingVar=%q, want empty", bindingVar)
	}
	if len(bindings) != 2 || bindings[0] != "user" || bindings[1] != "index" {
		t.Errorf("got bindings=%v, want [user index]", bindings)
	}
}

func TestParseBindingPattern_WhitespaceTolerated(t *testing.T) {
	_, bindings, err := parseBindingPattern("{  user ,  index  }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 2 || bindings[0] != "user" || bindings[1] != "index" {
		t.Errorf("got bindings=%v, want [user index]", bindings)
	}
}

func TestParseBindingPattern_SingleVariableDestructure(t *testing.T) {
	_, bindings, err := parseBindingPattern("{ item }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 1 || bindings[0] != "item" {
		t.Errorf("got bindings=%v, want [item]", bindings)
	}
}

func TestParseBindingPattern_InvalidEmpty(t *testing.T) {
	_, _, err := parseBindingPattern("{ }")
	if err == nil {
		t.Error("expected error for empty destructure, got nil")
	}
}

func TestParseBindingPattern_InvalidTrailingComma(t *testing.T) {
	_, _, err := parseBindingPattern("{ a, }")
	if err == nil {
		t.Error("expected error for trailing comma, got nil")
	}
}

func TestParseBindingPattern_InvalidStartsWithDigit(t *testing.T) {
	_, _, err := parseBindingPattern("123")
	if err == nil {
		t.Error("expected error for pattern starting with digit, got nil")
	}
}

func TestParseBindingPattern_InvalidSpacedIdentifiers(t *testing.T) {
	_, _, err := parseBindingPattern("{ a b }")
	if err == nil {
		t.Error("expected error for space-separated identifiers without comma, got nil")
	}
}

// --- parseSlotDirective tests ---

func TestParseSlotDirective_VSlot(t *testing.T) {
	name, ok := parseSlotDirective("v-slot")
	if !ok {
		t.Error("expected isSlotDirective=true for v-slot")
	}
	if name != "default" {
		t.Errorf("got name=%q, want %q", name, "default")
	}
}

func TestParseSlotDirective_VSlotNamed(t *testing.T) {
	name, ok := parseSlotDirective("v-slot:header")
	if !ok {
		t.Error("expected isSlotDirective=true for v-slot:header")
	}
	if name != "header" {
		t.Errorf("got name=%q, want %q", name, "header")
	}
}

func TestParseSlotDirective_HashNamed(t *testing.T) {
	name, ok := parseSlotDirective("#header")
	if !ok {
		t.Error("expected isSlotDirective=true for #header")
	}
	if name != "header" {
		t.Errorf("got name=%q, want %q", name, "header")
	}
}

func TestParseSlotDirective_HashDefault(t *testing.T) {
	name, ok := parseSlotDirective("#default")
	if !ok {
		t.Error("expected isSlotDirective=true for #default")
	}
	if name != "default" {
		t.Errorf("got name=%q, want %q", name, "default")
	}
}

func TestParseSlotDirective_NonSlot(t *testing.T) {
	name, ok := parseSlotDirective("class")
	if ok {
		t.Errorf("expected isSlotDirective=false for 'class', got name=%q", name)
	}
	if name != "" {
		t.Errorf("got name=%q, want empty string", name)
	}
}

func TestParseSlotDirective_VBind(t *testing.T) {
	_, ok := parseSlotDirective("v-bind:title")
	if ok {
		t.Error("expected isSlotDirective=false for v-bind:title")
	}
}

func TestRender_ComponentLayoutSlot(t *testing.T) {
	// <Layout title="My Blog"><p>content</p></Layout> renders Layout's template
	// with {{ title }} = "My Blog" and <slot /> filled with <p>content</p>.
	layout := mustParseComponent(t, "Layout.vue", `<div class="layout"><h1>{{ title }}</h1><slot /></div>`)
	main := mustParseComponent(t, "main.vue", `<Layout title="My Blog"><p>content</p></Layout>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Layout": layout}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<h1>My Blog</h1>") {
		t.Errorf("got %q, want <h1>My Blog</h1>", out)
	}
	if !strings.Contains(out, "<p>content</p>") {
		t.Errorf("got %q, want <p>content</p> in slot", out)
	}
}

func TestRender_NamedSlots(t *testing.T) {
	// Component with named slots receives content via <template v-slot:name>;
	// remaining children fill the default slot.
	// Note: explicit </slot> closing tags are required because the Go HTML
	// parser nests adjacent self-closing <slot/> elements inside each other.
	comp := mustParseComponent(t, "comp.vue",
		`<div><slot name="header"></slot><slot></slot><slot name="footer"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template v-slot:header><h1>Header</h1></template><p>Body</p><template v-slot:footer><em>Footer</em></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<h1>Header</h1>") {
		t.Errorf("got %q, want <h1>Header</h1> from header slot", out)
	}
	if !strings.Contains(out, "<p>Body</p>") {
		t.Errorf("got %q, want <p>Body</p> from default slot", out)
	}
	if !strings.Contains(out, "<em>Footer</em>") {
		t.Errorf("got %q, want <em>Footer</em> from footer slot", out)
	}
}

func TestRender_NamedSlotsHashSyntax(t *testing.T) {
	// #name shorthand for v-slot:name works the same way.
	comp := mustParseComponent(t, "comp.vue",
		`<div><slot name="header"></slot><slot></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template #header><h2>Title</h2></template><p>content</p></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<h2>Title</h2>") {
		t.Errorf("got %q, want <h2>Title</h2> from header slot", out)
	}
	if !strings.Contains(out, "<p>content</p>") {
		t.Errorf("got %q, want <p>content</p> from default slot", out)
	}
}

func TestRender_SlotFallbackWhenMissing(t *testing.T) {
	// <slot name="header">Default Header</slot> renders its fallback children
	// when no matching slot definition is provided.
	comp := mustParseComponent(t, "comp.vue", `<div><slot name="header"><span>Default</span></slot></div>`)
	main := mustParseComponent(t, "main.vue", `<Comp></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<span>Default</span>") {
		t.Errorf("got %q, want fallback '<span>Default</span>'", out)
	}
}

func TestRender_NamedSlotOverridesFallback(t *testing.T) {
	// Providing content for a named slot overrides its fallback children.
	comp := mustParseComponent(t, "comp.vue", `<div><slot name="header"><span>Default</span></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template v-slot:header><h1>Custom</h1></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<h1>Custom</h1>") {
		t.Errorf("got %q, want '<h1>Custom</h1>'", out)
	}
	if strings.Contains(out, "<span>Default</span>") {
		t.Errorf("got %q, fallback should be replaced by provided slot content", out)
	}
}

// --- scoped slot tests ---

func TestRender_ScopedSlotDestructured(t *testing.T) {
	// Child passes :user="theuser" on <slot>; caller receives via #default="{ user }".
	// Uses all-lowercase prop names because the HTML parser lowercases attribute keys.
	child := mustParseComponent(t, "child.vue", `<div><slot :user="theuser"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child :theuser="alice"><template #default="{ user }"><p>{{ user.name }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"alice": map[string]any{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>Alice</p>") {
		t.Errorf("got %q, want <p>Alice</p>", out)
	}
}

func TestRender_ScopedSlotSingleVar(t *testing.T) {
	// v-slot="slotProps" binds the entire slot props map; slotProps.msg is accessible.
	child := mustParseComponent(t, "child.vue", `<div><slot :msg="greeting"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child greeting="Hello"><template v-slot="slotProps"><p>{{ slotProps.msg }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>Hello</p>") {
		t.Errorf("got %q, want <p>Hello</p>", out)
	}
}

func TestRender_ScopedSlotInsideVFor(t *testing.T) {
	// Scoped slot inside v-for: each iteration passes different slot prop values.
	child := mustParseComponent(t, "child.vue", `<ul><li v-for="item in items"><slot :item="item"></slot></li></ul>`)
	main := mustParseComponent(t, "main.vue",
		`<List :items="items"><template #default="{ item }"><span>{{ item }}</span></template></List>`)
	out, err := NewRenderer(main).WithComponents(Registry{"List": child}).RenderString(map[string]any{
		"items": []any{"a", "b", "c"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{"<span>a</span>", "<span>b</span>", "<span>c</span>"} {
		if !strings.Contains(out, want) {
			t.Errorf("got %q, want it to contain %s", out, want)
		}
	}
}

func TestRender_ScopedSlotParentScopeAccessible(t *testing.T) {
	// Slot content can access both parent scope variables and slot props.
	// Uses all-lowercase names because the HTML parser lowercases attribute keys.
	child := mustParseComponent(t, "child.vue", `<div><slot :childval="42"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child><template #default="{ childval }"><p>{{ parentval }}-{{ childval }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"parentval": "hello",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>hello-42</p>") {
		t.Errorf("got %q, want <p>hello-42</p>", out)
	}
}

func TestRender_ScopedSlotPropOverridesParentVar(t *testing.T) {
	// Slot prop with the same name as a parent scope variable wins within slot content.
	// Uses all-lowercase names because the HTML parser lowercases attribute keys.
	child := mustParseComponent(t, "child.vue", `<div><slot :name="childname"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child childname="override"><template #default="{ name }"><p>{{ name }}</p></template></Child>`)
	// parent scope has name="parent", but slot prop name="override" should win
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"name": "parent",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>override</p>") {
		t.Errorf("got %q, want <p>override</p> (slot prop wins over parent scope)", out)
	}
}

func TestRender_NamedScopedSlot(t *testing.T) {
	// Named scoped slot: <slot name="item" :user="theuser" /> with <template #item="{ user }">.
	// Uses all-lowercase prop names because the HTML parser lowercases attribute keys.
	child := mustParseComponent(t, "child.vue", `<div><slot name="item" :user="theuser"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child :theuser="alice"><template #item="{ user }"><p>{{ user.name }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"alice": map[string]any{"name": "Alice"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>Alice</p>") {
		t.Errorf("got %q, want <p>Alice</p>", out)
	}
}

func TestRender_ScopedSlotNoBinding(t *testing.T) {
	// When slot has no binding pattern, slot props are discarded; parent scope is used.
	child := mustParseComponent(t, "child.vue", `<div><slot :ignored="42"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child><p>{{ parentVal }}</p></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"parentVal": "visible",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>visible</p>") {
		t.Errorf("got %q, want <p>visible</p>", out)
	}
}

func TestRender_ScopedSlotStaticAttr(t *testing.T) {
	// Static attributes on <slot> (other than name) are included as string slot props.
	child := mustParseComponent(t, "child.vue", `<div><slot label="static"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child><template #default="{ label }"><p>{{ label }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>static</p>") {
		t.Errorf("got %q, want <p>static</p>", out)
	}
}

func TestRender_ScopedSlotDestructuredMissingKey(t *testing.T) {
	// Destructured binding with a key not in slot props yields nil (renders as "null").
	child := mustParseComponent(t, "child.vue", `<div><slot :present="1"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child><template #default="{ present, missing }"><p>{{ present }}-{{ missing }}</p></template></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>1-null</p>") {
		t.Errorf("got %q, want <p>1-null</p>", out)
	}
}

// --- v-slot on component tag tests ---

func TestRender_VSlotOnComponentTag_Destructured(t *testing.T) {
	// v-slot="{ user, index }" on the component tag: all children are default slot content.
	// The main scoped slot example from the spec.
	child := mustParseComponent(t, "child.vue", `<ul><li v-for="(item, idx) in items"><slot :user="item" :index="idx"></slot></li></ul>`)
	main := mustParseComponent(t, "main.vue",
		`<List :items="items" v-slot="{ user, index }"><span>{{ user.name }}</span></List>`)
	out, err := NewRenderer(main).WithComponents(Registry{"List": child}).RenderString(map[string]any{
		"items": []any{
			map[string]any{"name": "Alice"},
			map[string]any{"name": "Bob"},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<span>Alice</span>") {
		t.Errorf("got %q, want <span>Alice</span>", out)
	}
	if !strings.Contains(out, "<span>Bob</span>") {
		t.Errorf("got %q, want <span>Bob</span>", out)
	}
}

func TestRender_VSlotOnComponentTag_HashDefault(t *testing.T) {
	// #default="{ item }" on component tag works identically to v-slot="{ item }".
	child := mustParseComponent(t, "child.vue", `<div><slot :item="theitem"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child :theitem="val" #default="{ item }"><p>{{ item }}</p></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(map[string]any{
		"val": "hello",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("got %q, want <p>hello</p>", out)
	}
}

func TestRender_VSlotOnComponentTag_NoBinding(t *testing.T) {
	// v-slot (no value) on component tag: all children are default slot, no binding.
	child := mustParseComponent(t, "child.vue", `<div><slot></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child v-slot><p>static content</p></Child>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>static content</p>") {
		t.Errorf("got %q, want <p>static content</p>", out)
	}
}

func TestRender_VSlotOnComponentTag_MixedError(t *testing.T) {
	// Mixing v-slot on the component tag with <template #header> children is invalid.
	// Note: the HTML parser lowercases tag names, so "Child" becomes "child" in the error.
	child := mustParseComponent(t, "child.vue", `<div><slot name="header"></slot><slot></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Child v-slot="{ x }"><template #header><h1>Title</h1></template></Child>`)
	_, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(nil)
	if err == nil {
		t.Fatal("expected error for mixed v-slot usage, got nil")
	}
	want := `v-slot on component tag cannot be mixed with named slot templates`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("got error %q, want it to contain %q", err.Error(), want)
	}
}

// --- binding pattern edge cases ---

func TestParseBindingPattern_ExtraWhitespaceEverywhere(t *testing.T) {
	// Extra whitespace outside and inside the braces is tolerated.
	_, bindings, err := parseBindingPattern("  {  a  ,  b  }  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bindings) != 2 || bindings[0] != "a" || bindings[1] != "b" {
		t.Errorf("got bindings=%v, want [a b]", bindings)
	}
}

func TestRender_VSlotOnTemplate_NoValue(t *testing.T) {
	// <template v-slot> (no = sign) creates the default slot with no binding;
	// slot content is rendered with the parent scope unchanged.
	comp := mustParseComponent(t, "comp.vue", `<div><slot :ignored="42"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template v-slot><p>{{ label }}</p></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(map[string]any{
		"label": "hello",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("got %q, want <p>hello</p>", out)
	}
}

// --- named slot edge cases ---

func TestRender_MultipleNamedSlotsSameName_LastWins(t *testing.T) {
	// When two <template #name> elements share the same slot name, the last
	// one in document order wins (overwrites the first).
	comp := mustParseComponent(t, "comp.vue", `<div><slot name="title"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template #title><h1>first</h1></template><template #title><h2>last</h2></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "<h1>first</h1>") {
		t.Errorf("got %q, first duplicate slot must be overwritten", out)
	}
	if !strings.Contains(out, "<h2>last</h2>") {
		t.Errorf("got %q, want <h2>last</h2> from last duplicate slot", out)
	}
}

func TestRender_TemplatePlainTransparent(t *testing.T) {
	// A <template> element without any directive renders its children
	// transparently — no <template> wrapper tag appears in the output.
	comp := mustParseComponent(t, "comp.vue", `<div><slot></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template><p>first</p><p>second</p></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "<template") {
		t.Errorf("got %q, bare <template> must not appear in output", out)
	}
	if !strings.Contains(out, "<p>first</p>") || !strings.Contains(out, "<p>second</p>") {
		t.Errorf("got %q, want both paragraphs in output", out)
	}
}

func TestRender_NamedSlotExplicitDefault(t *testing.T) {
	// <template #default> explicitly names the default slot — equivalent to
	// providing default slot content without a template wrapper.
	comp := mustParseComponent(t, "comp.vue", `<div><slot name="header"><em>fallback</em></slot><slot></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template #default><p>body</p></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// The header slot has no provided content, so its fallback renders.
	if !strings.Contains(out, "<em>fallback</em>") {
		t.Errorf("got %q, want header fallback <em>fallback</em>", out)
	}
	// The explicit #default content renders in the default slot.
	if !strings.Contains(out, "<p>body</p>") {
		t.Errorf("got %q, want <p>body</p> from #default slot", out)
	}
}

// --- scoped slot edge cases ---

func TestRender_ScopedSlot_ExtraKeysIgnored(t *testing.T) {
	// Slot props map has keys not in the destructuring binding pattern —
	// extra keys are silently ignored.
	comp := mustParseComponent(t, "comp.vue", `<div><slot :a="1" :b="2" :c="3"></slot></div>`)
	main := mustParseComponent(t, "main.vue",
		`<Comp><template #default="{ a, c }"><p>{{ a }}-{{ c }}</p></template></Comp>`)
	out, err := NewRenderer(main).WithComponents(Registry{"Comp": comp}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Only a and c are bound; b is ignored.
	if !strings.Contains(out, "<p>1-3</p>") {
		t.Errorf("got %q, want <p>1-3</p>", out)
	}
}

func TestRender_DeeplyNestedSlots(t *testing.T) {
	// Component A uses component B with named slots; component B uses
	// component C with scoped slots. Three levels of composition.
	//
	//   ComponentC: <div><slot :value="data"></slot></div>   (prop: data)
	//   ComponentB: <section><slot name="head"></slot>
	//               <ComponentC :data="msg"><template #default="{ value }">
	//                 <em>{{ value }}</em></template></ComponentC></section>
	//               (prop: msg)
	//   ComponentA: <article><ComponentB msg="hello">
	//               <template #head><h2>Title</h2></template>
	//               </ComponentB></article>
	compC := mustParseComponent(t, "C.vue", `<div><slot :value="data"></slot></div>`)
	compB := mustParseComponent(t, "B.vue",
		`<section><slot name="head"></slot><CompC :data="msg"><template #default="{ value }"><em>{{ value }}</em></template></CompC></section>`)
	compA := mustParseComponent(t, "A.vue",
		`<article><CompB msg="hello"><template #head><h2>Title</h2></template></CompB></article>`)

	reg := Registry{"CompC": compC, "CompB": compB}
	out, err := NewRenderer(compA).WithComponents(reg).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<article>") {
		t.Errorf("got %q, want <article> from ComponentA", out)
	}
	if !strings.Contains(out, "<h2>Title</h2>") {
		t.Errorf("got %q, want <h2>Title</h2> from named slot in ComponentB", out)
	}
	if !strings.Contains(out, "<em>hello</em>") {
		t.Errorf("got %q, want <em>hello</em> from scoped slot in ComponentC", out)
	}
}

func TestRender_VModelStripped(t *testing.T) {
	out := renderTemplate(t, `<input v-model="name" type="text" />`, map[string]any{"name": "Alice"})
	if strings.Contains(out, "v-model") {
		t.Errorf("v-model should be stripped from output, got: %s", out)
	}
	if !strings.Contains(out, `type="text"`) {
		t.Errorf("other attributes should be preserved, got: %s", out)
	}
}

func TestRender_VOnStripped(t *testing.T) {
	out := renderTemplate(t, `<button v-on:click="handleClick" id="btn">Click</button>`, map[string]any{})
	if strings.Contains(out, "v-on") {
		t.Errorf("v-on:click should be stripped from output, got: %s", out)
	}
	if !strings.Contains(out, `id="btn"`) {
		t.Errorf("other attributes should be preserved, got: %s", out)
	}
}

func TestRender_AtEventShorthandStripped(t *testing.T) {
	out := renderTemplate(t, `<button @click="handleClick" @mouseover="onHover" class="btn">Go</button>`, map[string]any{})
	if strings.Contains(out, "@click") || strings.Contains(out, "@mouseover") {
		t.Errorf("@event shorthands should be stripped, got: %s", out)
	}
	if !strings.Contains(out, `class="btn"`) {
		t.Errorf("class attribute should be preserved, got: %s", out)
	}
}

// --- TestRender_DynamicComponent_* ---

func TestRender_DynamicComponent_BasicLiteralComponent(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<article>card</article>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Card'"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<article>card</article>") {
		t.Errorf("got %q, want <article>card</article>", out)
	}
	if strings.Contains(out, "<component") {
		t.Errorf("got %q, literal <component> tag should not appear in output", out)
	}
}

func TestRender_DynamicComponent_BasicScopeVariable(t *testing.T) {
	banner := mustParseComponent(t, "banner.vue", `<section>banner</section>`)
	host := mustParseComponent(t, "host.vue", `<component :is="view"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Banner": banner}).RenderString(map[string]any{"view": "Banner"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<section>banner</section>") {
		t.Errorf("got %q, want <section>banner</section>", out)
	}
}

func TestRender_DynamicComponent_NativeHTMLElement(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component :is="'div'" class="box">text</component>`)
	out, err := NewRenderer(host).WithComponents(Registry{}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, `<div class="box">text</div>`) {
		t.Errorf("got %q, want <div class=\"box\">text</div>", out)
	}
}

func TestRender_DynamicComponent_NativeVoidElement(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component :is="'input'" type="text"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, `<input type="text">`) {
		t.Errorf("got %q, want <input type=\"text\">", out)
	}
	if strings.Contains(out, "</input>") {
		t.Errorf("got %q, void element should not have closing tag", out)
	}
}

func TestRender_DynamicComponent_PropsForwarding(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<div>{{ title }}</div>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Card'" :title="label"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(map[string]any{"label": "Hello"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("got %q, want 'Hello' in output", out)
	}
}

func TestRender_DynamicComponent_DefaultSlot(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<div><slot /></div>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Card'">slot content</component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "slot content") {
		t.Errorf("got %q, want 'slot content' in output", out)
	}
}

func TestRender_DynamicComponent_NamedSlots(t *testing.T) {
	layout := mustParseComponent(t, "layout.vue", `<div><slot name="header" /></div>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Layout'"><template #header>header text</template></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Layout": layout}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "header text") {
		t.Errorf("got %q, want 'header text' in output", out)
	}
}

func TestRender_DynamicComponent_InsideVFor(t *testing.T) {
	compA := mustParseComponent(t, "a.vue", `<span>A</span>`)
	compB := mustParseComponent(t, "b.vue", `<span>B</span>`)
	host := mustParseComponent(t, "host.vue", `<div v-for="item in items"><component :is="item.type"></component></div>`)
	reg := Registry{"A": compA, "B": compB}
	out, err := NewRenderer(host).WithComponents(reg).RenderString(map[string]any{
		"items": []any{
			map[string]any{"type": "A"},
			map[string]any{"type": "B"},
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<span>A</span>") {
		t.Errorf("got %q, want <span>A</span>", out)
	}
	if !strings.Contains(out, "<span>B</span>") {
		t.Errorf("got %q, want <span>B</span>", out)
	}
}

func TestRender_DynamicComponent_WithVIf_True(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<article>card</article>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Card'" v-if="show"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(map[string]any{"show": true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<article>card</article>") {
		t.Errorf("got %q, want card rendered", out)
	}
}

func TestRender_DynamicComponent_WithVIf_False(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<article>card</article>`)
	host := mustParseComponent(t, "host.vue", `<component :is="'Card'" v-if="show"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(map[string]any{"show": false})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "<article>") {
		t.Errorf("got %q, want no card output when v-if is false", out)
	}
}

func TestRender_DynamicComponent_VBindIs(t *testing.T) {
	card := mustParseComponent(t, "card.vue", `<article>card</article>`)
	host := mustParseComponent(t, "host.vue", `<component v-bind:is="'Card'"></component>`)
	out, err := NewRenderer(host).WithComponents(Registry{"Card": card}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<article>card</article>") {
		t.Errorf("got %q, want <article>card</article>", out)
	}
}

func TestRender_DynamicComponent_MissingIs(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component class="x"></component>`)
	_, err := NewRenderer(host).WithComponents(Registry{}).RenderString(nil)
	if err == nil {
		t.Fatal("expected error for missing :is, got nil")
	}
	if !strings.Contains(err.Error(), "<component>") || !strings.Contains(err.Error(), ":is") {
		t.Errorf("error %q should mention <component> and :is", err.Error())
	}
}

func TestRender_DynamicComponent_NonStringIs(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component :is="42"></component>`)
	_, err := NewRenderer(host).WithComponents(Registry{}).RenderString(nil)
	if err == nil {
		t.Fatal("expected error for non-string :is, got nil")
	}
	if !strings.Contains(err.Error(), "<component>") {
		t.Errorf("error %q should mention <component>", err.Error())
	}
}

func TestRender_DynamicComponent_EmptyStringIs(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component :is="empty"></component>`)
	_, err := NewRenderer(host).WithComponents(Registry{}).RenderString(map[string]any{"empty": ""})
	if err == nil {
		t.Fatal("expected error for empty string :is, got nil")
	}
	if !strings.Contains(err.Error(), "<component>") {
		t.Errorf("error %q should mention <component>", err.Error())
	}
}

func TestRender_DynamicComponent_UnknownComponentName(t *testing.T) {
	host := mustParseComponent(t, "host.vue", `<component :is="'no-such-comp'"></component>`)
	_, err := NewRenderer(host).WithComponents(Registry{}).RenderString(nil)
	if err == nil {
		t.Fatal("expected error for unknown component, got nil")
	}
	if !strings.Contains(err.Error(), "no-such-comp") {
		t.Errorf("error %q should contain the component name", err.Error())
	}
}

// --- location-aware render error tests ---

func TestRender_InterpolationError_IsRenderError(t *testing.T) {
	// A template with a member-access expression on a nil value should produce
	// a *RenderError wrapping the underlying expression error.
	// post is provided as nil in scope so validateProps passes, but post.Title
	// fails during interpolation (cannot access property of null).
	src := "<template>\n  <div class=\"card\">\n    {{ post.Title }}\n  </div>\n</template>"
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, renderErr := RenderString(c, map[string]any{"post": nil})
	if renderErr == nil {
		t.Fatal("expected render error, got nil")
	}
	var re *RenderError
	if !errors.As(renderErr, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", renderErr, renderErr)
	}
	if re.Expr == "" {
		t.Error("RenderError.Expr should be set for an interpolation error")
	}
}

func TestRender_InterpolationError_IncludesPathAndLine(t *testing.T) {
	// When the source is available and the expression is found, the error
	// message should include the file path and a line number.
	// post is nil in scope so validateProps passes but post.Title fails.
	src := "<template>\n  <div class=\"card\">\n    {{ post.Title }}\n  </div>\n</template>"
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, renderErr := RenderString(c, map[string]any{"post": nil})
	if renderErr == nil {
		t.Fatal("expected render error, got nil")
	}
	msg := renderErr.Error()
	// Should contain the file path.
	if !strings.Contains(msg, "Card.vue") {
		t.Errorf("error %q should contain file path Card.vue", msg)
	}
	// Should contain a line number (the expression is on line 3).
	if !strings.Contains(msg, ":3:") {
		t.Errorf("error %q should contain ':3:' for line 3", msg)
	}
}

// renderTemplateDebug is a helper that parses a template and renders it in debug mode.
func renderTemplateDebug(t *testing.T, tmpl string, scope map[string]any) string {
	t.Helper()
	src := "<template>" + tmpl + "</template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var sb strings.Builder
	dw := newDebugWriter(&sb)
	r := NewRenderer(c).withDebug(dw)
	if err := r.Render(dw, scope); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return sb.String()
}

func TestDebug_ComponentBoundaryComments(t *testing.T) {
	// When Debug is true, component boundaries are wrapped with HTML comments.
	childSrc := "<template><span>child</span></template>"
	child, err := ParseFile("Child.vue", childSrc)
	if err != nil {
		t.Fatalf("ParseFile child: %v", err)
	}

	parentSrc := "<template><Child /></template>"
	parent, err := ParseFile("Parent.vue", parentSrc)
	if err != nil {
		t.Fatalf("ParseFile parent: %v", err)
	}

	reg := Registry{"Child": child}
	var sb strings.Builder
	dw := newDebugWriter(&sb)
	r := NewRenderer(parent).WithComponents(reg).withDebug(dw)
	if err := r.Render(dw, map[string]any{}); err != nil {
		t.Fatalf("Render: %v", err)
	}

	out := sb.String()
	// Note: the HTML parser lowercases tag names, so "Child" becomes "child" in n.Data.
	if !strings.Contains(out, "<!-- [htmlc:debug] component=child") {
		t.Errorf("output missing component start comment, got:\n%s", out)
	}
	if !strings.Contains(out, "<!-- [htmlc:debug] /component=child -->") {
		t.Errorf("output missing component end comment, got:\n%s", out)
	}
}

func TestDebug_NoCommentsWhenDisabled(t *testing.T) {
	// When Debug is false (default), no debug comments appear in the output.
	childSrc := "<template><span>child</span></template>"
	child, err := ParseFile("Child.vue", childSrc)
	if err != nil {
		t.Fatalf("ParseFile child: %v", err)
	}

	parentSrc := "<template><Child /></template>"
	parent, err := ParseFile("Parent.vue", parentSrc)
	if err != nil {
		t.Fatalf("ParseFile parent: %v", err)
	}

	reg := Registry{"Child": child}
	out, err := NewRenderer(parent).WithComponents(reg).RenderString(map[string]any{})
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	if strings.Contains(out, "[htmlc:debug]") {
		t.Errorf("output should not contain debug comments when debug is disabled, got:\n%s", out)
	}
}

func TestDebug_VIfSkippedComment(t *testing.T) {
	// When Debug is true and a v-if evaluates to false, a skipped-node comment is emitted.
	scope := map[string]any{"show": false}
	out := renderTemplateDebug(t, `<p v-if="show">visible</p>`, scope)
	if !strings.Contains(out, `<!-- [htmlc:debug] v-if="show"`) {
		t.Errorf("output missing v-if skipped comment, got:\n%s", out)
	}
	if !strings.Contains(out, "node skipped") {
		t.Errorf("output missing 'node skipped' in v-if comment, got:\n%s", out)
	}
	// The element itself should NOT be rendered.
	if strings.Contains(out, "<p>") {
		t.Errorf("skipped element should not appear in output, got:\n%s", out)
	}
}

func TestDebug_VIfSkippedComment_NotEmittedWhenTrue(t *testing.T) {
	// When Debug is true and v-if evaluates to true, no skipped comment is emitted.
	scope := map[string]any{"show": true}
	out := renderTemplateDebug(t, `<p v-if="show">visible</p>`, scope)
	if strings.Contains(out, "node skipped") {
		t.Errorf("should not emit skipped comment for true v-if, got:\n%s", out)
	}
}

func TestDebug_ExpressionValueComment(t *testing.T) {
	// When Debug is true, expression interpolation emits an expr/value comment.
	scope := map[string]any{"title": "Hello World"}
	out := renderTemplateDebug(t, `<h1>{{ title }}</h1>`, scope)
	if !strings.Contains(out, `<!-- [htmlc:debug] expr="title" value=Hello World -->`) {
		t.Errorf("output missing expr value comment, got:\n%s", out)
	}
}

func TestDebug_ExpressionComment_NotEmittedWhenDisabled(t *testing.T) {
	// When Debug is false, no expression comment appears.
	scope := map[string]any{"title": "Hello World"}
	out := renderTemplate(t, `<h1>{{ title }}</h1>`, scope)
	if strings.Contains(out, "[htmlc:debug]") {
		t.Errorf("output should not contain debug comments when debug is disabled, got:\n%s", out)
	}
}

func TestRender_VBindSpreadMap(t *testing.T) {
	// Basic map spread onto HTML element; keys appear in sorted (deterministic) order.
	scope := map[string]any{
		"attrs": map[string]any{
			"hx-delete":  "/items/1",
			"hx-confirm": "Are you sure?",
			"hx-target":  "#list",
		},
	}
	out := renderTemplate(t, `<button v-bind="attrs">click</button>`, scope)
	want := `<button hx-confirm="Are you sure?" hx-delete="/items/1" hx-target="#list">click</button>`
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want it to contain %q", out, want)
	}
}

func TestRender_VBindSpreadNilNoOp(t *testing.T) {
	// nil spread is a no-op: element renders without extra attributes.
	scope := map[string]any{"nothing": nil}
	out := renderTemplate(t, `<div v-bind="nothing">x</div>`, scope)
	want := `<div>x</div>`
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want it to contain %q", out, want)
	}
}

func TestRender_VBindSpreadMergesClass(t *testing.T) {
	// class key in spread map is merged with static class attribute.
	scope := map[string]any{
		"extra": map[string]any{"class": "active"},
	}
	out := renderTemplate(t, `<div class="base" v-bind="extra">x</div>`, scope)
	want := `<div class="base active">x</div>`
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want it to contain %q", out, want)
	}
}

func TestRender_VBindSpreadBooleanAttr(t *testing.T) {
	// Boolean attrs in spread: truthy → present, falsy → omitted.
	scope := map[string]any{
		"attrs": map[string]any{
			"disabled": true,
			"required": false,
		},
	}
	out := renderTemplate(t, `<button v-bind="attrs">x</button>`, scope)
	if !strings.Contains(out, "disabled") {
		t.Errorf("got %q, want 'disabled' present", out)
	}
	if strings.Contains(out, "required") {
		t.Errorf("got %q, want 'required' omitted (falsy)", out)
	}
}

func TestRender_VBindSpreadStyle(t *testing.T) {
	// style key in spread map is merged with static style attribute.
	scope := map[string]any{
		"extra": map[string]any{"style": map[string]any{"fontSize": "14px"}},
	}
	out := renderTemplate(t, `<div style="color:red" v-bind="extra">x</div>`, scope)
	if !strings.Contains(out, "color:red") {
		t.Errorf("got %q, want static style 'color:red' preserved", out)
	}
	if !strings.Contains(out, "font-size:14px") {
		t.Errorf("got %q, want spread style 'font-size:14px' present", out)
	}
}

func TestRender_VBindSpreadComponentProps(t *testing.T) {
	// Spread map into child component props.
	child := mustParseComponent(t, "child.vue", `<div>{{ title }} {{ count }}</div>`)
	main := mustParseComponent(t, "main.vue", `<Child v-bind="props" />`)
	scope := map[string]any{
		"props": map[string]any{"title": "Hello", "count": float64(42)},
	}
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(scope)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `<div>Hello 42</div>`
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want it to contain %q", out, want)
	}
}

func TestRender_VBindSpreadComponentExplicitPropWins(t *testing.T) {
	// Explicit :prop binding should override spread value.
	child := mustParseComponent(t, "child.vue", `<div>{{ title }}</div>`)
	main := mustParseComponent(t, "main.vue", `<Child v-bind="props" :title="'Override'" />`)
	scope := map[string]any{
		"props": map[string]any{"title": "Spread"},
	}
	out, err := NewRenderer(main).WithComponents(Registry{"Child": child}).RenderString(scope)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := `<div>Override</div>`
	if !strings.Contains(out, want) {
		t.Errorf("got %q, want it to contain %q", out, want)
	}
}

func TestRender_VBindSpreadNonMapError(t *testing.T) {
	// Non-map value should return an error.
	scope := map[string]any{"notAMap": "hello"}
	src := "<template><div v-bind=\"notAMap\">x</div></template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, err = RenderString(c, scope)
	if err == nil {
		t.Fatal("expected error for non-map v-bind, got nil")
	}
	if !strings.Contains(err.Error(), "v-bind") {
		t.Errorf("error %q should mention 'v-bind'", err.Error())
	}
}

// TestRenderNode_StyleBlockVerbatim checks that a <style> element nested
// inside a template (e.g. in <head>) has its text content emitted verbatim,
// without HTML-escaping quotes or > characters.
func TestRenderNode_StyleBlockVerbatim(t *testing.T) {
	src := `<template>
<html>
  <head>
    <style>
      body { font-family: 'Outfit', sans-serif; }
      code { font-family: "SF Mono", monospace; }
      :not(pre) > code { color: red; }
    </style>
  </head>
  <body></body>
</html>
</template>`

	comp, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := NewRenderer(comp).Render(&buf, nil); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	for _, want := range []string{
		`font-family: 'Outfit'`,
		`font-family: "SF Mono"`,
		`:not(pre) > code`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, got)
		}
	}
	for _, bad := range []string{"&#39;", "&#34;", "&gt;"} {
		if strings.Contains(got, bad) {
			t.Errorf("output contains escaped entity %q in CSS; got:\n%s", bad, got)
		}
	}
}
