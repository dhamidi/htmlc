package radix

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// Like DropdownMenu.vue's own test, this module has no dependency on the
// root htmlc package, so a full render-based test (mounting ContextMenu.vue
// into a real htmlc.Engine and checking rendered HTML) is out of scope
// here — that proof is deliberately deferred to the examples/radix-demo
// commit, which does depend on root htmlc. These are content-sanity
// checks: they confirm the component's source file contains the markers
// this commit's design depends on.
func TestContextMenu_ContainsBaselineMarkup(t *testing.T) {
	src := readContextMenu(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-context-menu-trigger"`,
		"<slot></slot>",
		`class="radix-context-menu-content"`,
		`popover="auto"`,
		`role="menu"`,
		`aria-orientation="vertical"`,
		`v-for="(item, index) in items"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("ContextMenu.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestContextMenu_TriggerIsNotAPopovertargetInvoker confirms this component
// genuinely has no zero-JS opening mechanism: unlike DropdownMenu.vue's
// trigger <button popovertarget=...>, ContextMenu's trigger must not carry
// popovertarget wiring at all — there is no declarative equivalent for
// "open this popover on right-click" (see the file's header comment).
func TestContextMenu_TriggerIsNotAPopovertargetInvoker(t *testing.T) {
	src := readContextMenu(t)
	tpl := contextMenuTemplateBlock(t, src)

	for _, forbidden := range []string{"popovertarget", "popovertargetaction"} {
		if strings.Contains(tpl, forbidden) {
			t.Errorf("ContextMenu.vue <template> must not use %q; there is no declarative popovertarget equivalent for right-click triggering, so this component must not pretend to have one; template was:\n%s", forbidden, tpl)
		}
	}
}

// TestContextMenu_TriggerIsPlainNonInteractiveElement confirms the trigger
// region is a plain <div>, not a <button> — see the file's header comment's
// verified fact 2 (real Radix's own ContextMenuTrigger renders a plain
// Primitive.span with no aria-haspopup, unlike DropdownMenuTrigger's
// <button aria-haspopup="menu">).
func TestContextMenu_TriggerIsPlainNonInteractiveElement(t *testing.T) {
	src := readContextMenu(t)
	tpl := contextMenuTemplateBlock(t, src)

	triggerStart := strings.Index(tpl, `class="radix-context-menu-trigger"`)
	if triggerStart == -1 {
		t.Fatalf("ContextMenu.vue <template> missing trigger element; template was:\n%s", tpl)
	}
	tagStart := strings.LastIndex(tpl[:triggerStart], "<div")
	if tagStart == -1 {
		t.Fatalf("ContextMenu.vue <template> trigger must be a <div>; template was:\n%s", tpl)
	}
	triggerTag := scanTagFromContextMenu(t, tpl, tagStart)

	for _, forbidden := range []string{"aria-haspopup", "tabindex", `role="`} {
		if strings.Contains(triggerTag, forbidden) {
			t.Errorf("ContextMenu.vue trigger <div> must not carry %q; the trigger is a plain, non-interactive wrapper, not a widget (see header comment verified fact 2); trigger tag was:\n%s", forbidden, triggerTag)
		}
	}
}

func TestContextMenu_ContainsScopedStyle(t *testing.T) {
	src := readContextMenu(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("ContextMenu.vue missing expected <style scoped> block")
	}
}

func TestContextMenu_ContainsCustomElementEnhancement(t *testing.T) {
	src := readContextMenu(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "ContextMenu.vue" (verified by
		// actually running the algorithm, not guessed — see the file's
		// header comment), prefixed with "radix-" per radix.go's own
		// documented Mount{Prefix: "radix"} convention every sibling
		// component here already follows.
		"customElements.define('radix-context-menu'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("ContextMenu.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestContextMenu_ItemRendersRealMenuitemButton mirrors DropdownMenu.vue's
// own equivalent test: an "item"-type entry renders as a real
// <button role="menuitem">, carrying the item's id (as data-id) and
// disabled state.
func TestContextMenu_ItemRendersRealMenuitemButton(t *testing.T) {
	src := readContextMenu(t)
	tpl := contextMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`v-if="item.type === 'item'"`,
		"<button",
		`type="button"`,
		`role="menuitem"`,
		`:data-id="item.id"`,
		`:disabled="item.disabled"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("ContextMenu.vue <template> missing expected item marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestContextMenu_SeparatorItemMatchesSeparatorContract mirrors
// DropdownMenu.vue's own equivalent test.
func TestContextMenu_SeparatorItemMatchesSeparatorContract(t *testing.T) {
	src := readContextMenu(t)
	tpl := contextMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`v-else-if="item.type === 'separator'"`,
		`role="separator"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("ContextMenu.vue <template> missing expected separator-item marker %q; template was:\n%s", marker, tpl)
		}
	}

	separatorTag := contextMenuSeparatorTag(t, tpl)
	if strings.Contains(separatorTag, "aria-orientation") {
		t.Errorf("ContextMenu.vue separator must not carry aria-orientation; separator tag was:\n%s", separatorTag)
	}
}

// TestContextMenu_StaticTabindexBaselineSkipsLeadingNonInteractiveItems
// mirrors DropdownMenu.vue's own equivalent adversarial-review test: the
// static (pre-JS) baseline must not simply check `index === 0` — it must
// skip past leading separator/disabled entries when picking which item gets
// tabindex="0".
func TestContextMenu_StaticTabindexBaselineSkipsLeadingNonInteractiveItems(t *testing.T) {
	src := readContextMenu(t)
	tpl := contextMenuTemplateBlock(t, src)

	itemTag := contextMenuItemTag(t, tpl)

	for i := 0; i <= 5; i++ {
		idx := strconv.Itoa(i)
		lengthGuard := "items.length > " + idx
		typeGuard := "items[" + idx + "].type === 'item'"
		disabledGuard := "!items[" + idx + "].disabled"
		for _, marker := range []string{lengthGuard, typeGuard, disabledGuard} {
			if !strings.Contains(itemTag, marker) {
				t.Errorf("ContextMenu.vue <button> :tabindex missing expected guard %q for position %d; button tag was:\n%s", marker, i, itemTag)
			}
		}
	}

	if !strings.Contains(itemTag, ": -1)") {
		t.Errorf(`ContextMenu.vue <button> :tabindex missing expected "-1" fallback at the end of the bounded lookahead; button tag was:%s`, itemTag)
	}
	if strings.Contains(itemTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Error("ContextMenu.vue <button> :tabindex regressed to a bare `index === 0` check, which does not skip a leading separator/disabled item")
	}
}

// TestContextMenu_ContextMenuListenerPreventsDefaultUnconditionally is the
// process-mandated adversarial check for this component's own key
// difference from DropdownMenu.vue: the trigger's contextmenu listener must
// call event.preventDefault() as its very first, unconditional statement —
// not gated behind any condition that could let the native browser menu
// leak through on some path.
func TestContextMenu_ContextMenuListenerPreventsDefaultUnconditionally(t *testing.T) {
	src := readContextMenu(t)
	script := contextMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('contextmenu', this.#onContextMenu)",
		"removeEventListener('contextmenu', this.#onContextMenu)",
		"#onContextMenu = (event) => {",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected contextmenu-listener marker %q; script was:\n%s", marker, script)
		}
	}

	handler := contextMenuHandlerBody(t, script, "#onContextMenu = (event) => {")
	var firstStatement string
	for _, line := range strings.Split(handler, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			firstStatement = trimmed
			break
		}
	}
	if firstStatement != "event.preventDefault()" {
		t.Errorf("ContextMenu.vue #onContextMenu must call event.preventDefault() as its first, unconditional statement (found first statement %q); handler body was:\n%s", firstStatement, handler)
	}

	// event.preventDefault() must not appear only inside an `if` block
	// anywhere in the handler — i.e. there must be no conditional guard
	// wrapping it. A crude but effective check: the line immediately
	// preceding "event.preventDefault()" anywhere in the handler must never
	// be an `if (`-opening line.
	lines := strings.Split(handler, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "event.preventDefault()") {
			continue
		}
		if i > 0 && strings.Contains(strings.TrimSpace(lines[i-1]), "if (") {
			t.Errorf("ContextMenu.vue #onContextMenu appears to call event.preventDefault() conditionally (guarded by an `if` on the preceding line); handler body was:\n%s", handler)
		}
	}
}

// TestContextMenu_ShowPopoverCalledWithCursorCoordinatePositioning confirms
// the script drives .showPopover() itself (unlike DropdownMenu.vue, whose
// script must NOT call showPopover() because the native popovertarget
// invoker owns that job) and positions the content using
// event.clientX/event.clientY captured from the contextmenu event, with
// viewport edge-clamping applied on both axes.
func TestContextMenu_ShowPopoverCalledWithCursorCoordinatePositioning(t *testing.T) {
	src := readContextMenu(t)
	script := contextMenuScriptBlock(t, src)

	for _, marker := range []string{
		"this.#content.showPopover()",
		"this.#content.hidePopover()",
		"event.clientX",
		"event.clientY",
		"getBoundingClientRect()",
		"window.innerWidth",
		"window.innerHeight",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected showPopover/positioning marker %q; script was:\n%s", marker, script)
		}
	}

	// Edge-clamping must be applied on both axes: an overflow check against
	// the right/bottom edges, and a clamp against the left/top edges (< 0).
	for _, marker := range []string{
		"> viewportWidth",
		"> viewportHeight",
		"left < 0",
		"top < 0",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected edge-clamping marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestContextMenu_PopoverAttributeIsAuto confirms the content element uses
// popover="auto" (not "manual") — the same choice DropdownMenu.vue's/
// Popover.vue's own tests check, re-verified in this file's header comment
// specifically for the script-driven-open (not popovertarget-driven) case.
func TestContextMenu_PopoverAttributeIsAuto(t *testing.T) {
	src := readContextMenu(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("ContextMenu.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("ContextMenu.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestContextMenu_FocusesFirstItemOnOpen mirrors
// TestDropdownMenu_FocusesFirstItemOnOpen: the script hooks the content's
// native `toggle` event and, on the transition to "open", moves focus onto
// the first non-disabled item.
func TestContextMenu_FocusesFirstItemOnOpen(t *testing.T) {
	src := readContextMenu(t)
	script := contextMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('toggle', this.#onToggle)",
		"removeEventListener('toggle', this.#onToggle)",
		"event.newState !== 'open'",
		"#moveFocus(this.#items.indexOf(this.#focusable[0]))",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected focus-on-open marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestContextMenu_ArrowKeyNavigationSkipsSeparatorsAndDisabled mirrors
// DropdownMenu.vue's own equivalent test.
func TestContextMenu_ArrowKeyNavigationSkipsSeparatorsAndDisabled(t *testing.T) {
	src := readContextMenu(t)
	script := contextMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('keydown'",
		"#findFocusable",
		"'ArrowDown'",
		"'ArrowUp'",
		"'Home'",
		"'End'",
		"(index + step + n) % n",
		"classList.contains('radix-context-menu-menuitem')",
		"!el.disabled",
		".focus()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected arrow-key skip-logic marker %q; script was:\n%s", marker, script)
		}
	}

	for _, forbidden := range []string{"'ArrowLeft'", "'ArrowRight'"} {
		if strings.Contains(script, forbidden) {
			t.Errorf("ContextMenu.vue <script customelement> must not wire %q; a menu is always vertical; script was:\n%s", forbidden, script)
		}
	}
}

// TestContextMenu_ActivationDispatchesEventAndCloses mirrors
// DropdownMenu.vue's own equivalent adversarial check.
func TestContextMenu_ActivationDispatchesEventAndCloses(t *testing.T) {
	src := readContextMenu(t)
	script := contextMenuScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click', this.#onClick)",
		"#onClick = (event) => {",
		"closest('.radix-context-menu-menuitem')",
		"new CustomEvent('radix-context-menu-select'",
		"detail: { id: item.dataset.id }",
		"bubbles: true",
		"this.#content.hidePopover()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("ContextMenu.vue <script customelement> missing expected activation marker %q; script was:\n%s", marker, script)
		}
	}
}

func contextMenuTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("ContextMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("ContextMenu.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func contextMenuScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("ContextMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("ContextMenu.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// contextMenuHandlerBody extracts the body of a `name(event) => { ... }`
// (or similarly-shaped) arrow-function method, from just after its opening
// "{" up to its matching closing "}", by brace-depth counting — robust to
// nested braces (object literals, blocks) inside the handler.
func contextMenuHandlerBody(t *testing.T, script, marker string) string {
	t.Helper()
	start := strings.Index(script, marker)
	if start == -1 {
		t.Fatalf("could not find handler marker %q; script was:\n%s", marker, script)
	}
	braceStart := start + len(marker) // marker already ends with "{"
	depth := 1
	for i := braceStart; i < len(script); i++ {
		switch script[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return script[braceStart:i]
			}
		}
	}
	t.Fatalf("could not find matching closing brace for handler %q; script was:\n%s", marker, script)
	return ""
}

// contextMenuItemTag extracts the "item"-type <button ...> opening tag's
// full source text, the same scanning technique DropdownMenu.vue's own test
// helper (dropdownMenuItemTag) uses.
func contextMenuItemTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `v-if="item.type === 'item'"`)
	if marker == -1 {
		t.Fatalf("ContextMenu.vue <template> missing item branch; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<button")
	if start == -1 {
		t.Fatalf("ContextMenu.vue <template> could not find start of item <button>; template was:\n%s", tpl)
	}
	return scanTagFromContextMenu(t, tpl, start)
}

// contextMenuSeparatorTag extracts the separator <div ...> opening tag's
// full source text, the same scanning technique as contextMenuItemTag.
func contextMenuSeparatorTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, `v-else-if="item.type === 'separator'"`)
	if start == -1 {
		t.Fatalf("ContextMenu.vue <template> missing separator branch; template was:\n%s", tpl)
	}
	tagStart := strings.LastIndex(tpl[:start], "<div")
	if tagStart == -1 {
		t.Fatalf("ContextMenu.vue <template> could not find start of separator <div>; template was:\n%s", tpl)
	}
	return scanTagFromContextMenu(t, tpl, tagStart)
}

// scanTagFromContextMenu extracts an opening tag's full source text, up to
// and including the closing '>' of the opening tag, tracking whether it is
// inside a quoted attribute value so a literal '>' inside an attribute
// expression (e.g. the :tabindex "items.length > N" guards) doesn't
// terminate the scan early. Named distinctly from dropdownmenu_test.go's
// own scanTagFrom to avoid a duplicate-symbol collision within this shared
// package.
func scanTagFromContextMenu(t *testing.T, s string, start int) string {
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

func readContextMenu(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/ContextMenu.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/ContextMenu.vue) failed: %v", err)
	}
	return string(data)
}
