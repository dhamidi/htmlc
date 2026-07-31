package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like checkbox_test.go/switch_test.go/progress_test.go, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Slider.vue into a real htmlc.Engine and checking rendered HTML)
// is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestSlider_ContainsZeroJSBaseline(t *testing.T) {
	src := readSlider(t)

	for _, marker := range []string{
		"<template>",
		`type="range"`,
		`class="radix-slider-input"`,
		`class="radix-slider-track"`,
		`class="radix-slider-range"`,
		`:min="min"`,
		`:max="max"`,
		`:step="step"`,
		`:value="v"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Slider.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestSlider_LoopsOverValues confirms one native <input type="range"> is
// rendered per entry in the `values` prop — the load-bearing structural
// decision behind this component's whole multi-thumb design (see the file's
// header comment): a single-element array renders a plain one-thumb slider,
// a two-element array renders a min/max range slider, and this generalizes
// to N thumbs without a separate single-vs-multi-thumb component variant.
func TestSlider_LoopsOverValues(t *testing.T) {
	src := readSlider(t)
	tpl := sliderTemplateBlock(t, src)

	if !strings.Contains(tpl, `v-for="(v, index) in values"`) {
		t.Errorf(`Slider.vue <template> missing expected "v-for=\"(v, index) in values\"" loop over the values prop; template was:\n%s`, tpl)
	}

	inputTag := sliderInputTag(t, tpl)
	for _, marker := range []string{`type="range"`, `:min="min"`, `:max="max"`, `:step="step"`, `:value="v"`} {
		if !strings.Contains(inputTag, marker) {
			t.Errorf("Slider.vue <input> missing expected marker %q; input tag was:\n%s", marker, inputTag)
		}
	}

	// The v-for loop variable ("v") must actually be the identifier bound
	// to :value, not some other stray identifier — a copy/paste slip here
	// (e.g. binding :value to "value" instead of the loop var "v") would
	// silently make every thumb render the same value.
	if !strings.Contains(inputTag, `:value="v"`) {
		t.Errorf(`Slider.vue <input> must bind :value to the v-for loop variable "v"; input tag was:\n%s`, inputTag)
	}
}

// TestSlider_PointerEventsLayering verifies the CSS-only thumb-overlap
// disambiguation technique documented in Slider.vue's header comment: the
// host input's pointer-events are turned off, and then explicitly turned
// back on for both vendor thumb pseudo-elements. Both halves must be
// present — the input-level "none" alone (with no counteracting "auto" on
// the thumb pseudo-elements) is exactly the self-adversarial-review mistake
// called out in the header comment and the commit's process instructions:
// it would make the whole input, thumb included, unclickable rather than
// just its empty track.
func TestSlider_PointerEventsLayering(t *testing.T) {
	src := readSlider(t)

	inputRule := sliderCSSRule(t, src, ".radix-slider-input {")
	if !strings.Contains(inputRule, "pointer-events: none") {
		t.Errorf(".radix-slider-input rule missing expected \"pointer-events: none\"; rule was:\n%s", inputRule)
	}

	webkitThumbRule := sliderCSSRule(t, src, ".radix-slider-input::-webkit-slider-thumb {")
	if !strings.Contains(webkitThumbRule, "pointer-events: auto") {
		t.Errorf("::-webkit-slider-thumb rule missing expected counteracting \"pointer-events: auto\"; rule was:\n%s", webkitThumbRule)
	}

	mozThumbRule := sliderCSSRule(t, src, ".radix-slider-input::-moz-range-thumb {")
	if !strings.Contains(mozThumbRule, "pointer-events: auto") {
		t.Errorf("::-moz-range-thumb rule missing expected counteracting \"pointer-events: auto\"; rule was:\n%s", mozThumbRule)
	}

	// The empty-track pseudo-elements must stay inert (this is what makes
	// the fix work at all: an input's own track no longer shadows the thumb
	// of an input stacked underneath it).
	for _, selector := range []string{
		".radix-slider-input::-webkit-slider-runnable-track {",
		".radix-slider-input::-moz-range-track {",
	} {
		rule := sliderCSSRule(t, src, selector)
		if !strings.Contains(rule, "pointer-events: none") {
			t.Errorf("%s rule missing expected \"pointer-events: none\"; rule was:\n%s", selector, rule)
		}
	}
}

// TestSlider_HoverFocusActiveRaisesZIndex confirms the z-index bump that
// lets an overlapping/near-overlapping thumb be "dug out" from underneath
// another one, with zero JS — step 3 of the thumb-overlap fix documented in
// the header comment.
func TestSlider_HoverFocusActiveRaisesZIndex(t *testing.T) {
	src := readSlider(t)

	rule := sliderCSSRule(t, src, ".radix-slider-input:hover,")
	for _, marker := range []string{":hover", ":focus-visible", ":active", "z-index: 2"} {
		if !strings.Contains(rule, marker) {
			t.Errorf("expected hover/focus/active z-index rule missing marker %q; rule was:\n%s", marker, rule)
		}
	}

	baseRule := sliderCSSRule(t, src, ".radix-slider-input {")
	if !strings.Contains(baseRule, "z-index: 1") {
		t.Errorf(".radix-slider-input base rule missing expected default \"z-index: 1\" (needed for the hover/focus/active bump to have somewhere lower to raise from); rule was:\n%s", baseRule)
	}
}

// TestSlider_TrackFillPercentageMath verifies the visual range-fill bar's
// :style binding contains the expected percentage formula: left/width
// driven by (values[0] - min) / (max - min) * 100 for the low end and
// (values[values.length - 1] - min) / (max - min) * 100 for the high end,
// matching this file's header-comment hand trace (min=0, max=100,
// values=[25,75] => fill spanning 25%-75%).
func TestSlider_TrackFillPercentageMath(t *testing.T) {
	src := readSlider(t)
	tpl := sliderTemplateBlock(t, src)

	rangeIdx := strings.Index(tpl, `class="radix-slider-range"`)
	if rangeIdx == -1 {
		t.Fatalf(`Slider.vue <template> missing <span class="radix-slider-range">; template was:\n%s`, tpl)
	}
	// Grab the :style="{ ... }" binding's own source text. Scanning for the
	// tag's first ">" would stop early, inside the expression itself, since
	// the fill formula's "values.length > 1" comparison contains a bare ">"
	// well before the tag's real closing ">" — so this instead locates the
	// :style="{ opener and scans forward for the object literal's closing
	// "}" immediately followed by the attribute's closing quote.
	styleOpen := strings.Index(tpl[rangeIdx:], `:style="{`)
	if styleOpen == -1 {
		t.Fatalf(`Slider.vue .radix-slider-range missing expected :style="{...}" binding; template was:\n%s`, tpl)
	}
	styleStart := rangeIdx + styleOpen
	styleCloseRel := strings.Index(tpl[styleStart:], `}"`)
	if styleCloseRel == -1 {
		t.Fatalf(`Slider.vue could not find end of .radix-slider-range :style binding; template was:\n%s`, tpl)
	}
	rangeTag := tpl[styleStart : styleStart+styleCloseRel+2]

	for _, marker := range []string{
		"values[0]",
		"values[values.length - 1]",
		"- min) / (max - min) * 100",
		"values.length > 1",
	} {
		if !strings.Contains(rangeTag, marker) {
			t.Errorf(".radix-slider-range :style binding missing expected fill-percentage marker %q; binding was:\n%s", marker, rangeTag)
		}
	}
}

// TestSlider_NoCustomElementScript confirms this component ships with no
// <script customelement> block: every real interaction (drag, keyboard,
// focus, implicit ARIA) comes from the stacked native <input type="range">
// elements for free, and the thumb-overlap fix is pure CSS (see the file's
// header comment for the hand-traced reasoning) — there is nothing left for
// a script to do.
func TestSlider_NoCustomElementScript(t *testing.T) {
	src := readSlider(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("Slider.vue should not contain a <script customelement> block: dragging/keyboard/focus/ARIA all come from the stacked native <input type=\"range\"> elements for free, and thumb-overlap disambiguation is handled entirely in CSS — see the file's header comment")
	}
}

func TestSlider_ContainsScopedStyle(t *testing.T) {
	src := readSlider(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Slider.vue missing expected <style scoped> block")
	}
}

// TestSlider_InputStaysFocusable confirms the stacked native inputs are
// never pulled out of the tab order or accessibility tree — unlike
// Checkbox.vue's/Switch.vue's own hidden inputs (which are visually hidden
// but deliberately still focusable), Slider.vue's inputs are the entire
// visible+interactive surface, so there is no visually-hidden clip
// technique here to check, only that nothing disables them outright.
func TestSlider_InputStaysFocusable(t *testing.T) {
	src := readSlider(t)
	tpl := sliderTemplateBlock(t, src)
	inputTag := sliderInputTag(t, tpl)

	for _, forbidden := range []string{`tabindex="-1"`, `tabIndex = -1`, `aria-hidden`, "display: none", "display:none"} {
		if strings.Contains(inputTag, forbidden) {
			t.Errorf("Slider.vue <input> must stay focusable and visible, but its tag contains %q; input tag was:\n%s", forbidden, inputTag)
		}
	}
}

// sliderTemplateBlock extracts the <template>...</template> block's source
// text, excluding the header comment — same scoping technique
// switchTemplateBlock/checkboxTemplateBlock use in this package's other
// content-sanity tests.
func sliderTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Slider.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// sliderInputTag extracts the single <input ...> tag's own source text (its
// full opening/self-closing tag only) from a <template> block.
func sliderInputTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<input")
	if start == -1 {
		t.Fatalf("Slider.vue <template> missing <input>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[start:], "/>")
	if end == -1 {
		end = strings.Index(tpl[start:], ">")
		if end == -1 {
			t.Fatalf("Slider.vue could not find end of <input> tag; template was:\n%s", tpl)
		}
		return tpl[start : start+end+1]
	}
	return tpl[start : start+end+2]
}

// sliderCSSRule extracts a single CSS rule's source text (from its opening
// "selector {" marker through the matching closing "}").
func sliderCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("Slider.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("Slider.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}

func readSlider(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Slider.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Slider.vue) failed: %v", err)
	}
	return string(data)
}
