package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toggle.vue's/Tooltip.vue's own tests, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Toast.vue into a real htmlc.Engine and checking rendered HTML) is out of
// scope here — that proof is deliberately deferred to this commit's
// throwaway scratch render via examples/radix-demo's dependencies (see
// process notes), and to the examples/radix-demo commit that follows this
// series' own precedent. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.
func TestToast_ContainsBaselineMarkup(t *testing.T) {
	src := readToast(t)

	for _, marker := range []string{
		"<template>",
		`class="radix-toast"`,
		`:id="id"`,
		`role="status"`,
		`data-state="open"`,
		`:data-variant="variant"`,
		`:data-duration="duration"`,
		`tabindex="0"`,
		`v-if="title"`,
		`class="radix-toast-title"`,
		"{{ title }}",
		`v-if="description"`,
		`class="radix-toast-description"`,
		"{{ description }}",
		`class="radix-toast-close"`,
		`aria-label="Dismiss notification"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toast.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestToast_RoleIsAlwaysStatusNeverAlert confirms the root element's role
// is unconditionally "status" — per this file's header comment, real
// Radix's own source deliberately never uses role="alert" on a toast (a
// documented screen-reader "stuttering" bug avoidance), and this port
// preserves that finding rather than the more obvious-sounding
// role=status-vs-role=alert guess.
func TestToast_RoleIsAlwaysStatusNeverAlert(t *testing.T) {
	src := readToast(t)
	body := toastBodyAfterHeaderComment(t, src)

	if !strings.Contains(body, `role="status"`) {
		t.Errorf("Toast.vue root must carry a static role=\"status\"; got body:\n%s", body)
	}
	if strings.Contains(body, `role="alert"`) || strings.Contains(body, `:role=`) {
		t.Errorf("Toast.vue must not use role=\"alert\" or a dynamic :role binding — role is always \"status\" per the real source's own documented reasoning; got body:\n%s", body)
	}
}

// TestToast_AriaLiveDerivesFromVariant confirms aria-live is dynamically
// bound and switches between "assertive" (destructive) and "polite"
// (default) based on the variant prop — the one piece of the real ARIA
// contract that does vary per-toast (see header comment, "Accessibility
// mapping").
func TestToast_AriaLiveDerivesFromVariant(t *testing.T) {
	src := readToast(t)

	for _, marker := range []string{
		`:aria-live="variant === 'destructive' ? 'assertive' : 'polite'"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toast.vue missing expected aria-live derivation marker %q; got source:\n%s", marker, src)
		}
	}
}

func TestToast_ContainsCustomElementEnhancement(t *testing.T) {
	src := readToast(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-toast'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Toast.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestToast_AutoDismissTimerWiring confirms the timer is started from the
// rendered duration and fires the shared dismissal path with reason
// "timeout".
func TestToast_AutoDismissTimerWiring(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"window.setTimeout(() => this.#dismiss('timeout')",
		"this.#startTimer(this.#duration)",
		"Number(this.#root.dataset.duration)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected auto-dismiss marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToast_PauseOnHoverAndFocusWiring confirms both pointer (mouseenter/
// mouseleave) and focus (focusin/focusout) events are wired to the same
// shared pause/resume chokepoint, and that both are folded into one
// combined "should the timer be paused" decision — not two independent,
// potentially-conflicting pause mechanisms.
func TestToast_PauseOnHoverAndFocusWiring(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('mouseenter', this.#onPointerEnter)",
		"addEventListener('mouseleave', this.#onPointerLeave)",
		"addEventListener('focusin', this.#onFocusIn)",
		"addEventListener('focusout', this.#onFocusOut)",
		"#updatePauseState()",
		"this.#pointerInside || this.#focusInside || this.#dragging",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected pause-on-hover/focus marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToast_PauseResumeUsesRemainingTimeNotFullDuration is the content-
// level trace of this file's own self-adversarial review requirement:
// #pauseTimer must fold elapsed time into #remainingMs (not discard it),
// and #resumeTimer must restart from #remainingMs (not from the original
// #duration) — otherwise a hover-then-leave-before-expiry cycle would
// either double-fire or silently restart the full countdown instead of
// resuming where it left off.
func TestToast_PauseResumeUsesRemainingTimeNotFullDuration(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"#pauseTimer() {",
		"const elapsed = Date.now() - this.#timerStartedAt",
		"this.#remainingMs = Math.max(this.#remainingMs - elapsed, 0)",
		"#resumeTimer() {",
		"this.#startTimer(this.#remainingMs)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected pause/resume-remaining-time marker %q; script was:\n%s", marker, script)
		}
	}

	// #resumeTimer must not call #startTimer with this.#duration — that
	// would restart the full countdown instead of resuming the remainder.
	if strings.Contains(script, "#resumeTimer() {\n    if (this.#timerId !== null) return\n    this.#startTimer(this.#duration)") {
		t.Error("Toast.vue #resumeTimer must resume with #remainingMs, not restart with the full #duration")
	}
}

// TestToast_SwipeTrackingStructure confirms the pointerdown/pointermove/
// pointerup tracking triad exists on the toast's own root, uses pointer
// capture, converts horizontal drag distance into a visual slide via
// translateX, and dismisses (reason "swipe") only once past the
// documented threshold constant.
func TestToast_SwipeTrackingStructure(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"static SWIPE_THRESHOLD_PX = 50",
		"addEventListener('pointerdown', this.#onPointerDown)",
		"addEventListener('pointermove', this.#onPointerMove)",
		"addEventListener('pointerup', this.#onPointerUp)",
		"addEventListener('pointercancel', this.#onPointerCancel)",
		"this.#root.setPointerCapture(event.pointerId)",
		"translateX(",
		"Math.abs(deltaX) >= RadixToast.SWIPE_THRESHOLD_PX",
		"this.#dismiss('swipe')",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected swipe-tracking marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToast_CloseButtonDismissWiring confirms the close button routes
// through the exact same shared #dismiss path as the timer/swipe, tagged
// with reason "button".
func TestToast_CloseButtonDismissWiring(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click', this.#onCloseClick)",
		"#onCloseClick = () => {",
		"this.#dismiss('button')",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected close-button marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestToast_DismissDispatchesCancelableEventBeforeRemoving confirms the
// shared #dismiss path dispatches a cancelable, bubbling
// radix-toast-dismiss CustomEvent and only removes the element itself
// when that event was not canceled — the documented "cancelable event
// first, default removal second" design from this file's header comment.
func TestToast_DismissDispatchesCancelableEventBeforeRemoving(t *testing.T) {
	src := readToast(t)
	script := toastScriptBlock(t, src)

	for _, marker := range []string{
		"new CustomEvent('radix-toast-dismiss'",
		"bubbles: true",
		"cancelable: true",
		"const notCanceled = this.dispatchEvent(event)",
		"if (notCanceled) {\n      this.remove()",
		"if (this.#dismissed) return",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("Toast.vue <script customelement> missing expected dismissal marker %q; script was:\n%s", marker, script)
		}
	}
}

func TestToast_ContainsScopedStyle(t *testing.T) {
	src := readToast(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Toast.vue missing expected <style scoped> block")
	}
}

// toastBodyAfterHeaderComment returns the file's source with the leading
// header comment (whose prose deliberately discusses role="alert" and
// role="status" by name while explaining the design decision) stripped
// off, so markers can be searched for only in the actual template/style/
// script content below it. Mirrors the same "search after the last -->"
// technique toastScriptBlock/toggleScriptBlock use.
func toastBodyAfterHeaderComment(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Toast.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	return src[afterComment:]
}

// toastScriptBlock extracts the <script customelement>...</script> block's
// source text. It searches only after the header comment closes (the last
// "-->" before end of file) so it can't be fooled by this file's own
// header comment prose mentioning "<script customelement>" literally
// while documenting it. Mirrors toggleScriptBlock/tooltipScriptBlock.
func toastScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("Toast.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("Toast.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readToast(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Toast.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Toast.vue) failed: %v", err)
	}
	return string(data)
}
