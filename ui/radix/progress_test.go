package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like separator_test.go/aspectratio_test.go, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Progress.vue into a real htmlc.Engine and checking rendered HTML for both
// the determinate and indeterminate cases) is out of scope here. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestProgress_ContainsBaselineMarkers(t *testing.T) {
	src := readProgress(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Progress.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestProgress_HasNoCustomElementScript(t *testing.T) {
	src := readProgress(t)

	// Pure data binding — no interactive client-side state, so no JS
	// enhancement needed or expected, same as AspectRatio.vue/Separator.vue.
	if strings.Contains(src, "<script customelement>") {
		t.Error("Progress.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

// TestProgress_UsesCustomDivNotNativeProgressElement confirms this port
// chose the custom `<div role="progressbar">` + inner fill technique (per
// this file's header comment) rather than the native `<progress>` element,
// by checking the rendered wrapper is a <div> carrying role="progressbar"
// and that no literal <progress> tag appears anywhere in the template.
func TestProgress_UsesCustomDivNotNativeProgressElement(t *testing.T) {
	src := readProgress(t)
	wrapper := wrapperDivProgress(t, src)
	template := templateBlock(t, src)

	if !strings.Contains(wrapper, `role="progressbar"`) {
		t.Errorf(`Progress.vue wrapper div missing expected marker role="progressbar"; wrapper was:\n%s`, wrapper)
	}
	// Scoped to the <template> block specifically (not the whole file):
	// the header comment above <template> discusses the native <progress>
	// element in prose while explaining why it was *not* chosen, the same
	// "don't match prose" scoping AspectRatio.vue's own test uses.
	if strings.Contains(template, "<progress") {
		t.Error("Progress.vue's <template> should not use a literal <progress> element; this port deliberately uses a custom <div role=\"progressbar\">")
	}
}

// TestProgress_DeterminateAriaBindings confirms the wrapper div carries the
// three determinate-case ARIA value attributes: a static aria-valuemin="0",
// a dynamic :aria-valuemax bound to max, and a conditional :aria-valuenow
// that only resolves to a real number when indeterminate is falsy.
func TestProgress_DeterminateAriaBindings(t *testing.T) {
	src := readProgress(t)
	wrapper := wrapperDivProgress(t, src)

	for _, marker := range []string{
		`aria-valuemin="0"`,
		`:aria-valuemax="max"`,
		`:aria-valuenow="indeterminate ? undefined : value"`,
	} {
		if !strings.Contains(wrapper, marker) {
			t.Errorf("Progress.vue wrapper div missing expected marker %q; wrapper was:\n%s", marker, wrapper)
		}
	}
}

// TestProgress_IndeterminateOmitsAriaValuenow is the adversarial-review
// check called out by this commit's process: confirm the indeterminate
// case genuinely *omits* aria-valuenow (by binding it to the `undefined`
// literal, which this engine's falsy-value-omission rule for :attr
// bindings drops from the rendered output entirely — see README.md's
// v-bind notes) rather than merely zeroing it out. `:aria-valuenow="0"` or
// any other numeric placeholder would be an ARIA-spec violation: it would
// announce "0% complete" instead of "progress is happening, amount
// unknown". The binding must route through a real ternary that evaluates
// to the `undefined` literal on the indeterminate branch, not a bare
// `value` (or `value ?? undefined`, which would only fall through for
// null/undefined `value`, not for `indeterminate === true` with a real
// numeric `value` still set) that always renders some value.
func TestProgress_IndeterminateOmitsAriaValuenow(t *testing.T) {
	src := readProgress(t)
	wrapper := wrapperDivProgress(t, src)

	if !strings.Contains(wrapper, `:aria-valuenow="indeterminate ? undefined : value"`) {
		t.Fatalf(`Progress.vue wrapper div missing expected conditional marker :aria-valuenow="indeterminate ? undefined : value"; wrapper was:\n%s`, wrapper)
	}
	// Guard against a regression to an always-present placeholder value
	// (e.g. binding straight to :aria-valuenow="value" with no ternary, or
	// a literal aria-valuenow="0" attribute) anywhere in the wrapper.
	if strings.Contains(wrapper, `aria-valuenow="0"`) {
		t.Error(`Progress.vue wrapper div must not render a zeroed aria-valuenow="0" placeholder for the indeterminate case`)
	}
}

// TestProgress_FillWidthPercentageMath double-checks the fill-width
// arithmetic for the custom-div technique: (value / max) * 100, expressed
// as a percentage string via expr's string-concatenation `+` operator (see
// expr/doc.go's Type Coercion section: `+` concatenates when either operand
// is a string) — not, say, a mistaken max/value inversion or a missing
// *100 that would leave a fractional 0..1 value in a "%" unit.
func TestProgress_FillWidthPercentageMath(t *testing.T) {
	src := readProgress(t)

	if !strings.Contains(src, `:style="indeterminate ? {} : { width: (value / max * 100) + '%' }"`) {
		t.Errorf(`Progress.vue indicator div missing expected fill-width marker; source was:\n%s`, src)
	}
}

// TestProgress_DataStateReflectsAllThreeStates confirms the data-state
// attribute (used by the indeterminate sweep animation in <style scoped>)
// resolves to all three of Radix's documented progress states —
// "indeterminate", "complete", and "loading" — rather than only
// distinguishing indeterminate from a single generic "determinate" bucket.
func TestProgress_DataStateReflectsAllThreeStates(t *testing.T) {
	src := readProgress(t)

	marker := `indeterminate ? 'indeterminate' : (value >= max ? 'complete' : 'loading')`
	if count := strings.Count(src, marker); count != 2 {
		t.Errorf("Progress.vue expected data-state expression %q to appear exactly twice (wrapper + indicator), got %d", marker, count)
	}
}

// wrapperDivProgress extracts the single top-level <div ...>...</div>
// wrapper element's source text (the outer progressbar div, not its nested
// indicator div) so assertions can be scoped to it specifically, instead of
// matching markers anywhere in the whole file.
func wrapperDivProgress(t *testing.T, src string) string {
	t.Helper()
	tplStart := strings.Index(src, "<template>")
	if tplStart == -1 {
		t.Fatalf("Progress.vue missing <template>; source was:\n%s", src)
	}
	start := strings.Index(src[tplStart:], "<div")
	if start == -1 {
		t.Fatalf("Progress.vue missing expected wrapper <div>; source was:\n%s", src)
	}
	start += tplStart
	// The wrapper div's own opening tag ends at the first unescaped ">"
	// that closes it; find that, then find the *next* nested <div (the
	// indicator) to bound the wrapper's own attribute region.
	openEnd := strings.Index(src[start:], ">")
	if openEnd == -1 {
		t.Fatalf("Progress.vue could not find end of wrapper <div> opening tag; source was:\n%s", src)
	}
	return src[start : start+openEnd+1]
}

// templateBlock extracts the <template>...</template> block's source text
// so assertions can be scoped to actual markup, excluding the header
// comment that precedes it (which discusses, in prose, techniques this
// component deliberately did not choose).
func templateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Progress.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

func readProgress(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Progress.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Progress.vue) failed: %v", err)
	}
	return string(data)
}
