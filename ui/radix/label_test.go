package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like separator_test.go/aspectratio_test.go, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Label.vue into a real htmlc.Engine and checking rendered HTML) is out of
// scope here. These are content-sanity checks: they confirm the component's
// source file contains the markers this commit's design depends on.
func TestLabel_ContainsBaselineMarkers(t *testing.T) {
	src := readLabel(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Label.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestLabel_HasNoCustomElementScript confirms this component has no
// <script customelement> block: native <label> click-to-focus association
// needs no JS, and the double-click-text-selection fix is achieved entirely
// with CSS (user-select: none), so there is nothing left to progressively
// enhance.
func TestLabel_HasNoCustomElementScript(t *testing.T) {
	src := readLabel(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("Label.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

// TestLabel_ForAttributeBinding confirms the wrapper <label> binds the
// native `for` attribute to the "for" prop, and that the binding uses the
// documented `for ? for : undefined` fallback so an empty string opts out
// of emitting the attribute (nesting-only mode) instead of rendering a
// bogus for="" that would suppress the browser's native nested-control
// fallback.
func TestLabel_ForAttributeBinding(t *testing.T) {
	src := readLabel(t)
	wrapper := labelElement(t, src)

	if !strings.Contains(wrapper, `:for="for ? for : undefined"`) {
		t.Errorf(`Label.vue <label> missing expected marker :for="for ? for : undefined"; wrapper was:\n%s`, wrapper)
	}
	if !strings.Contains(wrapper, "<slot></slot>") {
		t.Errorf("Label.vue <label> missing expected default <slot></slot>; wrapper was:\n%s", wrapper)
	}
}

// TestLabel_UsesVNativeToAvoidSelfCycle confirms the wrapper <label> is
// marked v-native, matching Dialog.vue's identical <dialog v-native>
// precedent: this component's own name ("Label") auto-registers a
// lowercase "label" alias in the component registry, and the HTML parser
// lowercases the literal <label> tag in this template to the same name, so
// without v-native the tag would resolve back to this component itself
// (an infinite self-reference / "cycle detected" error).
func TestLabel_UsesVNativeToAvoidSelfCycle(t *testing.T) {
	src := readLabel(t)
	wrapper := labelElement(t, src)

	if !strings.Contains(wrapper, "v-native") {
		t.Errorf("Label.vue <label> missing expected v-native marker (needed to avoid resolving back to the Label component itself); wrapper was:\n%s", wrapper)
	}
}

// TestLabel_UserSelectNone confirms the key CSS fix for the
// double-click-selects-text papercut is present, including vendor-prefixed
// forms for broader browser coverage.
func TestLabel_UserSelectNone(t *testing.T) {
	src := readLabel(t)

	for _, marker := range []string{
		"user-select: none",
		"-webkit-user-select: none",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Label.vue missing expected style marker %q", marker)
		}
	}
}

// labelElement extracts the single top-level <label ...>...</label>
// wrapper element's source text so assertions can be scoped to it
// specifically, instead of matching markers anywhere in the whole file
// (including the header comment, which discusses the same attribute in
// prose).
func labelElement(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<label")
	if start == -1 {
		t.Fatalf("Label.vue missing expected wrapper <label>; source was:\n%s", src)
	}
	end := strings.Index(src[start:], "</label>")
	if end == -1 {
		t.Fatalf("Label.vue could not find wrapper </label>; source was:\n%s", src)
	}
	return src[start : start+end+len("</label>")]
}

func readLabel(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Label.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Label.vue) failed: %v", err)
	}
	return string(data)
}
