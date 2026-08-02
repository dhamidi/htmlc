package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like checkbox_test.go/tabs_test.go/progress_test.go, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Switch.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file
// contains the markers this commit's design depends on.
func TestSwitch_ContainsZeroJSBaseline(t *testing.T) {
	src := readSwitch(t)

	for _, marker := range []string{
		"<template>",
		`type="checkbox"`,
		`role="switch"`,
		"radix-switch-input",
		"radix-switch-track",
		`:checked="checked"`,
		`:disabled="disabled"`,
		`:required="required"`,
		`:id="id"`,
		`:name="name"`,
		`:value="value"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Switch.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestSwitch_RoleSwitchOnRealInput confirms role="switch" is a static
// attribute on the actual native <input type="checkbox">, not on the
// decorative track label — this is the load-bearing fact behind this
// component's whole no-JS design (see the file's header comment): the
// browser only computes aria-checked from the real checked state on the
// element that both the role and the checkbox behavior live on.
func TestSwitch_RoleSwitchOnRealInput(t *testing.T) {
	src := readSwitch(t)
	tpl := switchTemplateBlock(t, src)
	inputTag := switchInputTag(t, tpl)

	if !strings.Contains(inputTag, `role="switch"`) {
		t.Errorf(`Switch.vue <input> missing expected role="switch"; input tag was:\n%s`, inputTag)
	}
	if !strings.Contains(inputTag, `type="checkbox"`) {
		t.Errorf(`Switch.vue <input> missing expected type="checkbox"; input tag was:\n%s`, inputTag)
	}
}

func TestSwitch_ContainsScopedStyle(t *testing.T) {
	src := readSwitch(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Switch.vue missing expected <style scoped> block")
	}
}

// TestSwitch_NoCustomElementScript confirms this component ships with no
// <script customelement> block, and documents why in the assertion
// itself: role="switch" on a native <input type="checkbox"> is real,
// well-supported ARIA usage — the browser maps the input's checked state
// to aria-checked automatically, and both checked/unchecked states are
// reachable from plain markup (:checked/:checked="checked"), unlike
// Checkbox.vue's indeterminate state, which has no HTML attribute
// equivalent at all. There is nothing left for a script to do.
func TestSwitch_NoCustomElementScript(t *testing.T) {
	src := readSwitch(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("Switch.vue should not contain a <script customelement> block: role=\"switch\" on a native checkbox input needs no JS for correctness, and the thumb-slide transition already works off the real :checked pseudo-class with zero script involvement — see the file's header comment")
	}
}

// TestSwitch_SiblingSelectorDOMOrdering verifies the DOM ordering the
// :checked adjacent-sibling CSS selector depends on, matching the same
// care Checkbox.vue's own test takes: the native <input> must appear
// before, and immediately adjacent to (no intervening element), the
// <label class="radix-switch-track"> it is paired with, and the
// stylesheet must actually contain the adjacent-sibling selectors that
// assume that ordering.
func TestSwitch_SiblingSelectorDOMOrdering(t *testing.T) {
	src := readSwitch(t)
	tpl := switchTemplateBlock(t, src)

	inputIdx := strings.Index(tpl, "<input")
	labelIdx := strings.Index(tpl, "<label")
	if inputIdx == -1 || labelIdx == -1 {
		t.Fatalf("Switch.vue <template> missing <input> and/or <label>; template was:\n%s", tpl)
	}
	if inputIdx >= labelIdx {
		t.Fatalf("Switch.vue <template> must place <input> before <label> for the :checked + selector to reach it; template was:\n%s", tpl)
	}

	inputTag := switchInputTag(t, tpl)
	inputEnd := inputIdx + len(inputTag)
	between := tpl[inputEnd:labelIdx]
	if strings.TrimSpace(between) != "" {
		t.Errorf("Switch.vue <template> has non-whitespace content between <input> and <label>, which would break the adjacent-sibling selector; found %q", between)
	}

	for _, marker := range []string{
		".radix-switch-input:checked + .radix-switch-track",
		".radix-switch-input:focus-visible + .radix-switch-track",
		".radix-switch-input:disabled + .radix-switch-track",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Switch.vue missing expected sibling-selector CSS rule %q", marker)
		}
	}
}

// TestSwitch_ThumbTransitionIsCSSOnly confirms the thumb-slide animation
// is driven purely by a CSS `transition` on `transform`, keyed off the
// real :checked pseudo-class match — the specific claim this commit's
// self-adversarial review had to verify by hand before concluding no
// <script customelement> was needed even for visual polish.
func TestSwitch_ThumbTransitionIsCSSOnly(t *testing.T) {
	src := readSwitch(t)

	if !strings.Contains(src, "transition: transform") {
		t.Error(`Switch.vue missing expected "transition: transform" rule on the thumb pseudo-element`)
	}
	if !strings.Contains(src, ".radix-switch-input:checked + .radix-switch-track::after") {
		t.Error(`Switch.vue missing expected checked-state rule ".radix-switch-input:checked + .radix-switch-track::after" driving the thumb position`)
	}
	if !strings.Contains(src, "translateX") {
		t.Error(`Switch.vue missing expected "translateX" thumb-slide transform`)
	}
}

// TestSwitch_InputStaysInTabOrder mirrors Checkbox.vue's own check:
// confirm the visually-hidden clip technique used on the native <input>
// does not also pull it out of the tab order or the accessibility tree.
func TestSwitch_InputStaysInTabOrder(t *testing.T) {
	src := readSwitch(t)
	tpl := switchTemplateBlock(t, src)
	inputTag := switchInputTag(t, tpl)

	for _, forbidden := range []string{`tabindex="-1"`, `tabIndex = -1`, `aria-hidden`} {
		if strings.Contains(inputTag, forbidden) {
			t.Errorf("Switch.vue <input> must stay in the tab order and accessibility tree, but its tag contains %q; input tag was:\n%s", forbidden, inputTag)
		}
	}

	inputRule := switchCSSRule(t, src, ".radix-visually-hidden-input {")
	if strings.Contains(inputRule, "display: none") || strings.Contains(inputRule, "display:none") {
		t.Errorf(".radix-visually-hidden-input rule must not use display:none (would remove it from the tab order); rule was:\n%s", inputRule)
	}
	if !strings.Contains(inputRule, "clip-path") {
		t.Errorf(".radix-visually-hidden-input rule missing expected clip-path visually-hidden technique; rule was:\n%s", inputRule)
	}
}

// TestSwitch_LabelMarkedVNative confirms the <label> track is explicitly
// declared native via v-native. Without it, this package's own Label.vue
// (mounted alongside Switch.vue in the same "radix" component set)
// registers a lowercase "label" alias that a bare <label> tag here would
// resolve against instead of rendering as plain HTML — the exact
// collision Checkbox.vue's own commit discovered and fixed, applied here
// proactively per this commit's process instructions.
func TestSwitch_LabelMarkedVNative(t *testing.T) {
	src := readSwitch(t)
	tpl := switchTemplateBlock(t, src)

	labelIdx := strings.Index(tpl, "<label")
	if labelIdx == -1 {
		t.Fatalf("Switch.vue <template> missing <label>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[labelIdx:], ">")
	if end == -1 {
		t.Fatalf("Switch.vue could not find end of <label> opening tag; template was:\n%s", tpl)
	}
	labelTag := tpl[labelIdx : labelIdx+end+1]
	if !strings.Contains(labelTag, "v-native") {
		t.Errorf("Switch.vue <label> missing expected v-native marker; label tag was:\n%s", labelTag)
	}
}

// switchTemplateBlock extracts the <template>...</template> block's
// source text, excluding the header comment — same scoping technique
// checkbox_test.go's checkboxTemplateBlock uses.
func switchTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Switch.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// switchInputTag extracts the single <input ...> tag's own source text
// (its full opening/self-closing tag only) from a <template> block.
func switchInputTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<input")
	if start == -1 {
		t.Fatalf("Switch.vue <template> missing <input>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[start:], ">")
	if end == -1 {
		t.Fatalf("Switch.vue could not find end of <input> tag; template was:\n%s", tpl)
	}
	return tpl[start : start+end+1]
}

// switchCSSRule extracts a single CSS rule's source text (from its
// opening "selector {" marker through the matching closing "}").
func switchCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("Switch.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("Switch.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}

func readSwitch(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Switch.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Switch.vue) failed: %v", err)
	}
	return string(data)
}
