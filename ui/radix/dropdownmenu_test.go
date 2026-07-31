package radix

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// Like Toolbar.vue's/Popover.vue's own tests, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// DropdownMenu.vue into a real htmlc.Engine and checking rendered HTML) is
// out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestDropdownMenu_ContainsZeroJSBaseline(t *testing.T) {
	src := readDropdownMenu(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-dropdown-menu-trigger"`,
		"<slot></slot>",
		`class="radix-dropdown-menu-content"`,
		`popover="auto"`,
		`role="menu"`,
		`aria-orientation="vertical"`,
		`aria-haspopup="menu"`,
		`v-for="(item, index) in items"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("DropdownMenu.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestDropdownMenu_PopovertargetWiring confirms the trigger button and the
// content element are linked declaratively via popovertarget/id, with
// popovertargetaction="toggle" — the same zero-JS mechanism Popover.vue's
// own test (TestPopover_PopovertargetWiring) checks, reused here for
// DropdownMenu's trigger/content pair.
func TestDropdownMenu_PopovertargetWiring(t *testing.T) {
	src := readDropdownMenu(t)

	for _, marker := range []string{
		`:popovertarget="id"`,
		`popovertargetaction="toggle"`,
		`:id="id"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("DropdownMenu.vue missing expected popovertarget wiring marker %q; got source:\n%s", marker, src)
		}
	}
}

// TestDropdownMenu_PopoverAttributeIsAuto confirms the content element uses
// popover="auto" (not "manual"), the same light-dismiss/Escape/top-layer
// requirement Popover.vue's own test checks.
func TestDropdownMenu_PopoverAttributeIsAuto(t *testing.T) {
	src := readDropdownMenu(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("DropdownMenu.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("DropdownMenu.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestDropdownMenu_NoManualAriaExpanded mirrors
// TestPopover_NoManualAriaExpanded: the browser sets up an implicit
// aria-expanded/aria-details relationship for any popovertarget invoker
// automatically, so this component's <template> must not hand-set
// aria-expanded/aria-controls/aria-details on the trigger (which would go
// stale, since nothing in this component's own script updates it).
func TestDropdownMenu_NoManualAriaExpanded(t *testing.T) {
	src := readDropdownMenu(t)
	template := dropdownMenuTemplateBlock(t, src)

	for _, marker := range []string{
		"aria-expanded",
		"aria-controls",
		"aria-details",
	} {
		if strings.Contains(template, marker) {
			t.Errorf("DropdownMenu.vue <template> must not hand-set %q; the browser manages this implicitly for popovertarget invokers; template was:\n%s", marker, template)
		}
	}
}

func TestDropdownMenu_ContainsScopedStyle(t *testing.T) {
	src := readDropdownMenu(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("DropdownMenu.vue missing expected <style scoped> block")
	}
}

func TestDropdownMenu_ContainsCustomElementEnhancement(t *testing.T) {
	src := readDropdownMenu(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "DropdownMenu.vue" (verified
		// by actually running the algorithm, not guessed — see the file's
		// header comment), prefixed with "radix-" per radix.go's own
		// documented Mount{Prefix: "radix"} convention every sibling
		// component here already follows.
		"customElements.define('radix-dropdown-menu'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("DropdownMenu.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestDropdownMenu_ItemRendersRealMenuitemButton confirms an "item"-type
// entry renders as a real <button role="menuitem">, carrying the item's id
// (as data-id, so the activation event below can unambiguously identify
// it) and disabled state.
func TestDropdownMenu_ItemRendersRealMenuitemButton(t *testing.T) {
	src := readDropdownMenu(t)
	tpl := dropdownMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`v-if="item.type === 'item'"`,
		"<button",
		`type="button"`,
		`role="menuitem"`,
		`:data-id="item.id"`,
		`:disabled="item.disabled"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("DropdownMenu.vue <template> missing expected item marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestDropdownMenu_SeparatorItemMatchesSeparatorContract confirms a
// "separator"-type item renders role="separator" with no aria-orientation
// — matching Separator.vue's own documented convention that "horizontal" is
// aria-orientation's ARIA-implicit default, and (unlike Toolbar.vue's
// perpendicular-to-a-configurable-orientation separator) a menu is always
// vertical, so its separator is unconditionally the horizontal-divider
// case, with no orientation-driven ternary needed at all.
func TestDropdownMenu_SeparatorItemMatchesSeparatorContract(t *testing.T) {
	src := readDropdownMenu(t)
	tpl := dropdownMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`v-else-if="item.type === 'separator'"`,
		`role="separator"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("DropdownMenu.vue <template> missing expected separator-item marker %q; template was:\n%s", marker, tpl)
		}
	}

	separatorTag := dropdownMenuSeparatorTag(t, tpl)
	if strings.Contains(separatorTag, "aria-orientation") {
		t.Errorf("DropdownMenu.vue separator must not carry aria-orientation (horizontal is the ARIA-implicit default, and a menu is always vertical so its separator is always the horizontal case); separator tag was:\n%s", separatorTag)
	}
}

// TestDropdownMenu_StaticTabindexBaselineSkipsLeadingNonInteractiveItems is
// the same adversarial-review check Toolbar.vue's own test performs: the
// static (pre-JS) baseline must not simply check `index === 0` — it must
// skip past leading separator/disabled entries when picking which item gets
// tabindex="0". Confirms the bounded, unrolled lookahead expression
// inspects items[0] through items[5], each guarded by both
// `type === 'item'` and `!disabled`, before falling back to -1.
func TestDropdownMenu_StaticTabindexBaselineSkipsLeadingNonInteractiveItems(t *testing.T) {
	src := readDropdownMenu(t)
	tpl := dropdownMenuTemplateBlock(t, src)

	itemTag := dropdownMenuItemTag(t, tpl)

	for i := 0; i <= 5; i++ {
		idx := strconv.Itoa(i)
		lengthGuard := "items.length > " + idx
		typeGuard := "items[" + idx + "].type === 'item'"
		disabledGuard := "!items[" + idx + "].disabled"
		for _, marker := range []string{lengthGuard, typeGuard, disabledGuard} {
			if !strings.Contains(itemTag, marker) {
				t.Errorf("DropdownMenu.vue <button> :tabindex missing expected guard %q for position %d; button tag was:\n%s", marker, i, itemTag)
			}
		}
	}

	if !strings.Contains(itemTag, ": -1)") {
		t.Errorf(`DropdownMenu.vue <button> :tabindex missing expected "-1" fallback at the end of the bounded lookahead; button tag was:%s`, itemTag)
	}
	if !strings.Contains(itemTag, `:tabindex="index === (`) {
		t.Errorf(`DropdownMenu.vue <button> :tabindex must compare index against the computed first-interactive position; button tag was:%s`, itemTag)
	}
	if !strings.Contains(itemTag, `) ? '0' : '-1'"`) {
		t.Errorf(`DropdownMenu.vue <button> :tabindex must resolve to '0' for the matching index and '-1' otherwise; button tag was:%s`, itemTag)
	}

	// A leading separator must not make the whole chain regress to a bare
	// `index === 0` (which would incorrectly give the separator's own
	// position the tab stop, or give it to whatever sits at index 0
	// regardless of type/disabled).
	if strings.Contains(itemTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Error("DropdownMenu.vue <button> :tabindex regressed to a bare `index === 0` check, which does not skip a leading separator/disabled item")
	}
}

// TestDropdownMenu_ScriptRecomputesTabindexFromLiveDOM confirms the
// <script customelement> unconditionally recomputes the roving-tabindex
// baseline from the live DOM on connect, mirroring
// TestToolbar_ScriptRecomputesTabindexFromLiveDOM.
func TestDropdownMenu_ScriptRecomputesTabindexFromLiveDOM(t *testing.T) {
	src := readDropdownMenu(t)
	script := dropdownMenuScriptBlock(t, src)

	for _, marker := range []string{
		"#refreshFocusable",
		"#syncTabindex",
		"classList.contains('radix-dropdown-menu-menuitem')",
		"!el.disabled",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("DropdownMenu.vue <script customelement> missing expected baseline-recompute marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestDropdownMenu_ArrowKeyNavigationSkipsSeparatorsAndDisabled is the
// process-mandated hand-trace check: for items
// [item, separator, item, item], pressing ArrowDown from index 0 must land
// on index 2, not index 1 (the separator) — confirming #findFocusable
// actually walks past non-focusable entries instead of a plain modulo
// index step. Also confirms menus only wire the vertical arrow-key pair
// (ArrowDown/ArrowUp), not Toolbar.vue's horizontal ArrowLeft/ArrowRight —
// per this file's header comment, real Radix menus are always vertical.
func TestDropdownMenu_ArrowKeyNavigationSkipsSeparatorsAndDisabled(t *testing.T) {
	src := readDropdownMenu(t)
	script := dropdownMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('keydown'",
		"#findFocusable",
		"'ArrowDown'",
		"'ArrowUp'",
		"'Home'",
		"'End'",
		"(index + step + n) % n",
		"classList.contains('radix-dropdown-menu-menuitem')",
		"!el.disabled",
		".focus()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("DropdownMenu.vue <script customelement> missing expected arrow-key skip-logic marker %q; script was:\n%s", marker, script)
		}
	}

	for _, forbidden := range []string{"'ArrowLeft'", "'ArrowRight'"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("DropdownMenu.vue <script customelement> must not wire %q; a menu is always vertical (see header comment), unlike Toolbar.vue's configurable orientation; script was:\n%s", forbidden, script)
		}
	}

	// #findFocusable must be bounded (loop at most n times), guarding
	// against an infinite loop when a menu has no focusable item at all.
	if !strings.Contains(script, "i < n") {
		t.Error("DropdownMenu.vue <script customelement> #findFocusable must bound its walk (e.g. `i < n`) to avoid looping forever when no item is focusable")
	}
}

// TestDropdownMenu_FocusesFirstItemOnOpen confirms the script hooks the
// content's native `toggle` event (not `click` on the trigger — see
// Popover.vue's header comment for why `toggle` is the only correct hook
// point once popovertarget owns the open/close moment) and, on the
// transition to "open", moves focus onto the first non-disabled item — the
// documented APG Menu Button convention this file's header comment
// explains (and explicitly contrasts with upstream Radix's own
// keyboard-vs-pointer-modality nuance, deliberately not replicated here).
func TestDropdownMenu_FocusesFirstItemOnOpen(t *testing.T) {
	src := readDropdownMenu(t)
	script := dropdownMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('toggle', this.#onToggle)",
		"removeEventListener('toggle', this.#onToggle)",
		"event.newState !== 'open'",
		"#moveFocus(this.#items.indexOf(this.#focusable[0]))",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("DropdownMenu.vue <script customelement> missing expected focus-on-open marker %q; script was:\n%s", marker, script)
		}
	}

	// The native popovertarget invoker mechanism owns the entire
	// open/close lifecycle; this script must not call showPopover() itself
	// (mirrors Popover.vue's own forbidden-markers check), though it does
	// call hidePopover() itself for the activation-closes-the-menu
	// requirement (see TestDropdownMenu_ActivationDispatchesEventAndCloses)
	// — unlike Popover.vue, which never needs to close itself at all.
	if strings.Contains(script, ".showPopover()") {
		t.Errorf("DropdownMenu.vue <script customelement> must not call .showPopover(): the native popovertarget mechanism owns opening, not this script; script was:\n%s", script)
	}
}

// TestDropdownMenu_ActivationDispatchesEventAndCloses is the process-
// mandated adversarial check: the dispatched event's detail must
// unambiguously carry the *specific* activated item's id (read from that
// item's own data-id attribute), not just "something was clicked" — and
// activating an item must close the menu via hidePopover().
func TestDropdownMenu_ActivationDispatchesEventAndCloses(t *testing.T) {
	src := readDropdownMenu(t)
	script := dropdownMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click', this.#onClick)",
		"#onClick = (event) => {",
		"closest('.radix-dropdown-menu-menuitem')",
		"new CustomEvent('radix-dropdown-menu-select'",
		"detail: { id: item.dataset.id }",
		"bubbles: true",
		"this.#content.hidePopover()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("DropdownMenu.vue <script customelement> missing expected activation marker %q; script was:\n%s", marker, script)
		}
	}
}

func dropdownMenuTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("DropdownMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("DropdownMenu.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func dropdownMenuScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("DropdownMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("DropdownMenu.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// dropdownMenuItemTag extracts the "item"-type <button ...> opening tag's
// full source text, up to and including the closing '>' of the opening
// tag. Scans char-by-char tracking whether it is inside a quoted attribute
// value, since the :tabindex expression contains literal '>' characters
// (the "items.length > N" guards) that a naive strings.Index(..., ">")
// would stop at prematurely — the same technique Toolbar.vue's own test
// helper (toolbarButtonTag) uses.
func dropdownMenuItemTag(t *testing.T, tpl string) string {
	t.Helper()
	// The <template>'s first <button> is the always-present trigger; the
	// menuitem <button> (the one this helper needs) is the one guarded by
	// `v-if="item.type === 'item'"` inside the v-for loop, so search from
	// that marker rather than the first "<button" in the whole block.
	marker := strings.Index(tpl, `v-if="item.type === 'item'"`)
	if marker == -1 {
		t.Fatalf("DropdownMenu.vue <template> missing item branch; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<button")
	if start == -1 {
		t.Fatalf("DropdownMenu.vue <template> could not find start of item <button>; template was:\n%s", tpl)
	}
	return scanTagFrom(t, tpl, start)
}

// dropdownMenuSeparatorTag extracts the separator <div ...> opening tag's
// full source text, the same scanning technique as dropdownMenuItemTag.
func dropdownMenuSeparatorTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, `v-else-if="item.type === 'separator'"`)
	if start == -1 {
		t.Fatalf("DropdownMenu.vue <template> missing separator branch; template was:\n%s", tpl)
	}
	// Back up to the start of this <div ...> tag.
	tagStart := strings.LastIndex(tpl[:start], "<div")
	if tagStart == -1 {
		t.Fatalf("DropdownMenu.vue <template> could not find start of separator <div>; template was:\n%s", tpl)
	}
	return scanTagFrom(t, tpl, tagStart)
}

func scanTagFrom(t *testing.T, s string, start int) string {
	t.Helper()
	inQuote := byte(0)
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote != 0:
			if c == inQuote {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == '>':
			return s[start : i+1]
		}
	}
	t.Fatalf("could not find end of tag; source was:\n%s", s[start:])
	return ""
}

func readDropdownMenu(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/DropdownMenu.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/DropdownMenu.vue) failed: %v", err)
	}
	return string(data)
}
