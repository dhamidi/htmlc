package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Tooltip.vue's/Dialog.vue's own tests, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Popover.vue into a real htmlc.Engine and checking rendered HTML) is out
// of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestPopover_ContainsZeroJSBaseline(t *testing.T) {
	src := readPopover(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-popover-trigger"`,
		"<slot></slot>",
		`class="radix-popover-content"`,
		`popover="auto"`,
		`role="dialog"`,
		`aria-haspopup="dialog"`,
		`<slot name="content">`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Popover.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestPopover_PopovertargetWiring confirms the trigger button and the
// content element are linked declaratively via popovertarget/id, with
// popovertargetaction="toggle" — the zero-JS mechanism this component's
// entire open/close/dismiss story is built on (see the header comment for
// the verified spec citations). Both attributes must be bound off the same
// `id` prop so trigger and content always agree, and popovertargetaction
// must be the literal "toggle" action (not "show"/"hide") so a single
// click both opens and closes the popover.
func TestPopover_PopovertargetWiring(t *testing.T) {
	src := readPopover(t)

	for _, marker := range []string{
		`:popovertarget="id"`,
		`popovertargetaction="toggle"`,
		`:id="id"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Popover.vue missing expected popovertarget wiring marker %q; got source:\n%s", marker, src)
		}
	}
}

// TestPopover_PopoverAttributeIsAuto confirms the content element uses
// popover="auto" (not "manual") — see this file's header comment for why
// auto's light-dismiss-on-outside-click, Escape-to-close, and top-layer
// semantics are required for this component's zero-JS dismiss behavior.
func TestPopover_PopoverAttributeIsAuto(t *testing.T) {
	src := readPopover(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("Popover.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("Popover.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestPopover_NoManualAriaExpanded confirms the <template> block itself
// does not hand-set aria-expanded or aria-controls/aria-details on the
// trigger. Per this file's header comment, the browser sets up an implicit
// aria-details/aria-expanded relationship automatically once popovertarget
// links trigger and content — hand-setting a static aria-expanded in
// server-rendered markup would go stale the moment the popover's real
// state changes, since nothing in this component's own script updates it
// (that upkeep is the UA's job here, not this component's). This only
// inspects the <template>...</template> block, not the whole file — the
// header comment and script legitimately discuss these attribute names in
// prose while explaining why the template itself omits them.
func TestPopover_NoManualAriaExpanded(t *testing.T) {
	src := readPopover(t)
	template := popoverTemplateBlock(t, src)

	for _, marker := range []string{
		"aria-expanded",
		"aria-controls",
		"aria-details",
	} {
		if strings.Contains(template, marker) {
			t.Errorf("Popover.vue <template> must not hand-set %q; the browser manages this implicitly for popovertarget invokers (see header comment); template was:\n%s", marker, template)
		}
	}
}

func TestPopover_ContainsCustomElementEnhancement(t *testing.T) {
	src := readPopover(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-popover'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Popover.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestPopover_PositioningDrivenByToggleEvent confirms the script hooks the
// content's native `toggle` event (not `click` on the trigger) to trigger
// repositioning, and only repositions on the transition into the open
// state — see the header comment for why `toggle`, not `click`, is the
// only correct hook point once popovertarget owns the open/close moment.
func TestPopover_PositioningDrivenByToggleEvent(t *testing.T) {
	src := readPopover(t)
	script := popoverScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('toggle', this.#onToggle)",
		"removeEventListener('toggle', this.#onToggle)",
		"event.newState === 'open'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Popover.vue <script customelement> missing expected toggle-event marker %q; script was:\n%s", marker, script)
		}
	}

	// The script must not hook click on the trigger, nor call
	// showPopover/hidePopover itself — the native popovertarget invoker
	// mechanism owns the entire open/close lifecycle; this script only
	// reacts to it.
	for _, forbidden := range []string{
		"addEventListener('click'",
		".showPopover()",
		".hidePopover()",
	} {
		if strings.Contains(script, forbidden) {
			t.Errorf("Popover.vue <script customelement> must not contain %q; the native popovertarget mechanism owns open/close, not this script; script was:\n%s", forbidden, script)
		}
	}
}

// TestPopover_ContainsPositioningMath confirms the structure of the v1
// positioning logic: measuring both rects, viewport dimensions, a
// below-start-aligned default, a single-axis flip-if-no-room check, and
// edge-clamping on both axes. This can't execute real layout in a Go test,
// so it checks the expected structure is present in source instead.
func TestPopover_ContainsPositioningMath(t *testing.T) {
	src := readPopover(t)
	script := popoverScriptBlock(t, src)

	for _, marker := range []string{
		"getBoundingClientRect()",
		"window.innerWidth",
		"window.innerHeight",
		"triggerRect.bottom + gap",
		"triggerRect.top - gap - contentRect.height",
		"if (left < 0) left = 0",
		"if (left + contentRect.width > viewportWidth)",
		"if (top < 0) top = 0",
		"this.#content.style.top",
		"this.#content.style.left",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Popover.vue <script customelement> missing expected positioning marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestPopover_RecomputesOnResizeAndScroll confirms the v1 positioning is
// kept live against window resize/scroll while the popover is open,
// matching Tooltip.vue's identical recompute wiring.
func TestPopover_RecomputesOnResizeAndScroll(t *testing.T) {
	src := readPopover(t)
	script := popoverScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('resize', this.#reposition)",
		"addEventListener('scroll', this.#reposition",
		"removeEventListener('resize', this.#reposition)",
		"removeEventListener('scroll', this.#reposition",
		":popover-open",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Popover.vue <script customelement> missing expected resize/scroll marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestPopover_TriggerAndContentSlotStructure confirms the default slot
// fills the trigger button and a named "content" slot fills the popover's
// own body — the Dialog.vue-style slot-based content precedent this
// component follows (per the header comment) rather than Tooltip.vue's
// `content` string prop.
func TestPopover_TriggerAndContentSlotStructure(t *testing.T) {
	src := readPopover(t)

	triggerIdx := strings.Index(src, `class="radix-popover-trigger"`)
	contentIdx := strings.Index(src, `class="radix-popover-content"`)
	defaultSlotIdx := strings.Index(src, "<slot></slot>")
	namedSlotIdx := strings.Index(src, `<slot name="content">`)

	if triggerIdx == -1 || contentIdx == -1 || defaultSlotIdx == -1 || namedSlotIdx == -1 {
		t.Fatalf("Popover.vue missing expected trigger/content/slot structure; got source:\n%s", src)
	}

	if !(triggerIdx < defaultSlotIdx && defaultSlotIdx < contentIdx) {
		t.Errorf("Popover.vue's default slot must be nested inside the trigger button, before the content element; got source:\n%s", src)
	}
	if !(contentIdx < namedSlotIdx) {
		t.Errorf("Popover.vue's #content slot must be nested inside the content element; got source:\n%s", src)
	}
}

func TestPopover_ContainsScopedStyle(t *testing.T) {
	src := readPopover(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("Popover.vue missing expected <style scoped> block")
	}
}

// popoverScriptBlock extracts the <script customelement>...</script>
// block's source text. It searches only after the header comment closes
// (the last "-->" before end of file) so it can't be fooled by this file's
// own header comment prose mentioning script/event names literally while
// documenting them.
func popoverScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Popover.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Popover.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// popoverTemplateBlock extracts the <template>...</template> block's
// source text, the same searched-after-the-header-comment approach as
// popoverScriptBlock, so it can't be fooled by prose mentioning template
// markup literally while documenting it.
func popoverTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Popover.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Popover.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func readPopover(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Popover.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Popover.vue) failed: %v", err)
	}
	return string(data)
}
