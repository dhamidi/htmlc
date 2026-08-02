package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like tabs_test.go/progress_test.go, this module has no dependency on the
// root htmlc package, so a full render-based test (mounting Checkbox.vue
// into a real htmlc.Engine and checking rendered HTML) is out of scope
// here — that proof is deliberately deferred to the examples/radix-demo
// commit, which does depend on root htmlc. These are content-sanity
// checks: they confirm the component's source file contains the markers
// this commit's design depends on.
func TestCheckbox_ContainsZeroJSBaseline(t *testing.T) {
	src := readCheckbox(t)

	for _, marker := range []string{
		"<template>",
		`type="checkbox"`,
		"radix-checkbox-input",
		"radix-checkbox-box",
		`:checked="checked && !indeterminate"`,
		`:disabled="disabled"`,
		`:required="required"`,
		`:id="id"`,
		`:name="name"`,
		`:value="value"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Checkbox.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestCheckbox_ContainsScopedStyle(t *testing.T) {
	src := readCheckbox(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Checkbox.vue missing expected <style scoped> block")
	}
}

// TestCheckbox_ContainsCustomElementEnhancement confirms the
// <script customelement> block exists and does the one thing this
// component's header comment says is genuinely impossible any other way:
// assign `true` (a real, literal JS boolean, not a variable) to the native
// input's `.indeterminate` IDL property. HTML has no `indeterminate`
// attribute, so this JS-only assignment is this component's sole path to
// ever reaching the mixed state at all.
func TestCheckbox_ContainsCustomElementEnhancement(t *testing.T) {
	src := readCheckbox(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-checkbox'",
		".indeterminate = true",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Checkbox.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestCheckbox_AriaCheckedMixedOverride confirms the script explicitly sets
// aria-checked="mixed" alongside the .indeterminate assignment (this port's
// documented, verified-necessary belt-and-suspenders override — see this
// file's header comment on browser/AT inconsistency), and that it is
// cleared again rather than left stuck once indeterminate no longer holds.
func TestCheckbox_AriaCheckedMixedOverride(t *testing.T) {
	src := readCheckbox(t)

	for _, marker := range []string{
		`setAttribute('aria-checked', 'mixed')`,
		`removeAttribute('aria-checked')`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Checkbox.vue missing expected aria-checked marker %q", marker)
		}
	}
}

// TestCheckbox_IndeterminateDegradesToUncheckedBaseline is the adversarial-
// review check called out by this commit's process: an indeterminate
// checkbox must render as a plain unchecked box in the zero-JS baseline,
// never an accidentally-checked one. The native `checked` attribute must
// therefore be bound through a real `checked && !indeterminate` expression
// (not a bare `checked`, which would leak `checked=true` into the markup
// whenever a caller passes both `checked=true` and `indeterminate=true`).
func TestCheckbox_IndeterminateDegradesToUncheckedBaseline(t *testing.T) {
	src := readCheckbox(t)
	tpl := checkboxTemplateBlock(t, src)

	inputTag := checkboxInputTag(t, tpl)
	if !strings.Contains(inputTag, `:checked="checked && !indeterminate"`) {
		t.Errorf(`Checkbox.vue <input> missing expected marker :checked="checked && !indeterminate"; input tag was:\n%s`, inputTag)
	}
	// Guard against a regression to a bare, unconditional :checked="checked"
	// binding anywhere in the input tag, which would let checked=true leak
	// through even while indeterminate=true.
	if strings.Contains(inputTag, `:checked="checked"`) {
		t.Error(`Checkbox.vue <input> must not bind :checked="checked" unconditionally; indeterminate must force the baseline to unchecked`)
	}

	// The CSS painting the "mixed" dash must key off the real
	// :indeterminate pseudo-class, not a data attribute the template can
	// set unconditionally — that pseudo-class can only ever match once
	// script has actually run, which is exactly what keeps the zero-JS
	// baseline from showing a "mixed" look it hasn't earned.
	if !strings.Contains(src, ":indeterminate + .radix-checkbox-box") {
		t.Error(`Checkbox.vue missing expected CSS selector ":indeterminate + .radix-checkbox-box" — the mixed-state dash must be driven by the real :indeterminate pseudo-class, not a static attribute`)
	}
}

// TestCheckbox_SiblingSelectorDOMOrdering verifies the DOM ordering the
// :checked/:indeterminate adjacent-sibling CSS selectors below depend on
// (matching the same care Tabs.vue's own commit took for its own hidden
// radios + labels): the native <input> must appear before, and immediately
// adjacent to (no intervening element), the <label class="radix-checkbox-
// box"> it is paired with, and the stylesheet must actually contain the
// adjacent-sibling selectors that assume that ordering.
func TestCheckbox_SiblingSelectorDOMOrdering(t *testing.T) {
	src := readCheckbox(t)
	tpl := checkboxTemplateBlock(t, src)

	inputIdx := strings.Index(tpl, "<input")
	labelIdx := strings.Index(tpl, "<label")
	if inputIdx == -1 || labelIdx == -1 {
		t.Fatalf("Checkbox.vue <template> missing <input> and/or <label>; template was:\n%s", tpl)
	}
	if inputIdx >= labelIdx {
		t.Fatalf("Checkbox.vue <template> must place <input> before <label> for the :checked/:indeterminate + selector to reach it; template was:\n%s", tpl)
	}

	// Nothing but the input's own closing "/>"  and whitespace may sit
	// between the two tags, or the adjacent-sibling ("+") selector below
	// would not actually match in a real browser.
	inputTag := checkboxInputTag(t, tpl)
	inputEnd := inputIdx + len(inputTag)
	between := tpl[inputEnd:labelIdx]
	if strings.TrimSpace(between) != "" {
		t.Errorf("Checkbox.vue <template> has non-whitespace content between <input> and <label>, which would break the adjacent-sibling selector; found %q", between)
	}

	for _, marker := range []string{
		".radix-checkbox-input:checked + .radix-checkbox-box",
		".radix-checkbox-input:indeterminate + .radix-checkbox-box",
		".radix-checkbox-input:focus-visible + .radix-checkbox-box",
		".radix-checkbox-input:disabled + .radix-checkbox-box",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Checkbox.vue missing expected sibling-selector CSS rule %q", marker)
		}
	}
}

// TestCheckbox_InputStaysInTabOrder is the adversarial-review check called
// out by this commit's process, mirroring the one Tabs.vue's own commit
// performed for its hidden radios: confirm the visually-hidden clip
// technique used on the native <input> does not also pull it out of the
// tab order or the accessibility tree. Unlike Tabs.vue's own <script
// customelement> (which deliberately relocates ARIA tab semantics off its
// hidden radio and onto the visible label), this component's real
// interactive control *is* the native input, so nothing here — baseline or
// script — may ever mark it tabindex="-1" or aria-hidden.
func TestCheckbox_InputStaysInTabOrder(t *testing.T) {
	src := readCheckbox(t)
	tpl := checkboxTemplateBlock(t, src)
	inputTag := checkboxInputTag(t, tpl)

	for _, forbidden := range []string{`tabindex="-1"`, `tabIndex = -1`, `aria-hidden`} {
		if strings.Contains(inputTag, forbidden) {
			t.Errorf("Checkbox.vue <input> must stay in the tab order and accessibility tree, but its tag contains %q; input tag was:\n%s", forbidden, inputTag)
		}
	}

	// display:none on the input's own visually-hidden rule would remove it
	// from the tab order entirely; the clip-path/opacity/position technique
	// must be used instead (same requirement Tabs.vue's own hidden-input
	// CSS satisfies).
	inputRule := checkboxCSSRule(t, src, ".radix-visually-hidden-input {")
	if strings.Contains(inputRule, "display: none") || strings.Contains(inputRule, "display:none") {
		t.Errorf(".radix-visually-hidden-input rule must not use display:none (would remove it from the tab order); rule was:\n%s", inputRule)
	}
	if !strings.Contains(inputRule, "clip-path") {
		t.Errorf(".radix-visually-hidden-input rule missing expected clip-path visually-hidden technique; rule was:\n%s", inputRule)
	}
}

// TestCheckbox_LabelMarkedVNative confirms the <label> box is explicitly
// declared native via v-native. Without it, this package's own Label.vue
// (mounted alongside Checkbox.vue in the same "radix" component set)
// registers a lowercase "label" alias that a bare <label> tag here would
// resolve against instead of rendering as plain HTML — confirmed
// empirically against Tabs.vue's own (unrelated, pre-existing) <label>
// during this component's development. See this file's header comment.
func TestCheckbox_LabelMarkedVNative(t *testing.T) {
	src := readCheckbox(t)
	tpl := checkboxTemplateBlock(t, src)

	labelIdx := strings.Index(tpl, "<label")
	if labelIdx == -1 {
		t.Fatalf("Checkbox.vue <template> missing <label>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[labelIdx:], ">")
	if end == -1 {
		t.Fatalf("Checkbox.vue could not find end of <label> opening tag; template was:\n%s", tpl)
	}
	labelTag := tpl[labelIdx : labelIdx+end+1]
	if !strings.Contains(labelTag, "v-native") {
		t.Errorf("Checkbox.vue <label> missing expected v-native marker; label tag was:\n%s", labelTag)
	}
}

// checkboxTemplateBlock extracts the <template>...</template> block's
// source text, excluding the header comment (which discusses, in prose,
// techniques and traps this component deliberately reasons about) — same
// scoping technique Progress.vue's own test uses.
func checkboxTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Checkbox.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// checkboxInputTag extracts the single <input ...> tag's own source text
// (its full opening/self-closing tag only) from a <template> block, so
// assertions can be scoped to it specifically instead of matching anywhere
// in the whole template.
func checkboxInputTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<input")
	if start == -1 {
		t.Fatalf("Checkbox.vue <template> missing <input>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[start:], ">")
	if end == -1 {
		t.Fatalf("Checkbox.vue could not find end of <input> tag; template was:\n%s", tpl)
	}
	return tpl[start : start+end+1]
}

// checkboxCSSRule extracts a single CSS rule's source text (from its
// opening "selector {" marker through the matching closing "}") so
// assertions can be scoped to just that rule.
func checkboxCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("Checkbox.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("Checkbox.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}

func readCheckbox(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Checkbox.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Checkbox.vue) failed: %v", err)
	}
	return string(data)
}
