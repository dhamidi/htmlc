package htmlc

import (
	"path/filepath"
	"strings"
	"testing"
)

func highlightEngine(t *testing.T, tmpl string) *Engine {
	t.Helper()
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Host.vue"), "<template>"+tmpl+"</template>")
	e, err := New(Options{
		ComponentDir: dir,
		Directives:   DirectiveRegistry{"highlight": &VHighlight{}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// TestVHighlight_SetsBackground verifies that v-highlight sets the background
// style property on the host element.
func TestVHighlight_SetsBackground(t *testing.T) {
	e := highlightEngine(t, `<p v-highlight="'yellow'">text</p>`)
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "background:yellow") {
		t.Errorf("got %q, want background:yellow in style", out)
	}
}

// TestVHighlight_DynamicColour verifies that the expression is evaluated.
func TestVHighlight_DynamicColour(t *testing.T) {
	e := highlightEngine(t, `<span v-highlight="colour">text</span>`)
	out, err := e.RenderFragmentString("Host", map[string]any{"colour": "red"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "background:red") {
		t.Errorf("got %q, want background:red", out)
	}
}

// TestVHighlight_MergesExistingStyle verifies that an existing style attribute
// is preserved and the background is appended.
func TestVHighlight_MergesExistingStyle(t *testing.T) {
	e := highlightEngine(t, `<p style="color:blue" v-highlight="'green'">text</p>`)
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "color:blue") {
		t.Errorf("got %q, want color:blue preserved", out)
	}
	if !strings.Contains(out, "background:green") {
		t.Errorf("got %q, want background:green added", out)
	}
}

// TestVHighlight_EmptyValueNoOp verifies that an empty/nil expression leaves
// the element unchanged.
func TestVHighlight_EmptyValueNoOp(t *testing.T) {
	e := highlightEngine(t, `<p v-highlight="noColour">text</p>`)
	out, err := e.RenderFragmentString("Host", map[string]any{"noColour": ""})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if strings.Contains(out, "style") {
		t.Errorf("got %q, want no style attribute for empty colour", out)
	}
}

// TestVHighlight_NotRegisteredByDefault verifies that VHighlight is NOT
// auto-registered; it must be explicitly added via Options.Directives or
// Engine.RegisterDirective.
func TestVHighlight_NotRegisteredByDefault(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Host.vue"),
		"<template><p v-highlight=\"'yellow'\">text</p></template>")
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Without registration, v-highlight is an unknown attribute and passes
	// through as a plain HTML attribute (not an error, per directive semantics).
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The attribute should appear literally in the output, not as a style.
	if strings.Contains(out, "background:yellow") {
		t.Errorf("got %q, VHighlight should not apply without registration", out)
	}
}
