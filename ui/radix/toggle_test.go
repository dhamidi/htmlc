package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Switch.vue's/Checkbox.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Toggle.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file
// contains the markers this commit's design depends on.
func TestToggle_ContainsZeroJSBaseline(t *testing.T) {
	src := readToggle(t)

	for _, marker := range []string{
		"<template>",
		"<button",
		`type="button"`,
		`:aria-pressed="pressed ? 'true' : 'false'"`,
		`:data-state="pressed ? 'on' : 'off'"`,
		`:disabled="disabled"`,
		"<slot></slot>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toggle.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestToggle_ContainsCustomElementEnhancement confirms the load-bearing
// <script customelement> block exists at all — unlike Switch.vue, which
// has none because a native checkbox already handles everything, Toggle
// genuinely needs this block to make the click action work (see the
// file's header comment).
func TestToggle_ContainsCustomElementEnhancement(t *testing.T) {
	src := readToggle(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-toggle'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toggle.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestToggle_ClickHandlerFlipsPressedState confirms the click handler
// actually flips both aria-pressed and data-state between their two
// documented values, reading the current state off the real DOM attribute
// (not off some separate script-side variable that could drift out of
// sync with what got rendered).
func TestToggle_ClickHandlerFlipsPressedState(t *testing.T) {
	src := readToggle(t)
	script := toggleScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click'",
		"getAttribute('aria-pressed')",
		"setAttribute('aria-pressed'",
		"setAttribute('data-state'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toggle.vue <script customelement> missing expected click-handling marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToggle_DispatchesCustomEvent confirms the click handler dispatches
// a CustomEvent (not a plain Event) named radix-toggle-change, carrying
// the new pressed state in event.detail — the shape a future ToggleGroup
// component (or any consuming page's own script) needs to react to the
// change, per this commit's process instructions.
func TestToggle_DispatchesCustomEvent(t *testing.T) {
	src := readToggle(t)
	script := toggleScriptBlock(t, src)

	for _, marker := range []string{
		"new CustomEvent('radix-toggle-change'",
		"detail:",
		"pressed:",
		"bubbles: true",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toggle.vue <script customelement> missing expected CustomEvent marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToggle_NoDisabledGuardInClickHandler confirms the click handler
// does not duplicate disabled-checking logic in script. The native
// disabled <button> already suppresses its own 'click' dispatch per the
// HTML spec, so a second, script-side disabled check would be redundant
// at best and a second (driftable) source of truth at worst — see the
// file's header comment. This test guards against that regressing back
// in.
func TestToggle_NoDisabledGuardInClickHandler(t *testing.T) {
	src := readToggle(t)
	script := toggleScriptBlock(t, src)

	for _, forbidden := range []string{".disabled", "hasAttribute('disabled')"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("Toggle.vue <script customelement> should not contain a script-side disabled check (%q) — the native disabled <button> already suppresses click dispatch per the HTML spec; script was:\n%s", forbidden, script)
		}
	}
}

func TestToggle_ContainsScopedStyle(t *testing.T) {
	src := readToggle(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Toggle.vue missing expected <style scoped> block")
	}
}

// toggleScriptBlock extracts the <script customelement>...</script>
// block's source text. It searches only after the header comment closes
// (the last "-->" before end of file) so it can't be fooled by this file's
// own header comment prose mentioning "<script customelement>" literally
// while documenting it.
func toggleScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Toggle.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Toggle.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readToggle(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Toggle.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Toggle.vue) failed: %v", err)
	}
	return string(data)
}
