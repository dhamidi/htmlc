package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toolbar.vue's/DropdownMenu.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Menubar.vue into a real htmlc.Engine and checking rendered
// HTML) is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestMenubar_ContainsZeroJSBaseline(t *testing.T) {
	src := readMenubar(t)

	for _, marker := range []string{
		"<template>",
		`role="menubar"`,
		`class="radix-menubar-trigger"`,
		`role="menuitem"`,
		`aria-haspopup="menu"`,
		`class="radix-menubar-content"`,
		`popover="auto"`,
		`role="menu"`,
		`aria-orientation="vertical"`,
		`v-for="(menu, index) in menus"`,
		`v-for="item in menu.items"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Menubar.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestMenubar_PopovertargetWiring confirms each menu's trigger/content pair
// is linked declaratively via popovertarget/id — DropdownMenu.vue's own
// zero-JS mechanism, now applied per top-level menu.
func TestMenubar_PopovertargetWiring(t *testing.T) {
	src := readMenubar(t)

	for _, marker := range []string{
		`:popovertarget="menu.id"`,
		`popovertargetaction="toggle"`,
		`:id="menu.id"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Menubar.vue missing expected popovertarget wiring marker %q; got source:\n%s", marker, src)
		}
	}
}

func TestMenubar_PopoverAttributeIsAuto(t *testing.T) {
	src := readMenubar(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("Menubar.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("Menubar.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestMenubar_NoManualAriaExpanded mirrors DropdownMenu.vue's own test: the
// browser sets up an implicit aria-expanded/aria-details relationship for
// any popovertarget invoker automatically, so this component's <template>
// must not hand-set aria-expanded/aria-controls/aria-details on the
// trigger.
func TestMenubar_NoManualAriaExpanded(t *testing.T) {
	src := readMenubar(t)
	tpl := menubarTemplateBlock(t, src)

	for _, marker := range []string{
		"aria-expanded",
		"aria-controls",
		"aria-details",
	} {
		if !strings.Contains(tpl, marker) {
			continue
		}
		t.Errorf("Menubar.vue <template> must not hand-set %q; the browser manages this implicitly for popovertarget invokers; template was:\n%s", marker, tpl)
	}
}

func TestMenubar_ContainsScopedStyle(t *testing.T) {
	src := readMenubar(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Menubar.vue missing expected <style scoped> block")
	}
}

func TestMenubar_ContainsCustomElementEnhancement(t *testing.T) {
	src := readMenubar(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "Menubar.vue" (verified by
		// actually running the algorithm, not guessed — see the file's
		// header comment: "Menubar" has no lowercase-followed-by-uppercase
		// boundary at all, so it derives to the single unhyphenated
		// segment "menubar"), prefixed with "radix-" per radix.go's own
		// documented Mount{Prefix: "radix"} convention every sibling
		// component here already follows.
		"customElements.define('radix-menubar'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Menubar.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestMenubar_TopLevelRovingTabindexBaseline confirms the static baseline
// gives the *first* trigger tabindex="0" and every other trigger
// tabindex="-1" — Toolbar.vue's roving-tabindex convention, applied to a
// homogeneous top-level list (this v1's `menus` entries carry no per-menu
// `disabled` field, unlike Toolbar.vue's/DropdownMenu.vue's own
// heterogeneous item lists — see the file's header comment for why a plain
// `index === 0` check is correct here, not a bounded ternary chain).
func TestMenubar_TopLevelRovingTabindexBaseline(t *testing.T) {
	src := readMenubar(t)
	tpl := menubarTemplateBlock(t, src)

	triggerTag := menubarTriggerTag(t, tpl)
	if !strings.Contains(triggerTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Errorf("Menubar.vue trigger <button> missing expected top-level roving-tabindex baseline; button tag was:\n%s", triggerTag)
	}
}

// TestMenubar_InnerItemRendersRealMenuitemButton confirms an "item"-type
// entry inside a menu's own `items` renders as a real
// <button role="menuitem">, carrying the item's id (as data-id) and
// disabled state, and a static tabindex="-1" (see the file's header
// comment for why no baseline "first focusable item" computation is
// needed here, unlike DropdownMenu.vue's own item baseline).
func TestMenubar_InnerItemRendersRealMenuitemButton(t *testing.T) {
	src := readMenubar(t)
	tpl := menubarTemplateBlock(t, src)

	for _, marker := range []string{
		`v-if="item.type === 'item'"`,
		"<button",
		`type="button"`,
		`role="menuitem"`,
		`:data-id="item.id"`,
		`:disabled="item.disabled"`,
		`tabindex="-1"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("Menubar.vue <template> missing expected inner-item marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestMenubar_SeparatorItemMatchesSeparatorContract confirms a
// "separator"-type inner item renders role="separator" with no
// aria-orientation, matching DropdownMenu.vue's own contract (a menu is
// always vertical, so its separator is unconditionally the horizontal-
// divider case).
func TestMenubar_SeparatorItemMatchesSeparatorContract(t *testing.T) {
	src := readMenubar(t)
	tpl := menubarTemplateBlock(t, src)

	for _, marker := range []string{
		`v-else-if="item.type === 'separator'"`,
		`role="separator"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("Menubar.vue <template> missing expected separator-item marker %q; template was:\n%s", marker, tpl)
		}
	}

	separatorTag := menubarSeparatorTag(t, tpl)
	if strings.Contains(separatorTag, "aria-orientation") {
		t.Errorf("Menubar.vue separator must not carry aria-orientation; separator tag was:\n%s", separatorTag)
	}
}

// TestMenubar_ArrowDownOpensClosedTrigger confirms the trigger-level
// keydown handler opens a closed submenu on ArrowDown — the one top-level
// key with no native <button> activation to piggyback on (Enter/Space
// already fire the trigger's own popovertargetaction="toggle" natively).
func TestMenubar_ArrowDownOpensClosedTrigger(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	for _, marker := range []string{
		"#onTriggerKeydown",
		"'ArrowDown'",
		"#openMenu(index)",
		"showPopover()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Menubar.vue <script customelement> missing expected ArrowDown-opens marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestMenubar_AdjacentSwitchClosesCurrentAndOpensNext is the most important
// test in this file: the process-mandated hand-trace check for the one
// genuinely new piece of logic this component needed. For a 3-menu bar
// with the middle menu open, pressing ArrowRight must close the currently
// open submenu (hidePopover on the current menu) and open the *adjacent*
// trigger's own submenu (showPopover, via #openMenu) — not just move which
// trigger has tabindex="0" the way a plain Toolbar.vue would. This checks
// the #navigate method's actual open-submenu branch does both steps, in
// the close-then-open order, gated on #openIndex !== -1.
func TestMenubar_AdjacentSwitchClosesCurrentAndOpensNext(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	navigate := menubarMethodBody(t, script, "#navigate(fromIndex, direction) {")

	for _, marker := range []string{
		"(fromIndex + direction + n) % n",
		"if (this.#openIndex !== -1) {",
		"this.#menus[this.#openIndex].content.hidePopover()",
		"this.#openMenu(nextIndex)",
	} {
		if !strings.Contains(navigate, marker) {
			t.Errorf("Menubar.vue #navigate missing expected adjacent-switch marker %q; #navigate body was:\n%s", marker, navigate)
		}
	}

	// The close call must textually precede the open call: this is a
	// close-then-open sequence, not open-then-close (which would rely on
	// native auto-popover-stacking to clean up the old one, an implicit
	// and less deterministic ordering for a script-driven keyboard switch —
	// see the file's header comment).
	closeIdx := strings.Index(navigate, "hidePopover()")
	openIdx := strings.Index(navigate, "#openMenu(nextIndex)")
	if closeIdx == -1 || openIdx == -1 || closeIdx > openIdx {
		t.Errorf("Menubar.vue #navigate must call hidePopover() on the current menu BEFORE #openMenu(nextIndex); #navigate body was:\n%s", navigate)
	}

	// When no menu is open, #navigate must fall back to a plain
	// roving-tabindex focus move (Toolbar.vue's own Left/Right behavior)
	// with no popover calls at all.
	elseIdx := strings.Index(navigate, "} else {")
	if elseIdx == -1 {
		t.Fatalf("Menubar.vue #navigate missing expected else branch for the no-menu-open case; #navigate body was:\n%s", navigate)
	}
	elseBranch := navigate[elseIdx:]
	for _, marker := range []string{
		"this.#syncTriggerTabindex(nextIndex)",
		"this.#triggers[nextIndex].focus()",
	} {
		if !strings.Contains(elseBranch, marker) {
			t.Errorf("Menubar.vue #navigate's no-menu-open branch missing expected plain-focus-move marker %q; branch was:\n%s", marker, elseBranch)
		}
	}
	if strings.Contains(elseBranch, "Popover()") {
		t.Errorf("Menubar.vue #navigate's no-menu-open branch must not call show/hidePopover() — pressing Left/Right with nothing open must only move focus, not open anything; branch was:\n%s", elseBranch)
	}
}

// TestMenubar_ContentKeydownAlsoTriggersAdjacentSwitch confirms the
// far-more-common real-world path also reaches the same #navigate logic:
// once a submenu is open, keyboard focus lives *inside* its content (per
// this file's own focus-on-open behavior), so Left/Right must be handled
// by a keydown listener on the content element itself, not only on the
// top-level trigger.
func TestMenubar_ContentKeydownAlsoTriggersAdjacentSwitch(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	onContentKeydown := menubarMethodBody(t, script, "#onContentKeydown = (event, index) => {")
	for _, marker := range []string{
		"'ArrowRight'",
		"'ArrowLeft'",
		"this.#navigate(index, event.key === 'ArrowRight' ? 1 : -1)",
	} {
		if !strings.Contains(onContentKeydown, marker) {
			t.Errorf("Menubar.vue #onContentKeydown missing expected adjacent-switch dispatch marker %q; #onContentKeydown body was:\n%s", marker, onContentKeydown)
		}
	}
}

// TestMenubar_FocusOnOpenTracksRovingStopAndFirstItem confirms
// #onContentToggle both moves the roving-tabindex stop onto the
// newly-opened trigger and focuses that submenu's first non-disabled item
// — the toggle-driven bookkeeping the adjacent-switch behavior depends on
// (a submenu opened via #openMenu's showPopover() call gets its first item
// focused here, not inside #navigate itself).
func TestMenubar_FocusOnOpenTracksRovingStopAndFirstItem(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	onToggle := menubarMethodBody(t, script, "#onContentToggle = (event, index) => {")
	for _, marker := range []string{
		"event.newState === 'open'",
		"this.#openIndex = index",
		"this.#syncTriggerTabindex(index)",
		"#refreshFocusable",
		"#moveItemFocus(menu, menu.items.indexOf(menu.focusable[0]))",
	} {
		if !strings.Contains(onToggle, marker) {
			t.Errorf("Menubar.vue #onContentToggle missing expected focus-on-open marker %q; #onContentToggle body was:\n%s", marker, onToggle)
		}
	}

	if !strings.Contains(onToggle, "this.#openIndex = -1") {
		t.Errorf("Menubar.vue #onContentToggle missing expected close-transition bookkeeping (\"this.#openIndex = -1\"); #onContentToggle body was:\n%s", onToggle)
	}
}

// TestMenubar_ItemNavigationSkipsSeparatorsAndDisabled is the process-
// mandated hand-trace check for in-submenu navigation: for a menu's items
// [item, separator, item, item], pressing ArrowDown from index 0 must land
// on index 2, not index 1 (the separator) — confirming #findFocusableItem
// walks past non-focusable entries instead of a plain modulo index step.
func TestMenubar_ItemNavigationSkipsSeparatorsAndDisabled(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	for _, marker := range []string{
		"#findFocusableItem",
		"'ArrowDown'",
		"'ArrowUp'",
		"'Home'",
		"'End'",
		"(index + step + n) % n",
		"classList.contains('radix-menubar-menuitem')",
		"!el.disabled",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Menubar.vue <script customelement> missing expected in-submenu navigation marker %q; script was:\n%s", marker, script)
		}
	}

	if !strings.Contains(script, "i < n") {
		t.Error("Menubar.vue <script customelement> #findFocusableItem must bound its walk (e.g. `i < n`) to avoid looping forever when no item is focusable")
	}
}

// TestMenubar_ActivationDispatchesEventAndCloses is the process-mandated
// adversarial check: the dispatched event's detail must unambiguously
// carry both which menu and which specific item was activated, and
// activating an item must close that submenu via hidePopover().
func TestMenubar_ActivationDispatchesEventAndCloses(t *testing.T) {
	src := readMenubar(t)
	script := menubarScriptBlock(t, src)

	for _, marker := range []string{
		"#onContentClick = (event, index) => {",
		"closest('.radix-menubar-menuitem')",
		"new CustomEvent('radix-menubar-select'",
		"detail: { menuId: menu.id, id: item.dataset.id }",
		"bubbles: true",
		"menu.content.hidePopover()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Menubar.vue <script customelement> missing expected activation marker %q; script was:\n%s", marker, script)
		}
	}
}

func menubarTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Menubar.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Menubar.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func menubarScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Menubar.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Menubar.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// menubarMethodBody extracts a method/arrow-function body's source text,
// starting just after `header` (e.g. "#navigate(fromIndex, direction) {")
// and ending at the matching closing brace, tracked via simple brace-depth
// counting (sufficient here: this file's script has no string/template
// literal containing an unbalanced '{' or '}').
func menubarMethodBody(t *testing.T, script, header string) string {
	t.Helper()
	start := strings.Index(script, header)
	if start == -1 {
		t.Fatalf("Menubar.vue <script customelement> missing expected method header %q; script was:\n%s", header, script)
	}
	bodyStart := start + len(header)
	depth := 1
	for i := bodyStart; i < len(script); i++ {
		switch script[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return script[bodyStart:i]
			}
		}
	}
	t.Fatalf("Menubar.vue <script customelement> could not find end of method %q; script was:\n%s", header, script)
	return ""
}

// menubarTriggerTag extracts the trigger <button ...> opening tag's full
// source text, up to and including the closing '>' of the opening tag.
func menubarTriggerTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `class="radix-menubar-trigger"`)
	if marker == -1 {
		t.Fatalf("Menubar.vue <template> missing trigger button; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<button")
	if start == -1 {
		t.Fatalf("Menubar.vue <template> could not find start of trigger <button>; template was:\n%s", tpl)
	}
	return scanMenubarTagFrom(t, tpl, start)
}

// menubarSeparatorTag extracts the separator <div ...> opening tag's full
// source text.
func menubarSeparatorTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, `v-else-if="item.type === 'separator'"`)
	if start == -1 {
		t.Fatalf("Menubar.vue <template> missing separator branch; template was:\n%s", tpl)
	}
	tagStart := strings.LastIndex(tpl[:start], "<div")
	if tagStart == -1 {
		t.Fatalf("Menubar.vue <template> could not find start of separator <div>; template was:\n%s", tpl)
	}
	return scanMenubarTagFrom(t, tpl, tagStart)
}

func scanMenubarTagFrom(t *testing.T, s string, start int) string {
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

func readMenubar(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Menubar.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Menubar.vue) failed: %v", err)
	}
	return string(data)
}
