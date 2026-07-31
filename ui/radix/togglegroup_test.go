package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toggle.vue's/RadioGroup.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting ToggleGroup.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestToggleGroup_ContainsZeroJSBaseline(t *testing.T) {
	src := readToggleGroup(t)

	for _, marker := range []string{
		"<template>",
		`role="group"`,
		`data-orientation="horizontal"`,
		`:data-type="type"`,
		`v-for="(item, index) in items"`,
		"<button",
		`type="button"`,
		`:data-value="item.id"`,
		`:aria-pressed="(type === 'single' ? item.id === value : item.pressed) ? 'true' : 'false'"`,
		`:data-state="(type === 'single' ? item.id === value : item.pressed) ? 'on' : 'off'"`,
		`:tabindex="type === 'single' ? (item.id === value ? '0' : (value === '' && index === 0 ? '0' : '-1')) : (index === 0 ? '0' : '-1')"`,
		`:disabled="disabled"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("ToggleGroup.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestToggleGroup_ContainsScopedStyle(t *testing.T) {
	src := readToggleGroup(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("ToggleGroup.vue missing expected <style scoped> block")
	}
}

func TestToggleGroup_ContainsCustomElementEnhancement(t *testing.T) {
	src := readToggleGroup(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-toggle-group'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("ToggleGroup.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestToggleGroup_ClickHandlerTogglesOwnState confirms the click handler
// flips the clicked button's own aria-pressed/data-state off the real DOM
// attribute (not a separate script-side variable that could drift), the
// same self-contained toggle-flip Toggle.vue's own script performs.
func TestToggleGroup_ClickHandlerTogglesOwnState(t *testing.T) {
	src := readToggleGroup(t)
	script := toggleGroupScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click'",
		"getAttribute('aria-pressed')",
		"setAttribute('aria-pressed'",
		"setAttribute('data-state'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ToggleGroup.vue <script customelement> missing expected click-handling marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToggleGroup_SingleModeUnpressesOthers confirms single mode's mutual
// exclusivity is actually implemented: pressing a new item must un-press
// every other button in the group, not just press the clicked one and
// leave a stale previously-pressed button also marked pressed.
func TestToggleGroup_SingleModeUnpressesOthers(t *testing.T) {
	src := readToggleGroup(t)
	script := toggleGroupScriptBlock(t, src)

	if !strings.Contains(script, "this.#type === 'single'") {
		t.Error("ToggleGroup.vue <script customelement> click handler must branch on single-mode exclusivity")
	}
	if !strings.Contains(script, "this.#buttons") || !strings.Contains(script, "other") {
		t.Errorf("ToggleGroup.vue <script customelement> must iterate the group's other buttons to un-press them in single mode; script was:\n%s", script)
	}
}

// TestToggleGroup_DispatchesGroupChangeEvent confirms the group-level
// CustomEvent fires on selection change and carries both the active `type`
// and the resulting `value` (a single id for single mode, an array of ids
// for multiple mode).
func TestToggleGroup_DispatchesGroupChangeEvent(t *testing.T) {
	src := readToggleGroup(t)
	script := toggleGroupScriptBlock(t, src)

	for _, marker := range []string{
		"new CustomEvent('radix-toggle-group-change'",
		"detail:",
		"type: this.#type",
		"value",
		"bubbles: true",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ToggleGroup.vue <script customelement> missing expected CustomEvent marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToggleGroup_ArrowKeyRovingTabindexWraps confirms the arrow-key
// navigation logic exists and wraps at both ends, matching Tabs.vue's own
// established wrap-around convention ((index + 1) % length /
// (index - 1 + length) % length) rather than a non-wrapping clamp.
func TestToggleGroup_ArrowKeyRovingTabindexWraps(t *testing.T) {
	src := readToggleGroup(t)
	script := toggleGroupScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('keydown'",
		"'ArrowRight'",
		"'ArrowLeft'",
		"(index + 1) % this.#buttons.length",
		"(index - 1 + this.#buttons.length) % this.#buttons.length",
		"setAttribute('tabindex'",
		".focus()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ToggleGroup.vue <script customelement> missing expected roving-tabindex marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToggleGroup_NoDisabledGuardInClickHandler mirrors Toggle.vue's own
// identical regression guard: a disabled <button> never dispatches 'click'
// per the HTML spec, so the click delegation below must not duplicate that
// check in script.
func TestToggleGroup_NoDisabledGuardInClickHandler(t *testing.T) {
	src := readToggleGroup(t)
	script := toggleGroupScriptBlock(t, src)

	for _, forbidden := range []string{".disabled", "hasAttribute('disabled')"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("ToggleGroup.vue <script customelement> should not contain a script-side disabled check (%q); script was:\n%s", forbidden, script)
		}
	}
}

func toggleGroupScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("ToggleGroup.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("ToggleGroup.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readToggleGroup(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/ToggleGroup.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/ToggleGroup.vue) failed: %v", err)
	}
	return string(data)
}
