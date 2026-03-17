package htmlc

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestRenderFragment_ComponentCycle verifies that RenderFragment terminates
// with an error when a component's template directly references itself,
// rather than looping forever.
//
// Minimal reproduction: a component named "SelfRef" whose template contains
// <self-ref>, which resolves (via kebab-to-Pascal) back to "SelfRef".
// Before the cycle-detection fix, this caused infinite recursion.
func TestRenderFragment_ComponentCycle(t *testing.T) {
	memFS := fstest.MapFS{
		"SelfRef.vue": {Data: []byte(`<template><self-ref></self-ref></template>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		var buf strings.Builder
		done <- e.RenderFragment(&buf, "SelfRef", nil)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for self-referential component, got nil")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error should mention cycle, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RenderFragment did not terminate within 3s (cycle not detected)")
	}
}

// TestRenderFragment_IndirectCycle verifies that RenderFragment terminates
// when two components reference each other (A → B → A).
func TestRenderFragment_IndirectCycle(t *testing.T) {
	memFS := fstest.MapFS{
		"Alpha.vue": {Data: []byte(`<template><beta></beta></template>`)},
		"Beta.vue":  {Data: []byte(`<template><alpha></alpha></template>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		var buf strings.Builder
		done <- e.RenderFragment(&buf, "Alpha", nil)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error for mutually recursive components, got nil")
		}
		if !strings.Contains(err.Error(), "cycle") {
			t.Errorf("error should mention cycle, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RenderFragment did not terminate within 3s (cycle not detected)")
	}
}
