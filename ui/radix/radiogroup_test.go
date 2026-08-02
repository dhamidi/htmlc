package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like checkbox_test.go/switch_test.go/tabs_test.go, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting RadioGroup.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestRadioGroup_ContainsZeroJSBaseline(t *testing.T) {
	src := readRadioGroup(t)

	for _, marker := range []string{
		"<template>",
		`role="radiogroup"`,
		`type="radio"`,
		"radix-radiogroup-input",
		"radix-radiogroup-item-label",
		`v-for="item in items"`,
		`:checked="item.value === value"`,
		`:disabled="disabled"`,
		`:name="name"`,
		`:value="item.value"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("RadioGroup.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestRadioGroup_ContainsScopedStyle(t *testing.T) {
	src := readRadioGroup(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("RadioGroup.vue missing expected <style scoped> block")
	}
}

// TestRadioGroup_RoleOnContainerDiv confirms role="radiogroup" lives on a
// plain <div>, not a <fieldset> — matching Radix's own documented choice
// (see the file's header comment on why <fieldset>'s harder-to-override
// default browser chrome is avoided).
func TestRadioGroup_RoleOnContainerDiv(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)

	divIdx := strings.Index(tpl, "<div")
	if divIdx == -1 {
		t.Fatalf("RadioGroup.vue <template> missing <div>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[divIdx:], ">")
	if end == -1 {
		t.Fatalf("RadioGroup.vue could not find end of <div> opening tag; template was:\n%s", tpl)
	}
	divTag := tpl[divIdx : divIdx+end+1]
	if !strings.Contains(divTag, `role="radiogroup"`) {
		t.Errorf(`RadioGroup.vue <div> missing expected role="radiogroup"; div tag was:\n%s`, divTag)
	}
	if strings.Contains(tpl, "<fieldset") {
		t.Error("RadioGroup.vue should not use <fieldset>; Radix's own choice, and this port's, is role=\"radiogroup\" on a <div> — see the file's header comment")
	}
}

// TestRadioGroup_NameIsRequiredPerInstanceProp confirms `name` is a real,
// per-instance prop bound via `:name="name"` on each item's hidden input —
// never a hardcoded literal the way Accordion.vue's/Tabs.vue's own shared
// group name is. This is the load-bearing difference this component's
// header comment calls out: multiple independent <RadioGroup> instances on
// one page must not share one native radio group.
func TestRadioGroup_NameIsRequiredPerInstanceProp(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)
	inputTag := radioGroupInputTag(t, tpl)

	if !strings.Contains(inputTag, `:name="name"`) {
		t.Errorf(`RadioGroup.vue <input> missing expected :name="name" binding; input tag was:\n%s`, inputTag)
	}
	// Guard against a regression to a hardcoded literal name, the
	// Accordion.vue/Tabs.vue pattern this component deliberately does not
	// follow (see the file's header comment).
	for _, literal := range []string{`name="radix-radiogroup"`, `name="radix-accordion"`, `name="radix-tabs"`} {
		if strings.Contains(inputTag, literal) {
			t.Errorf("RadioGroup.vue <input> must not use a hardcoded literal name %q; name must be the real, required, per-instance `name` prop", literal)
		}
	}
}

// TestRadioGroup_CheckedLogicMatchesItemValue is the adversarial-review
// check called out by this commit's process: `:checked` must be bound
// through a real `item.value === value` comparison, so that an unset
// `value` prop (which renders as this codebase's truthy
// "[missing: value]" placeholder — see component.go's validateProps and
// this file's header comment) can never accidentally match a real item and
// leave something checked that shouldn't be.
func TestRadioGroup_CheckedLogicMatchesItemValue(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)
	inputTag := radioGroupInputTag(t, tpl)

	if !strings.Contains(inputTag, `:checked="item.value === value"`) {
		t.Errorf(`RadioGroup.vue <input> missing expected marker :checked="item.value === value"; input tag was:\n%s`, inputTag)
	}
}

// TestRadioGroup_ItemIdsAreUnique confirms each item's hidden input/label
// pair builds its element id from both this instance's `name` prop and the
// item's own `id` field — the real bug risk process step 4 calls out
// explicitly: in a v-for loop, if `item.id` were not actually used to build
// a unique DOM id (or `name` were omitted from it), multiple radios could
// end up with colliding ids, breaking label association both within one
// instance and across multiple <RadioGroup> instances on the same page.
func TestRadioGroup_ItemIdsAreUnique(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)
	inputTag := radioGroupInputTag(t, tpl)

	const wantIDExpr = `'radix-radiogroup-' + name + '-' + item.id`
	if !strings.Contains(inputTag, `:id="`+wantIDExpr+`"`) {
		t.Errorf("RadioGroup.vue <input> missing expected unique id expression %q; input tag was:\n%s", wantIDExpr, inputTag)
	}

	labelIdx := strings.Index(tpl, "<label")
	if labelIdx == -1 {
		t.Fatalf("RadioGroup.vue <template> missing <label>; template was:\n%s", tpl)
	}
	labelEnd := strings.Index(tpl[labelIdx:], ">")
	if labelEnd == -1 {
		t.Fatalf("RadioGroup.vue could not find end of <label> opening tag; template was:\n%s", tpl)
	}
	labelTag := tpl[labelIdx : labelIdx+labelEnd+1]
	if !strings.Contains(labelTag, `:for="`+wantIDExpr+`"`) {
		t.Errorf("RadioGroup.vue <label> missing expected matching :for expression %q; label tag was:\n%s", wantIDExpr, labelTag)
	}
}

// TestRadioGroup_SiblingSelectorDOMOrdering verifies the DOM ordering the
// :checked adjacent-sibling CSS selector depends on, matching the same
// care Checkbox.vue's/Switch.vue's own tests take: within each item, the
// native <input> must appear before, and immediately adjacent to (no
// intervening element), the <label class="radix-radiogroup-item-label">
// it is paired with, and the stylesheet must actually contain the
// adjacent-sibling selectors that assume that ordering.
func TestRadioGroup_SiblingSelectorDOMOrdering(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)

	inputIdx := strings.Index(tpl, "<input")
	labelIdx := strings.Index(tpl, "<label")
	if inputIdx == -1 || labelIdx == -1 {
		t.Fatalf("RadioGroup.vue <template> missing <input> and/or <label>; template was:\n%s", tpl)
	}
	if inputIdx >= labelIdx {
		t.Fatalf("RadioGroup.vue <template> must place <input> before <label> for the :checked + selector to reach it; template was:\n%s", tpl)
	}

	inputTag := radioGroupInputTag(t, tpl)
	inputEnd := inputIdx + len(inputTag)
	between := tpl[inputEnd:labelIdx]
	if strings.TrimSpace(between) != "" {
		t.Errorf("RadioGroup.vue <template> has non-whitespace content between <input> and <label>, which would break the adjacent-sibling selector; found %q", between)
	}

	for _, marker := range []string{
		".radix-radiogroup-input:checked + .radix-radiogroup-item-label::before",
		".radix-radiogroup-input:focus-visible + .radix-radiogroup-item-label::before",
		".radix-radiogroup-input:disabled + .radix-radiogroup-item-label",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("RadioGroup.vue missing expected sibling-selector CSS rule %q", marker)
		}
	}
}

// TestRadioGroup_InputStaysInTabOrder mirrors Checkbox.vue's/Switch.vue's
// own checks: confirm the visually-hidden clip technique used on each
// native <input> does not also pull it out of the tab order or the
// accessibility tree, and that the native radio-group keyboard roving this
// component's zero-JS design depends on stays intact.
func TestRadioGroup_InputStaysInTabOrder(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)
	inputTag := radioGroupInputTag(t, tpl)

	for _, forbidden := range []string{`tabindex="-1"`, `tabIndex = -1`, `aria-hidden`} {
		if strings.Contains(inputTag, forbidden) {
			t.Errorf("RadioGroup.vue <input> must stay in the tab order and accessibility tree, but its tag contains %q; input tag was:\n%s", forbidden, inputTag)
		}
	}

	inputRule := radioGroupCSSRule(t, src, ".radix-visually-hidden-input {")
	if strings.Contains(inputRule, "display: none") || strings.Contains(inputRule, "display:none") {
		t.Errorf(".radix-visually-hidden-input rule must not use display:none (would remove it from the tab order); rule was:\n%s", inputRule)
	}
	if !strings.Contains(inputRule, "clip-path") {
		t.Errorf(".radix-visually-hidden-input rule missing expected clip-path visually-hidden technique; rule was:\n%s", inputRule)
	}
}

// TestRadioGroup_LabelMarkedVNative confirms each item's <label> is
// explicitly declared native via v-native. Without it, this package's own
// Label.vue (mounted alongside RadioGroup.vue in the same "radix" component
// set) registers a lowercase "label" alias that a bare <label> tag here
// would resolve against instead of rendering as plain HTML — the exact
// collision Checkbox.vue's own commit discovered and fixed, applied here
// proactively per this commit's process instructions.
func TestRadioGroup_LabelMarkedVNative(t *testing.T) {
	src := readRadioGroup(t)
	tpl := radioGroupTemplateBlock(t, src)

	labelIdx := strings.Index(tpl, "<label")
	if labelIdx == -1 {
		t.Fatalf("RadioGroup.vue <template> missing <label>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[labelIdx:], ">")
	if end == -1 {
		t.Fatalf("RadioGroup.vue could not find end of <label> opening tag; template was:\n%s", tpl)
	}
	labelTag := tpl[labelIdx : labelIdx+end+1]
	if !strings.Contains(labelTag, "v-native") {
		t.Errorf("RadioGroup.vue <label> missing expected v-native marker; label tag was:\n%s", labelTag)
	}
}

// TestRadioGroup_NoCustomElementScript confirms this component ships with
// no <script customelement> block, and documents why in the assertion
// itself: every piece of this component's real interactive behavior
// (role="radio" state, checked/unchecked, Space/click activation,
// arrow-key roving selection, single Tab stop into/out of the group) is
// intrinsic, free browser behavior of a same-`name` group of native radio
// inputs — the same conclusion Switch.vue's own commit reached for
// role="switch". There is nothing left for a script to do.
func TestRadioGroup_NoCustomElementScript(t *testing.T) {
	src := readRadioGroup(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("RadioGroup.vue should not contain a <script customelement> block: a same-name group of native radio inputs already gives correct role/checked-state/keyboard-roving behavior with zero JS, the same conclusion Switch.vue reached for role=\"switch\" — see the file's header comment")
	}
}

// radioGroupTemplateBlock extracts the <template>...</template> block's
// source text, excluding the header comment — same scoping technique
// checkbox_test.go's/switch_test.go's own helpers use.
func radioGroupTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("RadioGroup.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// radioGroupInputTag extracts the single <input ...> tag's own source text
// (its full opening/self-closing tag only) from a <template> block.
func radioGroupInputTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<input")
	if start == -1 {
		t.Fatalf("RadioGroup.vue <template> missing <input>; template was:\n%s", tpl)
	}
	end := strings.Index(tpl[start:], ">")
	if end == -1 {
		t.Fatalf("RadioGroup.vue could not find end of <input> tag; template was:\n%s", tpl)
	}
	return tpl[start : start+end+1]
}

// radioGroupCSSRule extracts a single CSS rule's source text (from its
// opening "selector {" marker through the matching closing "}").
func radioGroupCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("RadioGroup.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("RadioGroup.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}

func readRadioGroup(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/RadioGroup.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/RadioGroup.vue) failed: %v", err)
	}
	return string(data)
}
