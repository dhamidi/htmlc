package htmlc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestIntegration_FullPipeline exercises the complete rendering pipeline through
// the Engine layer: discovers .vue files from a temp directory, renders a
// component that uses v-if/v-else, v-for, :class, mustache interpolation, and
// scoped styles, then asserts the resulting HTML output covers every subsystem.
func TestIntegration_FullPipeline(t *testing.T) {
	memFS := fstest.MapFS{
		"Featured.vue": &fstest.MapFile{Data: []byte(`<template><div :class="{ active: isActive, disabled: isDisabled }">
  <h1>{{ title }}</h1>
  <p v-if="show">Visible content</p>
  <p v-else>Hidden content</p>
  <ul>
    <li v-for="item in items">{{ item }}</li>
  </ul>
</div></template>
<style scoped>
.active { color: green; }
.disabled { color: grey; }
</style>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Featured", map[string]any{
		"title":      "Hello World",
		"isActive":   true,
		"isDisabled": false,
		"show":       true,
		"items":      []any{"alpha", "beta", "gamma"},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	// mustache interpolation
	if !strings.Contains(out, "Hello World") {
		t.Errorf("mustache: want 'Hello World' in output:\n%s", out)
	}

	// :class binding: isActive=true → "active" present; isDisabled=false → omitted
	if !strings.Contains(out, `class="active"`) {
		t.Errorf(":class: want class=\"active\" in output:\n%s", out)
	}

	// v-if: show=true → "Visible content" rendered; v-else branch suppressed
	if !strings.Contains(out, "Visible content") {
		t.Errorf("v-if: want 'Visible content' in output:\n%s", out)
	}
	if strings.Contains(out, "Hidden content") {
		t.Errorf("v-else: 'Hidden content' must not appear in output:\n%s", out)
	}

	// v-for: all three items rendered as <li> elements (scope attr may be present on the tag)
	for _, item := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(out, ">"+item+"</li>") {
			t.Errorf("v-for: want >%s</li> in output:\n%s", item, out)
		}
	}

	// scoped styles: prepended <style> block rewrites selectors with the scope attribute
	scopeAttr := ScopeID("Featured.vue")
	scopeSelector := "[" + scopeAttr + "]"
	if !strings.Contains(out, scopeSelector) {
		t.Errorf("scoped style: want CSS scope selector %q in output:\n%s", scopeSelector, out)
	}

	// HTML elements carry the scope attribute added by the renderer
	if !strings.Contains(out, " "+scopeAttr) {
		t.Errorf("scoped attr: want scope attribute %q on elements in output:\n%s", scopeAttr, out)
	}
}

// TestIntegration_NestedComponentsWithSlots verifies that a parent component
// can compose a child component and inject content through the default slot,
// exercising the Engine's registry, the component renderer, and slot injection.
func TestIntegration_NestedComponentsWithSlots(t *testing.T) {
	memFS := fstest.MapFS{
		"Card.vue": &fstest.MapFile{Data: []byte(`<template><div class="card"><slot></slot></div></template>`)},
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><Card><span>Slot content</span></Card></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	// Card's wrapper div must be present
	if !strings.Contains(out, `class="card"`) {
		t.Errorf("slot: want card wrapper in output:\n%s", out)
	}

	// Slot content must appear inside the Card div
	if !strings.Contains(out, "<span>Slot content</span>") {
		t.Errorf("slot: want '<span>Slot content</span>' in output:\n%s", out)
	}

	// Full composed structure: slot content is nested inside the card wrapper
	if !strings.Contains(out, `class="card"><span>Slot content</span></div>`) {
		t.Errorf("composition: want slot content inside card wrapper in output:\n%s", out)
	}
}

// TestIntegration_ServeComponentHTTP uses httptest to exercise ServeComponent
// end-to-end and asserts the HTTP status code, Content-Type header, and body.
// The data function injects a dynamic greeting name that must appear in the
// rendered output, verifying that the data func is wired through correctly.
func TestIntegration_ServeComponentHTTP(t *testing.T) {
	memFS := fstest.MapFS{
		"Greeting.vue": &fstest.MapFile{Data: []byte(`<template><section class="greeting"><h1>Hello, {{ name }}!</h1></section></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h := e.ServeComponent("Greeting", func(r *http.Request) map[string]any {
		return map[string]any{"name": "World"}
	})
	req := httptest.NewRequest(http.MethodGet, "/greeting", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want \"text/html; charset=utf-8\"", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `class="greeting"`) {
		t.Errorf("body: want class=\"greeting\" in:\n%s", body)
	}
	if !strings.Contains(body, "Hello, World!") {
		t.Errorf("body: want 'Hello, World!' in:\n%s", body)
	}
}

// TestIntegration_VIfWithLength verifies that v-if="posts.length > 0" renders
// the element when the slice is non-empty and hides it when the slice is empty.
func TestIntegration_VIfWithLength(t *testing.T) {
	memFS := fstest.MapFS{
		"Posts.vue": &fstest.MapFile{Data: []byte(`<template><div><p v-if="posts.length > 0">Has posts</p><p v-else>No posts</p></div></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Non-empty slice: the v-if branch must render.
	out, err := e.RenderFragmentString(context.Background(), "Posts", map[string]any{
		"posts": []any{"first", "second"},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString (non-empty): %v", err)
	}
	if !strings.Contains(out, "Has posts") {
		t.Errorf("non-empty: want 'Has posts' in output:\n%s", out)
	}
	if strings.Contains(out, "No posts") {
		t.Errorf("non-empty: 'No posts' must not appear in output:\n%s", out)
	}

	// Empty slice: the v-else branch must render.
	out, err = e.RenderFragmentString(context.Background(), "Posts", map[string]any{
		"posts": []any{},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString (empty): %v", err)
	}
	if strings.Contains(out, "Has posts") {
		t.Errorf("empty: 'Has posts' must not appear in output:\n%s", out)
	}
	if !strings.Contains(out, "No posts") {
		t.Errorf("empty: want 'No posts' in output:\n%s", out)
	}
}

// TestIntegration_LayoutPattern exercises the three-slot layout pattern:
// a Layout component with named header/main/footer slots is used by a Page
// component that injects dynamic content into each slot.
func TestIntegration_LayoutPattern(t *testing.T) {
	memFS := fstest.MapFS{
		"Layout.vue": &fstest.MapFile{Data: []byte(`<template><div class="layout"><header><slot name="header"></slot></header><main><slot></slot></main><footer><slot name="footer"></slot></footer></div></template>`)},
		"Page.vue":   &fstest.MapFile{Data: []byte(`<template><Layout><template #header><h1>{{ title }}</h1></template><p>{{ body }}</p><template #footer><small>{{ copy }}</small></template></Layout></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", map[string]any{
		"title": "My Blog",
		"body":  "Welcome!",
		"copy":  "2024",
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	if !strings.Contains(out, `class="layout"`) {
		t.Errorf("layout wrapper: want class=\"layout\" in output:\n%s", out)
	}
	if !strings.Contains(out, "<h1>My Blog</h1>") {
		t.Errorf("header slot: want <h1>My Blog</h1> in output:\n%s", out)
	}
	if !strings.Contains(out, "<p>Welcome!</p>") {
		t.Errorf("default slot: want <p>Welcome!</p> in output:\n%s", out)
	}
	if !strings.Contains(out, "<small>2024</small>") {
		t.Errorf("footer slot: want <small>2024</small> in output:\n%s", out)
	}
}

// TestIntegration_RenderlessListPattern exercises the renderless component
// pattern: a UserList component iterates its items internally and exposes
// each item and its index through a named scoped slot.
func TestIntegration_RenderlessListPattern(t *testing.T) {
	memFS := fstest.MapFS{
		"UserList.vue": &fstest.MapFile{Data: []byte(`<template><ul><li v-for="(user, index) in users"><slot name="item" :user="user" :index="index"></slot></li></ul></template>`)},
		"Page.vue":     &fstest.MapFile{Data: []byte(`<template><UserList :users="users"><template #item="{ user, index }"><span>{{ index }}: {{ user.name }}</span></template></UserList></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", map[string]any{
		"users": []any{
			map[string]any{"name": "Alice"},
			map[string]any{"name": "Bob"},
		},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	if !strings.Contains(out, "<ul>") {
		t.Errorf("list: want <ul> in output:\n%s", out)
	}
	if !strings.Contains(out, "<span>0: Alice</span>") {
		t.Errorf("item 0: want <span>0: Alice</span> in output:\n%s", out)
	}
	if !strings.Contains(out, "<span>1: Bob</span>") {
		t.Errorf("item 1: want <span>1: Bob</span> in output:\n%s", out)
	}
}

// TestIntegration_NestedParentChildGrandchild verifies a three-level component
// hierarchy where each level uses slots: Parent provides content to Child via
// a named slot, and Child provides content to Grandchild via a named slot.
func TestIntegration_NestedParentChildGrandchild(t *testing.T) {
	memFS := fstest.MapFS{
		"Grandchild.vue": &fstest.MapFile{Data: []byte(`<template><span class="gc"><slot name="data"></slot></span></template>`)},
		"Child.vue":      &fstest.MapFile{Data: []byte(`<template><div class="child"><slot name="main"></slot><Grandchild><template #data><em>gc-content</em></template></Grandchild></div></template>`)},
		"Parent.vue":     &fstest.MapFile{Data: []byte(`<template><article><Child><template #main><strong>parent-content</strong></template></Child></article></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	if !strings.Contains(out, "<article>") {
		t.Errorf("parent: want <article> in output:\n%s", out)
	}
	if !strings.Contains(out, `class="child"`) {
		t.Errorf("child: want class=\"child\" in output:\n%s", out)
	}
	if !strings.Contains(out, "<strong>parent-content</strong>") {
		t.Errorf("parent slot: want <strong>parent-content</strong> in output:\n%s", out)
	}
	if !strings.Contains(out, `class="gc"`) {
		t.Errorf("grandchild: want class=\"gc\" in output:\n%s", out)
	}
	if !strings.Contains(out, "<em>gc-content</em>") {
		t.Errorf("grandchild slot: want <em>gc-content</em> in output:\n%s", out)
	}
}

// TestIntegration_ReloadPicksUpChanges verifies that Reload:true causes the
// Engine to re-parse a component file when its modification time advances,
// exercising the full path from file-system change through to rendered output.
func TestIntegration_ReloadPicksUpChanges(t *testing.T) {
	memFS := fstest.MapFS{
		"Live.vue": &fstest.MapFile{Data: []byte(`<template><p>version one</p></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: ".", Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (before reload): %v", err)
	}
	if !strings.Contains(out, "version one") {
		t.Errorf("before reload: want 'version one' in output:\n%s", out)
	}

	// Simulate file modification by updating the MapFile with new content and a
	// future ModTime so the reload check sees it as changed.
	memFS["Live.vue"] = &fstest.MapFile{
		Data:    []byte(`<template><p>version two</p></template>`),
		ModTime: time.Now().Add(time.Second),
	}

	out, err = e.RenderFragmentString(context.Background(), "Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (after reload): %v", err)
	}
	if !strings.Contains(out, "version two") {
		t.Errorf("after reload: want 'version two' in output:\n%s", out)
	}
	if strings.Contains(out, "version one") {
		t.Errorf("after reload: 'version one' must not appear in output:\n%s", out)
	}
}

// TestIntegration_CamelCasePropViaSlot verifies that a camelCase prop passed
// with v-bind inside a parent component's slot resolves correctly. The HTML
// parser lowercases attribute names (e.g. :submitLabel → :submitlabel), so the
// engine must recover the original casing when injecting into the child scope.
func TestIntegration_CamelCasePropViaSlot(t *testing.T) {
	memFS := fstest.MapFS{
		"Layout.vue": &fstest.MapFile{Data: []byte(`<template><div class="layout"><slot></slot></div></template>`)},
		"Inner.vue":  &fstest.MapFile{Data: []byte(`<template><span>{{ myProp }}</span></template>`)},
		"Page.vue":   &fstest.MapFile{Data: []byte(`<template><Layout><Inner :myProp="myProp" /></Layout></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", map[string]any{
		"myProp": "hello",
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	if !strings.Contains(out, "<span>hello</span>") {
		t.Errorf("camelCase prop via slot: want <span>hello</span> in output:\n%s", out)
	}
}

func TestIntegration_DynamicComponent_BasicResolution(t *testing.T) {
	memFS := fstest.MapFS{
		"Banner.vue": &fstest.MapFile{Data: []byte(`<template><section class="banner"><slot></slot></section></template>`)},
		"Page.vue":   &fstest.MapFile{Data: []byte(`<template><component :is="widgetType">hello</component></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", map[string]any{"widgetType": "Banner"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, `class="banner"`) {
		t.Errorf("dynamic component: want Banner output, got:\n%s", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("dynamic component: want slot content 'hello', got:\n%s", out)
	}
}

// TestIntegration_SelfClosingComponentTag verifies that a self-closing custom
// component tag (<PostImage ... />) renders identically to the explicit
// open/close form (<PostImage ...></PostImage>).
func TestIntegration_SelfClosingComponentTag(t *testing.T) {
	memFS := fstest.MapFS{
		"PostImage.vue":    &fstest.MapFile{Data: []byte(`<template><img :src="src" :alt="alt" /></template>`)},
		"PageSelfClose.vue": &fstest.MapFile{Data: []byte(`<template><PostImage src="/hero.jpg" alt="Hero" /><p>Caption here</p></template>`)},
		"PageExplicit.vue": &fstest.MapFile{Data: []byte(`<template><PostImage src="/hero.jpg" alt="Hero"></PostImage><p>Caption here</p></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	selfCloseOut, err := e.RenderFragmentString(context.Background(), "PageSelfClose", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (self-close): %v", err)
	}
	explicitOut, err := e.RenderFragmentString(context.Background(), "PageExplicit", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (explicit): %v", err)
	}

	if selfCloseOut != explicitOut {
		t.Errorf("self-close and explicit forms produce different output:\nself-close: %s\nexplicit:   %s", selfCloseOut, explicitOut)
	}

	// The <p>Caption here</p> must be a sibling, not swallowed as a child of PostImage.
	if !strings.Contains(selfCloseOut, "<p>Caption here</p>") {
		t.Errorf("self-close: <p>Caption here</p> was swallowed; output:\n%s", selfCloseOut)
	}
}

// TestIntegration_SelfClosingComponentWarning verifies that parsing a .vue file
// with a self-closing custom component tag sets Component.Warnings, and that
// ValidateAll surfaces those warnings as ValidationError entries.
func TestIntegration_SelfClosingComponentWarning(t *testing.T) {
	memFS := fstest.MapFS{
		"Icon.vue": &fstest.MapFile{Data: []byte(`<template><span class="icon"></span></template>`)},
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><Icon /></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	found := false
	for _, ve := range errs {
		// The engine registers both "Page" and "page" (lowercase alias); either
		// may appear here depending on map iteration order.
		if strings.EqualFold(ve.Component, "Page") && strings.Contains(ve.Message, "auto-corrected") {
			found = true
		}
	}
	if !found {
		t.Errorf("ValidateAll: expected a warning for self-closing component in Page, got: %v", errs)
	}
}

func TestIntegration_DynamicComponent_ReloadPicksUpNewTemplate(t *testing.T) {
	memFS := fstest.MapFS{
		"Widget.vue": &fstest.MapFile{Data: []byte(`<template><p>version one</p></template>`)},
		"Page.vue":   &fstest.MapFile{Data: []byte(`<template><component :is="'Widget'"></component></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: ".", Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (v1): %v", err)
	}
	if !strings.Contains(out, "version one") {
		t.Errorf("reload: want 'version one' in initial output:\n%s", out)
	}

	// Simulate file modification by updating the MapFile with new content and a
	// future ModTime so the reload check sees it as changed.
	memFS["Widget.vue"] = &fstest.MapFile{
		Data:    []byte(`<template><p>version two</p></template>`),
		ModTime: time.Now().Add(time.Second),
	}

	out, err = e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (v2): %v", err)
	}
	if !strings.Contains(out, "version two") {
		t.Errorf("reload: want 'version two' after reload, got:\n%s", out)
	}
}

// TestIntegration_FontFaceQuotesPreserved verifies that a component with a
// global <style> block containing an @font-face rule emits quoted string values
// byte-for-byte in the rendered output, with no HTML-escaping or quote removal.
func TestIntegration_FontFaceQuotesPreserved(t *testing.T) {
	memFS := fstest.MapFS{
		"Fonts.vue": &fstest.MapFile{Data: []byte(`<template><p>hello</p></template>
<style>
@font-face {
  font-family: "My Font";
  src: url("font.woff2") format("woff2");
}
p { font-family: "My Font"; }
</style>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Fonts", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	for _, want := range []string{`"My Font"`, `"font.woff2"`, `format("woff2")`} {
		if !strings.Contains(out, want) {
			t.Errorf("font-face: want %q in output:\n%s", want, out)
		}
	}
}

// TestIntegration_ScopedFontFaceQuotesPreserved verifies that a scoped
// <style> block with @font-face emits quoted values verbatim. @-rules are
// passed through without selector rewriting, so the output must be identical.
func TestIntegration_ScopedFontFaceQuotesPreserved(t *testing.T) {
	memFS := fstest.MapFS{
		"ScopedFonts.vue": &fstest.MapFile{Data: []byte(`<template><p>hello</p></template>
<style scoped>
@font-face {
  font-family: "My Font";
  src: url("font.woff2") format("woff2");
}
p { font-family: "My Font"; }
</style>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "ScopedFonts", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	for _, want := range []string{`"My Font"`, `"font.woff2"`, `format("woff2")`} {
		if !strings.Contains(out, want) {
			t.Errorf("scoped font-face: want %q in output:\n%s", want, out)
		}
	}
}

// TestIntegration_CSSContentSpecialCharsPreserved verifies that &, <, and >
// characters inside CSS content property values pass through the pipeline
// without HTML-entity encoding or other modification.
func TestIntegration_CSSContentSpecialCharsPreserved(t *testing.T) {
	memFS := fstest.MapFS{
		"Icons.vue": &fstest.MapFile{Data: []byte(`<template><span class="arrow"></span></template>
<style>
.arrow::before { content: "a > b & c < d"; }
</style>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Icons", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	want := `"a > b & c < d"`
	if !strings.Contains(out, want) {
		t.Errorf("CSS special chars: want %q in output:\n%s", want, out)
	}
}

// TestIntegration_NestedSlotChain_NoInfiniteRecursion is a regression test for
// the infinite-recursion bug that occurred when a middle component's template
// passed slot content containing a <slot /> to an inner component.
//
// Layout: Outer → Middle → Inner, where:
//   - Outer passes <a href="/">Home</a> as slot content to Middle.
//   - Middle's template wraps <slot /> in a <div> and passes that to Inner.
//   - Inner's template wraps <slot /> in a <div>.
//
// Expected output: <div><div><a href="/">Home</a></div></div>
func TestIntegration_NestedSlotChain_NoInfiniteRecursion(t *testing.T) {
	memFS := fstest.MapFS{
		"Inner.vue":  &fstest.MapFile{Data: []byte(`<template><div><slot /></div></template>`)},
		"Middle.vue": &fstest.MapFile{Data: []byte(`<template><Inner><div><slot /></div></Inner></template>`)},
		"Outer.vue":  &fstest.MapFile{Data: []byte(`<template><Middle><a href="/">Home</a></Middle></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Outer", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	// The slot chain must resolve: Inner's slot → Middle's <div><slot/></div>,
	// which then resolves its slot → Outer's <a href="/">Home</a>.
	if !strings.Contains(out, `<a href="/">Home</a>`) {
		t.Errorf("nested slot chain: want anchor in output:\n%s", out)
	}
	if !strings.Contains(out, "<div><div>") {
		t.Errorf("nested slot chain: want nested divs in output:\n%s", out)
	}
}

// TestIntegration_NestedSlotChain_NamedSlot verifies that the fix also works
// when named slots are used in the chain.
func TestIntegration_NestedSlotChain_NamedSlot(t *testing.T) {
	memFS := fstest.MapFS{
		"InnerNamed.vue":  &fstest.MapFile{Data: []byte(`<template><section><slot name="body" /></section></template>`)},
		"MiddleNamed.vue": &fstest.MapFile{Data: []byte(`<template><InnerNamed><template #body><p><slot name="content" /></p></template></InnerNamed></template>`)},
		"OuterNamed.vue":  &fstest.MapFile{Data: []byte(`<template><MiddleNamed><template #content>hello</template></MiddleNamed></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "OuterNamed", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	// The named slot chain: InnerNamed#body → MiddleNamed's <p><slot name="content"/></p>
	// → OuterNamed's "hello".
	if !strings.Contains(out, "<p>hello</p>") {
		t.Errorf("named slot chain: want <p>hello</p> in output:\n%s", out)
	}
	if !strings.Contains(out, "<section>") {
		t.Errorf("named slot chain: want <section> wrapper in output:\n%s", out)
	}
}

// TestIntegration_NestedSlotChain_SlotProps verifies that slot props thread
// correctly through a three-component chain.
func TestIntegration_NestedSlotChain_SlotProps(t *testing.T) {
	memFS := fstest.MapFS{
		// InnerProps emits a static slot prop msg="hello".
		"InnerProps.vue": &fstest.MapFile{Data: []byte(`<template><slot msg="hello" /></template>`)},
		// MiddleProps consumes InnerProps' msg and re-emits it as label via slot props.
		"MiddleProps.vue": &fstest.MapFile{Data: []byte(`<template><InnerProps v-slot="{ msg }"><slot :label="msg" /></InnerProps></template>`)},
		// OuterProps consumes MiddleProps' label and renders it in a span.
		"OuterProps.vue": &fstest.MapFile{Data: []byte(`<template><MiddleProps v-slot="{ label }"><span>{{ label }}</span></MiddleProps></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "OuterProps", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	// Slot props should thread: InnerProps emits msg="hello", MiddleProps rebinds as
	// label="hello", OuterProps renders <span>hello</span>.
	if !strings.Contains(out, "<span>hello</span>") {
		t.Errorf("slot props chain: want <span>hello</span> in output:\n%s", out)
	}
}
