package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting VisuallyHidden.vue into a real htmlc.Engine
// and checking the rendered HTML) is out of scope here — that proof is
// deliberately deferred to the examples/radix-demo commit, which does
// depend on root htmlc. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.
func TestVisuallyHidden_ContainsSlot(t *testing.T) {
	src := readVisuallyHidden(t)

	for _, marker := range []string{
		"<template>",
		"<slot></slot>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("VisuallyHidden.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestVisuallyHidden_ContainsClippingTechnique(t *testing.T) {
	src := readVisuallyHidden(t)

	for _, marker := range []string{
		"position: absolute",
		"width: 1px",
		"height: 1px",
		"overflow: hidden",
		"clip: rect(0 0 0 0)",
		"clip-path: inset(50%)",
		"white-space: nowrap",
		"margin: -1px",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("VisuallyHidden.vue missing expected clipping-technique marker %q", marker)
		}
	}
}

func TestVisuallyHidden_DoesNotHideFromAssistiveTech(t *testing.T) {
	src := readVisuallyHidden(t)

	// display:none and visibility:hidden both remove content from the
	// accessibility tree, which would defeat this component's entire
	// purpose. Guard against ever reintroducing either by accident.
	for _, forbidden := range []string{
		"display: none",
		"display:none",
		"visibility: hidden",
		"visibility:hidden",
	} {
		if strings.Contains(src, forbidden) {
			t.Errorf("VisuallyHidden.vue must not use %q — it would hide content from assistive technology too", forbidden)
		}
	}
}

func TestVisuallyHidden_ContainsScopedStyle(t *testing.T) {
	src := readVisuallyHidden(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("VisuallyHidden.vue missing expected <style scoped> block")
	}
}

func TestVisuallyHidden_HasNoCustomElementScript(t *testing.T) {
	src := readVisuallyHidden(t)

	// Pure CSS technique — no JavaScript enhancement needed or expected.
	if strings.Contains(src, "<script customelement>") {
		t.Error("VisuallyHidden.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

func readVisuallyHidden(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/VisuallyHidden.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/VisuallyHidden.vue) failed: %v", err)
	}
	return string(data)
}
