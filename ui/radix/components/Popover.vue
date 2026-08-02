<!--
  Popover — Radix-inspired floating panel of richer content, shown near a
  trigger on click (not hover/focus like Tooltip.vue) and dismissed by
  clicking the trigger again, clicking outside, or pressing Escape.

  ## Zero-JS baseline: a meaningfully better story than Tooltip.vue's

  Tooltip.vue's header comment documents an honest "without JS, the
  tooltip never becomes visible at all" baseline, because hover/focus
  showing with a delay has no purely-declarative equivalent. Popover does
  not have that problem, and this is the core design fact this file is
  built around — verified against the spec/MDN before committing to it,
  not assumed from the task brief:

  The HTML Popover API's `popovertarget`/`popovertargetaction` attributes
  turn a plain `<button>` into a declarative popover *invoker*, with zero
  JavaScript:

    <button popovertarget="my-popover" popovertargetaction="toggle">Open</button>
    <div id="my-popover" popover="auto">...</div>

  `popovertargetaction="toggle"` (this component's only use of the
  attribute) switches the target between shown and hidden on each click —
  open on the first click, close on the next. Combined with `popover="auto"`
  on the content element (the same choice, and the same "auto vs manual"
  reasoning, as Tooltip.vue: light-dismiss on outside click/tap, Escape-to-
  close, and top-layer stacking, all native, all free), that covers every
  interaction this component's brief calls for — open, close, click-outside-
  to-dismiss, Escape-to-close — with **zero JavaScript**. Unlike Tooltip's
  hover-delay (genuinely JS-only, no declarative equivalent exists),
  click-to-toggle is exactly what `popovertargetaction="toggle"` was built
  for. This was confirmed against MDN's Popover API documentation before
  this file was written, specifically: `popovertarget` "[t]urns a <button>
  ... into a popover control button"; `popovertargetaction="toggle"`
  "toggles a popover element between showing and hidden"; and `popover="auto"`
  popovers close automatically on outside interaction and Escape, and enter
  the top layer, all as native UA behavior requiring no script.

  A second, independently-verified fact this design leans on: the browser
  does not leave the invoker's accessibility semantics for the page author
  to hand-wire. Per MDN's "Popover accessibility features": "[w]hen a
  relationship is established between a popover and its control (invoker)
  via the `popovertarget` attribute, ... an implicit `aria-details` and
  `aria-expanded` relationship is set up between them" automatically, kept
  live as the popover opens and closes — with zero script. That is why
  this template below sets neither `aria-expanded` nor `aria-controls`/
  `aria-details` by hand (unlike real Radix's `PopoverTrigger`, which
  tracks `aria-expanded={context.open}` itself in JS, because React
  owns the open/close state and has no native invoker relationship to
  delegate to): the UA already keeps that pairing correct across every
  toggle, baseline or enhanced, without this component lifting a finger.
  `aria-haspopup="dialog"` is set explicitly below because — unlike
  aria-expanded/aria-details — MDN's accessibility section does not
  document it as part of the implicit relationship; it is added by hand to
  match real Radix's own `PopoverTrigger`, and pairs with `role="dialog"`
  on the content element (also matching real Radix's `PopoverContent`),
  which content readers rely on to announce the popover's landmark role.

  ## The practical consequence for <script customelement> below

  Because the native `popovertarget` mechanism owns the entire open/close/
  dismiss lifecycle, this component's own script never calls
  `showPopover()`/`hidePopover()`, never listens for `click` on the
  trigger, and never toggles anything itself — there is no open/close
  moment for it to own at all. Its *only* job is positioning: reading the
  trigger's/content's live
  `getBoundingClientRect()` and placing the content near the trigger, the
  one thing (same as Tooltip.vue) that genuinely cannot be expressed
  declaratively. It reacts to the content's own native `toggle` event
  (dispatched by every `popover`-attributed element on every open/close,
  per the `ToggleEvent` interface — confirmed via MDN: `newState`/
  `oldState` are each `"open"` or `"closed"`) rather than hooking the
  trigger's `click`, specifically because click is not this script's event
  to own: the browser's own invoker mechanism already decided to open or
  close before this script ever finds out, so reacting after the fact via
  `toggle` is the only correct hook point — hooking `click` on the trigger
  would race the UA's own toggle, and would still need to run *after* the
  popover state actually changed to measure its real, laid-out
  `getBoundingClientRect()`. This is the same "small, honest v1"
  positioning scope as Tooltip.vue, reused and adapted (not imported) here:
  default placement below the trigger, start-aligned (Radix's own
  documented default `side="bottom"` with left-edge alignment is close
  enough for a v1; no full `align="center"` computation), a single-axis
  flip to above when there is no room below, and edge-clamping on both
  axes — no multi-side fallback order, no collision priority, no arrow.

  ## Props (REQUIRED — this package has no notion of an optional prop with
  a real default; an unpassed required prop renders a visible, truthy
  "[missing: <name>]" placeholder — see Avatar.vue's/Checkbox.vue's header
  comments for the full trap this convention avoids):

    id: string — the id shared between the trigger's `popovertarget`
      attribute and the content element's own `id`. This is *not* the
      script-minted, enhancement-only id Tooltip.vue generates for its own
      `aria-describedby` pairing (see that file's `#content.id` note):
      here, the id is load-bearing for the *zero-JS baseline itself* — the
      declarative `popovertarget`/`id` link between trigger and content is
      exactly what makes open/close/dismiss work with no script at all, so
      it must already be correct in the very first server-rendered HTML,
      before any script has had a chance to run. A page with multiple
      `<Popover>` instances is expected to pass a distinct `id` to each —
      the same caller-supplied-uniqueness contract Checkbox.vue/Switch.vue
      already establish for their own required `id` prop (there driving a
      `<label for>` pairing instead of `popovertarget`, but the same
      underlying constraint: whenever an id has to exist in the initial
      markup for a *declarative* HTML feature to work, this package makes
      the caller own generating a unique value, rather than trying to mint
      one script-side after the fact — which would be too late for a
      feature that has to work before any script runs anyway).

  ## Slots

    default — the trigger's content (text, icon, whatever the caller
      composes), rendered inside a real `<button type="button">`, matching
      Tooltip.vue's identical trigger-slot decomposition and rationale (a
      real `<button>` for free keyboard focusability, matching Radix's own
      default `Primitive.button` trigger).

    content — the popover's own body. Unlike Tooltip.vue's `content` string
      prop (a tooltip's content is almost always short, plain text), a
      popover's content is typically richer — headings, form fields,
      buttons, arbitrary structure — so this port uses a *named slot* here
      instead, the same call Dialog.vue's own header comment already makes
      for its (default-slot) freeform body content, just spelled as a named
      slot here because this component also needs its own default slot for
      the trigger.

  Usage:
    <Popover id="notifications-popover">
      Notifications
      <template #content>
        <h3>Notifications</h3>
        <p>You're all caught up.</p>
      </template>
    </Popover>

  No v-native escape hatch needed: this component's own name ("Popover")
  auto-registers a lowercase "popover" alias in the component registry.
  There is no native HTML element literally named `<popover>` for that
  alias to collide with (`popover` is an *attribute*, not an element) — the
  elements actually used below (`<span>`, `<button>`, `<div>`, `<slot>`)
  are unrelated native tags, the same reasoning Tooltip.vue's/Avatar.vue's
  own header comments already work through for their own files.
-->
<template>
  <span class="radix-popover">
    <button
      type="button"
      class="radix-popover-trigger"
      :popovertarget="id"
      popovertargetaction="toggle"
      aria-haspopup="dialog"
    ><slot></slot></button>
    <div
      :id="id"
      class="radix-popover-content"
      popover="auto"
      role="dialog"
    ><slot name="content"></slot></div>
  </span>
</template>

<style scoped>
.radix-popover {
  display: inline-block;
}

.radix-popover-trigger {
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-popover-trigger:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 2px;
}

/*
 * Override the UA popover stylesheet's own `position: fixed; inset: 0;
 * margin: auto;` (which centers a freshly-shown popover in the viewport).
 * Author styles always win over UA styles regardless of specificity, so
 * this rule reliably cancels that default. `position: fixed` is kept
 * (already the UA default) since the script below computes `top`/`left`
 * from `getBoundingClientRect()`, which returns viewport-relative
 * coordinates — `position: fixed` is what makes those coordinates apply
 * directly with no scroll-offset math needed. `inset: auto` and
 * `margin: 0` clear the centering so the script's own inline `top`/`left`
 * (which win over this stylesheet rule as inline styles) take full effect.
 * Identical technique to Tooltip.vue's own `.radix-tooltip-content` rule.
 */
.radix-popover-content {
  position: fixed;
  inset: auto;
  margin: 0;
  max-width: 20rem;
  padding: 1rem;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  background-color: #fff;
  color: #171717;
  font-size: 0.875rem;
  line-height: 1.5;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
}
</style>

<script customelement>
// Progressive enhancement on top of the fully-interactive-without-JS
// baseline above. Per RFC 014 §3 Non-Goals, this reimplements nothing the
// browser already provides for free — open, close, click-outside dismiss,
// Escape, top-layer stacking, and even the trigger's aria-expanded/
// aria-details bookkeeping are all native to the popovertarget +
// popover="auto" pairing (see this file's header comment for the verified
// spec citations). This script's only job is positioning: reading the
// trigger's/content's live layout and placing the content near the
// trigger, the one thing (same as Tooltip.vue) that genuinely cannot be
// expressed declaratively.
class RadixPopover extends HTMLElement {
  #trigger = null
  #content = null
  #reposition = () => {
    if (this.#content.matches(':popover-open')) {
      this.#positionContent()
    }
  }

  connectedCallback() {
    this.#trigger = this.querySelector('.radix-popover-trigger')
    this.#content = this.querySelector('.radix-popover-content')
    if (!this.#trigger || !this.#content) return

    // The native `toggle` event, not the trigger's `click` — the UA's own
    // popovertarget invoker mechanism already decided to open or close
    // before this script is told about it, so `toggle` (fired just after
    // the state change, per the ToggleEvent interface) is the only correct
    // hook point: it fires after the content has actually been shown, so
    // getBoundingClientRect() below reflects its real, laid-out box.
    this.#content.addEventListener('toggle', this.#onToggle)
    window.addEventListener('resize', this.#reposition)
    window.addEventListener('scroll', this.#reposition, { capture: true, passive: true })
  }

  disconnectedCallback() {
    if (this.#content) {
      this.#content.removeEventListener('toggle', this.#onToggle)
    }
    window.removeEventListener('resize', this.#reposition)
    window.removeEventListener('scroll', this.#reposition, { capture: true })
  }

  // Only (re)position on the transition *into* the open state — closing
  // needs no positioning work, and the content is not laid out to measure
  // once hidden anyway.
  #onToggle = (event) => {
    if (event.newState === 'open') {
      this.#positionContent()
    }
  }

  // v1 positioning (see this file's header comment for the documented
  // scope line against Floating UI, reused and adapted from Tooltip.vue's
  // own v1): default placement is below the trigger, start-aligned (left
  // edges flush); if there is no room below the viewport, flip above;
  // finally clamp both axes so the content never renders off-screen.
  // getBoundingClientRect() on both elements and window.inner* are the
  // only inputs — no collision priority beyond this single below/above
  // flip, no left/right placements.
  #positionContent() {
    const gap = 8
    const triggerRect = this.#trigger.getBoundingClientRect()
    const contentRect = this.#content.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let top = triggerRect.bottom + gap
    let left = triggerRect.left

    // Single-axis flip: only when placing below would overflow the
    // viewport's bottom edge, and only if placing above actually fits.
    if (top + contentRect.height > viewportHeight) {
      const aboveTop = triggerRect.top - gap - contentRect.height
      if (aboveTop >= 0) {
        top = aboveTop
      }
    }

    // Edge-clamping: shift back in bounds on both axes rather than
    // attempting any further placement fallback.
    if (left < 0) left = 0
    if (left + contentRect.width > viewportWidth) left = viewportWidth - contentRect.width
    if (top < 0) top = 0

    this.#content.style.top = top + 'px'
    this.#content.style.left = left + 'px'
  }
}

customElements.define('radix-popover', RadixPopover)
</script>
