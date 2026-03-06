package htmlc

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegration_FullPipeline exercises the complete rendering pipeline through
// the Engine layer: discovers .vue files from a temp directory, renders a
// component that uses v-if/v-else, v-for, :class, mustache interpolation, and
// scoped styles, then asserts the resulting HTML output covers every subsystem.
func TestIntegration_FullPipeline(t *testing.T) {
	dir := t.TempDir()
	compPath := filepath.Join(dir, "Featured.vue")
	writeVue(t, compPath, `<template><div :class="{ active: isActive, disabled: isDisabled }">
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
</style>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Featured", map[string]any{
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
	scopeAttr := ScopeID(compPath)
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Card.vue"),
		`<template><div class="card"><slot></slot></div></template>`)
	writeVue(t, filepath.Join(dir, "Page.vue"),
		`<template><Card><span>Slot content</span></Card></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", nil)
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeting.vue"),
		`<template><section class="greeting"><h1>Hello, {{ name }}!</h1></section></template>`)

	e, err := New(Options{ComponentDir: dir})
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Posts.vue"),
		`<template><div><p v-if="posts.length > 0">Has posts</p><p v-else>No posts</p></div></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Non-empty slice: the v-if branch must render.
	out, err := e.RenderFragmentString("Posts", map[string]any{
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
	out, err = e.RenderFragmentString("Posts", map[string]any{
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Layout.vue"),
		`<template><div class="layout"><header><slot name="header"></slot></header><main><slot></slot></main><footer><slot name="footer"></slot></footer></div></template>`)
	writeVue(t, filepath.Join(dir, "Page.vue"),
		`<template><Layout><template #header><h1>{{ title }}</h1></template><p>{{ body }}</p><template #footer><small>{{ copy }}</small></template></Layout></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", map[string]any{
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "UserList.vue"),
		`<template><ul><li v-for="(user, index) in users"><slot name="item" :user="user" :index="index"></slot></li></ul></template>`)
	writeVue(t, filepath.Join(dir, "Page.vue"),
		`<template><UserList :users="users"><template #item="{ user, index }"><span>{{ index }}: {{ user.name }}</span></template></UserList></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", map[string]any{
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
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Grandchild.vue"),
		`<template><span class="gc"><slot name="data"></slot></span></template>`)
	writeVue(t, filepath.Join(dir, "Child.vue"),
		`<template><div class="child"><slot name="main"></slot><Grandchild><template #data><em>gc-content</em></template></Grandchild></div></template>`)
	writeVue(t, filepath.Join(dir, "Parent.vue"),
		`<template><article><Child><template #main><strong>parent-content</strong></template></Child></article></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Parent", nil)
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
	dir := t.TempDir()
	p := filepath.Join(dir, "Live.vue")
	writeVue(t, p, `<template><p>version one</p></template>`)

	e, err := New(Options{ComponentDir: dir, Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (before reload): %v", err)
	}
	if !strings.Contains(out, "version one") {
		t.Errorf("before reload: want 'version one' in output:\n%s", out)
	}

	// Ensure the mtime advances before overwriting the file.
	time.Sleep(10 * time.Millisecond)
	writeVue(t, p, `<template><p>version two</p></template>`)

	out, err = e.RenderFragmentString("Live", nil)
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
