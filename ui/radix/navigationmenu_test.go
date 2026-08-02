package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toolbar.vue's/DropdownMenu.vue's/Menubar.vue's own tests, this
// module has no dependency on the root htmlc package, so a full
// render-based test (mounting NavigationMenu.vue into a real htmlc.Engine
// and checking rendered HTML) is out of scope here — that proof is
// deliberately deferred to the examples/radix-demo commit, which does
// depend on root htmlc. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.

// TestNavigationMenu_RootIsNavWithNoRedundantRole confirms the root
// element is a real <nav> and that no explicit role="navigation" is
// hand-set anywhere — <nav> already carries an implicit ARIA landmark
// role, so a hand-set role would be redundant (see the file's header
// comment, fact 1, verified against the real Radix source rather than
// assumed).
func TestNavigationMenu_RootIsNavWithNoRedundantRole(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	if !strings.Contains(tpl, "<nav") {
		t.Errorf("NavigationMenu.vue <template> missing expected <nav> root; template was:\n%s", tpl)
	}
	if strings.Contains(tpl, `role="navigation"`) {
		t.Errorf("NavigationMenu.vue <template> must not hand-set role=\"navigation\" on <nav> — the native element already implies it; template was:\n%s", tpl)
	}
	if !strings.Contains(tpl, `aria-label="Main"`) {
		t.Errorf("NavigationMenu.vue <nav> missing expected aria-label=\"Main\" (verified against the real Radix source); template was:\n%s", tpl)
	}
}

// TestNavigationMenu_DoesNotUseMenuRolePattern confirms this component
// deliberately does NOT carry over DropdownMenu.vue's/Menubar.vue's
// role="menu"/role="menuitem"/aria-haspopup="menu" ARIA-menu-button
// pattern — verified absent from the real Radix source (see the file's
// header comment, facts 2-5), not merely omitted by oversight.
func TestNavigationMenu_DoesNotUseMenuRolePattern(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`role="menu"`,
		`role="menuitem"`,
		`aria-haspopup`,
	} {
		if strings.Contains(tpl, marker) {
			t.Errorf("NavigationMenu.vue <template> must not carry the menu-button ARIA pattern marker %q (verified absent from real Radix's own NavigationMenu source); template was:\n%s", marker, tpl)
		}
	}
}

// TestNavigationMenu_NoManualAriaExpanded mirrors DropdownMenu.vue's own
// test: the browser sets up an implicit aria-expanded/aria-details
// relationship for any popovertarget invoker automatically, so this
// component's <template> must not hand-set aria-expanded/aria-controls on
// the trigger.
func TestNavigationMenu_NoManualAriaExpanded(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	for _, marker := range []string{
		"aria-expanded",
		"aria-controls",
		"aria-details",
	} {
		if strings.Contains(tpl, marker) {
			t.Errorf("NavigationMenu.vue <template> must not hand-set %q; the browser manages this implicitly for popovertarget invokers; template was:\n%s", marker, tpl)
		}
	}
}

// TestNavigationMenu_FlyoutItemHasPopovertargetWiring confirms a
// content-bearing item's trigger/content pair is linked declaratively via
// popovertarget/id — DropdownMenu.vue's own zero-JS mechanism, applied
// per item here — and that the content is a real popover="auto" element
// carrying the v-html binding and an aria-labelledby back to its own
// trigger.
func TestNavigationMenu_FlyoutItemHasPopovertargetWiring(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	for _, marker := range []string{
		`v-if="item.content"`,
		`:id="item.id + '-trigger'"`,
		`:popovertarget="item.id"`,
		`popovertargetaction="toggle"`,
		`class="radix-navigation-menu-trigger radix-navigation-menu-toplevel"`,
		`:id="item.id"`,
		`class="radix-navigation-menu-content"`,
		`popover="auto"`,
		`:aria-labelledby="item.id + '-trigger'"`,
		`v-html="item.content"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("NavigationMenu.vue <template> missing expected flyout-item marker %q; template was:\n%s", marker, tpl)
		}
	}

	if strings.Contains(tpl, `popover="manual"`) {
		t.Errorf("NavigationMenu.vue flyout content must not use popover=\"manual\"; template was:\n%s", tpl)
	}
}

// TestNavigationMenu_PlainLinkItemHasNoPopoverMachinery is the
// self-adversarial check this component's own process mandates: a plain
// top-level link (no `content`) must render as an ordinary <a> with
// absolutely no popovertarget/popover attributes leaking in from the
// flyout branch, and must not itself carry the flyout's own classes.
func TestNavigationMenu_PlainLinkItemHasNoPopoverMachinery(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	linkTag := navigationMenuLinkTag(t, tpl)
	for _, marker := range []string{
		"popovertarget",
		"popovertargetaction",
		"popover",
		"radix-navigation-menu-trigger",
		"radix-navigation-menu-content",
	} {
		if strings.Contains(linkTag, marker) {
			t.Errorf("NavigationMenu.vue plain-link <a> must not carry %q; link tag was:\n%s", marker, linkTag)
		}
	}

	for _, marker := range []string{
		"<a",
		`:href="item.href"`,
		`class="radix-navigation-menu-link radix-navigation-menu-toplevel"`,
	} {
		if !strings.Contains(linkTag, marker) {
			t.Errorf("NavigationMenu.vue plain-link <a> missing expected marker %q; link tag was:\n%s", marker, linkTag)
		}
	}

	// The plain-link branch must be the v-else counterpart of the
	// content-truthy check, never its own independent (and therefore
	// possibly overlapping) condition.
	if !strings.Contains(tpl, "v-else") {
		t.Errorf("NavigationMenu.vue <template> missing expected v-else plain-link branch; template was:\n%s", tpl)
	}
}

// TestNavigationMenu_TopLevelRovingTabindexBaseline confirms the static
// baseline gives the *first* top-level item tabindex="0" and every other
// item tabindex="-1", on BOTH the trigger and the link branch — Menubar.vue's
// simpler homogeneous-list convention (see the file's header comment for
// why a plain `index === 0` check is correct here, not a bounded ternary
// chain), and the process-mandated check that links and triggers are
// treated identically for the tabindex walk (a common bug is only
// counting one element type).
func TestNavigationMenu_TopLevelRovingTabindexBaseline(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)

	triggerTag := navigationMenuTriggerTag(t, tpl)
	if !strings.Contains(triggerTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Errorf("NavigationMenu.vue trigger <button> missing expected roving-tabindex baseline; button tag was:\n%s", triggerTag)
	}

	linkTag := navigationMenuLinkTag(t, tpl)
	if !strings.Contains(linkTag, `:tabindex="index === 0 ? '0' : '-1'"`) {
		t.Errorf("NavigationMenu.vue link <a> missing expected roving-tabindex baseline; link tag was:\n%s", linkTag)
	}
}

func TestNavigationMenu_ContainsScopedStyle(t *testing.T) {
	src := readNavigationMenu(t)

	if !strings.Contains(src, "<style>") {
		t.Error("NavigationMenu.vue missing expected <style scoped> block")
	}
}

func TestNavigationMenu_ContainsCustomElementEnhancement(t *testing.T) {
	src := readNavigationMenu(t)

	for _, marker := range []string{
		"<script customelement>",
		// The tag name must be exactly what component.go's real
		// deriveCustomElementTag derives for "NavigationMenu.vue"
		// (verified by actually running the algorithm, not guessed — see
		// the file's header comment: the single lowercase-followed-by-
		// uppercase boundary "n"->"M" splits it into "navigation-menu"),
		// prefixed with "radix-" per radix.go's own documented
		// Mount{Prefix: "radix"} convention every sibling component here
		// already follows.
		"customElements.define('radix-navigation-menu'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("NavigationMenu.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestNavigationMenu_RovingTabindexScriptTreatsLinksAndTriggersAlike is
// the process-mandated self-adversarial check applied to the script
// itself: the item collection must select a class present on BOTH
// branches, not a class scoped to only one of them, or the roving-tabindex
// walk would silently skip half the top-level row.
func TestNavigationMenu_RovingTabindexScriptTreatsLinksAndTriggersAlike(t *testing.T) {
	src := readNavigationMenu(t)
	tpl := navigationMenuTemplateBlock(t, src)
	script := navigationMenuScriptBlock(t, src)

	if !strings.Contains(script, "querySelectorAll('.radix-navigation-menu-toplevel')") {
		t.Errorf("NavigationMenu.vue <script customelement> must collect #items via the shared '.radix-navigation-menu-toplevel' class so links and triggers are walked identically; script was:\n%s", script)
	}

	triggerTag := navigationMenuTriggerTag(t, tpl)
	linkTag := navigationMenuLinkTag(t, tpl)
	if !strings.Contains(triggerTag, "radix-navigation-menu-toplevel") {
		t.Errorf("NavigationMenu.vue trigger <button> missing shared 'radix-navigation-menu-toplevel' class; button tag was:\n%s", triggerTag)
	}
	if !strings.Contains(linkTag, "radix-navigation-menu-toplevel") {
		t.Errorf("NavigationMenu.vue link <a> missing shared 'radix-navigation-menu-toplevel' class; link tag was:\n%s", linkTag)
	}
}

// TestNavigationMenu_LeftRightWrapsAroundAndNeverTouchesPopoverState is
// the hand-trace check for this file's one piece of real logic: Left/Right
// (and Home/End) must move focus with the established wraparound formula,
// and — per the file's own verified-not-assumed conclusion that the
// Menubar-style "close current, open adjacent" behavior does NOT apply to
// NavigationMenu — the keydown handler must never call show/hidePopover().
func TestNavigationMenu_LeftRightWrapsAroundAndNeverTouchesPopoverState(t *testing.T) {
	src := readNavigationMenu(t)
	script := navigationMenuScriptBlock(t, src)

	for _, marker := range []string{
		"'ArrowRight'",
		"'ArrowLeft'",
		"'Home'",
		"'End'",
		"(fromIndex + step + n) % n",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("NavigationMenu.vue <script customelement> missing expected roving-tabindex marker %q; script was:\n%s", marker, script)
		}
	}

	if strings.Contains(script, "Popover()") {
		t.Errorf("NavigationMenu.vue <script customelement> must never call show/hidePopover() — this v1 verified NavigationMenu's real keyboard contract has no Menubar-style adjacent-open behavior (see header comment); script was:\n%s", script)
	}
}

func navigationMenuTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("NavigationMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("NavigationMenu.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func navigationMenuScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("NavigationMenu.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("NavigationMenu.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// navigationMenuTriggerTag extracts the flyout trigger <button ...>
// opening tag's full source text.
func navigationMenuTriggerTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `class="radix-navigation-menu-trigger radix-navigation-menu-toplevel"`)
	if marker == -1 {
		t.Fatalf("NavigationMenu.vue <template> missing trigger button; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<button")
	if start == -1 {
		t.Fatalf("NavigationMenu.vue <template> could not find start of trigger <button>; template was:\n%s", tpl)
	}
	return scanNavigationMenuTagFrom(t, tpl, start)
}

// navigationMenuLinkTag extracts the plain-link <a ...> opening tag's full
// source text.
func navigationMenuLinkTag(t *testing.T, tpl string) string {
	t.Helper()
	marker := strings.Index(tpl, `class="radix-navigation-menu-link radix-navigation-menu-toplevel"`)
	if marker == -1 {
		t.Fatalf("NavigationMenu.vue <template> missing plain-link <a>; template was:\n%s", tpl)
	}
	start := strings.LastIndex(tpl[:marker], "<a")
	if start == -1 {
		t.Fatalf("NavigationMenu.vue <template> could not find start of plain-link <a>; template was:\n%s", tpl)
	}
	return scanNavigationMenuTagFrom(t, tpl, start)
}

func scanNavigationMenuTagFrom(t *testing.T, s string, start int) string {
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

func readNavigationMenu(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/NavigationMenu.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/NavigationMenu.vue) failed: %v", err)
	}
	return string(data)
}
