<!--
  Toast — Radix-inspired transient notification.

  ## Re-scoped, deliberately, from what "Toast" means in real Radix

  Real Radix Toast is three parts working together: `ToastProvider` (a
  React context holding shared config — duration, swipe direction/threshold,
  a label), `ToastViewport` (one shared, app-wide fixed-position region
  where toasts render, with its own queue/limit and pause-on-hover/focus
  coordination across every mounted toast), and `Toast` itself (one
  notification). That whole provider/queue/viewport architecture assumes a
  client-side JS framework where toasts are *dynamically created at
  runtime* in response to some later event (a fetch response, a form
  submission) — a fundamentally different shape from every other
  ui/radix component so far, all of which render known, correct state at
  the moment of a server response. htmlc has no client-side state/queue
  system, and — per Toolbar.vue's/Menubar.vue's own design notes — this
  library's established convention is not to invent one just for this
  port.

  **What this file actually is**: the markup and self-contained behavior
  for *one already-triggered, already-rendered* toast notification —
  auto-dismiss timer, pause-on-hover/focus, swipe-to-dismiss, a dismiss
  button, and correct role/aria-live semantics. It does not ship a
  `ToastViewport` equivalent, and it has no notion of a queue, a limit, or
  "one toast replaces another." Getting a `<Toast>` onto the page in the
  first place — deciding *when* one is needed and inserting its rendered
  markup into the DOM — is the caller's job, via whichever mechanism fits
  their app:
    - a server-rendered out-of-band swap, using this repository's own
      hypermedia/htmx or hypermedia/turbo modules (built earlier in this
      same project) to OOB-swap a rendered `<Toast>` into a fixed-position
      container the page already has; or
    - plain client-side `document.createElement`/`insertAdjacentHTML`
      insertion from the caller's own script (e.g. inside a `fetch()`
      response handler).
  Either way, the consuming page provides its own simple fixed-position
  container for toasts to land in, e.g.:

    <div
      id="toast-region"
      role="region"
      aria-label="Notifications"
      style="position: fixed; bottom: 1rem; right: 1rem; display: flex;
             flex-direction: column; gap: 0.5rem; z-index: 999;"
    ></div>

  and appends/prepends rendered `<Toast>` instances into it however fits
  their own stack (newest-on-top vs. newest-on-bottom, a max-visible cap,
  etc.) — all of that orchestration is explicitly out of scope here, the
  same "caller owns this" line Toolbar.vue/Menubar.vue draw for their own
  omitted orchestration layers.

  ## Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Avatar.vue's/Toggle.vue's header comments for
  the "[missing: <name>]" placeholder trap this convention avoids):

    id:          string — required so a consumer's own removal logic can
                 target this exact toast directly (e.g.
                 `document.getElementById(id).remove()`), independent of
                 whatever event-based dismissal this file's own script
                 does (see "Dismissal" below). Rendered as the root
                 element's `id` attribute.
    title:       string — the toast's heading. Pass "" to omit (rendered
                 conditionally via `v-if`, the same "empty string skips
                 the element" convention NavigationMenu.vue's/
                 Separator.vue's own `v-if` on a plain prop already use).
    description: string — the toast's body copy. Also optional-by-empty-
                 string, same as `title`.
    duration:    number — auto-dismiss timing, in milliseconds. Radix's
                 own real default (`ToastProviderProps`' `duration = 5000`)
                 is 5000ms. This port has no provider to supply that default
                 implicitly, so callers pass it explicitly (5000 is the
                 value to pass for parity with Radix's own default).
                 Real Radix additionally accepts `Infinity` for "never
                 auto-dismiss"; Go has no clean equivalent to serialize
                 into a numeric prop, so this port uses `0` for that same
                 "persistent, no timer" meaning instead (documented
                 deviation) — the script below's `#startTimer` treats a
                 falsy duration as "don't start a timer," mirroring real
                 Radix's own `if (!duration || duration === Infinity)
                 return` guard in spirit.
    variant:     string — `"default"` | `"destructive"`. Drives the
                 root's `role`/`aria-live` — see "Accessibility mapping"
                 below for the mapping this port actually implements,
                 which is a documented simplification of what the real
                 source does, not a one-to-one port of it.

  ## Accessibility mapping — *not* what a naive reading of "role=status vs
  role=alert" would suggest

  Reading toast.tsx directly (not transcribed below, only the finding):
  the *visible* toast element (`Primitive.li`) in real Radix carries no
  `role`/`aria-live` attribute of its own at all. Screen-reader
  announcement instead happens through a completely separate,
  `VisuallyHidden`-wrapped clone of the toast's text content
  (`ToastAnnounce`, driven by `getAnnounceTextContent`), and *that* hidden
  clone is where `role`/`aria-live` actually live — with a comment in the
  source spelling out the reasoning: "Toasts are always role=status to
  avoid stuttering issues with role=alert in SRs." In other words, real
  Radix deliberately *never* uses `role="alert"` anywhere in this
  component, precisely because of a known screen-reader bug class — the
  variable part is only `aria-live` (`"assertive"` vs `"polite"`), toggled
  by a `type` prop (`"foreground"` | `"background"`, default
  `"foreground"` → `"assertive"`) that has nothing to do with visual
  severity/color (it's about whether the toast resulted from the user's
  own action or not).

  This port does not reproduce the separate-hidden-announce-clone
  architecture (a second, content-duplicating DOM subtree per toast is a
  lot of machinery for a v1 port to carry, and — see "Dismissal" below —
  this file's dismiss button already needs the visible root to be the
  thing assistive tech can reach directly). Instead, `role="status"` and
  a variant-derived `aria-live` are placed directly on the visible root:
  `variant="destructive"` → `aria-live="assertive"` (interrupting,
  matching the urgency a "destructive" toast is meant to convey);
  `variant="default"` → `aria-live="polite"` (non-interrupting). Per the
  source finding above, `role` itself is `"status"` for *both* variants —
  never `"alert"` — deliberately, to avoid exactly the SR-stuttering
  failure mode Radix's own source comment documents. A reader expecting
  a role=status-vs-role=alert split per variant (a reasonable-sounding
  guess, and this component's own starting assumption before reading the
  source) would be reproducing a real, documented accessibility bug —
  this is the honest finding after checking, not the guess.

  ## Dismissal — cancelable event first, then a default removal

  Every dismissal path (auto-dismiss timer, swipe past the threshold,
  clicking the close button) funnels through one `#dismiss(reason)`
  method in the script below, which:
    1. dispatches a bubbling, *cancelable* `radix-toast-dismiss`
       CustomEvent (`detail: { id, reason }`, `reason` one of `"timeout"`,
       `"swipe"`, `"button"`) — giving a listening caller a chance to
       react (log the dismissal, clean up related state) or to fully take
       over removal (e.g. play its own exit animation before removing the
       node) by calling `event.preventDefault()`;
    2. removes this element from the DOM itself (`this.remove()`) — but
       only if nothing called `preventDefault()` on that event.
  This is a "which is more useful/composable" design call: a pure "always
  dispatch an event, never touch the DOM" design (Toggle.vue's own choice
  for its click handler)
  would leave a toast that nobody happens to be listening for stuck in
  the DOM forever, which is a real usability regression for a component
  whose entire point is disappearing on its own. A pure "always remove
  directly, no event" design would leave a caller wanting to log
  dismissals or animate an exit with no hook to do so. The cancelable-
  event-with-a-default-action pattern (the same shape as the platform's
  own cancelable events, e.g. `beforeunload`) gets both: correct,
  listener-free behavior out of the box, and a documented override point
  for a caller who wants more control.

  ## `<script customelement>` (load-bearing — a toast that doesn't
  auto-dismiss or respond to swipe isn't really a toast)

  See the script block's own comments for the auto-dismiss timer, the
  pause-on-hover/focus logic (including the remaining-time-not-full-
  duration resume this file's own self-review specifically hand-traced —
  see the method comments on `#pauseTimer`/`#resumeTimer`), and the
  swipe-to-dismiss pointer tracking (a simpler, single-axis, bidirectional
  version of the drag-tracking pattern ScrollArea.vue's thumb-drag already
  established in this library).

  Swipe direction: real Radix's own default (`ToastProviderProps`'
  `swipeThreshold = 50`, verified from the source — the same value this
  port uses) additionally restricts *which* direction counts as a
  dismissing swipe (`swipeDirection`, defaulting to `"right"`), because
  the real ToastProvider knows which screen corner its ToastViewport sits
  in and only the direction that visually slides a toast off toward that
  edge should dismiss it. This port has no provider/viewport and so no
  way to know where on the caller's page a given toast is stacked —
  guessing a single fixed direction would be less honest than supporting
  both: swiping past the threshold in *either* horizontal direction
  dismisses here, a documented, deliberate scope simplification, not an
  oversight.

  ## No v-native escape hatch needed

  This component's own name ("Toast") auto-registers a lowercase "toast"
  alias in the component registry. There is no native HTML element
  literally named `<toast>` for that alias to collide with — the elements
  actually used below (`<li>`, `<div>`, `<p>`, `<button>`) are unrelated
  native tags, the same reasoning Avatar.vue's/Tooltip.vue's header
  comments work through for their own files.

  Usage (rendered by the caller into their own fixed-position container,
  e.g. via an OOB swap or plain DOM insertion — see above):

    <Toast
      id="toast-1"
      title="Saved"
      description="Your changes have been saved."
      :duration="5000"
      variant="default"
    />

    <Toast
      id="toast-2"
      title="Something went wrong"
      description="Could not save your changes. Try again."
      :duration="5000"
      variant="destructive"
    />
-->
<template>
  <Tokens></Tokens>
  <li
    class="radix-toast"
    :id="id"
    role="status"
    :aria-live="variant === 'destructive' ? 'assertive' : 'polite'"
    data-state="open"
    :data-variant="variant"
    :data-duration="duration"
    tabindex="0"
  >
    <div class="radix-toast-content">
      <p v-if="title" class="radix-toast-title">{{ title }}</p>
      <p v-if="description" class="radix-toast-description">{{ description }}</p>
    </div>
    <button v-native type="button" class="radix-toast-close" aria-label="Dismiss notification">
      <span aria-hidden="true">&times;</span>
    </button>
  </li>
</template>

<style>
.radix-toast {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  width: 100%;
  max-width: 24rem;
  padding: 0.9rem 1rem;
  /* Without this, the browser's default content-box model adds the
     padding and border on top of the 100% width, pushing the toast past
     its container's right edge — invisible on a wide page with generous
     margin either side, but a real overflow on a narrow (mobile) viewport.
     Found via an actual mobile-width screenshot, not caught by markup
     review. */
  box-sizing: border-box;
  background-color: var(--radix-sand-1);
  border: 1px solid var(--radix-sand-6);
  border-radius: var(--radix-radius-4);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
  touch-action: none; /* dragging the toast itself should not also pan the page */
  user-select: none;
}

.radix-toast[data-variant='destructive'] {
  border-color: var(--radix-ruby-9);
}

.radix-toast[data-variant='destructive'] .radix-toast-title {
  color: var(--radix-ruby-11);
}

.radix-toast-content {
  flex: 1;
  min-width: 0;
}

.radix-toast-title {
  margin: 0 0 0.2rem;
  font-weight: 600;
  font-size: 0.9rem;
}

.radix-toast-description {
  margin: 0;
  font-size: 0.85rem;
  color: var(--radix-sand-11);
}

.radix-toast-close {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.5rem;
  height: 1.5rem;
  padding: 0;
  background: none;
  border: none;
  border-radius: var(--radix-radius-2);
  color: var(--radix-sand-11);
  font-size: 1.1rem;
  line-height: 1;
  cursor: pointer;
}

.radix-toast-close:hover {
  background-color: rgba(0, 0, 0, 0.06);
  color: var(--radix-sand-12);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-toast-close:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

/* Focus outline on the toast root itself, reached via Tab (tabindex="0"
   above) — also the state that pauses the auto-dismiss timer, see the
   script block below. */
.radix-toast:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the server-rendered, already-open
// toast markup above. Per RFC 014 §3 Non-Goals, this does not reimplement
// anything the browser already provides for free — there is nothing to
// reimplement here, since (unlike Dialog's <dialog>/Tooltip's popover)
// there is no native "toast" primitive at all. Everything a toast is
// actually known for — timing out, pausing while inspected, being swiped
// away, closing on click — is JS-driven by design, so this block is not
// optional polish; it is the component.
class RadixToast extends HTMLElement {
  // Radix's own real default swipeThreshold (ToastProviderProps).
  static SWIPE_THRESHOLD_PX = 50

  #root = null
  #closeButton = null
  #dismissed = false

  // Auto-dismiss timer bookkeeping. #remainingMs starts as the full
  // duration and is only ever decremented by #pauseTimer while a timer
  // was actually running; #timerStartedAt records when the *current*
  // countdown segment began, so #pauseTimer can compute exactly how much
  // of it has elapsed so far.
  #duration = 0
  #remainingMs = 0
  #timerId = null
  #timerStartedAt = 0

  // Independent "why should the timer be paused right now" flags —
  // pause takes effect the moment any is true; resume only happens once
  // *all* of them are false again, so e.g. leaving with the pointer while
  // still keyboard-focused correctly keeps the timer paused.
  #pointerInside = false
  #focusInside = false
  #dragging = false

  // Swipe-to-dismiss pointer tracking (single horizontal axis).
  #dragStartX = 0

  connectedCallback() {
    this.#root = this.querySelector('.radix-toast')
    this.#closeButton = this.querySelector('.radix-toast-close')
    if (!this.#root) return

    this.#duration = Number(this.#root.dataset.duration) || 0
    this.#remainingMs = this.#duration

    this.#root.addEventListener('mouseenter', this.#onPointerEnter)
    this.#root.addEventListener('mouseleave', this.#onPointerLeave)
    this.#root.addEventListener('focusin', this.#onFocusIn)
    this.#root.addEventListener('focusout', this.#onFocusOut)
    this.#root.addEventListener('pointerdown', this.#onPointerDown)
    this.#root.addEventListener('pointermove', this.#onPointerMove)
    this.#root.addEventListener('pointerup', this.#onPointerUp)
    this.#root.addEventListener('pointercancel', this.#onPointerCancel)
    if (this.#closeButton) {
      this.#closeButton.addEventListener('click', this.#onCloseClick)
    }

    this.#startTimer(this.#duration)
  }

  disconnectedCallback() {
    this.#clearTimer()
    if (this.#root) {
      this.#root.removeEventListener('mouseenter', this.#onPointerEnter)
      this.#root.removeEventListener('mouseleave', this.#onPointerLeave)
      this.#root.removeEventListener('focusin', this.#onFocusIn)
      this.#root.removeEventListener('focusout', this.#onFocusOut)
      this.#root.removeEventListener('pointerdown', this.#onPointerDown)
      this.#root.removeEventListener('pointermove', this.#onPointerMove)
      this.#root.removeEventListener('pointerup', this.#onPointerUp)
      this.#root.removeEventListener('pointercancel', this.#onPointerCancel)
    }
    if (this.#closeButton) {
      this.#closeButton.removeEventListener('click', this.#onCloseClick)
    }
  }

  // --- Auto-dismiss timer -------------------------------------------

  #clearTimer() {
    if (this.#timerId !== null) {
      window.clearTimeout(this.#timerId)
      this.#timerId = null
    }
  }

  // Starts (or restarts) the countdown for `ms` milliseconds. A falsy
  // duration (0, per this file's documented Infinity-substitute — see
  // header comment) means "never auto-dismiss," matching real Radix's
  // own `if (!duration || duration === Infinity) return` guard in spirit.
  #startTimer(ms) {
    this.#clearTimer()
    if (!ms) return
    this.#timerStartedAt = Date.now()
    this.#timerId = window.setTimeout(() => this.#dismiss('timeout'), ms)
  }

  // Pauses the running timer, if one is running, and folds the elapsed
  // time of *this* countdown segment into #remainingMs. Guarded by the
  // #timerId !== null check so calling this more than once in a row
  // (e.g. mouseenter firing while already focus-paused) is a no-op rather
  // than double-subtracting elapsed time.
  #pauseTimer() {
    if (this.#timerId === null) return
    const elapsed = Date.now() - this.#timerStartedAt
    this.#remainingMs = Math.max(this.#remainingMs - elapsed, 0)
    window.clearTimeout(this.#timerId)
    this.#timerId = null
  }

  // Resumes with whatever #remainingMs was left standing at, never the
  // full original #duration — this is the specific dangling-timer/
  // restart-not-resume bug this file's own self-review hand-traced (the
  // same class of check Tooltip.vue's commit performed for its own
  // hover-delay timer): hover, then leave before the *original* duration
  // would have elapsed, must resume with the remaining time, not restart
  // the full duration and not double-fire. Guarded by the #timerId ===
  // null check so an already-running timer is never clobbered/restarted
  // by a redundant resume call (e.g. focusin firing on top of an
  // already-hovering pointer).
  #resumeTimer() {
    if (this.#timerId !== null) return
    this.#startTimer(this.#remainingMs)
  }

  // Single chokepoint every pause/resume-triggering handler below calls
  // into: pause wins the moment any of the three reasons is true; resume
  // only once every one of them is false again.
  #updatePauseState() {
    const shouldPause = this.#pointerInside || this.#focusInside || this.#dragging
    if (shouldPause) {
      this.#pauseTimer()
    } else {
      this.#resumeTimer()
    }
  }

  #onPointerEnter = () => {
    this.#pointerInside = true
    this.#updatePauseState()
  }

  #onPointerLeave = () => {
    this.#pointerInside = false
    this.#updatePauseState()
  }

  // focusin/focusout (unlike focus/blur) bubble, so a single pair of
  // listeners on the root catches focus landing on/leaving any
  // descendant (e.g. the close button), not just the root itself.
  #onFocusIn = () => {
    this.#focusInside = true
    this.#updatePauseState()
  }

  #onFocusOut = () => {
    this.#focusInside = false
    this.#updatePauseState()
  }

  // --- Swipe-to-dismiss (single horizontal axis, both directions) ----

  #onPointerDown = (event) => {
    // Ignore secondary mouse buttons; a real primary click/tap always
    // reports button 0 (touch/pen pointerdown events report 0 too).
    if (event.button !== 0) return
    // Let the close button handle its own click normally — starting a
    // drag from it would capture the pointer and suppress the button's
    // own click event.
    if (event.target.closest && event.target.closest('.radix-toast-close')) return

    this.#dragStartX = event.clientX
    this.#dragging = true
    this.#updatePauseState()
    this.#root.setPointerCapture(event.pointerId)
    this.#root.setAttribute('data-swipe', 'start')
  }

  #onPointerMove = (event) => {
    if (!this.#dragging) return
    const deltaX = event.clientX - this.#dragStartX
    this.#root.setAttribute('data-swipe', 'move')
    this.#root.style.transform = 'translateX(' + deltaX + 'px)'
    this.#root.style.opacity = String(
      Math.max(1 - Math.abs(deltaX) / (RadixToast.SWIPE_THRESHOLD_PX * 3), 0.25)
    )
  }

  #onPointerUp = (event) => {
    if (!this.#dragging) return
    this.#dragging = false
    if (this.#root.hasPointerCapture(event.pointerId)) {
      this.#root.releasePointerCapture(event.pointerId)
    }

    const deltaX = event.clientX - this.#dragStartX
    if (Math.abs(deltaX) >= RadixToast.SWIPE_THRESHOLD_PX) {
      // Past the threshold: finish sliding fully off in the same
      // direction the drag was already headed, then dismiss.
      this.#root.setAttribute('data-swipe', 'end')
      const direction = deltaX < 0 ? -1 : 1
      this.#root.style.transform = 'translateX(' + direction * this.#root.offsetWidth + 'px)'
      this.#dismiss('swipe')
      return
    }

    // Under threshold: snap back and resume normal pause/resume state.
    this.#root.setAttribute('data-swipe', 'cancel')
    this.#root.style.transform = ''
    this.#root.style.opacity = ''
    this.#updatePauseState()
  }

  #onPointerCancel = (event) => {
    if (!this.#dragging) return
    this.#dragging = false
    if (this.#root.hasPointerCapture(event.pointerId)) {
      this.#root.releasePointerCapture(event.pointerId)
    }
    this.#root.setAttribute('data-swipe', 'cancel')
    this.#root.style.transform = ''
    this.#root.style.opacity = ''
    this.#updatePauseState()
  }

  // --- Manual dismiss button -----------------------------------------

  #onCloseClick = () => {
    this.#dismiss('button')
  }

  // --- Shared dismissal path -------------------------------------------
  // See this file's header comment, "Dismissal," for why this dispatches
  // a cancelable event before removing rather than doing only one or the
  // other.
  #dismiss(reason) {
    if (this.#dismissed) return
    this.#dismissed = true
    this.#clearTimer()

    const event = new CustomEvent('radix-toast-dismiss', {
      detail: { id: this.#root.id, reason },
      bubbles: true,
      cancelable: true,
    })
    const notCanceled = this.dispatchEvent(event)
    if (notCanceled) {
      this.remove()
    }
  }
}

customElements.define('radix-toast', RadixToast)
</script>
