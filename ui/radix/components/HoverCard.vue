<!--
  HoverCard — Radix-inspired floating preview card shown near a trigger on
  hover/focus, hidden otherwise. Architecturally this is Tooltip.vue's
  hover-delay + popover="auto" + scoped-down positioning approach, reused
  (not imported) here and adapted in two ways: richer slot-based content
  (this component's content is a profile card, not a short string) and a
  genuine close delay, which Tooltip.vue does not need.

  ## Zero-JS baseline

  Same category of degradation as Tooltip.vue, for the same reason: hover/
  focus-triggered showing with a delay has no purely-declarative HTML
  equivalent (an element carrying the `popover` attribute is hidden by
  default until `.showPopover()` is called, or a declarative `popovertarget`
  invoker fires it — and `popovertarget` has no concept of a delay). So,
  honestly: **without JS, this hover card never becomes visible at all.**
  The trigger still renders as ordinary, fully usable content (whatever the
  caller composes into the default slot, typically a link) — only the
  floating preview card is unreachable.

  ## `popover="auto"`, not `"manual"`

  Confirmed for this component specifically, not assumed by analogy: a
  HoverCard is the same interaction category as Tooltip — content shown on
  hover/focus, not on click, and expected to disappear the moment the user
  interacts elsewhere. Real Radix's own HoverCardContent wraps a
  DismissableLayer (closes on outside pointerdown/focus and Escape) and
  relies on Popper for positioning with no manual cross-instance
  coordination of its own. `popover="auto"` reproduces both of those for
  free: light-dismiss on outside click/tap plus Escape-to-close, and the
  browser's own "only one auto popover open at a time" top-layer rule
  matches a floating preview card that should not stack with an unrelated
  tooltip/popover/hover-card also left open elsewhere on the page.
  `manual` would require reimplementing both from scratch, as Tooltip.vue's
  own header comment already argues for that component; the same argument
  holds here.

  ## Props (REQUIRED — this package has no notion of an optional prop with a
  real default; see Avatar.vue's/Tooltip.vue's header comments for the full
  "[missing: <name>]" placeholder trap this convention avoids, and why the
  script-side fallbacks below are a defensive backstop, not a substitute for
  documenting these as required):

    openDelayMs: number — hover/focus-to-show delay, in milliseconds. Read
      from Radix's own source (`packages/react/hover-card/src/hover-card.tsx`,
      the `HoverCardProps` default `openDelay = 700`) rather than guessed —
      it happens to match Tooltip's own 700ms default, but the two are
      independently-documented defaults for different components, not one
      copied from the other. Falls back to the same 700 script-side
      (`Number(...) || 700`) whenever the `data-open-delay-ms` attribute is
      missing or not a valid number.
    closeDelayMs: number — hover/focus-out-to-hide delay, in milliseconds.
      Also read from Radix's own source (`closeDelay = 300` in that same
      `HoverCardProps` default), and this is the value that most
      distinguishes HoverCard from Tooltip.vue: Tooltip hides *immediately*
      on mouseleave/blur (see Tooltip.vue's `#onLeave`), because its content
      is short static text with nothing worth reaching for. A HoverCard's
      content is typically a rich, often-interactive preview (profile info,
      a link, a follow button) that the user may want to move the pointer
      into — hiding it the instant the pointer leaves the trigger would make
      that content practically unreachable by mouse. Falls back to 300
      script-side (`Number(...) || 300`) on a missing/invalid
      `data-close-delay-ms` attribute.

  ## Slots

    default — the trigger's content, rendered inside a real
      `<button type="button">`, matching Tooltip.vue's/Popover.vue's
      identical trigger-slot decomposition (a real `<button>` for free
      keyboard focusability). Real Radix's own `HoverCardTrigger` renders an
      `<a>` by default (hovering a link is the canonical HoverCard usage —
      a username or profile link that previews a card), but this port
      reuses Tooltip.vue's `<button>`-wrapped-slot shape rather than
      introducing a third trigger-element convention into this package;
      callers who want an actual link still compose an `<a>` as this slot's
      content (the same way Tooltip.vue's own header comment documents
      composing plain, non-interactive content into its trigger slot).

    content — the card's own body. Unlike Tooltip.vue's `content` string
      prop (a tooltip's content is almost always short, plain text), a
      HoverCard's content is typically much richer — an avatar, a name, a
      bio, stats, a follow button — so, matching Popover.vue's identical
      call for its own richer content, this port uses a named `#content`
      slot here instead of a string prop.

  ## `<script customelement>` (load-bearing)

  Same two irreducibly-JS jobs as Tooltip.vue: delay timing and positioning
  (see that file's header comment for why neither has a declarative
  equivalent). HoverCard adds a third: tracking *where the pointer is*
  across two separate elements (trigger and content) so that moving from
  one into the other does not trigger a close.

  ## The close-delay / "pointer entered content" behavior (read from Radix,
  not copied)

  Real Radix's `HoverCard` root wires the *same* `onOpen`/`onClose` handlers
  onto both `HoverCardTrigger` (`onPointerEnter`/`onPointerLeave`) and
  `HoverCardContent` (also `onPointerEnter`/`onPointerLeave`) — i.e. hovering
  the content itself is treated exactly like hovering the trigger: it cancels
  any pending close and (re)enters the open state, while leaving the content
  starts the same close-delay timer leaving the trigger would. This is the
  behavior this port replicates (independently reimplemented for this
  package's custom-element/popover-API model, not transcribed): both
  `#trigger` and `#content` get `mouseenter`/`mouseleave` (and `focus`/
  `blur` on the trigger) listeners routed through the *same* `#onEnter`/
  `#onLeave` handlers below. `#onEnter` always clears any pending close timer
  first — so entering the content while a close timer from leaving the
  trigger is still ticking cancels it, matching Radix's own
  `clearTimeout(openTimerRef.current)`-on-close /
  `clearTimeout(closeTimerRef.current)`-on-open symmetry. Concretely: trigger
  hover starts the open-delay timer; leaving the trigger for the content
  starts the close-delay timer; entering the content before that timer fires
  cancels it (via the shared `#onEnter` calling `#clearCloseTimer()`); only
  leaving *both* the trigger and the content, with no re-entry into either
  before the close-delay elapses, actually hides the card. Not ported: real
  Radix's additional guards against closing while text is being selected
  inside the content or while a pointer is held down on it
  (`hasSelectionRef`/`isPointerDownOnContentRef`) — a documented scope cut,
  the same "small, honest v1" spirit as Tooltip.vue's own documented cuts
  (no pointer grace-area polygon, no cross-instance skipDelayDuration).

  Not ported from Radix in this v1 (documented scope cuts, matching
  Tooltip.vue's/Popover.vue's own): full Floating UI collision detection
  (this file reuses Tooltip.vue's below-centered + single-axis-flip +
  edge-clamp v1 instead), the convex-hull pointer grace-area, and
  `skipDelayDuration`.

  Usage:
    <HoverCard :openDelayMs="700" :closeDelayMs="300">
      <a href="/users/kentcdodds">@kentcdodds</a>
      <template #content>
        <strong>Kent C. Dodds</strong>
        <p>Teaching people development skills.</p>
      </template>
    </HoverCard>

  No v-native escape hatch needed: this component's own name ("HoverCard")
  auto-registers a lowercase "hovercard" alias in the component registry.
  There is no native HTML element literally named `<hovercard>` for that
  alias to collide with — the elements actually used below (`<span>`,
  `<button>`, `<div>`, `<slot>`) are unrelated native tags, the same
  reasoning Tooltip.vue's/Popover.vue's/Avatar.vue's own header comments
  already work through for their own files.
-->
<template>
  <span class="radix-hover-card">
    <button v-native type="button" class="radix-hover-card-trigger"><slot></slot></button>
    <div
      class="radix-hover-card-content"
      popover="auto"
      :data-open-delay-ms="openDelayMs"
      :data-close-delay-ms="closeDelayMs"
    ><slot name="content"></slot></div>
  </span>
</template>

<style scoped>
.radix-hover-card {
  display: inline-block;
}

.radix-hover-card-trigger {
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-hover-card-trigger:focus-visible {
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
 * Identical technique to Tooltip.vue's/Popover.vue's own content rules.
 */
.radix-hover-card-content {
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
// Positioning logic (getBoundingClientRect() math, edge-clamp, below→above
// flip, resize/scroll re-position wiring) is intentionally near-identical
// to Popover.vue's/Tooltip.vue's own <script customelement> blocks,
// differing only in center-align vs left-align. This package has no
// mechanism for one component's script to import/reference another's (each
// <script customelement> block is compiled to its own standalone,
// content-hashed file — see customelement_collector.go / the README's
// "Custom Elements" section), so this is deliberate, tracked duplication,
// not undiscovered drift.
//
// Progressive enhancement on top of the never-visible-without-JS baseline
// above. Per RFC 014 §3 Non-Goals, this reimplements nothing the browser
// already provides for free (top-layer stacking, light-dismiss, Escape —
// all native to popover="auto") — it only adds what a popover attribute
// genuinely cannot express: hover/focus open-delay and close-delay timing
// (tracked across *both* the trigger and the content, see this file's
// header comment for why), and positioning the content relative to its
// trigger.
let nextContentId = 0

class RadixHoverCard extends HTMLElement {
  #trigger = null
  #content = null
  #openTimer = null
  #closeTimer = null
  #reposition = () => {
    if (this.#content.matches(':popover-open')) {
      this.#positionContent()
    }
  }

  connectedCallback() {
    this.#trigger = this.querySelector('.radix-hover-card-trigger')
    this.#content = this.querySelector('.radix-hover-card-content')
    if (!this.#trigger || !this.#content) return

    // Wire an aria-describedby relationship between the trigger and its
    // content, matching Tooltip.vue's identical script-minted-id approach.
    // A stable id can't be baked into the template at author time (multiple
    // <HoverCard> instances on one page would collide), so a fresh one is
    // minted here, once per connected instance.
    if (!this.#content.id) {
      this.#content.id = 'radix-hover-card-content-' + nextContentId++
    }
    this.#trigger.setAttribute('aria-describedby', this.#content.id)

    // Both the trigger AND the content get the same enter/leave handlers —
    // this is the load-bearing part of this component's design (see header
    // comment): hovering the content itself must behave exactly like
    // hovering the trigger, so that moving the pointer from one into the
    // other never closes the card.
    this.#trigger.addEventListener('mouseenter', this.#onEnter)
    this.#trigger.addEventListener('focus', this.#onEnter)
    this.#trigger.addEventListener('mouseleave', this.#onLeave)
    this.#trigger.addEventListener('blur', this.#onLeave)
    this.#trigger.addEventListener('keydown', this.#onKeyDown)
    this.#content.addEventListener('mouseenter', this.#onEnter)
    this.#content.addEventListener('mouseleave', this.#onLeave)
    this.#content.addEventListener('keydown', this.#onKeyDown)
    window.addEventListener('resize', this.#reposition)
    window.addEventListener('scroll', this.#reposition, { capture: true, passive: true })
  }

  disconnectedCallback() {
    this.#clearOpenTimer()
    this.#clearCloseTimer()
    if (this.#trigger) {
      this.#trigger.removeEventListener('mouseenter', this.#onEnter)
      this.#trigger.removeEventListener('focus', this.#onEnter)
      this.#trigger.removeEventListener('mouseleave', this.#onLeave)
      this.#trigger.removeEventListener('blur', this.#onLeave)
      this.#trigger.removeEventListener('keydown', this.#onKeyDown)
    }
    if (this.#content) {
      this.#content.removeEventListener('mouseenter', this.#onEnter)
      this.#content.removeEventListener('mouseleave', this.#onLeave)
      this.#content.removeEventListener('keydown', this.#onKeyDown)
    }
    window.removeEventListener('resize', this.#reposition)
    window.removeEventListener('scroll', this.#reposition, { capture: true })
  }

  #clearOpenTimer() {
    if (this.#openTimer !== null) {
      window.clearTimeout(this.#openTimer)
      this.#openTimer = null
    }
  }

  #clearCloseTimer() {
    if (this.#closeTimer !== null) {
      window.clearTimeout(this.#closeTimer)
      this.#closeTimer = null
    }
  }

  // Hover/focus entry — fires for the trigger's mouseenter/focus AND the
  // content's own mouseenter (see connectedCallback). Any pending close
  // timer is cancelled first: this is what makes moving the pointer from
  // the trigger into the content (or back again) never close the card, the
  // core behavior this component adds on top of Tooltip.vue. Restarting the
  // open timer on every entry (rather than only when no timer is pending)
  // keeps this correct even if mouseenter/focus both fire in quick
  // succession for the same interaction.
  #onEnter = () => {
    this.#clearCloseTimer()
    if (this.#content.matches(':popover-open')) return
    this.#clearOpenTimer()
    const openDelayMs = Number(this.#content.dataset.openDelayMs) || 700
    this.#openTimer = window.setTimeout(() => {
      this.#openTimer = null
      this.#show()
    }, openDelayMs)
  }

  // Hover/focus exit — fires for the trigger's mouseleave/blur AND the
  // content's own mouseleave. Cancels any pending open timer (the same
  // dangling-timeout fix Tooltip.vue's #onLeave documents) and, if the card
  // is open, starts the close-delay timer rather than hiding immediately —
  // giving the pointer time to re-enter either element (handled by
  // #onEnter above) before the card actually closes.
  #onLeave = () => {
    this.#clearOpenTimer()
    this.#clearCloseTimer()
    const closeDelayMs = Number(this.#content.dataset.closeDelayMs) || 300
    this.#closeTimer = window.setTimeout(() => {
      this.#closeTimer = null
      this.#hide()
    }, closeDelayMs)
  }

  #onKeyDown = (event) => {
    if (event.key === 'Escape') {
      this.#clearOpenTimer()
      this.#clearCloseTimer()
      this.#hide()
    }
  }

  #show() {
    if (!this.#content.matches(':popover-open')) {
      this.#content.showPopover()
    }
    this.#positionContent()
  }

  #hide() {
    if (this.#content.matches(':popover-open')) {
      this.#content.hidePopover()
    }
  }

  // v1 positioning (see this file's header comment for the documented scope
  // line against Floating UI): reused from Tooltip.vue's identical
  // below-centered default, single-axis flip, and edge-clamping — not
  // reimplemented from scratch.
  #positionContent() {
    const gap = 8
    const triggerRect = this.#trigger.getBoundingClientRect()
    const contentRect = this.#content.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let top = triggerRect.bottom + gap
    let left = triggerRect.left + (triggerRect.width - contentRect.width) / 2

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

customElements.define('radix-hover-card', RadixHoverCard)
</script>
