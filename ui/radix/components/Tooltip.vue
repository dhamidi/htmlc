<!--
  Tooltip — Radix-inspired floating label shown near a trigger on
  hover/focus, hidden otherwise.

  This is the first component in this series built on genuine dynamic
  positioning rather than native document flow (Dialog's <dialog>,
  Collapsible's <details>, Accordion/Tabs' native controls). Real Radix
  computes tooltip placement with Floating UI, a full-featured positioning
  engine with flip/shift/collision-detection across every side and a
  configurable priority order. Reimplementing that from scratch is a large,
  separate library's worth of work and explicitly out of scope for this
  port (see this commit's brief) — this file instead ships a small, honest
  v1: fixed below-centered placement, plus a single-axis flip (below → above
  when there's no room below) and edge-clamping on both axes. There is no
  multi-side fallback order, no collision priority, and no arrow/pointer
  element. Document, don't hide, that scope line.

  ## Zero-JS baseline

  The content element below carries the native `popover` *attribute*
  (`popover="auto"`, see the "auto vs manual" note further down). Per the
  Popover API spec, an element with the `popover` attribute is hidden by
  default (`display: none` via the UA stylesheet) until something calls
  `.showPopover()` on it, or a declarative `popovertarget` invoker button
  triggers it. This component uses neither a declarative invoker (its
  trigger needs a *delay*, which the Invoker Commands/popovertarget API has
  no concept of) nor any CSS-only reveal — hover/focus-triggered showing
  with a delay is inherently a JS-driven interaction pattern with no
  purely-declarative equivalent. So, honestly: **without JS, this tooltip
  never becomes visible at all.** The trigger still renders as ordinary,
  fully usable content (whatever the caller composes into the default
  slot) — only the floating label itself is unreachable. This is the same
  category of degradation Toggle.vue's header comment documents for its own
  inert-without-JS baseline, and unlike Dialog's `open` attribute (which
  *can* express "start open" declaratively), there is no equivalent
  "start shown" lever for a hover-triggered tooltip to fall back on.

  ## `popover="auto"`, not `"manual"`

  The two popover types differ in exactly the ways that matter here:
    - `auto`: light-dismisses on outside click/tap and on Escape, and
      opening one `auto` popover closes any other currently-open `auto`
      popover automatically (the browser's own "top layer" group
      semantics) — all with zero JS.
    - `manual`: no light-dismiss of any kind; the page's own script is
      solely responsible for every close, and multiple `manual` popovers
      can be open at once with no coordination between them.
  A tooltip should disappear the moment the user interacts elsewhere, and
  real Radix explicitly closes a tooltip on the trigger's own `pointerdown`
  (clicking a tooltip's trigger is not "using" the tooltip) — behavior
  `auto`'s light-dismiss-on-outside-click reproduces for free, since the
  trigger itself is not a declared popover invoker and so counts as
  "outside" for light-dismiss purposes. Real Radix also explicitly closes
  any previously-open tooltip when a new one opens (`document.dispatchEvent(
  new CustomEvent('tooltip.open'))`, listened for by every mounted
  tooltip's content) — `auto`'s built-in "only one auto popover open at a
  time" rule gives this the same outcome without this component having to
  hand-roll that coordination itself. `manual` would require reimplementing
  both of those from scratch; `auto` is the correct choice here.

  ## Props (both REQUIRED — this package has no notion of an optional prop
  with a real default; see Avatar.vue's header comment for the full
  "[missing: <name>]" placeholder trap this convention avoids, and why a
  script-side fallback below is a defensive backstop, not a substitute for
  documenting these as required):

    content: string — the tooltip's text. A tooltip's content is almost
      always short, plain text (unlike Dialog's/Collapsible's freeform
      body content, which justified a slot), so this port uses a simple
      string prop here rather than a named `#content` slot — one fewer
      thing for a caller to wire up for the common case. This is a
      deliberate departure from Radix's own composable
      `<Tooltip.Content>{children}</Tooltip.Content>`, documented here as
      this port's own call per the brief, not an oversight.
    delayMs: number — hover/focus-to-show delay, in milliseconds. Radix's
      own documented default (`DEFAULT_DELAY_DURATION` in
      TooltipProvider) is 700ms; this port uses the same 700ms as the
      script-side fallback the `data-delay-ms` attribute reader below
      falls back to whenever the attribute is missing or not a valid
      number (`Number(...) || 700`, the same shape as Avatar.vue's
      `Number(...) || 0` — see that file's header comment for why `??`
      cannot be used here). Pass 0 for "show immediately, no delay".

  ## Slot

    default — the trigger's content (text, icon, whatever the caller
    composes), rendered inside a real `<button type="button">`. A `<button>`
    is used rather than an arbitrary wrapper so the trigger is keyboard-
    focusable out of the box with zero extra work, matching Radix's own
    default `Primitive.button` trigger element.

  ## No v-native escape hatch needed

  This component's own name ("Tooltip") auto-registers a lowercase
  "tooltip" alias in the component registry. There is no native HTML
  element literally named `<tooltip>` for that alias to collide with — the
  elements actually used below (`<span>`, `<button>`, `<div>`, `<slot>`)
  are unrelated native tags, the same reasoning Avatar.vue's header comment
  works through for its own file.

  ## <script customelement> (load-bearing)

  Positioning math genuinely cannot be expressed declaratively — there is
  no CSS-only way to read a trigger's live `getBoundingClientRect()` and
  clamp against the viewport — so, like Dialog's showModal() sequencing,
  this is JS doing something markup alone cannot. See the script block's
  own comments for: the show/hide delay timers (and the dangling-timeout
  fix — clearing a pending "show" timer the moment the pointer leaves
  before it fires, a real and common bug in hover-delay implementations),
  the positioning formula, and the resize/scroll recompute.

  Not ported from Radix in this v1 (documented scope cuts, not oversights):
  the pointer "grace area" that keeps a tooltip open while the cursor
  travels from trigger to content (a convex-hull point-in-polygon
  algorithm in the real source) is skipped — this v1's content is not
  independently hoverable, matching Radix's own `disableHoverableContent`
  branch. `TooltipProvider`'s cross-tooltip `skipDelayDuration` (re-hovering
  a *different* trigger shortly after closing one skips its own delay) is
  also skipped — every trigger here always waits its own full `delayMs`.

  Usage:
    <Tooltip content="Saved to your library" :delayMs="700">Save</Tooltip>

  The default slot fills this component's own `<button>` trigger element —
  it is not a separate wrapper around a caller-supplied `<button>` of its
  own. Passing another interactive element (a nested `<button>`, an `<a>`,
  etc.) as slot content would render invalid nested-interactive-content
  HTML (a real bug this port's own scratch-render self-check caught before
  this commit, the same kind of check this codebase's process consistently
  runs before landing a component). Compose an icon, text, or a plain
  `<span>` here instead.
-->
<template>
  <span class="radix-tooltip">
    <button type="button" class="radix-tooltip-trigger"><slot></slot></button>
    <div
      class="radix-tooltip-content"
      popover="auto"
      role="tooltip"
      :data-delay-ms="delayMs"
    >{{ content }}</div>
  </span>
</template>

<style scoped>
.radix-tooltip {
  display: inline-block;
}

.radix-tooltip-trigger {
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-tooltip-trigger:focus-visible {
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
 */
.radix-tooltip-content {
  position: fixed;
  inset: auto;
  margin: 0;
  max-width: 20rem;
  padding: 0.35rem 0.6rem;
  border: none;
  border-radius: 6px;
  background-color: #171717;
  color: #fff;
  font-size: 0.8125rem;
  line-height: 1.4;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
}
</style>

<script customelement>
// Positioning logic (getBoundingClientRect() math, edge-clamp, below→above
// flip, resize/scroll re-position wiring) is intentionally near-identical
// to Popover.vue's/HoverCard.vue's own <script customelement> blocks,
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
// all native to popover="auto") — it only adds the two things a popover
// attribute genuinely cannot: hover/focus delay timing, and positioning
// the content relative to its trigger.
let nextContentId = 0

class RadixTooltip extends HTMLElement {
  #trigger = null
  #content = null
  #openTimer = null
  #reposition = () => {
    if (this.#content.matches(':popover-open')) {
      this.#positionContent()
    }
  }

  connectedCallback() {
    this.#trigger = this.querySelector('.radix-tooltip-trigger')
    this.#content = this.querySelector('.radix-tooltip-content')
    if (!this.#trigger || !this.#content) return

    // Wire an aria-describedby relationship between the trigger and its
    // content. A stable id can't be baked into the template at author
    // time (multiple <Tooltip> instances on one page would collide), so a
    // fresh one is minted here, once per connected instance.
    if (!this.#content.id) {
      this.#content.id = 'radix-tooltip-content-' + nextContentId++
    }
    this.#trigger.setAttribute('aria-describedby', this.#content.id)

    this.#trigger.addEventListener('mouseenter', this.#onEnter)
    this.#trigger.addEventListener('focus', this.#onEnter)
    this.#trigger.addEventListener('mouseleave', this.#onLeave)
    this.#trigger.addEventListener('blur', this.#onLeave)
    this.#trigger.addEventListener('keydown', this.#onKeyDown)
    window.addEventListener('resize', this.#reposition)
    window.addEventListener('scroll', this.#reposition, { capture: true, passive: true })
  }

  disconnectedCallback() {
    this.#clearOpenTimer()
    if (this.#trigger) {
      this.#trigger.removeEventListener('mouseenter', this.#onEnter)
      this.#trigger.removeEventListener('focus', this.#onEnter)
      this.#trigger.removeEventListener('mouseleave', this.#onLeave)
      this.#trigger.removeEventListener('blur', this.#onLeave)
      this.#trigger.removeEventListener('keydown', this.#onKeyDown)
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

  // Hover/focus entry: (re)start the show delay. Restarting on every entry
  // — rather than only when no timer is pending — keeps this correct even
  // if mouseenter/focus both fire in quick succession for the same
  // interaction (e.g. a mouse click that also moves focus).
  #onEnter = () => {
    this.#clearOpenTimer()
    const delayMs = Number(this.#content.dataset.delayMs) || 700
    this.#openTimer = window.setTimeout(() => {
      this.#openTimer = null
      this.#show()
    }, delayMs)
  }

  // Hover/focus exit: this is the fix for the classic dangling-timeout bug
  // in hover-delay implementations — if the pointer/focus leaves *before*
  // the delay above elapses, the pending timer must be cancelled here, or
  // it fires later and shows a tooltip whose trigger is no longer
  // hovered/focused at all. Also hides the tooltip if it was already open.
  #onLeave = () => {
    this.#clearOpenTimer()
    this.#hide()
  }

  #onKeyDown = (event) => {
    if (event.key === 'Escape') {
      this.#clearOpenTimer()
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

  // v1 positioning (see this file's header comment for the documented
  // scope line against Floating UI): default placement is below-centered
  // on the trigger; if there is no room below the viewport, flip above;
  // finally clamp both axes so the content never renders off-screen.
  // `getBoundingClientRect()` on both elements and `window.inner*` are the
  // only inputs — no collision priority beyond this single below/above
  // flip, no left/right placements.
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

customElements.define('radix-tooltip', RadixTooltip)
</script>
