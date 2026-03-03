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

	out, err := e.RenderFragment("Featured", map[string]any{
		"title":      "Hello World",
		"isActive":   true,
		"isDisabled": false,
		"show":       true,
		"items":      []any{"alpha", "beta", "gamma"},
	})
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderFragment("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderFragment("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragment (before reload): %v", err)
	}
	if !strings.Contains(out, "version one") {
		t.Errorf("before reload: want 'version one' in output:\n%s", out)
	}

	// Ensure the mtime advances before overwriting the file.
	time.Sleep(10 * time.Millisecond)
	writeVue(t, p, `<template><p>version two</p></template>`)

	out, err = e.RenderFragment("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragment (after reload): %v", err)
	}
	if !strings.Contains(out, "version two") {
		t.Errorf("after reload: want 'version two' in output:\n%s", out)
	}
	if strings.Contains(out, "version one") {
		t.Errorf("after reload: 'version one' must not appear in output:\n%s", out)
	}
}
