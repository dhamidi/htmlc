package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like separator_test.go, this module has no dependency on the root htmlc
// package, so a full render-based test (mounting AspectRatio.vue into a
// real htmlc.Engine and checking rendered HTML) is out of scope here. These
// are content-sanity checks: they confirm the component's source file
// contains the markers this commit's design depends on, scoped to the
// relevant wrapper markup rather than loose whole-file substring checks.
func TestAspectRatio_ContainsBaselineMarkers(t *testing.T) {
	src := readAspectRatio(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AspectRatio.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestAspectRatio_HasNoCustomElementScript(t *testing.T) {
	src := readAspectRatio(t)

	// Purely CSS — no JS enhancement needed or expected, same as
	// VisuallyHidden.vue and Separator.vue.
	if strings.Contains(src, "<script customelement>") {
		t.Error("AspectRatio.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

// TestAspectRatio_WrapperUsesModernAspectRatioProperty confirms the
// wrapper element binds `ratio` into the CSS `aspect-ratio` property via
// object-syntax :style (which this codebase's :style implementation
// converts from camelCase `aspectRatio` to kebab-case `aspect-ratio` at
// render time — see README.md's v-bind notes and renderer.go's
// camelToKebab), scoped to the wrapper element specifically.
func TestAspectRatio_WrapperUsesModernAspectRatioProperty(t *testing.T) {
	src := readAspectRatio(t)
	wrapper := wrapperDiv(t, src)

	if !strings.Contains(wrapper, `:style="{ aspectRatio: ratio }"`) {
		t.Errorf(`AspectRatio.vue wrapper div missing expected marker :style="{ aspectRatio: ratio }"; wrapper was:\n%s`, wrapper)
	}
	if !strings.Contains(wrapper, `data-radix-aspect-ratio-wrapper=""`) {
		t.Errorf(`AspectRatio.vue wrapper div missing expected marker data-radix-aspect-ratio-wrapper=""; wrapper was:\n%s`, wrapper)
	}
	if !strings.Contains(wrapper, "<slot></slot>") {
		t.Errorf("AspectRatio.vue wrapper div missing expected default <slot></slot>; wrapper was:\n%s", wrapper)
	}
}

// TestAspectRatio_DoesNotUseLegacyPaddingBottomTrick confirms this port
// chose the modern single-element `aspect-ratio` CSS property technique
// (per this file's header comment) rather than upstream Radix's classic
// padding-bottom-percentage + absolutely-positioned-inner-element trick,
// which would need a second wrapped element and different CSS properties
// entirely.
func TestAspectRatio_DoesNotUseLegacyPaddingBottomTrick(t *testing.T) {
	src := readAspectRatio(t)
	wrapper := wrapperDiv(t, src)

	for _, marker := range []string{
		"padding-bottom",
		"paddingBottom",
		"position: absolute",
		"position:absolute",
	} {
		if strings.Contains(wrapper, marker) {
			t.Errorf("AspectRatio.vue wrapper div should not use the legacy padding-bottom trick; found unexpected marker %q in wrapper:\n%s", marker, wrapper)
		}
	}
}

// TestAspectRatio_StyleEstablishesDefiniteWidth confirms the CSS gives the
// wrapper a definite width to compute its aspect-ratio-derived height
// from, and clips overflowing slotted content instead of leaking it
// outside the ratio-constrained box.
func TestAspectRatio_StyleEstablishesDefiniteWidth(t *testing.T) {
	src := readAspectRatio(t)

	for _, marker := range []string{
		"width: 100%",
		"overflow: hidden",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AspectRatio.vue missing expected style marker %q", marker)
		}
	}
}

// wrapperDiv extracts the single top-level <div ...>...</div> wrapper
// element's source text so assertions can be scoped to it specifically,
// instead of matching markers anywhere in the whole file (including doc
// comments, which mention some of the same technique names in prose while
// explaining why they were *not* chosen).
func wrapperDiv(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<div")
	if start == -1 {
		t.Fatalf("AspectRatio.vue missing expected wrapper <div>; source was:\n%s", src)
	}
	end := strings.Index(src[start:], "</div>")
	if end == -1 {
		t.Fatalf("AspectRatio.vue could not find wrapper </div>; source was:\n%s", src)
	}
	return src[start : start+end+len("</div>")]
}

func readAspectRatio(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/AspectRatio.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/AspectRatio.vue) failed: %v", err)
	}
	return string(data)
}
