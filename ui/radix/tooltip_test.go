package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toggle.vue's/Dialog.vue's own tests, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Tooltip.vue into a real htmlc.Engine and checking rendered HTML) is out
// of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestTooltip_ContainsZeroJSBaseline(t *testing.T) {
	src := readTooltip(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-tooltip-trigger"`,
		"<slot></slot>",
		`class="radix-tooltip-content"`,
		`popover="auto"`,
		`role="tooltip"`,
		`:data-delay-ms="delayMs"`,
		"{{ content }}",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Tooltip.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestTooltip_PopoverAttributeIsAuto confirms the content element uses
// popover="auto" (not "manual") — see this file's header comment for why
// "auto"'s light-dismiss-on-outside-click and single-open-popover-at-a-time
// semantics are the correct choice for a tooltip, matching what real
// Radix's TooltipTrigger/TooltipContent do by hand.
func TestTooltip_PopoverAttributeIsAuto(t *testing.T) {
	src := readTooltip(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("Tooltip.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("Tooltip.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

func TestTooltip_ContainsCustomElementEnhancement(t *testing.T) {
	src := readTooltip(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-tooltip'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Tooltip.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestTooltip_TriggerEventWiring confirms the trigger listens for the four
// documented hover/focus/dismiss events, and that both mouseenter and
// focus route through the same delayed-show path while mouseleave/blur/
// Escape route through the same immediate-hide path.
func TestTooltip_TriggerEventWiring(t *testing.T) {
	src := readTooltip(t)
	script := tooltipScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('mouseenter', this.#onEnter)",
		"addEventListener('focus', this.#onEnter)",
		"addEventListener('mouseleave', this.#onLeave)",
		"addEventListener('blur', this.#onLeave)",
		"addEventListener('keydown', this.#onKeyDown)",
		"event.key === 'Escape'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Tooltip.vue <script customelement> missing expected event-wiring marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestTooltip_ShowsAndHidesViaPopoverMethods confirms the enhancement
// calls the real Popover API methods (not, say, a style.display toggle)
// to open/close the content, matching the popover="auto" baseline.
func TestTooltip_ShowsAndHidesViaPopoverMethods(t *testing.T) {
	src := readTooltip(t)
	script := tooltipScriptBlock(t, src)

	for _, marker := range []string{
		".showPopover()",
		".hidePopover()",
		":popover-open",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Tooltip.vue <script customelement> missing expected popover-method marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestTooltip_NoDanglingTimeoutOnEarlyLeave confirms the show-delay timer
// is tracked in an instance field and explicitly cleared both when the
// pointer/focus leaves before the delay elapses (#onLeave) and on
// disconnect — the fix for the classic dangling-setTimeout bug in
// hover-delay implementations (a timer that fires after its trigger is no
// longer hovered/focused).
func TestTooltip_NoDanglingTimeoutOnEarlyLeave(t *testing.T) {
	src := readTooltip(t)
	script := tooltipScriptBlock(t, src)

	for _, marker := range []string{
		"#openTimer = null",
		"window.setTimeout(",
		"window.clearTimeout(this.#openTimer)",
		"#clearOpenTimer()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Tooltip.vue <script customelement> missing expected timer-cleanup marker %q; script was:\n%s", marker, script)
		}
	}

	// #onLeave must clear the timer before anything else — confirm the
	// clear call appears inside the #onLeave handler body, not just
	// somewhere else in the file.
	onLeaveIdx := strings.Index(script, "#onLeave = () => {")
	if onLeaveIdx == -1 {
		t.Fatalf("Tooltip.vue <script customelement> missing #onLeave handler; script was:\n%s", script)
	}
	onLeaveBody := script[onLeaveIdx:]
	end := strings.Index(onLeaveBody, "\n\n")
	if end != -1 {
		onLeaveBody = onLeaveBody[:end]
	}
	if !strings.Contains(onLeaveBody, "#clearOpenTimer()") {
		t.Errorf("Tooltip.vue #onLeave handler must clear the pending show timer; handler body was:\n%s", onLeaveBody)
	}

	if !strings.Contains(script, "disconnectedCallback() {\n    this.#clearOpenTimer()") {
		t.Errorf("Tooltip.vue disconnectedCallback must clear the open timer on disconnect; script was:\n%s", script)
	}
}

// TestTooltip_ContainsPositioningMath confirms the structure of the v1
// positioning logic: measuring both rects, viewport dimensions, a
// below-centered default, a single-axis flip-if-no-room check, and
// edge-clamping on both axes. This can't execute real layout in a Go test,
// so it checks the expected structure is present in source instead.
func TestTooltip_ContainsPositioningMath(t *testing.T) {
	src := readTooltip(t)
	script := tooltipScriptBlock(t, src)

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
			t.Errorf("Tooltip.vue <script customelement> missing expected positioning marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestTooltip_RecomputesOnResizeAndScroll confirms the v1 positioning is
// kept live against window resize/scroll, per this commit's process
// instructions ("recompute on window resize/scroll if you have time/scope
// for it").
func TestTooltip_RecomputesOnResizeAndScroll(t *testing.T) {
	src := readTooltip(t)
	script := tooltipScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('resize', this.#reposition)",
		"addEventListener('scroll', this.#reposition",
		"removeEventListener('resize', this.#reposition)",
		"removeEventListener('scroll', this.#reposition",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Tooltip.vue <script customelement> missing expected resize/scroll marker %q; script was:\n%s", marker, script)
		}
	}
}

func TestTooltip_ContainsScopedStyle(t *testing.T) {
	src := readTooltip(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Tooltip.vue missing expected <style scoped> block")
	}
}

// tooltipScriptBlock extracts the <script customelement>...</script>
// block's source text. It searches only after the header comment closes
// (the last "-->" before end of file) so it can't be fooled by this file's
// own header comment prose mentioning script/event names literally while
// documenting them.
func tooltipScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Tooltip.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Tooltip.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readTooltip(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Tooltip.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Tooltip.vue) failed: %v", err)
	}
	return string(data)
}
