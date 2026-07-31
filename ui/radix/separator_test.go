package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting Separator.vue into a real htmlc.Engine and
// checking the rendered HTML for both the decorative and non-decorative
// cases) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on, including — since this
// component has real conditional attribute logic, unlike VisuallyHidden's
// static markup — both branches of the decorative/non-decorative split.
func TestSeparator_ContainsBaselineMarkers(t *testing.T) {
	src := readSeparator(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Separator.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestSeparator_HasNoCustomElementScript(t *testing.T) {
	src := readSeparator(t)

	// Purely semantic/static — no JS enhancement needed or expected, same
	// as VisuallyHidden.vue.
	if strings.Contains(src, "<script customelement>") {
		t.Error("Separator.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

// TestSeparator_NonDecorativeBranch confirms the branch of the template
// that renders when decorative is falsy carries role="separator" and a
// conditional aria-orientation binding — and, critically, does NOT also
// carry aria-hidden, which would contradict "meaningfully part of the
// page's content structure".
func TestSeparator_NonDecorativeBranch(t *testing.T) {
	src := readSeparator(t)
	branch := nonDecorativeBranch(t, src)

	for _, marker := range []string{
		`role="separator"`,
		`:aria-orientation="orientation === 'vertical' ? 'vertical' : undefined"`,
	} {
		if !strings.Contains(branch, marker) {
			t.Errorf("Separator.vue non-decorative (v-else) branch missing expected marker %q; branch was:\n%s", marker, branch)
		}
	}
	if strings.Contains(branch, "aria-hidden") {
		t.Errorf("Separator.vue non-decorative (v-else) branch must not carry aria-hidden; branch was:\n%s", branch)
	}
}

// TestSeparator_DecorativeBranch confirms the branch of the template that
// renders when decorative is truthy (v-if="decorative") carries
// role="none" (matching Radix Primitives' own real behavior — this removes
// the "separator" semantic without pulling the element out of the
// accessibility tree entirely, unlike aria-hidden) and, critically, never
// carries role="separator" or aria-orientation — a decorative separator
// that still announced role="separator" would be an accessibility
// regression, the opposite of what "decorative" means.
func TestSeparator_DecorativeBranch(t *testing.T) {
	src := readSeparator(t)
	branch := decorativeBranch(t, src)

	if !strings.Contains(branch, `v-if="decorative"`) {
		t.Fatalf("Separator.vue missing expected v-if=\"decorative\" branch; source was:\n%s", src)
	}
	if !strings.Contains(branch, `role="none"`) {
		t.Errorf(`Separator.vue decorative (v-if) branch missing expected marker role="none"; branch was:\n%s`, branch)
	}
	if strings.Contains(branch, "aria-hidden") {
		t.Errorf("Separator.vue decorative (v-if) branch must not carry aria-hidden; branch was:\n%s", branch)
	}
	if strings.Contains(branch, `role="separator"`) {
		t.Errorf("Separator.vue decorative (v-if) branch must not carry role=\"separator\"; branch was:\n%s", branch)
	}
	if strings.Contains(branch, "aria-orientation") {
		t.Errorf("Separator.vue decorative (v-if) branch must not carry aria-orientation; branch was:\n%s", branch)
	}
}

// TestSeparator_BothBranchesShareOrientationData confirms both branches
// expose the raw orientation via data-orientation, independent of the
// decorative/role split, so CSS (and any consumer script) can still tell
// horizontal from vertical regardless of accessibility mode.
func TestSeparator_BothBranchesShareOrientationData(t *testing.T) {
	src := readSeparator(t)

	if count := strings.Count(src, `:data-orientation="orientation"`); count != 2 {
		t.Errorf(`Separator.vue expected :data-orientation="orientation" to appear exactly twice (once per branch), got %d`, count)
	}
}

// TestSeparator_DefaultAppearance confirms the CSS gives sensible thin-line
// sizing for both orientations, and documents the "vertical needs a
// height-establishing parent" gotcha in prose near the relevant rule.
func TestSeparator_DefaultAppearance(t *testing.T) {
	src := readSeparator(t)

	for _, marker := range []string{
		`[data-orientation='horizontal']`,
		`[data-orientation='vertical']`,
		"width: 100%",
		"height: 100%",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Separator.vue missing expected style marker %q", marker)
		}
	}
}

// nonDecorativeBranch extracts the <div v-else ...></div> element's source
// text so assertions can be scoped to that branch specifically, instead of
// matching markers anywhere in the whole file (which would pass even if a
// marker ended up in the wrong branch).
func nonDecorativeBranch(t *testing.T, src string) string {
	t.Helper()
	idx := strings.Index(src, "v-else")
	if idx == -1 {
		t.Fatalf("Separator.vue missing expected v-else branch; source was:\n%s", src)
	}
	// Walk backward to the start of this <div ...> and forward to its
	// closing </div>, giving just this one element's markup.
	start := strings.LastIndex(src[:idx], "<div")
	end := strings.Index(src[idx:], "</div>")
	if start == -1 || end == -1 {
		t.Fatalf("Separator.vue could not isolate v-else branch; source was:\n%s", src)
	}
	return src[start : idx+end+len("</div>")]
}

// decorativeBranch is the v-if analogue of nonDecorativeBranch.
func decorativeBranch(t *testing.T, src string) string {
	t.Helper()
	idx := strings.Index(src, `v-if="decorative"`)
	if idx == -1 {
		t.Fatalf(`Separator.vue missing expected v-if="decorative" branch; source was:\n%s`, src)
	}
	start := strings.LastIndex(src[:idx], "<div")
	end := strings.Index(src[idx:], "</div>")
	if start == -1 || end == -1 {
		t.Fatalf("Separator.vue could not isolate v-if branch; source was:\n%s", src)
	}
	return src[start : idx+end+len("</div>")]
}

func readSeparator(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Separator.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Separator.vue) failed: %v", err)
	}
	return string(data)
}
