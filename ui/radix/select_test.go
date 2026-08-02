package radix

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
)

// Like DropdownMenu.vue's/Checkbox.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting Select.vue into a real htmlc.Engine and checking rendered HTML)
// is out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestSelect_ContainsZeroJSBaseline(t *testing.T) {
	src := readSelect(t)

	for _, marker := range []string{
		"<template>",
		"<select",
		`v-native`,
		`class="radix-select-native radix-visually-hidden-input"`,
		`:name="name"`,
		`:id="id + '-native'"`,
		`v-for="item in items"`,
		`:value="item.value"`,
		`:selected="item.value === value"`,
		`:disabled="item.disabled"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Select.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestSelect_NativeSelectMarkedVNative confirms the native <select> tag is
// explicitly declared native via v-native. Without it, this component's own
// name ("Select") registers a lowercase "select" alias in the component
// registry that a bare <select> tag here would resolve back to (this
// component would try to render itself) instead of being treated as plain
// HTML — the exact self-reference-cycle trap Label.vue's/Dialog.vue's own
// header comments document for their own same-named native tags, hitting
// this file directly since "Select" and native <select> share the same
// lowercase spelling. See this file's header comment.
func TestSelect_NativeSelectMarkedVNative(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)

	selectTag := selectNativeSelectTag(t, tpl)
	if !strings.Contains(selectTag, "v-native") {
		t.Errorf("Select.vue <select> missing expected v-native marker; select tag was:\n%s", selectTag)
	}
}

// TestSelect_NativeSelectStaysInTabOrder is the adversarial-review check
// called out by this commit's process, mirroring
// TestCheckbox_InputStaysInTabOrder: confirm the visually-hidden clip
// technique used on the native <select> does not also pull it out of the
// tab order or the accessibility tree. Unlike upstream Radix's own real
// bridging <select> (aria-hidden, tabindex="-1" — read, not transcribed,
// from select.tsx's SelectBubbleInput), this port's native <select> is NOT
// a degraded fallback and must stay exactly as reachable/operable as any
// other real form control — see this file's header comment, "Two real,
// independently-operable controls."
func TestSelect_NativeSelectStaysInTabOrder(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)
	selectTag := selectNativeSelectTag(t, tpl)

	for _, forbidden := range []string{`tabindex="-1"`, `tabIndex = -1`, `aria-hidden`} {
		if strings.Contains(selectTag, forbidden) {
			t.Errorf("Select.vue <select> must stay in the tab order and accessibility tree, but its tag contains %q; select tag was:\n%s", forbidden, selectTag)
		}
	}

	// display:none on the native select's visually-hidden rule would remove
	// it from the tab order entirely; the clip-path/opacity/position
	// technique must be used instead (same requirement Checkbox.vue's own
	// hidden-input CSS satisfies).
	rule := tokensCSSRule(t, ".radix-visually-hidden-input {")
	if strings.Contains(rule, "display: none") || strings.Contains(rule, "display:none") {
		t.Errorf(".radix-visually-hidden-input rule must not use display:none (would remove it from the tab order); rule was:\n%s", rule)
	}
	if !strings.Contains(rule, "clip-path") {
		t.Errorf(".radix-visually-hidden-input rule missing expected clip-path visually-hidden technique; rule was:\n%s", rule)
	}
	if strings.Contains(rule, "disabled") {
		t.Errorf(".radix-visually-hidden-input rule must not disable the native select; rule was:\n%s", rule)
	}
}

// TestSelect_PopovertargetWiring confirms the custom trigger button and the
// custom listbox content element are linked declaratively via
// popovertarget/id, with popovertargetaction="toggle" — the same zero-JS
// mechanism Popover.vue's/DropdownMenu.vue's own tests check.
func TestSelect_PopovertargetWiring(t *testing.T) {
	src := readSelect(t)

	for _, marker := range []string{
		`:popovertarget="id"`,
		`popovertargetaction="toggle"`,
		`:id="id"`,
		`popover="auto"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Select.vue missing expected popovertarget wiring marker %q", marker)
		}
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("Select.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestSelect_NoManualAriaExpanded mirrors DropdownMenu.vue's own test: the
// browser sets up an implicit aria-expanded/aria-details relationship for
// any popovertarget invoker automatically, so this component's <template>
// must not hand-set aria-expanded/aria-controls/aria-details on the
// trigger.
func TestSelect_NoManualAriaExpanded(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)

	for _, marker := range []string{"aria-expanded", "aria-controls", "aria-details"} {
		if strings.Contains(tpl, marker) {
			t.Errorf("Select.vue <template> must not hand-set %q; the browser manages this implicitly for popovertarget invokers; template was:\n%s", marker, tpl)
		}
	}
}

// TestSelect_ListboxAndOptionRoles confirms the custom visual layer uses
// role="listbox"/role="option" (not DropdownMenu.vue's role="menu"/
// role="menuitem" vocabulary), with aria-selected wired per-item and
// aria-haspopup="listbox" on the trigger — verified against Radix's real
// select.tsx (read, not transcribed): SelectPosition renders
// role="listbox" with no aria-orientation, and SelectItemImpl renders
// role="option"/aria-selected.
func TestSelect_ListboxAndOptionRoles(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)

	for _, marker := range []string{
		`role="listbox"`,
		`role="option"`,
		`:aria-selected="item.value === value"`,
		`aria-haspopup="listbox"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("Select.vue <template> missing expected listbox/option marker %q; template was:\n%s", marker, tpl)
		}
	}

	// A listbox's ARIA-implicit default orientation is already vertical
	// (verified against select.tsx, not assumed — see header comment), so
	// no aria-orientation override should appear on the content element.
	contentTag := selectContentTag(t, tpl)
	if strings.Contains(contentTag, "aria-orientation") {
		t.Errorf("Select.vue listbox content must not carry aria-orientation (vertical is the ARIA-implicit default for a listbox); content tag was:\n%s", contentTag)
	}
}

// TestSelect_TriggerLabelComputedFromItemsAndValue confirms the trigger's
// displayed label is computed server-side via a nested <template v-for>/
// <template v-if> pair matching item.value === value, not left blank or
// requiring script to populate it on first paint.
func TestSelect_TriggerLabelComputedFromItemsAndValue(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)

	triggerTag := "radix-select-trigger"
	idx := strings.Index(tpl, triggerTag)
	if idx == -1 {
		t.Fatalf("Select.vue <template> missing trigger button; template was:\n%s", tpl)
	}
	// Scope to the trigger <button>...</button> element.
	closeIdx := strings.Index(tpl[idx:], "</button>")
	if closeIdx == -1 {
		t.Fatalf("Select.vue <template> missing trigger </button> close tag; template was:\n%s", tpl)
	}
	triggerBlock := tpl[idx : idx+closeIdx]

	for _, marker := range []string{
		`<template v-for="item in items">`,
		`<template v-if="item.value === value">`,
		`{{ item.label }}`,
	} {
		if !strings.Contains(triggerBlock, marker) {
			t.Errorf("Select.vue trigger <button> missing expected label-lookup marker %q; trigger block was:\n%s", marker, triggerBlock)
		}
	}
}

func TestSelect_ContainsScopedStyle(t *testing.T) {
	src := readSelect(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Select.vue missing expected <style scoped> block")
	}
}

func TestSelect_ContainsCustomElementEnhancement(t *testing.T) {
	src := readSelect(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "radix/Select.vue" (verified
		// by actually running the algorithm, not guessed — see the file's
		// header comment), matching every sibling component's own
		// Mount{Prefix: "radix"} convention.
		"customElements.define('radix-select'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Select.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestSelect_StaticTabindexBaselineSkipsLeadingDisabledItems mirrors
// DropdownMenu.vue's own adversarial tabindex-baseline check: the static
// (pre-JS) baseline must not simply check `index === 0` — it must skip past
// leading disabled items when picking which option gets tabindex="0".
func TestSelect_StaticTabindexBaselineSkipsLeadingDisabledItems(t *testing.T) {
	src := readSelect(t)
	tpl := selectTemplateBlock(t, src)

	itemTag := selectItemTag(t, tpl)

	for i := 0; i <= 5; i++ {
		idx := strconv.Itoa(i)
		lengthGuard := "items.length > " + idx
		disabledGuard := "!items[" + idx + "].disabled"
		for _, marker := range []string{lengthGuard, disabledGuard} {
			if !strings.Contains(itemTag, marker) {
				t.Errorf("Select.vue <div role=\"option\"> :tabindex missing expected guard %q for position %d; tag was:\n%s", marker, i, itemTag)
			}
		}
	}

	if !strings.Contains(itemTag, ": -1)") {
		t.Errorf(`Select.vue option :tabindex missing expected "-1" fallback at the end of the bounded lookahead; tag was:%s`, itemTag)
	}
	if strings.Contains(itemTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Error("Select.vue option :tabindex regressed to a bare `index === 0` check, which does not skip a leading disabled item")
	}
}

// TestSelect_SelectionSyncsNativeSelectValueAndDispatchesChange is the
// process-mandated adversarial check: selecting a custom option must
// update the *native* <select>'s own .value (not just the visual trigger
// text — this is the one thing that would make the form silently submit
// the wrong value if it were wrong), and must dispatch a real, bubbling
// native 'change' event on it so external listeners still fire correctly.
func TestSelect_SelectionSyncsNativeSelectValueAndDispatchesChange(t *testing.T) {
	src := readSelect(t)
	script := selectScriptBlock(t, src)

	for _, marker := range []string{
		"#activateItem(item)",
		"#nativeSelect.value = item.dataset.value",
		"new Event('change', { bubbles: true })",
		"#nativeSelect.dispatchEvent(",
		"#content.hidePopover()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Select.vue <script customelement> missing expected selection-sync marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestSelect_NativeChangeDrivesTriggerAndAriaSelectedSync confirms the
// script listens for the native select's own 'change' event and, from it,
// re-derives the trigger's displayed label and every option's
// aria-selected from the native select's live .value — the mechanism that
// gives both the custom-driven selection path (#activateItem's own
// dispatched event) and any external change to the native select real
// (if partial — see header comment) reverse sync.
func TestSelect_NativeChangeDrivesTriggerAndAriaSelectedSync(t *testing.T) {
	src := readSelect(t)
	script := selectScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('change', this.#onNativeChange)",
		"#syncFromNative",
		"#nativeSelect.value",
		"#trigger.textContent",
		"setAttribute('aria-selected'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Select.vue <script customelement> missing expected native-change-sync marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestSelect_KeyboardNavigationMarkers confirms Up/Down roving (reusing
// DropdownMenu.vue's/Toolbar.vue's own wrap-around/skip-disabled formula)
// and Enter/Space activation are wired.
func TestSelect_KeyboardNavigationMarkers(t *testing.T) {
	src := readSelect(t)
	script := selectScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('keydown'",
		"#findFocusable",
		"'ArrowDown'",
		"'ArrowUp'",
		"'Home'",
		"'End'",
		"(index + step + n) % n",
		"'Enter'",
		"' '",
		"#activateItem(item)",
		".focus()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Select.vue <script customelement> missing expected keyboard-nav marker %q; script was:\n%s", marker, script)
		}
	}

	if !strings.Contains(script, "i < n") {
		t.Error("Select.vue <script customelement> #findFocusable must bound its walk (e.g. `i < n`) to avoid looping forever when no option is focusable")
	}
}

// TestSelect_NeverCallsShowPopover mirrors DropdownMenu.vue's/Popover.vue's
// own forbidden-marker check: the native popovertarget invoker mechanism
// owns opening the listbox; this script must not call showPopover() itself,
// though it does call hidePopover() itself on activation.
func TestSelect_NeverCallsShowPopover(t *testing.T) {
	src := readSelect(t)
	script := selectScriptBlock(t, src)

	if strings.Contains(script, ".showPopover()") {
		t.Errorf("Select.vue <script customelement> must not call .showPopover(): the native popovertarget mechanism owns opening, not this script; script was:\n%s", script)
	}
}

// selectTemplateBlock extracts the outer SFC <template>...</template>
// section's source text. Unlike DropdownMenu.vue's own analogous helper
// (which can afford to stop at the *first* "</template>" it finds, since
// none of its own assertions ever need content past its single nested
// <template v-for>'s own close), this component's trigger button computes
// its label via a *nested* <template v-for>/<template v-if> pair (see
// header comment), so the first "</template>" in the source is that inner
// fragment's own close, not the outer section's true close. The correct,
// outer close is instead the *last* "</template>" occurring before the
// next section marker ("<style").
func selectTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Select.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	if start == -1 {
		t.Fatalf("Select.vue missing <template> after header comment; source was:\n%s", body)
	}
	styleStart := strings.Index(body, "<style")
	if styleStart == -1 {
		t.Fatalf("Select.vue missing <style> section after <template>; source was:\n%s", body)
	}
	relEnd := strings.LastIndex(body[start:styleStart], "</template>")
	if relEnd == -1 {
		t.Fatalf("Select.vue missing </template> before <style>; source was:\n%s", body)
	}
	end := start + relEnd
	return body[start : end+len("</template>")]
}

func selectScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Select.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Select.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// selectNativeSelectTag extracts the native <select ...> opening tag's full
// source text.
func selectNativeSelectTag(t *testing.T, tpl string) string {
	t.Helper()
	start := strings.Index(tpl, "<select")
	if start == -1 {
		t.Fatalf("Select.vue <template> missing <select>; template was:\n%s", tpl)
	}
	return scanSelectTagFrom(t, tpl, start)
}

// selectContentTag extracts the listbox content <div ...> opening tag's
// full source text — the one carrying role="listbox", identified by
// searching from the popover="auto" marker (distinguishing it from the
// outer wrapping <span> and the per-item <div role="option">s).
func selectContentTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `popover="auto"`)
	if marker == -1 {
		t.Fatalf("Select.vue <template> missing popover=\"auto\" content element; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<div")
	if start == -1 {
		t.Fatalf("Select.vue <template> could not find start of listbox content <div>; template was:\n%s", tpl)
	}
	return scanSelectTagFrom(t, tpl, start)
}

// selectItemTag extracts the per-item option <div role="option" ...>
// opening tag's full source text. Scans char-by-char tracking whether it is
// inside a quoted attribute value, since the :tabindex expression contains
// literal '>' characters (the "items.length > N" guards) that a naive
// strings.Index(..., ">") would stop at prematurely — the same technique
// DropdownMenu.vue's own test helper (dropdownMenuItemTag) uses.
func selectItemTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `role="option"`)
	if marker == -1 {
		t.Fatalf("Select.vue <template> missing role=\"option\" item; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<div")
	if start == -1 {
		t.Fatalf("Select.vue <template> could not find start of option <div>; template was:\n%s", tpl)
	}
	return scanSelectTagFrom(t, tpl, start)
}

func scanSelectTagFrom(t *testing.T, s string, start int) string {
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

// selectCSSRule extracts a single CSS rule's source text (from its opening
// "selector {" marker through the matching closing "}") so assertions can
// be scoped to just that rule.
func selectCSSRule(t *testing.T, src, selectorOpen string) string {
	t.Helper()
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("Select.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("Select.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}

func readSelect(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Select.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Select.vue) failed: %v", err)
	}
	return string(data)
}
