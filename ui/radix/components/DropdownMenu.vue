<!--
  DropdownMenu — Radix-inspired action menu, opened from a trigger button
  and closed by selecting an item, clicking outside, or pressing Escape.
  This file synthesizes two patterns this package already established
  separately, rather than inventing a third:

    - Popover.vue's zero-JS `popovertarget`/`popovertargetaction="toggle"`
      open/close/light-dismiss mechanism (see that file's header comment
      for the verified MDN citations this reuses as-is: `popovertarget`
      turns a `<button>` into a declarative popover invoker,
      `popovertargetaction="toggle"` flips it open/closed on each click,
      and `popover="auto"` gives outside-click dismiss, Escape-to-close,
      and top-layer stacking, all native, all free).
    - Toolbar.vue's `items: Array<{type, ...}>` + roving-tabindex +
      separator-skip arrow-key navigation (see that file's header comment
      for why this package's expression language forces a bounded,
      unrolled ternary chain for the static tabindex baseline, and why a
      plain modulo index walk is not enough once some entries are not
      focusable at all — both reused verbatim here, only the item-type
      name and the arrow-key axis differ).

  Applied to APG menu semantics (`role="menu"`/`role="menuitem"`) instead
  of `role="toolbar"`/plain buttons, and to `popovertarget` instead of
  Toolbar's plain always-visible group.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/menu/src/menu.tsx` and
  `packages/react/dropdown-menu/src/dropdown-menu.tsx` in the read-only
  clone. Three facts this file's design depends on, confirmed there rather
  than assumed:

    1. `DropdownMenuTrigger` renders `aria-haspopup="menu"` on its
       `<button>` (matched below) — unlike Popover.vue's
       `aria-haspopup="dialog"`, a menu is the ARIA-documented haspopup
       value for this component. No `aria-expanded`/`aria-controls` is
       hand-set here either, for the identical reason Popover.vue's
       template omits them: the browser sets up an implicit
       `aria-expanded`/`aria-details` relationship for any
       `popovertarget` invoker automatically (see Popover.vue's header
       comment for the MDN citation), so hand-setting a static value here
       would just go stale.
    2. `MenuContent` renders `role="menu"` with `aria-orientation="vertical"`
       hardcoded (menu.tsx: `role="menu" aria-orientation="vertical"` on
       the content, and `RovingFocusGroup.Root ... orientation="vertical"`
       one level up) — confirming menus are conventionally vertical, not a
       configurable axis the way Toolbar.vue's `orientation` prop is. This
       is why this file's roving-tabindex script (below) only wires
       ArrowUp/ArrowDown, with no horizontal variant and no `orientation`
       prop at all — a deliberate, verified simplification versus
       Toolbar.vue's two-axis design, not an oversight.
    3. `menu.tsx`'s own `FIRST_KEYS`/`LAST_KEYS` constants
       (`['ArrowDown', 'PageUp', 'Home']` / `['ArrowUp', 'PageDown', 'End']`)
       confirm Home/End are real, in-scope menu keys in upstream Radix, not
       a speculative addition — implemented below (PageUp/PageDown are not,
       a small additional v1 scope cut consistent with this package's
       "honest, bounded v1" convention elsewhere).

  ## Focus-on-open: first item, not "leave focus on the trigger"

  Per the WAI-ARIA APG Menu Button pattern this component (and upstream
  Radix) is built on, a menu itself is not normally part of the page's tab
  sequence — the documented convention is that opening the menu moves
  focus onto it, landing on its first item (Down Arrow/click) so that
  arrow keys work immediately with no extra keypress required. This file
  implements exactly that: `<script customelement>` below moves focus to
  the first non-disabled item whenever the content's native `toggle`
  event reports a transition to `"open"`.

  One real nuance from Radix's own source, read but *not* replicated here,
  documented as a deliberate v1 simplification: `menu.tsx`'s
  `RovingFocusGroup.Root` wires `onEntryFocus` to `event.preventDefault()`
  "only focus first item when using keyboard" (line ~546) — i.e. upstream
  Radix tracks pointer-vs-keyboard modality globally and only autofocuses
  the first item for a keyboard-triggered open, leaving a mouse-triggered
  open focused on the content wrapper itself instead. This package has no
  existing cross-component modality-tracking primitive (nothing in
  Popover.vue/HoverCard.vue/Toolbar.vue needs one), and the native
  `toggle` event this script hooks (the only correct hook point once
  `popovertarget` owns the open decision — see Popover.vue's header
  comment) does not itself carry that information. Given that, this port
  applies the simpler, still-documented APG convention uniformly —
  always focus the first non-disabled item on open, regardless of how the
  menu was opened — rather than leaving mouse-opened menus with no
  keyboard-reachable focus at all until an extra keypress. This is a
  legitimate, intentionally-scoped divergence from upstream's own
  modality-aware nuance, not a missed detail.

  ## `items[].type`: "item" and "separator"

  Matching Toolbar.vue's `items: Array<{type, ...}>` convention (see that
  file's header comment for why an `items`-array is used at all instead of
  composed child components — the same slot/context limitation applies
  here unchanged):
    - "item":      an actionable menu action — `id`, `label`, `disabled`.
                   Rendered as `<button role="menuitem">`.
    - "separator": a visual/semantic divider between groups of actions —
                   `role="separator"`, matching Separator.vue's/
                   Toolbar.vue's own contract. Unlike Toolbar.vue's
                   separator (which is drawn *perpendicular* to a
                   configurable `orientation`), a menu is always vertical
                   (see above), so its separator is always the horizontal-
                   divider case — and per Separator.vue's own documented
                   convention, "horizontal" is `aria-orientation`'s
                   ARIA-implicit default value for `role="separator"`, so
                   it is correctly omitted here unconditionally (no
                   `orientation`-driven ternary needed, unlike Toolbar.vue,
                   because there is only one possible case).

  Radix's real DropdownMenu also offers checkbox items, radio items,
  sub-menus, and labels. None of those are ported in this v1 — same
  "small, honest v1" scope discipline as Popover.vue's/HoverCard.vue's own
  documented cuts, and each is believed to be a self-contained, additive
  follow-up (a new `items[].type` branch) rather than blocked by anything
  in this file's structure.

  ## Props (REQUIRED — this package has no notion of an optional prop with
  a real default; see Popover.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):

    id:    string — shared between the trigger's `popovertarget` and the
           content element's own `id`, exactly Popover.vue's own `id` prop
           and rationale: this id is load-bearing for the zero-JS baseline
           itself (the declarative `popovertarget`/`id` link is what makes
           open/close/dismiss work with no script at all), so it must
           already be correct in the very first server-rendered HTML. A
           page with multiple `<DropdownMenu>` instances must pass a
           distinct `id` to each, the same caller-owns-uniqueness contract
           Popover.vue/Checkbox.vue/Switch.vue already establish.
    items: Array<{ id, label, disabled, type }> — see above.

  Slot:
    default — the trigger's content, rendered inside a real
      `<button type="button">`, matching Popover.vue's/HoverCard.vue's
      identical trigger-slot decomposition.

  ## Positioning: an intentional v1 scope cut, unlike Popover.vue/
  HoverCard.vue

  Those two files' own `<script customelement>` blocks exist *specifically*
  to compute `getBoundingClientRect()`-based placement next to their
  trigger — documented there as the one thing a `popover` attribute
  genuinely cannot express declaratively. This component's own script
  scope is menu keyboard semantics only:
  roving-tabindex navigation, activation, and open-focus management — not
  floating placement. So, deliberately, this file does not reuse Popover's
  positioning script, and does not override the UA popover stylesheet's
  default `position: fixed; inset: 0; margin: auto` centering the way
  Popover.vue's `.radix-popover-content` rule does. The practical
  consequence: this v1's menu content opens centered in the viewport
  rather than anchored below its trigger. This is an explicit, documented
  limitation, not a silent gap — porting Popover.vue's positioning
  approach onto this component is a believed-straightforward, additive
  follow-up, not attempted here to keep this component's script scope
  narrowly focused on menu keyboard semantics.

  ## `<script customelement>` (load-bearing, not just polish)

  1. Roving-tabindex baseline recompute + arrow-key (Up/Down) navigation
     among items, skipping separators/disabled entries — Toolbar.vue's
     exact `#findFocusable` walk (bounded to `#items.length` steps, wraps
     at both ends via `(index + step + n) % n`), reused unchanged except
     for the fixed vertical axis (no `orientation`-conditional key choice,
     see above) and the `radix-dropdown-menu-menuitem` class name it
     filters on instead of `radix-toolbar-button`.
  2. Home/End jump to the first/last non-disabled item, reusing the exact
     same "search from just before index 0 / just after the last index"
     technique Toolbar.vue's own header comment documents.
  3. Activation: a `click` listener on the content element (delegated, so
     it also fires for a real `<button>`'s own native Enter/Space
     activation behavior — a disabled `<button>` never dispatches `click`
     at all per the HTML spec, exactly the reasoning Toggle.vue's header
     comment already works through for its own click-based design, so no
     extra `disabled` guard is needed here either) dispatches a
     `radix-dropdown-menu-select` `CustomEvent` whose `detail.id` is read
     from the *specific* activated item's own `data-id` attribute — not a
     generic "something was clicked" signal, mirroring Toggle.vue's
     `radix-toggle-change` event-dispatch precedent — and then calls
     `.hidePopover()` to close the menu.
  4. On the content's native `toggle` event transitioning to `"open"`,
     moves focus to the first non-disabled item (see "Focus-on-open"
     above) — the same `toggle`-not-`click` hook-point reasoning
     Popover.vue's header comment already establishes (the native
     `popovertarget` invoker mechanism decides open/close before this
     script is told about it, so reacting to the content's own `toggle`
     event is the only correct hook, and it fires only after the content
     is actually shown).

  Usage:
    <DropdownMenu
      id="file-menu"
      :items="[
        { type: 'item', id: 'new', label: 'New File' },
        { type: 'item', id: 'open', label: 'Open...' },
        { type: 'separator' },
        { type: 'item', id: 'close', label: 'Close', disabled: true },
      ]"
    >
      File
    </DropdownMenu>

  No v-native escape hatch needed: this component's own name
  ("DropdownMenu") auto-registers a lowercase "dropdownmenu" alias in the
  component registry. There is no native HTML element literally named
  `<dropdownmenu>` for that alias to collide with — the elements actually
  used below (`<span>`, `<button>`, `<div>`, `<slot>`) are unrelated native
  tags, the same reasoning Popover.vue's/HoverCard.vue's own header
  comments already work through for their own files.

  ## Custom-element tag name

  This file's own base name, "DropdownMenu.vue", derives to
  `radix-dropdown-menu` under the standard `Mount{Prefix: "radix"}` this
  package assumes — see radix.go's header comment for the derivation
  algorithm.
-->
<template>
  <span class="radix-dropdown-menu">
    <button
      v-native
      type="button"
      class="radix-dropdown-menu-trigger"
      :popovertarget="id"
      popovertargetaction="toggle"
      aria-haspopup="menu"
    ><slot></slot></button>
    <div
      :id="id"
      class="radix-dropdown-menu-content"
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
          class="radix-dropdown-menu-item radix-dropdown-menu-menuitem"
          :data-id="item.id"
          :disabled="item.disabled"
          :tabindex="index === (items.length > 0 && items[0].type === 'item' && !items[0].disabled ? 0 : items.length > 1 && items[1].type === 'item' && !items[1].disabled ? 1 : items.length > 2 && items[2].type === 'item' && !items[2].disabled ? 2 : items.length > 3 && items[3].type === 'item' && !items[3].disabled ? 3 : items.length > 4 && items[4].type === 'item' && !items[4].disabled ? 4 : items.length > 5 && items[5].type === 'item' && !items[5].disabled ? 5 : -1) ? '0' : '-1'"
        >{{ item.label }}</button>
        <div
          v-else-if="item.type === 'separator'"
          class="radix-dropdown-menu-item radix-dropdown-menu-separator"
          role="separator"
        ></div>
      </template>
    </div>
  </span>
</template>

<style>
/*
 * ui/radix design tokens — mirrors real Radix Colors (@radix-ui/colors,
 * gray/blue/red scales, light mode, fetched from the published package's
 * own CSS source, not guessed) and Radix Themes' default "medium" radius
 * scale / scaling=1 spacing scale (fetched from
 * @radix-ui/themes/.../tokens/{radius,space}.css). Prefixed --radix-*
 * (unlike upstream's own unprefixed --gray-1/--blue-9/--radius-3/--space-4)
 * because this is a mounted library, not an app shell: an unprefixed
 * custom property would leak into, and could collide with, a consuming
 * project's own global CSS custom properties of the same generic name.
 *
 * Every component in this package includes this exact block, verbatim, in
 * its own unscoped <style> section. htmlc's StyleCollector dedupes style
 * contributions by exact (scope, CSS text) match (see style.go's
 * StyleCollector.Add) — global/unscoped contributions use an empty scope,
 * so two components emitting byte-identical unscoped CSS text collapse
 * into a single entry in the final page's stylesheet, regardless of how
 * many of the 30+ components in this package a given page actually mounts.
 * This is the mechanism, not an assumption: confirmed by reading
 * style.go's Add directly before adopting this pattern.
 *
 * This block is the single canonical source of truth (edit it here, in
 * Button.vue, first — see radix.go's own header comment, "Design
 * tokens", for the cross-reference) — every other component file's own
 * copy must then be updated to match exactly, since the dedup only
 * collapses text that is byte-identical everywhere it appears. Space
 * tokens are defined for completeness/future use; not yet adopted for
 * every padding/margin/gap declaration across this package (a deliberate
 * v1 scope cut).
 */
:root {
  /* Radix Colors, gray scale, light mode */
  --radix-gray-1: #fcfcfc;
  --radix-gray-2: #f9f9f9;
  --radix-gray-3: #f0f0f0;
  --radix-gray-4: #e8e8e8;
  --radix-gray-5: #e0e0e0;
  --radix-gray-6: #d9d9d9;
  --radix-gray-7: #cecece;
  --radix-gray-8: #bbbbbb;
  --radix-gray-9: #8d8d8d;
  --radix-gray-10: #838383;
  --radix-gray-11: #646464;
  --radix-gray-12: #202020;

  /* Radix Colors, blue scale, light mode */
  --radix-blue-1: #fbfdff;
  --radix-blue-2: #f4faff;
  --radix-blue-3: #e6f4fe;
  --radix-blue-4: #d5efff;
  --radix-blue-5: #c2e5ff;
  --radix-blue-6: #acd8fc;
  --radix-blue-7: #8ec8f6;
  --radix-blue-8: #5eb1ef;
  --radix-blue-9: #0090ff;
  --radix-blue-10: #0588f0;
  --radix-blue-11: #0d74ce;
  --radix-blue-12: #113264;

  /* Radix Colors, red scale, light mode */
  --radix-red-1: #fffcfc;
  --radix-red-2: #fff7f7;
  --radix-red-3: #feebec;
  --radix-red-4: #ffdbdc;
  --radix-red-5: #ffcdce;
  --radix-red-6: #fdbdbe;
  --radix-red-7: #f4a9aa;
  --radix-red-8: #eb8e90;
  --radix-red-9: #e5484d;
  --radix-red-10: #dc3e42;
  --radix-red-11: #ce2c31;
  --radix-red-12: #641723;

  /* Radix Themes radius scale, radius-factor=1 ("medium"), scaling=1 */
  --radix-radius-1: 3px;
  --radix-radius-2: 4px;
  --radix-radius-3: 6px;
  --radix-radius-4: 8px;
  --radix-radius-5: 12px;
  --radix-radius-6: 16px;
  --radix-radius-thumb: 9999px;

  /* Radix Themes spacing scale, scaling=1 */
  --radix-space-1: 4px;
  --radix-space-2: 8px;
  --radix-space-3: 12px;
  --radix-space-4: 16px;
  --radix-space-5: 24px;
  --radix-space-6: 32px;
  --radix-space-7: 40px;
  --radix-space-8: 48px;
  --radix-space-9: 64px;
}

/*
 * Shared visually-hidden-input technique (was previously repeated
 * verbatim, under a different class name, in Checkbox.vue/RadioGroup.vue/
 * Select.vue/Switch.vue/Tabs.vue — same dedup mechanism as above). A
 * component using this pairs it as a *second* class alongside its own
 * component-specific class (e.g. class="radix-checkbox-input
 * radix-visually-hidden-input"), so this class only ever contributes the
 * hiding declarations; every compound/sibling selector a component needs
 * (:checked + .foo, :focus-visible + .foo, etc.) keeps using its own
 * specific class, unaffected by this one.
 */
.radix-visually-hidden-input {
  position: absolute;
  opacity: 0;
  width: 1px;
  height: 1px;
  margin: -1px;
  padding: 0;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}

.radix-dropdown-menu {
  display: inline-block;
}

.radix-dropdown-menu-trigger {
  font: inherit;
  color: inherit;
  background-color: #fff;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
  padding: 0.4rem 0.7rem;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-dropdown-menu-trigger:focus-visible,
.radix-dropdown-menu-menuitem:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

/*
 * See this file's header comment ("Positioning: an intentional v1 scope
 * cut") — unlike Popover.vue's/HoverCard.vue's own `.radix-*-content`
 * rules, this deliberately does NOT override the UA popover stylesheet's
 * default centering, so no `position`/`inset`/`margin` overrides appear
 * here.
 */
.radix-dropdown-menu-content {
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
 * `display` is gated on `:popover-open`, not set unconditionally above: an
 * author-level `display` declaration on a `[popover]` element — at any
 * specificity — wins the cascade over the UA stylesheet's own
 * `[popover]:not(:popover-open) { display: none; }` rule, because author
 * origin beats UA origin regardless of selector specificity. An
 * unconditional `display: flex` here would therefore make this menu
 * render permanently visible, even in its closed, zero-JS baseline state
 * — found and fixed via an actual rendered screenshot showing exactly
 * that, not caught by markup-only inspection.
 */
.radix-dropdown-menu-content:popover-open {
  display: flex;
}

.radix-dropdown-menu-menuitem {
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

.radix-dropdown-menu-menuitem:hover:not(:disabled) {
  background-color: var(--radix-gray-3);
}

.radix-dropdown-menu-menuitem:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-dropdown-menu-separator {
  background-color: currentColor;
  opacity: 0.2;
  width: 100%;
  height: 1px;
  flex-shrink: 0;
  margin: 0.25rem 0;
}
</style>

<script customelement>
// Progressive enhancement on top of the zero-JS popovertarget baseline
// above (open/close/light-dismiss are all native, per Popover.vue's header
// comment). Per RFC 014 §3 Non-Goals, this reimplements nothing the
// browser already provides for free — what genuinely has no native
// equivalent, and is therefore this script's actual job, is: roving-
// tabindex arrow-key navigation across menu items (a plain <button> and
// <div role="separator"> have no native cross-sibling roving-tabindex
// behavior, the same conclusion Toolbar.vue's own header comment reaches),
// moving focus onto the menu's first item when it opens (see this file's
// header comment's "Focus-on-open" section for the documented APG
// convention this follows), and dispatching an unambiguous activation
// event carrying which specific item was selected.
class RadixDropdownMenu extends HTMLElement {
  #content = null
  #items = [] // every item element (menuitems AND separators), in DOM order
  #focusable = [] // subset of #items: enabled menuitem buttons only, in DOM order

  connectedCallback() {
    this.#content = this.querySelector('.radix-dropdown-menu-content')
    if (!this.#content) return

    this.#items = Array.from(this.querySelectorAll('.radix-dropdown-menu-item'))
    this.#refreshFocusable()

    // Recompute the tab stop from the live DOM unconditionally, the same
    // genuine correctness fix (not just a style refresh) Toolbar.vue's own
    // connectedCallback documents for `items` arrays longer than the
    // static baseline's bound.
    this.#syncTabindex()

    this.#content.addEventListener('toggle', this.#onToggle)
    this.#content.addEventListener('click', this.#onClick)
    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    if (this.#content) {
      this.#content.removeEventListener('toggle', this.#onToggle)
      this.#content.removeEventListener('click', this.#onClick)
    }
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #refreshFocusable() {
    this.#focusable = this.#items.filter((el) => el.classList.contains('radix-dropdown-menu-menuitem') && !el.disabled)
  }

  #syncTabindex() {
    this.#focusable.forEach((el, i) => el.setAttribute('tabindex', i === 0 ? '0' : '-1'))
  }

  // Moves focus onto the first non-disabled item whenever the content
  // transitions to open — see this file's header comment's "Focus-on-open"
  // section for why this applies uniformly (not gated on pointer-vs-
  // keyboard modality, a deliberate, documented simplification of
  // upstream Radix's own nuance there). No action on close: the native
  // popovertarget/popover machinery already owns returning focus to the
  // trigger, the same "nothing to do here" conclusion Popover.vue's own
  // #onToggle reaches for its own (positioning-only) close case.
  #onToggle = (event) => {
    if (event.newState !== 'open') return
    this.#refreshFocusable()
    this.#syncTabindex()
    if (this.#focusable.length === 0) return
    this.#moveFocus(this.#items.indexOf(this.#focusable[0]))
  }

  // Delegated click listener on the content element: fires for a real
  // mouse click AND for a real <button>'s own native Enter/Space
  // activation (both dispatch 'click' per the HTML spec) — so this one
  // handler covers "Enter/Space on a menu item: activate it" with no
  // separate keydown-based activation logic needed. No `disabled` guard
  // is needed either: a disabled
  // <button> never dispatches 'click' at all, the exact reasoning
  // Toggle.vue's header comment already works through for its own
  // click-based design.
  #onClick = (event) => {
    const item = event.target.closest('.radix-dropdown-menu-menuitem')
    // Ignore clicks from anywhere else, and ignore menuitems belonging to
    // a nested radix-dropdown-menu, if any (mirrors Toolbar.vue's own
    // closest(...) !== this guard).
    if (!item || item.closest('radix-dropdown-menu') !== this) return

    // `detail.id` is read from this *specific* activated item's own
    // data-id attribute, not a generic "something was clicked" signal —
    // this is what lets a listener unambiguously know which item fired,
    // mirroring Toggle.vue's own `radix-toggle-change` event-dispatch
    // precedent (detail carrying the specific new state, not just "it was
    // clicked").
    this.dispatchEvent(
      new CustomEvent('radix-dropdown-menu-select', {
        detail: { id: item.dataset.id },
        bubbles: true,
      })
    )
    this.#content.hidePopover()
  }

  #onKeydown = (event) => {
    const item = event.target.closest('.radix-dropdown-menu-menuitem')
    if (!item || item.closest('radix-dropdown-menu') !== this) return

    const index = this.#items.indexOf(item)
    if (index === -1) return

    let target
    switch (event.key) {
      case 'ArrowDown': target = this.#findFocusable(index, 1); break
      case 'ArrowUp': target = this.#findFocusable(index, -1); break
      // Search forward from just before the first item / backward from
      // just after the last item, so these naturally resolve to the
      // first/last *focusable* item rather than the first/last item —
      // the exact technique Toolbar.vue's own header comment documents.
      case 'Home': target = this.#findFocusable(-1, 1); break
      case 'End': target = this.#findFocusable(this.#items.length, -1); break
      default: return
    }
    if (target === -1) return

    event.preventDefault()
    this.#moveFocus(target)
  }

  // Walks #items (menuitems AND separators) one step at a time in `step`
  // direction, wrapping at both ends — Toolbar.vue's own
  // `#findFocusable`, reused unchanged (same wrap formula
  // `(index + step + n) % n`, same "keep stepping past anything not an
  // enabled menuitem instead of stopping on whatever index is next"
  // logic, same bound of at most #items.length steps so a menu with no
  // focusable item at all terminates instead of looping forever).
  #findFocusable(fromIndex, step) {
    const n = this.#items.length
    if (n === 0) return -1
    let index = fromIndex
    for (let i = 0; i < n; i++) {
      index = (index + step + n) % n
      const el = this.#items[index]
      if (el.classList.contains('radix-dropdown-menu-menuitem') && !el.disabled) return index
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

customElements.define('radix-dropdown-menu', RadixDropdownMenu)
</script>
