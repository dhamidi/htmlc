package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Tooltip.vue's/Popover.vue's own tests, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// HoverCard.vue into a real htmlc.Engine and checking rendered HTML) is out
// of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file contains
// the markers this commit's design depends on.
func TestHoverCard_ContainsZeroJSBaseline(t *testing.T) {
	src := readHoverCard(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-hover-card-trigger"`,
		"<slot></slot>",
		`class="radix-hover-card-content"`,
		`popover="auto"`,
		`:data-open-delay-ms="openDelayMs"`,
		`:data-close-delay-ms="closeDelayMs"`,
		`<slot name="content">`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("HoverCard.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestHoverCard_PopoverAttributeIsAuto confirms the content element uses
// popover="auto" (not "manual") — same interaction category as Tooltip.vue
// (hover-triggered, expected to light-dismiss on outside click and Escape,
// and to obey the browser's single-open-auto-popover rule), per this file's
// header comment.
func TestHoverCard_PopoverAttributeIsAuto(t *testing.T) {
	src := readHoverCard(t)

	if !strings.Contains(src, `popover="auto"`) {
		t.Errorf("HoverCard.vue content element must carry popover=\"auto\"; got source:\n%s", src)
	}
	if strings.Contains(src, `popover="manual"`) {
		t.Errorf("HoverCard.vue content element must not use popover=\"manual\"; got source:\n%s", src)
	}
}

// TestHoverCard_NoInventedRole confirms the <template> block does not set an
// explicit `role` on the content element. Real Radix's own
// HoverCardContent/HoverCardTrigger set no `role` or `aria-*` attribute at
// all (verified directly against packages/react/hover-card/src/hover-card.tsx
// before this component was written) — unlike Tooltip's role="tooltip" (that
// role forbids interactive content, which a HoverCard's richer body can
// contain) or Popover's role="dialog" (an explicit, source-verified choice
// specific to that component). Copying either by analogy without checking
// the real HoverCard source would misrepresent this component's semantics
// to assistive technology, so this test locks in that it was not copied.
// Only checks the <template> block, not the whole file — the header comment
// legitimately discusses role names in prose while explaining this cut.
func TestHoverCard_NoInventedRole(t *testing.T) {
	src := readHoverCard(t)
	template := hoverCardTemplateBlock(t, src)

	if strings.Contains(template, "role=") {
		t.Errorf("HoverCard.vue <template> must not set an explicit role attribute; real Radix's HoverCard sets none (verified against source); template was:\n%s", template)
	}
}

func TestHoverCard_ContainsCustomElementEnhancement(t *testing.T) {
	src := readHoverCard(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-hover-card'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("HoverCard.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestHoverCard_TriggerAndContentSlotStructure confirms the default slot
// fills the trigger button and a named "content" slot fills the card's own
// body — matching Popover.vue's identical richer-content precedent (per the
// header comment) rather than Tooltip.vue's `content` string prop.
func TestHoverCard_TriggerAndContentSlotStructure(t *testing.T) {
	src := readHoverCard(t)

	triggerIdx := strings.Index(src, `class="radix-hover-card-trigger"`)
	contentIdx := strings.Index(src, `class="radix-hover-card-content"`)
	defaultSlotIdx := strings.Index(src, "<slot></slot>")
	namedSlotIdx := strings.Index(src, `<slot name="content">`)

	if triggerIdx == -1 || contentIdx == -1 || defaultSlotIdx == -1 || namedSlotIdx == -1 {
		t.Fatalf("HoverCard.vue missing expected trigger/content/slot structure; got source:\n%s", src)
	}

	if !(triggerIdx < defaultSlotIdx && defaultSlotIdx < contentIdx) {
		t.Errorf("HoverCard.vue's default slot must be nested inside the trigger button, before the content element; got source:\n%s", src)
	}
	if !(contentIdx < namedSlotIdx) {
		t.Errorf("HoverCard.vue's #content slot must be nested inside the content element; got source:\n%s", src)
	}
}

// TestHoverCard_TriggerEventWiring confirms the trigger listens for the
// documented hover/focus/dismiss events, routed through the shared
// #onEnter/#onLeave handlers, matching Tooltip.vue's identical wiring.
func TestHoverCard_TriggerEventWiring(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	for _, marker := range []string{
		"this.#trigger.addEventListener('mouseenter', this.#onEnter)",
		"this.#trigger.addEventListener('focus', this.#onEnter)",
		"this.#trigger.addEventListener('mouseleave', this.#onLeave)",
		"this.#trigger.addEventListener('blur', this.#onLeave)",
		"this.#trigger.addEventListener('keydown', this.#onKeyDown)",
		"event.key === 'Escape'",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("HoverCard.vue <script customelement> missing expected trigger event-wiring marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestHoverCard_ContentEventWiring confirms the content element ALSO gets
// mouseenter/mouseleave listeners routed through the same #onEnter/#onLeave
// handlers as the trigger. This is the crux of this component's whole
// design: without listeners on the content element too, hovering the card's
// own (often-interactive) body would always close it, defeating the point
// of a close delay in the first place. See this file's header comment for
// the hand-traced scenario this locks in.
func TestHoverCard_ContentEventWiring(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	for _, marker := range []string{
		"this.#content.addEventListener('mouseenter', this.#onEnter)",
		"this.#content.addEventListener('mouseleave', this.#onLeave)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("HoverCard.vue <script customelement> missing expected content event-wiring marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestHoverCard_ShowsAndHidesViaPopoverMethods confirms the enhancement
// calls the real Popover API methods (not, say, a style.display toggle) to
// open/close the content, matching the popover="auto" baseline.
func TestHoverCard_ShowsAndHidesViaPopoverMethods(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	for _, marker := range []string{
		".showPopover()",
		".hidePopover()",
		":popover-open",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("HoverCard.vue <script customelement> missing expected popover-method marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestHoverCard_OpenDelayDefaultsTo700 confirms the open-delay fallback
// matches Radix's own documented default (openDelay = 700 in
// packages/react/hover-card/src/hover-card.tsx's HoverCardProps), read from
// source rather than guessed.
func TestHoverCard_OpenDelayDefaultsTo700(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	if !strings.Contains(script, "Number(this.#content.dataset.openDelayMs) || 700") {
		t.Errorf("HoverCard.vue <script customelement> missing expected open-delay-default marker; script was:\n%s", script)
	}
}

// TestHoverCard_CloseDelayDefaultsTo300 confirms the close-delay fallback
// matches Radix's own documented default (closeDelay = 300 in that same
// HoverCardProps), read from source rather than guessed. This is the delay
// Tooltip.vue does not have at all — Tooltip hides immediately on
// mouseleave/blur.
func TestHoverCard_CloseDelayDefaultsTo300(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	if !strings.Contains(script, "Number(this.#content.dataset.closeDelayMs) || 300") {
		t.Errorf("HoverCard.vue <script customelement> missing expected close-delay-default marker; script was:\n%s", script)
	}
}

// TestHoverCard_LeaveStartsCloseTimerInsteadOfHidingImmediately confirms
// #onLeave starts a close-delay timer rather than calling #hide()
// synchronously — the behavior that most distinguishes this component from
// Tooltip.vue's #onLeave (which hides immediately).
func TestHoverCard_LeaveStartsCloseTimerInsteadOfHidingImmediately(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	onLeaveIdx := strings.Index(script, "#onLeave = () => {")
	if onLeaveIdx == -1 {
		t.Fatalf("HoverCard.vue <script customelement> missing #onLeave handler; script was:\n%s", script)
	}
	onLeaveBody := script[onLeaveIdx:]
	end := strings.Index(onLeaveBody, "\n\n")
	if end != -1 {
		onLeaveBody = onLeaveBody[:end]
	}

	setTimeoutIdx := strings.Index(onLeaveBody, "this.#closeTimer = window.setTimeout(")
	if setTimeoutIdx == -1 {
		t.Fatalf("HoverCard.vue #onLeave handler must start a close-delay timer, not hide immediately; handler body was:\n%s", onLeaveBody)
	}

	// #hide() must only appear *inside* the setTimeout callback (i.e. after
	// the setTimeout( call itself in source order) — never as a synchronous
	// call in #onLeave's own top-level body.
	hideIdx := strings.Index(onLeaveBody, "this.#hide()")
	if hideIdx == -1 || hideIdx < setTimeoutIdx {
		t.Errorf("HoverCard.vue #onLeave handler must call #hide() only from within the close-delay timer's callback, not synchronously; handler body was:\n%s", onLeaveBody)
	}
}

// TestHoverCard_EnterCancelsPendingCloseTimer is the core "pointer entered
// content before the close delay elapsed" behavior this component adds on
// top of Tooltip.vue, hand-traced in this commit's own self-review: since
// #onEnter is wired to both the trigger (see
// TestHoverCard_TriggerEventWiring) and the content
// (TestHoverCard_ContentEventWiring), moving the pointer from the trigger
// into the content fires the content's mouseenter, which must clear
// whatever close timer the trigger's mouseleave started moments earlier —
// otherwise the card would close underneath the user's own cursor while
// they're still hovering it.
func TestHoverCard_EnterCancelsPendingCloseTimer(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	onEnterIdx := strings.Index(script, "#onEnter = () => {")
	if onEnterIdx == -1 {
		t.Fatalf("HoverCard.vue <script customelement> missing #onEnter handler; script was:\n%s", script)
	}
	onEnterBody := script[onEnterIdx:]
	end := strings.Index(onEnterBody, "\n\n")
	if end != -1 {
		onEnterBody = onEnterBody[:end]
	}

	if !strings.Contains(onEnterBody, "this.#clearCloseTimer()") {
		t.Errorf("HoverCard.vue #onEnter handler must clear any pending close timer; handler body was:\n%s", onEnterBody)
	}

	// #clearCloseTimer must be the first statement in the handler body, so
	// that no early return before it can leave a close timer dangling.
	firstStatement := strings.TrimSpace(strings.SplitN(onEnterBody, "\n", 2)[1])
	if !strings.HasPrefix(firstStatement, "this.#clearCloseTimer()") {
		t.Errorf("HoverCard.vue #onEnter handler must clear the pending close timer as its first statement; handler body was:\n%s", onEnterBody)
	}
}

// TestHoverCard_NoDanglingTimersOnEarlyLeaveOrDisconnect confirms both the
// open-delay and close-delay timers are tracked in instance fields and
// explicitly cleared: #onEnter clears any pending close timer, #onLeave
// clears any pending open timer, and disconnectedCallback clears both — the
// same dangling-setTimeout fix Tooltip.vue's own test locks in, extended
// here to HoverCard's second timer.
func TestHoverCard_NoDanglingTimersOnEarlyLeaveOrDisconnect(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	for _, marker := range []string{
		"#openTimer = null",
		"#closeTimer = null",
		"window.setTimeout(",
		"window.clearTimeout(this.#openTimer)",
		"window.clearTimeout(this.#closeTimer)",
		"#clearOpenTimer()",
		"#clearCloseTimer()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("HoverCard.vue <script customelement> missing expected timer-cleanup marker %q; script was:\n%s", marker, script)
		}
	}

	if !strings.Contains(script, "disconnectedCallback() {\n    this.#clearOpenTimer()\n    this.#clearCloseTimer()") {
		t.Errorf("HoverCard.vue disconnectedCallback must clear both timers on disconnect; script was:\n%s", script)
	}
}

// TestHoverCard_EscapeClearsBothTimers confirms Escape hides the card and
// clears both pending timers, not just one.
func TestHoverCard_EscapeClearsBothTimers(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	onKeyDownIdx := strings.Index(script, "#onKeyDown = (event) => {")
	if onKeyDownIdx == -1 {
		t.Fatalf("HoverCard.vue <script customelement> missing #onKeyDown handler; script was:\n%s", script)
	}
	onKeyDownBody := script[onKeyDownIdx:]
	end := strings.Index(onKeyDownBody, "\n\n")
	if end != -1 {
		onKeyDownBody = onKeyDownBody[:end]
	}

	for _, marker := range []string{
		"this.#clearOpenTimer()",
		"this.#clearCloseTimer()",
		"this.#hide()",
	} {
		if !strings.Contains(onKeyDownBody, marker) {
			t.Errorf("HoverCard.vue #onKeyDown handler missing expected marker %q; handler body was:\n%s", marker, onKeyDownBody)
		}
	}
}

// TestHoverCard_ContainsPositioningMath confirms the structure of the v1
// positioning logic: measuring both rects, viewport dimensions, a
// below-centered default, a single-axis flip-if-no-room check, and
// edge-clamping on both axes — reused from Tooltip.vue's own v1. This can't
// execute real layout in a Go test, so it checks the expected structure is
// present in source instead.
func TestHoverCard_ContainsPositioningMath(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

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
			t.Errorf("HoverCard.vue <script customelement> missing expected positioning marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestHoverCard_RecomputesOnResizeAndScroll confirms the v1 positioning is
// kept live against window resize/scroll, matching Tooltip.vue's/
// Popover.vue's identical recompute wiring.
func TestHoverCard_RecomputesOnResizeAndScroll(t *testing.T) {
	src := readHoverCard(t)
	script := hoverCardScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('resize', this.#reposition)",
		"addEventListener('scroll', this.#reposition",
		"removeEventListener('resize', this.#reposition)",
		"removeEventListener('scroll', this.#reposition",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("HoverCard.vue <script customelement> missing expected resize/scroll marker %q; script was:\n%s", marker, script)
		}
	}
}

func TestHoverCard_ContainsScopedStyle(t *testing.T) {
	src := readHoverCard(t)

	if !strings.Contains(src, "<style>") {
		t.Error("HoverCard.vue missing expected <style scoped> block")
	}
}

// hoverCardScriptBlock extracts the <script customelement>...</script>
// block's source text. It searches only after the header comment closes
// (the last "-->" before end of file) so it can't be fooled by this file's
// own header comment prose mentioning script/event names literally while
// documenting them.
func hoverCardScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("HoverCard.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("HoverCard.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

// hoverCardTemplateBlock extracts the <template>...</template> block's
// source text, the same searched-after-the-header-comment approach as
// hoverCardScriptBlock, so it can't be fooled by prose mentioning template
// markup literally while documenting it.
func hoverCardTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("HoverCard.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("HoverCard.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

func readHoverCard(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/HoverCard.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/HoverCard.vue) failed: %v", err)
	}
	return string(data)
}
