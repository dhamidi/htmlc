<!--
  ContextMenu — Radix-inspired action menu, opened by right-clicking (or
  a platform-equivalent long-press, not ported — see below) inside a
  trigger region, and closed by selecting an item, clicking outside, or
  pressing Escape. Shares its role="menu"/"menuitem" internals — item/
  separator structure, roving-tabindex arrow-key navigation, activation —
  almost entirely with DropdownMenu.vue's own script. Rewritten fresh
  here (not copy-pasted across files), the logic matches, because both
  components render the exact same underlying menu semantics.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/context-menu/src/context-menu.tsx`
  in the read-only clone. Two facts this file's design depends on,
  confirmed there rather than assumed:

    1. `ContextMenuContent` renders `<MenuPrimitive.Content ... side="right"
       sideOffset={2} align="start" />` — the *same* `MenuPrimitive.Content`
       component `DropdownMenuContent` renders (confirmed in
       DropdownMenu.vue's own header comment, which already read
       `menu.tsx`'s `role="menu" aria-orientation="vertical"` hardcoding on
       that shared primitive). So ContextMenu's menu-internals really are
       the same `role="menu"` structure as DropdownMenu.vue's, not merely
       "similar" — the two components share one content primitive upstream.
       This file's `side`/`sideOffset`/`align` popper props are Floating-UI-
       specific configuration with no equivalent in this package's own v1
       positioning approach (see "Positioning" below) and are not ported.
    2. `ContextMenuTrigger` renders a plain `Primitive.span` (not a
       `<button>`), and sets no `aria-haspopup` at all — unlike
       `DropdownMenuTrigger`'s `<button aria-haspopup="menu">`. This makes
       sense once the underlying interaction is understood: a "menu
       button" (APG's own term) is a widget you operate directly — you
       focus it, then activate it — so it earns `aria-haspopup` and lives
       in the tab order. A context-menu trigger region is not that kind of
       widget; it is an arbitrary area of already-existing page content
       (an image, a list row, a whole panel) that *incidentally* also
       responds to a right-click. It has no independent focusable/
       activatable identity of its own to annotate. This file follows that
       distinction: the trigger below is a plain, non-interactive `<div>`
       wrapping the caller's slotted content, with no `tabindex`,
       `role`, or `aria-haspopup` added.

  ## The key difference from DropdownMenu.vue: no zero-JS baseline for
  opening

  Every popovertarget-based component in this package so far (Popover.vue,
  DropdownMenu.vue, etc.) opens with zero JavaScript:
  `popovertarget`/`popovertargetaction="toggle"` turn a
  `<button>` into a declarative popover invoker, verified via MDN in
  Popover.vue's own header comment. There is no equivalent declarative
  attribute for "open this popover on **right-click** instead of left-
  click" — `contextmenu` is a script-only DOM event with no HTML attribute
  surface at all (unlike `click`, which `popovertarget` intercepts). So,
  honestly, unlike every prior file in this series: **opening this menu
  genuinely requires JavaScript from the very first interaction.** Without
  JS, the `<script customelement>` below never runs, no listener is ever
  attached to the trigger region, and right-clicking inside it does
  nothing special at all — the browser's own native context menu simply
  appears, exactly as it would over any other page content. That is a
  completely reasonable, non-broken degradation, not "nothing happens":
  the trigger region's slotted content is still fully present, still fully
  usable, and a context menu (the browser's own) is still available on it
  — just not *this* component's custom one. This is documented here
  honestly, the same way Tooltip.vue's header comment documents its own
  "without JS, never becomes visible at all" baseline for a different
  reason (hover-delay has no declarative equivalent either).

  ## Props (REQUIRED — this package has no notion of an optional prop with
  a real default; see Popover.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):

    items: Array<{ id, label, disabled, type }> — identical shape to
      DropdownMenu.vue's own `items` prop; see that file's header comment
      for the "item"/"separator" contract, reused unchanged here.

  Slot:
    default — the trigger region's content: whatever the caller wants a
      right-click to act on (an image, a card, a table row, a whole
      panel). Rendered inside `<div class="radix-context-menu-trigger">`,
      not a `<button>` — see verified fact 2 above for why a plain,
      non-interactive wrapper is correct here, unlike every other trigger
      in this package.

  Unlike DropdownMenu.vue's/Popover.vue's `id` prop, this component takes
  no `id` at all. Those props exist because a declarative `popovertarget`/
  `id` pair is what makes their zero-JS baseline work — the link has to be
  correct in the very first server-rendered HTML, before any script runs.
  This component has no such declarative link to begin with (see above):
  its `<script customelement>` locates its own trigger/content via
  `querySelector` on class name, the same way it locates menu items, and
  drives `.showPopover()`/`.hidePopover()` directly by element reference.
  An `id` prop here would have nothing load-bearing to attach to, so, per
  this package's "no vestigial required props" discipline, it is omitted.

  ## `popover="auto"`, verified for the script-driven-open case specifically

  Every other file in this package that chose `popover="auto"` did so for
  a popover opened via a declarative `popovertarget` invoker. This
  component's content is instead opened by this file's own script calling
  `.showPopover()` directly — a meaningfully different invocation path, so
  the "auto vs manual" choice was re-verified for *this* case rather than
  assumed to carry over unchanged. Per MDN's "Using the Popover API" guide
  (fetched and read before writing this file): light-dismiss-on-outside-
  click and Escape-to-close are described as inherent behaviors of the
  `auto` popover *state* itself ("The popover can be 'light dismissed' —
  this means that you can hide the popover by clicking outside it. The
  popover can also be closed, using browser-specific mechanisms such as
  pressing the Esc key.") — properties of the element's `popover="auto"`
  attribute, not of whichever mechanism happened to call `.showPopover()`.
  Nothing in that behavior is gated on "was this shown by a
  `popovertarget` invoker": an `auto` popover shown programmatically still
  light-dismisses and still closes on Escape. So `popover="auto"` is the
  correct choice here too, for the same underlying reason (outside-click
  dismiss and Escape-to-close, all native, all free) — verified, not
  assumed, for this component's script-first invocation path.

  That correct choice has one genuinely nasty consequence specific to this
  component, found via a real trusted-input regression (right-click via CDP
  `Input.dispatchMouseEvent({button: 'right'})`, both `mousePressed` and
  `mouseReleased`, against a live examples/radix-demo instance — not
  reasoned from the spec) and root-caused with a standalone `<dialog>`-free
  popover repro before being fixed: **the right-click gesture that opens
  this menu also light-dismisses it**, on the same gesture's own trailing
  pointerup/mouseup. `Popover.vue`'s/`DropdownMenu.vue`'s declarative
  `popovertarget` trigger is recognized by the browser as the popover's own
  associated invoker and is exempted from counting as "outside" for that
  same click — but this component opens imperatively via `.showPopover()`
  from `#onContextMenu`, so there is no such invoker association, and
  Chromium (HeadlessChrome/151.0.7922.71, the engine available in this
  environment) cannot tell "the gesture that opened this" from "a
  subsequent click outside it". The repro isolated the actual trigger for
  the dismiss: it is not about the right-click itself, and stopping the
  `contextmenu` event's propagation, or intercepting the trailing
  `pointerup`/`mouseup` with `preventDefault()`/`stopPropagation()`, made no
  difference. What did matter was timing: with positioning deferred to the
  content's own `toggle` event (as this file originally did, and as
  Popover.vue's/Tooltip.vue's own `#onToggle` still correctly does for
  their non-racing invocation paths), the content is still sitting at its
  unpositioned default location (`getBoundingClientRect()` reads all-zero)
  at the moment the same gesture's trailing pointerup/mouseup is processed
  — so that pointerup lands nowhere near the popover's (not yet real)
  bounding box, reads as "outside", and light-dismisses it, immediately
  followed by the deferred `toggle`-driven positioning that never gets to
  matter. Fix, verified the same way: `#onContextMenu` now also does a
  synchronous, un-clamped "rough" placement of the content at the cursor
  point *before* calling `.showPopover()` (see the code below and
  "Positioning" below) — by the time the browser processes the trailing
  pointerup/mouseup, the content already occupies real, cursor-adjacent
  screen space, so that pointerup reads as landing inside it, not outside,
  and the light-dismiss never fires. `#onToggle`'s existing `toggle`-driven
  call to `#positionAtPoint()` still runs afterward and still owns the
  real, edge-clamped final position — the rough placement only has to
  survive the one trailing pointerup, not be pixel-accurate. Honest
  trade-off: because the rough placement is intentionally un-clamped (real
  size isn't known synchronously — see "Positioning" below), right-clicking
  very close to the viewport's right or bottom edge can show the content
  spilling slightly off-screen for the single frame between `showPopover()`
  and the `toggle`-driven clamp correcting it, rather than being clamped
  from the very first paint. Not a functional regression (nothing is
  unusable or misclicked), and not verified to be visually perceptible at
  normal frame rates, but recorded here rather than glossed over.

  One additional, genuinely different consequence of driving `showPopover()`
  from script here (absent from every popovertarget-based file, where the
  browser's own invoker mechanism guarantees a clean toggle): per the HTML
  Standard's "check popover validity" algorithm (fetched and read before
  writing this file), calling `.showPopover()` on a popover element whose
  visibility state is already "showing" is a silent no-op — it does not
  throw, and it does not do anything. That matters because right-clicking
  a *second* spot while this menu is already open must still reposition it
  at the new cursor location; a bare `.showPopover()` call would silently
  fail to do that. `#onContextMenu` below therefore explicitly
  `.hidePopover()`s first when already open, forcing a fresh open (and
  with it, a fresh `toggle` event, which is what actually repositions the
  content — see "Positioning" below). This is a deliberate, simpler v1
  substitute for real Radix's own finer-grained behavior (re-anchoring its
  virtual anchor point without closing at all, read in
  `context-menu.tsx`'s `ContextMenuTrigger`, not replicated here) — a
  visible close-then-reopen flash on re-trigger, traded for not having to
  reimplement Radix's virtual-anchor machinery in this v1.

  ## Positioning: cursor-anchored, not trigger-rect-anchored

  Tooltip.vue/Popover.vue compute placement from the *trigger's*
  `getBoundingClientRect()` — a fixed rectangle that does not move once
  rendered. This component has no such fixed rectangle to anchor to: the
  whole point of a context menu is that it opens whichever specific point
  the user right-clicked, which can be anywhere inside a large trigger
  region and differs on every open. So the anchor here is a **point**
  (`event.clientX`/`event.clientY`, captured in `#onContextMenu`), not a
  rect — a meaningfully different positioning basis, not a copy-paste of
  Popover.vue's/Tooltip.vue's trigger-rect math with the serial numbers
  filed off. The same "small, honest v1" scope discipline still applies
  though, reused from those two files: no Floating-UI-equivalent
  multi-side fallback order, no collision priority, no arrow — just the
  cursor point as the content's natural top-left corner, clamped back in
  bounds on both axes if the menu's real, laid-out size would overflow the
  viewport's right or bottom edge.

  This is done in two phases, not one — see the `popover="auto"` section
  above for the real, verified bug this two-phase split fixes. Phase 1: a
  synchronous, un-clamped placement of the content's top-left corner at the
  cursor point, done in `#onContextMenu` itself, before `.showPopover()` is
  even called — this is what has to exist before the opening gesture's own
  trailing pointerup/mouseup is processed, or Chromium light-dismisses the
  popover this same handler just opened (see `popover="auto"` section).
  Phase 2: the real, edge-clamped final placement, computed in
  `#positionAtPoint()` and applied on the content's own `toggle` event (not
  synchronously in `#onContextMenu` — this part is unchanged from before
  the fix, for the identical reason Popover.vue's header comment documents:
  the content has to actually be shown and laid out before
  `getBoundingClientRect()` reflects its real size, which phase 1's rough
  placement does not need because it does no clamping).

  ## Not ported in this v1 (documented scope cuts, not oversights)

  - Long-press-to-open on touch/pen (real Radix's `ContextMenuTrigger`
    wires a 700ms `pointerdown` timer via `whenTouchOrPen`, read in
    `context-menu.tsx`, not replicated here) — this v1 is a `contextmenu`-
    only trigger, matching the "each new `items[].type`/interaction mode
    is a self-contained, additive follow-up" discipline DropdownMenu.vue's
    own header comment already establishes for its own cuts.
  - Precisely because long-press is not ported: this file deliberately
    does **not** set `-webkit-touch-callout: none` on the trigger (the CSS
    property real Radix's `ContextMenuTrigger` does set, to suppress iOS's
    own native long-press callout menu). Real Radix can safely suppress it
    because it *also* implements the long-press-to-open handler that
    replaces it. This port implements no such handler, so disabling the
    native iOS callout here would leave a touch user with **no** path to
    any context menu at all on that trigger — neither this component's
    (no long-press support) nor the platform's own (suppressed) — an
    actual regression, not a harmless copy of upstream's CSS. Left alone,
    iOS's native long-press menu still works over this trigger, matching
    this file's own "the native context menu is a reasonable fallback"
    story above.
  - Disabled trigger state, checkbox/radio items, sub-menus, and labels —
    the same "small, honest v1" cuts DropdownMenu.vue's own header comment
    already documents for its identical `items[].type` contract.

  ## `<script customelement>` (load-bearing for the open interaction
  itself, unlike DropdownMenu.vue's script, which only adds keyboard
  navigation on top of an already-fully-working `popovertarget` toggle)

  1. `contextmenu` on the trigger region: `event.preventDefault()`
     unconditionally suppresses the native browser menu, the pointer
     position is captured, and the content is opened via `.showPopover()`
     (closing it first if already open — see above).
  2. On the content's native `toggle` event transitioning to `"open"`:
     position the content at the captured cursor point with viewport
     edge-clamping (see "Positioning" above), then move focus to the first
     non-disabled item — the identical APG Menu Button "focus lands on the
     first item when the menu opens" convention DropdownMenu.vue's header
     comment documents and applies uniformly (not gated on pointer-vs-
     keyboard modality), reused here for the same reason: this package has
     no cross-component modality-tracking primitive to gate it on.
  3. Roving-tabindex arrow-key (Up/Down, Home/End) navigation among items,
     skipping separators/disabled entries, and click-delegated activation
     dispatching a `radix-context-menu-select` `CustomEvent` (detail.id
     from the activated item's own `data-id`) followed by
     `.hidePopover()` — the same logic as DropdownMenu.vue's own script,
     rewritten fresh for this file's own class names and custom-element
     tag, not copy-pasted across files.

  Usage:
    <ContextMenu
      :items="[
        { type: 'item', id: 'copy', label: 'Copy' },
        { type: 'item', id: 'paste', label: 'Paste' },
        { type: 'separator' },
        { type: 'item', id: 'delete', label: 'Delete', disabled: true },
      ]"
    >
      <div class="card">Right-click me</div>
    </ContextMenu>

  No v-native escape hatch needed: this component's own name
  ("ContextMenu") auto-registers a lowercase "contextmenu" alias in the
  component registry. There is no native HTML element literally named
  `<contextmenu>` (the old, long-removed HTML5 `<menu type="context">`
  companion tag was dropped from the living standard years ago and was
  never itself a bare `<contextmenu>` element) for that alias to collide
  with — the elements actually used below (`<div>`, `<button>`, `<slot>`)
  are unrelated native tags, the same reasoning DropdownMenu.vue's/
  Popover.vue's own header comments already work through for their own
  files.

  ## Custom-element tag name

  This file's own base name, "ContextMenu.vue", derives to
  `radix-context-menu` under the standard `Mount{Prefix: "radix"}` this
  package assumes — see radix.go's header comment for the derivation
  algorithm.
-->
<template>
  <Tokens></Tokens>
  <div class="radix-context-menu">
    <div class="radix-context-menu-trigger"><slot></slot></div>
    <div
      class="radix-context-menu-content"
      popover="auto"
      role="menu"
      aria-orientation="vertical"
    >
      <template v-for="(item, index) in items">
        <button
          v-native
          v-if="item.type === 'item'"
          type="button"
          role="menuitem"
          class="radix-context-menu-item radix-context-menu-menuitem"
          :data-id="item.id"
          :disabled="item.disabled"
          :tabindex="index === (items.length > 0 && items[0].type === 'item' && !items[0].disabled ? 0 : items.length > 1 && items[1].type === 'item' && !items[1].disabled ? 1 : items.length > 2 && items[2].type === 'item' && !items[2].disabled ? 2 : items.length > 3 && items[3].type === 'item' && !items[3].disabled ? 3 : items.length > 4 && items[4].type === 'item' && !items[4].disabled ? 4 : items.length > 5 && items[5].type === 'item' && !items[5].disabled ? 5 : -1) ? '0' : '-1'"
        >{{ item.label }}</button>
        <div
          v-else-if="item.type === 'separator'"
          class="radix-context-menu-item radix-context-menu-separator"
          role="separator"
        ></div>
      </template>
    </div>
  </div>
</template>

<style>
.radix-context-menu {
  display: block;
}

/*
 * A plain, non-interactive wrapper — no cursor/hover affordance styling
 * here, unlike every button-based trigger in this package. Right-clicking
 * is not a "this looks clickable" left-click affordance; the trigger
 * region is whatever the caller composes into the default slot, styled by
 * the caller, not by this component.
 */
.radix-context-menu-trigger {
  display: block;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-context-menu-menuitem:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

/*
 * Override the UA popover stylesheet's own `position: fixed; inset: 0;
 * margin: auto;` (which centers a freshly-shown popover in the viewport) —
 * the identical technique Popover.vue's/Tooltip.vue's own `.radix-*-content`
 * rules use, `position: fixed` kept so the script's `getBoundingClientRect()`-
 * derived, viewport-relative `top`/`left` apply directly with no scroll-
 * offset math needed.
 */
.radix-context-menu-content {
  position: fixed;
  inset: auto;
  margin: 0;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 10rem;
  padding: 0.35rem;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-4);
  background-color: #fff;
  color: var(--radix-gray-12);
  font-size: 0.875rem;
  line-height: 1.5;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
}

/*
 * `display` is gated on `:popover-open`, not set unconditionally above:
 * an author-level `display` declaration on a `[popover]` element — at any
 * specificity — wins the cascade over the UA stylesheet's own
 * `[popover]:not(:popover-open) { display: none; }` rule, because author
 * origin beats UA origin regardless of selector specificity. An
 * unconditional `display: flex` here would therefore make this menu
 * render permanently visible, even in its closed, zero-JS baseline state
 * — found and fixed via an actual rendered screenshot showing exactly
 * that, not caught by markup-only inspection.
 */
.radix-context-menu-content:popover-open {
  display: flex;
}

.radix-context-menu-menuitem {
  display: block;
  width: 100%;
  text-align: left;
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  border-radius: var(--radix-radius-2);
  padding: 0.4rem 0.6rem;
  cursor: pointer;
}

.radix-context-menu-menuitem:hover:not(:disabled) {
  background-color: var(--radix-gray-3);
}

.radix-context-menu-menuitem:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-context-menu-separator {
  background-color: currentColor;
  opacity: 0.2;
  width: 100%;
  height: 1px;
  flex-shrink: 0;
  margin: 0.25rem 0;
}
</style>

<script customelement>
// Unlike DropdownMenu.vue's script — pure progressive enhancement layered
// on an already-fully-working popovertarget toggle — this script is
// load-bearing for the open interaction itself: there is no declarative
// equivalent for "open this popover on right-click" (see this file's
// header comment), so without this script running, right-clicking the
// trigger region does nothing but show the browser's own native menu.
// What this script owns: suppressing that native menu and opening this
// one instead, positioning it at the cursor with edge-clamping, and — once
// open — the identical roving-tabindex/activation/close-on-select menu
// semantics DropdownMenu.vue's own script implements (rewritten fresh
// here, not copy-pasted, because the underlying menu structure really is
// the same shared MenuPrimitive.Content upstream — see this file's header
// comment's verified-facts section).
class RadixContextMenu extends HTMLElement {
  #trigger = null
  #content = null
  #items = [] // every item element (menuitems AND separators), in DOM order
  #focusable = [] // subset of #items: enabled menuitem buttons only, in DOM order
  #point = { x: 0, y: 0 } // last captured cursor position, viewport-relative

  connectedCallback() {
    this.#trigger = this.querySelector('.radix-context-menu-trigger')
    this.#content = this.querySelector('.radix-context-menu-content')
    if (!this.#trigger || !this.#content) return

    this.#items = Array.from(this.querySelectorAll('.radix-context-menu-item'))
    this.#refreshFocusable()

    // Recompute the tab stop from the live DOM unconditionally, the same
    // genuine correctness fix Toolbar.vue's/DropdownMenu.vue's own
    // connectedCallback documents for `items` arrays longer than the
    // static baseline's bound.
    this.#syncTabindex()

    this.#trigger.addEventListener('contextmenu', this.#onContextMenu)
    this.#content.addEventListener('toggle', this.#onToggle)
    this.#content.addEventListener('click', this.#onClick)
    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    if (this.#trigger) {
      this.#trigger.removeEventListener('contextmenu', this.#onContextMenu)
    }
    if (this.#content) {
      this.#content.removeEventListener('toggle', this.#onToggle)
      this.#content.removeEventListener('click', this.#onClick)
    }
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #refreshFocusable() {
    this.#focusable = this.#items.filter((el) => el.classList.contains('radix-context-menu-menuitem') && !el.disabled)
  }

  #syncTabindex() {
    this.#focusable.forEach((el, i) => el.setAttribute('tabindex', i === 0 ? '0' : '-1'))
  }

  // Suppresses the browser's native context menu and opens this one
  // instead. event.preventDefault() is the very first, unconditional
  // statement in this handler — every contextmenu event delivered to the
  // trigger region reaches it before anything else runs, so there is no
  // code path here that lets the native menu leak through.
  //
  // Also does a synchronous, un-clamped "rough" placement of the content at
  // the cursor point BEFORE calling showPopover() — see this file's header
  // comment's "Positioning"/`popover="auto"` sections for why this matters:
  // without it, the right-click gesture's own trailing pointerup/mouseup
  // immediately light-dismisses the popover this same handler just opened.
  // #onToggle below still re-positions with real edge-clamping once the
  // content is actually laid out; this rough placement only has to survive
  // long enough for that trailing pointerup, not be pixel-perfect.
  #onContextMenu = (event) => {
    event.preventDefault()
    this.#point = { x: event.clientX, y: event.clientY }

    // Re-triggering at a new spot while already open: per the HTML
    // Standard's "check popover validity" algorithm (verified before
    // writing this file, see header comment), calling showPopover() on a
    // popover that is already showing is a silent no-op — it would not
    // reposition anything. Close first so the showPopover() call below
    // always starts a fresh open (and fires a fresh 'toggle' event, which
    // is what actually repositions the content — see #onToggle).
    if (this.#content.matches(':popover-open')) {
      this.#content.hidePopover()
    }
    // Rough placement, synchronous and un-clamped (real size isn't known
    // yet — see header comment). This alone is what keeps the menu open
    // across the opening gesture's own trailing pointerup; see #onToggle
    // for the real, edge-clamped placement. Offset 4px up-and-left of the
    // exact cursor point, not placed AT it: verified empirically (via
    // document.elementFromPoint() probed in a small pixel grid around the
    // cursor, against the real popover's rendered box) that positioning
    // the content's top-left corner exactly at the cursor coordinate is
    // not enough — Chromium's real hit-testing of the freshly-shown
    // top-layer popover was found to disagree with its own
    // getBoundingClientRect() by 1-2px right where a fractional-pixel
    // cursor coordinate (the ordinary case; window.innerWidth/Height-scaled
    // event.clientX/Y are rarely whole numbers) lands exactly on the box's
    // reported edge, so the cursor point tests as just outside the popover
    // and the trailing pointerup light-dismisses it — the same bug this
    // whole placement exists to fix, just moved a couple of pixels rather
    // than fixed. A 4px inward offset comfortably clears that margin.
    const roughOffsetPx = 4
    this.#content.style.left = (this.#point.x - roughOffsetPx) + 'px'
    this.#content.style.top = (this.#point.y - roughOffsetPx) + 'px'
    this.#content.showPopover()
  }

  // Positions the content and moves focus onto the first item whenever it
  // transitions to open. Reacting to the content's own 'toggle' event
  // (not doing this synchronously inside #onContextMenu) matters here for
  // the same reason it matters in Popover.vue's #onToggle: the content
  // has to actually be shown and laid out before getBoundingClientRect()
  // in #positionAtPoint reflects its real size — this call to
  // #positionAtPoint() is what turns #onContextMenu's rough, un-clamped
  // placement above into the real, edge-clamped final position. No action
  // on close: the popover machinery already owns returning focus to
  // wherever it was before the menu opened.
  #onToggle = (event) => {
    if (event.newState !== 'open') return
    this.#positionAtPoint()
    this.#refreshFocusable()
    this.#syncTabindex()
    if (this.#focusable.length === 0) return
    this.#moveFocus(this.#items.indexOf(this.#focusable[0]))
  }

  // Cursor-anchored positioning (see this file's header comment's
  // "Positioning" section for why this differs from Popover.vue's/
  // Tooltip.vue's trigger-rect-anchored math): the captured cursor point
  // is the content's natural top-left corner, clamped back in bounds on
  // both axes if the content's real, laid-out size would overflow the
  // viewport's right or bottom edge — the same "shift back in bounds
  // rather than a further placement fallback" discipline as Popover.vue's/
  // Tooltip.vue's own #positionContent, with no side-flip step (a cursor
  // point, unlike a trigger rect, has no "other side" to flip to).
  #positionAtPoint() {
    const contentRect = this.#content.getBoundingClientRect()
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let left = this.#point.x
    let top = this.#point.y

    if (left + contentRect.width > viewportWidth) left = viewportWidth - contentRect.width
    if (top + contentRect.height > viewportHeight) top = viewportHeight - contentRect.height
    if (left < 0) left = 0
    if (top < 0) top = 0

    this.#content.style.left = left + 'px'
    this.#content.style.top = top + 'px'
  }

  // Delegated click listener on the content element: fires for a real
  // mouse click AND for a real <button>'s own native Enter/Space
  // activation (both dispatch 'click' per the HTML spec) — identical
  // reasoning to DropdownMenu.vue's own #onClick. No `disabled` guard
  // needed either: a disabled <button> never dispatches 'click' at all.
  #onClick = (event) => {
    const item = event.target.closest('.radix-context-menu-menuitem')
    // Ignore clicks from anywhere else, and ignore menuitems belonging to
    // a nested radix-context-menu, if any.
    if (!item || item.closest('radix-context-menu') !== this) return

    this.dispatchEvent(
      new CustomEvent('radix-context-menu-select', {
        detail: { id: item.dataset.id },
        bubbles: true,
      })
    )
    this.#content.hidePopover()
  }

  #onKeydown = (event) => {
    const item = event.target.closest('.radix-context-menu-menuitem')
    if (!item || item.closest('radix-context-menu') !== this) return

    const index = this.#items.indexOf(item)
    if (index === -1) return

    let target
    switch (event.key) {
      case 'ArrowDown': target = this.#findFocusable(index, 1); break
      case 'ArrowUp': target = this.#findFocusable(index, -1); break
      case 'Home': target = this.#findFocusable(-1, 1); break
      case 'End': target = this.#findFocusable(this.#items.length, -1); break
      default: return
    }
    if (target === -1) return

    event.preventDefault()
    this.#moveFocus(target)
  }

  // Walks #items (menuitems AND separators) one step at a time in `step`
  // direction, wrapping at both ends — identical to DropdownMenu.vue's own
  // #findFocusable (same wrap formula, same "keep stepping past anything
  // not an enabled menuitem" logic, same bound of at most #items.length
  // steps).
  #findFocusable(fromIndex, step) {
    const n = this.#items.length
    if (n === 0) return -1
    let index = fromIndex
    for (let i = 0; i < n; i++) {
      index = (index + step + n) % n
      const el = this.#items[index]
      if (el.classList.contains('radix-context-menu-menuitem') && !el.disabled) return index
    }
    return -1
  }

  #moveFocus(index) {
    this.#focusable.forEach((el) => el.setAttribute('tabindex', '-1'))
    const el = this.#items[index]
    el.setAttribute('tabindex', '0')
    el.focus()
  }
}

customElements.define('radix-context-menu', RadixContextMenu)
</script>
