<!--
  NavigationMenu — Radix-inspired site-navigation bar: a row of top-level
  items that are either plain links or flyout triggers opening a panel of
  arbitrary rich content, with roving-tabindex across the top-level row.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/navigation-menu/src/navigation-menu.tsx`
  in the read-only clone. This file's single most important verified fact —
  the one the whole design brief hinges on — is that NavigationMenu's real
  ARIA contract is *not* DropdownMenu.vue's/Menubar.vue's menu-button
  pattern. Grepping `navigation-menu.tsx` line by line for every `role`,
  `aria-haspopup`, and JSX element it actually renders turns up:

    1. Root (`NavigationMenu`): renders `<Primitive.nav aria-label="Main"
       data-orientation dir ... />` — a real `<nav>` element. No `role` prop
       is passed anywhere. Confirms the design brief's suspicion: this
       relies on `<nav>`'s own implicit ARIA landmark role ("navigation",
       per the HTML-ARIA mapping every modern browser/AT already applies to
       a bare `<nav>` element) rather than a hand-set `role="navigation"` —
       the same "don't fight/duplicate native semantics" discipline
       Separator.vue's own header comment documents for its later
       role="separator"-vs-aria-hidden correction. This file's `<nav>`
       below carries no `role` attribute for the same reason.
    2. `NavigationMenuList`: renders `<Primitive.ul>` — no role at all
       (native `<ul>`); items are addressed as a normal HTML list, not a
       `role="menu"` tree. This file's `<ul>`/`<li>` below match that.
    3. `NavigationMenuTrigger`: renders `<Primitive.button id disabled
       data-disabled data-state aria-expanded aria-controls ... />`. Two
       things confirmed absent by direct inspection: **no `aria-haspopup`
       at all** (unlike `DropdownMenuTrigger`'s `aria-haspopup="menu"` —
       DropdownMenu.vue's own header comment already documents that one; a
       nav trigger is a plain disclosure button, not a menu-button per
       ARIA's own taxonomy) and **no `role="menuitem"`**. `aria-expanded`/
       `aria-controls` *are* real, hand-managed React state upstream — but
       this port reuses Popover.vue's/DropdownMenu.vue's own already-
       established, already-verified reasoning for why a `popovertarget`-
       based port omits them anyway: a `popovertarget` invoker gets an
       implicit `aria-expanded`/`aria-details` relationship to its target
       from the browser automatically (see either of those files' header
       comments for the full citation), so hand-setting a static value
       here would just go stale the instant the popover opens or closes.
    4. `NavigationMenuContent`/`NavigationMenuContentImpl`: renders a
       `DismissableLayer` with only `id`/`aria-labelledby`/
       `data-orientation`/`data-motion` — **no `role` at all**, in
       particular no `role="menu"`. `aria-labelledby={triggerId}` *is* a
       real, verified relationship upstream (content labelled by its own
       trigger's id) — genuinely useful for a panel of arbitrary rich
       content, and reproduced below via a derived id
       (`item.id + '-trigger'` on the trigger, `aria-labelledby` on the
       content), the same derived-id-pair convention Accordion.vue's own
       header comment already establishes for its own
       summary/panel `-summary`/`-panel` ids.
    5. `NavigationMenuLink`: renders `<Primitive.a data-active aria-current
       ... />` — a plain anchor, **no `role="menuitem"`**. Confirms plain
       top-level links need no menu-button-family ARIA at all, exactly the
       design brief's expectation.

  Net conclusion, verified rather than assumed: this component's real ARIA
  contract is built on `nav`/list/link semantics — a `<nav>` (implicit
  landmark role, no explicit `role`), a plain `<ul>`/`<li>` list, plain
  `<a>` links, and disclosure `<button>`s with no `aria-haspopup`/
  `role="menuitem"` at all — genuinely different from DropdownMenu.vue's/
  Menubar.vue's `role="menu"`/`role="menuitem"`/`aria-haspopup="menu"`
  menu-button pattern, not a stylistic variant of it. Nothing in this file
  carries over that pattern.

  ## Props (REQUIRED — this package has no notion of an optional prop with
  a real default; see Popover.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):

    items: Array<{ id, label, href, content }> —
      - id:      stable, unique-within-this-instance identifier. Shared
                 between the flyout content's own `id`/`popovertarget`
                 value (DropdownMenu.vue's own `id`-prop contract, applied
                 per item here) and, suffixed `-trigger`, the flyout
                 trigger button's own `id` (used only for the content's
                 `aria-labelledby`, per fact 4 above — Accordion.vue's own
                 derived-id-pair convention). Not otherwise read for a
                 plain-link item.
      - label:   the visible top-level text, for either a link or a
                 trigger button.
      - href:    a plain link's destination. Read only when `content` is
                 absent (see below).
      - content: rich HTML for a flyout panel, inserted via `v-html` —
                 Accordion.vue's own precedent for arbitrary caller-trusted
                 rich content inside a `v-for` loop (this templating
                 language has no per-item dynamic slot mechanism to express
                 "arbitrary composed children, one flyout panel per item"
                 any other way — the identical constraint Toolbar.vue's own
                 header comment already works through at length for its own
                 `items`-array design, reused here without re-litigating).

      **Discriminator**: an item with `content` set (a non-empty string is
      expr's own JS-style-truthy value; an omitted/undefined `content` is
      expr's falsy `UndefinedValue` — see Toolbar.vue's header comment for
      the identical "omitted field reads as falsy, not a
      '[missing: ...]' placeholder" reasoning, which applies unchanged to a
      plain v-for item-scope member access like this one) is a **flyout
      trigger**: rendered as a `popovertarget`-toggled `<button>` plus its
      own `popover="auto"` content `<div v-html="item.content">`, DropdownMenu.vue's
      exact trigger/content wiring pattern, reused unchanged at the
      template level (only the ARIA attributes differ — see above). An
      item with no `content` (regardless of whether `href` is set) is a
      **plain link**: a real `<a :href="item.href">` with *no* popover
      machinery at all — no `popovertarget`, no `popover`, no second
      element rendered for it. The template below expresses this as a
      single `v-if="item.content"` / `v-else` pair per item, so the two
      branches cannot both fire for the same item and a plain-link item's
      markup has no path through which a stray `popovertarget`/`popover`
      attribute could leak in from the flyout branch (self-adversarially
      re-checked after writing this file — see this component's Go test
      file for the automated version of the same check).

  ## Static roving-tabindex baseline: Menubar.vue's simpler homogeneous
  case, not Toolbar.vue's/DropdownMenu.vue's bounded ternary chain

  Unlike Toolbar.vue's/DropdownMenu.vue's own item lists (which mix
  separators and independently-`disabled` entries, forcing a bounded,
  unrolled ternary chain to find "the first real tab stop" — see either
  file's header comment), this component's top-level `items` has no such
  heterogeneity: this v1 exposes no per-item `disabled` field, and *every*
  entry renders as exactly one real top-level interactive control — either
  a `<button>` (flyout trigger) or an `<a href>` (plain link), never a
  separator or an inert placeholder. The true first tab stop is therefore
  unconditionally `items[0]`'s own control, exactly the same conclusion
  Menubar.vue's own header comment reaches for its own homogeneous
  `menus` array — so the template below uses a plain
  `index === 0 ? '0' : '-1'` check on *both* branches (link and trigger
  alike), not a ternary chain.

  ## What's cut from this v1, and why (scope discipline, not oversight)

  Real Radix's NavigationMenu is genuinely one of the more elaborate
  components in this whole family — every prior floating/menu component in
  this batch (Tooltip.vue, Popover.vue, HoverCard.vue for Floating UI's
  full collision engine; DropdownMenu.vue/Menubar.vue for positioning, see
  either file's own "Positioning: an intentional v1 scope cut" section)
  already declined to reimplement the elaborate part of its real upstream
  counterpart in favor of a solid, honestly-scoped core. This file follows
  the same discipline for NavigationMenu's own three headline elaborate
  features, verified present in the source read above and deliberately not
  ported:

    1. **Shared animated viewport** (`NavigationMenuViewport` +
       `ViewportContentMounter`): upstream can portal every flyout's
       content into one shared, size-animated `<div>` positioned below the
       trigger row, so switching between open flyouts cross-fades/resizes
       smoothly instead of each panel being its own independent box. This
       file has no viewport at all — every flyout is its own independent
       `popover="auto"` element, opening centered in the viewport per the
       UA popover stylesheet's default (the same "Positioning: an
       intentional v1 scope cut" DropdownMenu.vue's/Menubar.vue's own
       header comments already document and give the identical rationale
       for: this component's own script scope is kept to roving-tabindex
       navigation, not floating/shared-viewport placement).
    2. **Sliding active-indicator** (`NavigationMenuIndicator`): upstream
       can render a small marker that slides/resizes under the currently
       active top-level trigger via `ResizeObserver`-driven
       `offsetWidth`/`offsetLeft` measurement (ROOT_CONTENT_DISMISS-scoped
       CSS custom properties `--radix-navigation-menu-indicator-translate-*`).
       Not ported: it is a purely decorative animation layer with no
       semantic/keyboard content, additive on top of everything else this
       file implements, and not part of this v1's brief.
    3. **Hover-intent open/close** (`activationMode="automatic"`,
       `delayDuration`/`skipDelayDuration`, the `onTriggerEnter`/
       `onTriggerLeave`/`onContentEnter`/`onContentLeave` timer state
       machine on lines ~146-209 of the read source): upstream's default
       mode opens a flyout on pointer hover after a short delay and closes
       it after the pointer leaves both trigger and content, with a
       separate "skip the delay" window for quickly moving between
       triggers. None of that timer/pointer-tracking machinery is
       reimplemented here — this v1 is `DropdownMenu.vue`'s own
       "manual"-equivalent instead (click a `popovertarget`-toggled trigger
       to open/close, exactly like every other `popovertarget`-based
       component in this package), matching Tooltip.vue's own explicit
       "no hover-intent timing" scope cut for the identical reason: no
       existing primitive in this package tracks pointer dwell time, and
       building one is a real, separate, additive feature, not a small
       add-on to this commit.

  A fourth, smaller cut, decided by directly checking the source rather
  than assumed from Menubar.vue's structural similarity (per this
  component's own design brief): **no Menubar.vue-style "close current
  flyout, open adjacent one" behavior on top-level Left/Right.** Grepping
  `navigation-menu.tsx`'s real `FocusGroupItem` (the shared roving-focus
  handler for every top-level trigger *and* link) shows its own `onKeyDown`
  only ever calls `.focus()` on the next/previous candidate — it never
  touches `context.value`/open state at all. The adjacent-open-on-arrow-key
  behavior Menubar.vue ports (`menubar.tsx`'s own `MenubarContent`
  `onKeyDown`, see that file's header comment fact 3) has no counterpart
  here: NavigationMenu's real "open the adjacent trigger's flyout"
  experience comes entirely from `activationMode="automatic"`'s
  *hover*-intent path (`onTriggerEnter` above), not from any keyboard
  handler — and hover-intent is itself already cut (#3 above). So, verified
  rather than assumed: this is genuinely a Menubar-specific keyboard
  nuance, not a NavigationMenu one, and this file's own Left/Right handler
  (below) is a plain roving-tabindex focus move only — it never calls
  `showPopover()`/`hidePopover()`.

  One more small, deliberate divergence, made consciously per this
  component's own design brief ("reuse the established wrap-around
  formula") rather than by exactly replicating upstream: the real
  `FocusGroupItem.onKeyDown` read above does *not* wrap at either end
  (it slices the candidate list to `candidateNodes.slice(currentIndex + 1)`
  before focusing, so Left/Right/Home/End walk off the end of the list and
  simply do nothing once you're on the first/last item — no wrap), and
  treats *all four* arrow keys as equivalent forward/backward moves
  regardless of the list's own horizontal layout. This file instead reuses
  this package's own established, already-verified-in-three-other-files
  roving-tabindex convention (Toolbar.vue/DropdownMenu.vue/Menubar.vue: a
  bounded `(index + step + n) % n` wraparound walk, Left/Right only for a
  horizontal top-level row) for consistency with every sibling component in
  this package, rather than introducing a fourth, one-off, non-wrapping,
  all-arrow-keys navigation scheme solely for this file. This is a
  conscious, brief-directed scope choice, not a missed detail — documented
  here per this component's own design brief's explicit instruction to
  record it either way.

  ## `<script customelement>` (load-bearing, not just polish)

  Per RFC 014 §3 Non-Goals, this reimplements nothing the browser already
  provides for free. A plain `<a>` and `<button>` have no native
  cross-sibling roving-tabindex behavior (the same conclusion every
  roving-tabindex file in this package already reaches), so that is
  this script's whole job:

  1. On connect, recomputes and re-applies the roving-tabindex baseline
     from the live DOM — Toolbar.vue's/DropdownMenu.vue's/Menubar.vue's own
     "never trust the static baseline alone" discipline, reused unchanged
     (here it is a provable no-op given the homogeneous item list — see
     above — but kept for the same consistency reason Menubar.vue's own
     header comment gives for its own no-op recompute).
  2. Left/Right/Home/End roving-tabindex navigation among the top-level
     items (links and triggers treated identically as "interactive" — see
     the self-adversarial note below), wrapping at both ends via the
     Toolbar.vue-established `(index + step + n) % n` formula. No `open`/
     `close` popover calls anywhere in this handler — see the "what's cut"
     section above for why that is a verified, deliberate omission, not a
     missed case.
  3. No click/activation handling of any kind: a real `<a href>` and a
     real `<button popovertarget>` already provide fully correct, native
     click/Enter/Space-driven navigation and open/close behavior with zero
     script (Popover.vue's/DropdownMenu.vue's own established
     `popovertarget` zero-JS baseline, reused unchanged) — there is nothing
     for this script to add on top of either.

  Self-adversarial check (per this commit's own process): a common bug in
  this exact shape of roving-tabindex walk is counting only `<button>`
  elements and silently skipping `<a>` elements (or vice versa) when
  collecting "the interactive top-level items" to walk over. This file's
  `#items` query below (`.radix-navigation-menu-toplevel`) is a single
  shared class present on *both* the trigger `<button>` and the plain-link
  `<a>` — deliberately, so the walk and the tabindex sync treat every
  top-level item as equally focusable, regardless of which element it
  renders as. Re-checked directly against the rendered template below
  after writing this file: both branches carry the class.

  Usage:
    <NavigationMenu
      :items="[
        { id: 'home', label: 'Home', href: '/' },
        {
          id: 'products',
          label: 'Products',
          content: '<a href=\"/widgets\">Widgets</a><a href=\"/gadgets\">Gadgets</a>',
        },
        { id: 'about', label: 'About', href: '/about' },
      ]"
    />

  No v-native escape hatch needed: this component's own name
  ("NavigationMenu") auto-registers a lowercase "navigationmenu" alias in
  the component registry. There is no native HTML element literally named
  `<navigationmenu>` for that alias to collide with — the elements actually
  used below (`<nav>`, `<ul>`, `<li>`, `<a>`, `<button>`, `<div>`) are
  unrelated native tags, the same reasoning every other file in this
  package's own header comment already works through for its own name.

  ## Custom-element tag name

  This file's own base name, "NavigationMenu.vue", derives to
  `radix-navigation-menu` under the standard `Mount{Prefix: "radix"}` this
  package assumes — see radix.go's header comment for the derivation
  algorithm.
-->
<template>
  <nav class="radix-navigation-menu" aria-label="Main">
    <ul class="radix-navigation-menu-list">
      <template v-for="(item, index) in items">
        <li v-if="item.content" class="radix-navigation-menu-item">
          <button
            type="button"
            :id="item.id + '-trigger'"
            class="radix-navigation-menu-trigger radix-navigation-menu-toplevel"
            :popovertarget="item.id"
            popovertargetaction="toggle"
            :tabindex="index === 0 ? '0' : '-1'"
          >{{ item.label }}</button>
          <div
            :id="item.id"
            class="radix-navigation-menu-content"
            popover="auto"
            :aria-labelledby="item.id + '-trigger'"
            v-html="item.content"
          ></div>
        </li>
        <li v-else class="radix-navigation-menu-item">
          <a
            :href="item.href"
            class="radix-navigation-menu-link radix-navigation-menu-toplevel"
            :tabindex="index === 0 ? '0' : '-1'"
          >{{ item.label }}</a>
        </li>
      </template>
    </ul>
  </nav>
</template>

<style scoped>
.radix-navigation-menu {
  display: block;
}

.radix-navigation-menu-list {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 0.15rem;
  margin: 0;
  padding: 0.25rem;
  list-style: none;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
}

.radix-navigation-menu-item {
  display: inline-block;
}

.radix-navigation-menu-trigger,
.radix-navigation-menu-link {
  display: inline-flex;
  align-items: center;
  font: inherit;
  color: inherit;
  text-decoration: none;
  background-color: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  padding: 0.4rem 0.7rem;
  cursor: pointer;
}

.radix-navigation-menu-trigger:hover,
.radix-navigation-menu-link:hover {
  background-color: #f5f5f5;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-navigation-menu-trigger:focus-visible,
.radix-navigation-menu-link:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 2px;
}

/*
 * Positioning: the same intentional v1 scope cut DropdownMenu.vue's/
 * Menubar.vue's own header comments document ("Positioning: an
 * intentional v1 scope cut" / "shared animated viewport" above) — this
 * deliberately does NOT override the UA popover stylesheet's default
 * `position: fixed; inset: 0; margin: auto` centering, so a flyout opens
 * centered in the viewport rather than anchored below its trigger or
 * routed through a shared viewport.
 */
.radix-navigation-menu-content {
  min-width: 12rem;
  padding: 0.75rem;
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
// Progressive enhancement on top of the zero-JS popovertarget/plain-<a>
// baseline above (every flyout already opens/closes/light-dismisses with
// no script, per Popover.vue's/DropdownMenu.vue's established
// popovertarget mechanism, and every plain link already navigates with no
// script at all). What genuinely has no native equivalent, and is
// therefore this script's whole job, is roving-tabindex arrow-key
// navigation across the top-level row — see this file's header comment's
// "What's cut from this v1" section for the verified, deliberate absence
// of any open/close logic here.
class RadixNavigationMenu extends HTMLElement {
  #items = [] // every top-level control (both triggers AND links), in DOM order

  connectedCallback() {
    const menu = this.querySelector('.radix-navigation-menu')
    if (!menu) return

    this.#items = Array.from(this.querySelectorAll('.radix-navigation-menu-toplevel'))

    // Recompute the tab stop from the live DOM unconditionally — a
    // provable no-op here (this file's baseline has no heterogeneity to
    // outgrow, see header comment), kept only for the same "never trust
    // the static baseline alone" discipline every roving-tabindex file in
    // this package already follows.
    this.#syncTabindex(0)

    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #syncTabindex(index) {
    this.#items.forEach((el, i) => el.setAttribute('tabindex', i === index ? '0' : '-1'))
  }

  #onKeydown = (event) => {
    const item = event.target.closest('.radix-navigation-menu-toplevel')
    // Ignore keys from anywhere else (e.g. inside an open flyout's own
    // rich content), and ignore items belonging to a nested
    // radix-navigation-menu, if any (mirrors every sibling roving-tabindex
    // file's own closest(...) !== this guard).
    if (!item || item.closest('radix-navigation-menu') !== this) return

    const index = this.#items.indexOf(item)
    if (index === -1) return

    let target
    switch (event.key) {
      case 'ArrowRight': target = this.#step(index, 1); break
      case 'ArrowLeft': target = this.#step(index, -1); break
      case 'Home': target = this.#step(-1, 1); break
      case 'End': target = this.#step(this.#items.length, -1); break
      default: return
    }
    if (target === -1) return

    event.preventDefault()
    this.#moveFocus(target)
  }

  // Every top-level item is always focusable (no separators, no per-item
  // disabled field in this v1 — see header comment), so this is a plain
  // wraparound modulo walk (Toolbar.vue's own `(index + step + n) % n`
  // formula) with no "skip non-focusable entries" loop needed — unlike
  // Toolbar.vue's/DropdownMenu.vue's own #findFocusable, which does need
  // that loop for their heterogeneous item lists.
  #step(fromIndex, step) {
    const n = this.#items.length
    if (n === 0) return -1
    return (fromIndex + step + n) % n
  }

  #moveFocus(index) {
    this.#syncTabindex(index)
    this.#items[index].focus()
  }
}

customElements.define('radix-navigation-menu', RadixNavigationMenu)
</script>
