package radix

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// Like ToggleGroup.vue's/Tabs.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Toolbar.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestToolbar_ContainsZeroJSBaseline(t *testing.T) {
	src := readToolbar(t)

	for _, marker := range []string{
		"<template>",
		`role="toolbar"`,
		`:aria-orientation="orientation"`,
		`:data-orientation="orientation"`,
		`v-for="(item, index) in items"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toolbar.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestToolbar_ContainsScopedStyle(t *testing.T) {
	src := readToolbar(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Toolbar.vue missing expected <style scoped> block")
	}
}

func TestToolbar_ContainsCustomElementEnhancement(t *testing.T) {
	src := readToolbar(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-toolbar'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toolbar.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestToolbar_ButtonItemRendersRealButton confirms a "button"-type item
// renders as a real <button type="button">, carrying the item's id (for a
// caller's own click-delegation) and disabled state.
func TestToolbar_ButtonItemRendersRealButton(t *testing.T) {
	src := readToolbar(t)
	tpl := toolbarTemplateBlock(t, src)

	for _, marker := range []string{
		`v-if="item.type === 'button'"`,
		"<button",
		`type="button"`,
		`:data-id="item.id"`,
		`:disabled="item.disabled"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("Toolbar.vue <template> missing expected button-item marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestToolbar_SeparatorItemMatchesSeparatorContract confirms a
// "separator"-type item is rendered per Separator.vue's own ARIA contract:
// role="separator", with aria-orientation present only for the "vertical"
// case (the ARIA-implicit default for role="separator" is "horizontal", so
// it is omitted then) — and that its orientation is perpendicular to the
// toolbar's own `orientation`, matching Radix's own documented
// ToolbarSeparator behavior.
func TestToolbar_SeparatorItemMatchesSeparatorContract(t *testing.T) {
	src := readToolbar(t)
	tpl := toolbarTemplateBlock(t, src)

	for _, marker := range []string{
		`v-else-if="item.type === 'separator'"`,
		`role="separator"`,
		`:data-orientation="orientation === 'vertical' ? 'horizontal' : 'vertical'"`,
		`:aria-orientation="orientation === 'vertical' ? undefined : 'vertical'"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("Toolbar.vue <template> missing expected separator-item marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestToolbar_StaticTabindexBaselineSkipsLeadingNonInteractiveItems is the
// adversarial-review check this commit's process calls out explicitly: the
// static (pre-JS) baseline must not simply check `index === 0` the way
// ToggleGroup.vue's homogeneous item list could — it must skip past
// leading separator/disabled entries when picking which button gets
// tabindex="0". This confirms the bounded, unrolled lookahead expression
// (see the file's header comment for why expr cannot express an
// unbounded search) actually inspects items[0] through items[5], each
// guarded by both `type === 'button'` and `!disabled`, before falling back
// to -1 (no static tab stop).
func TestToolbar_StaticTabindexBaselineSkipsLeadingNonInteractiveItems(t *testing.T) {
	src := readToolbar(t)
	tpl := toolbarTemplateBlock(t, src)

	buttonTag := toolbarButtonTag(t, tpl)

	// Every position from 0 through 5 must be guarded by both a
	// length check and a real interactivity check (type === 'button' and
	// !disabled), not just an index comparison — this is what makes the
	// baseline correctly skip a leading separator or a leading disabled
	// button, unlike a plain `index === 0`.
	for i := 0; i <= 5; i++ {
		idx := strconv.Itoa(i)
		lengthGuard := "items.length > " + idx
		typeGuard := "items[" + idx + "].type === 'button'"
		disabledGuard := "!items[" + idx + "].disabled"
		for _, marker := range []string{lengthGuard, typeGuard, disabledGuard} {
			if !strings.Contains(buttonTag, marker) {
				t.Errorf("Toolbar.vue <button> :tabindex missing expected guard %q for position %d; button tag was:\n%s", marker, i, buttonTag)
			}
		}
	}

	// The chain must fall back to -1 (no item picked) beyond its bound,
	// and the outer comparison must gate on `index ===` so exactly one
	// item (if any) resolves to tabindex="0".
	if !strings.Contains(buttonTag, ": -1)") {
		t.Errorf(`Toolbar.vue <button> :tabindex missing expected "-1" fallback at the end of the bounded lookahead; button tag was:%s`, buttonTag)
	}
	if !strings.Contains(buttonTag, `:tabindex="index === (`) {
		t.Errorf(`Toolbar.vue <button> :tabindex must compare index against the computed first-interactive position; button tag was:%s`, buttonTag)
	}
	if !strings.Contains(buttonTag, `) ? '0' : '-1'"`) {
		t.Errorf(`Toolbar.vue <button> :tabindex must resolve to '0' for the matching index and '-1' otherwise; button tag was:%s`, buttonTag)
	}

	// A leading separator must not make the whole chain regress to a bare
	// `index === 0` (which would incorrectly give the separator's own
	// position the tab stop, or give it to whatever sits at index 0
	// regardless of type/disabled).
	if strings.Contains(buttonTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Error("Toolbar.vue <button> :tabindex regressed to a bare `index === 0` check, which does not skip a leading separator/disabled item")
	}
}

// TestToolbar_ScriptRecomputesTabindexFromLiveDOM confirms the
// <script customelement> unconditionally recomputes the roving-tabindex
// baseline from the live DOM on connect — the real fix for `items` arrays
// longer than the static baseline's bound (see the file's header comment).
func TestToolbar_ScriptRecomputesTabindexFromLiveDOM(t *testing.T) {
	src := readToolbar(t)
	script := toolbarScriptBlock(t, src)

	for _, marker := range []string{
		"#refreshFocusable",
		"#syncTabindex",
		"classList.contains('radix-toolbar-button')",
		"!el.disabled",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toolbar.vue <script customelement> missing expected baseline-recompute marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToolbar_ArrowKeyNavigationSkipsSeparatorsAndDisabled is the
// process-mandated hand-trace check: for items
// [button, separator, button, button], pressing "next" (ArrowRight, the
// horizontal-orientation default) from index 0 must land on index 2, not
// index 1 (the separator) — confirming #findFocusable actually walks past
// non-focusable entries instead of a plain modulo index step.
func TestToolbar_ArrowKeyNavigationSkipsSeparatorsAndDisabled(t *testing.T) {
	src := readToolbar(t)
	script := toolbarScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('keydown'",
		"#findFocusable",
		"'ArrowRight'",
		"'ArrowLeft'",
		"'ArrowDown'",
		"'ArrowUp'",
		"'Home'",
		"'End'",
		"(index + step + n) % n",
		"classList.contains('radix-toolbar-button')",
		"!el.disabled",
		".focus()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toolbar.vue <script customelement> missing expected arrow-key skip-logic marker %q; script was:\n%s", marker, script)
		}
	}

	// #findFocusable must be bounded (loop at most n times), guarding
	// against an infinite loop when a toolbar has no focusable item at
	// all — a real risk once the walk can step past non-focusable entries
	// rather than always advancing exactly one logical position.
	if !strings.Contains(script, "i < n") {
		t.Error("Toolbar.vue <script customelement> #findFocusable must bound its walk (e.g. `i < n`) to avoid looping forever when no item is focusable")
	}
}

// TestToolbar_NoDisabledGuardNeededForClicks documents (and guards) the
// deliberate absence of a click handler: a plain action button has no
// pressed/unpressed state for this component to own, unlike
// ToggleGroup.vue's items — see the file's header comment. There is
// nothing here for a click listener to do that the native <button> does
// not already provide for free.
func TestToolbar_NoDisabledGuardNeededForClicks(t *testing.T) {
	src := readToolbar(t)
	script := toolbarScriptBlock(t, src)

	if strings.Contains(script, "addEventListener('click'") {
		t.Error("Toolbar.vue <script customelement> should not need a click handler: a plain action button has no state for this component to own beyond what the native <button> already provides for free — see the file's header comment")
	}
}

func toolbarTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Toolbar.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// toolbarButtonTag extracts the <button ...> tag's own full opening-tag
// source text from a <template> block, up to and including the closing
// '>' of the opening tag. Scans char-by-char tracking whether it is inside
// a quoted attribute value, since Toolbar.vue's own :tabindex expression
// contains a literal '>' (the "items.length > N" guards) that a naive
// strings.Index(..., ">") would stop at prematurely.
func toolbarButtonTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<button")
	if start == -1 {
		t.Fatalf("Toolbar.vue <template> missing <button>; template was:\n%s", tpl)
	}
	inQuote := byte(0)
	for i := start; i < len(tpl); i++ {
		c := tpl[i]
		switch {
		case inQuote != 0:
			if c == inQuote {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == '>':
			return tpl[start : i+1]
		}
	}
	t.Fatalf("Toolbar.vue could not find end of <button> opening tag; template was:\n%s", tpl)
	return ""
}

func toolbarScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Toolbar.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Toolbar.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readToolbar(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Toolbar.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Toolbar.vue) failed: %v", err)
	}
	return string(data)
}
