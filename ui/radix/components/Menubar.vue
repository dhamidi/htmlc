<!--
  Menubar — a horizontal bar of top-level triggers, each opening its own
  DropdownMenu.vue-shaped submenu, with roving-tabindex navigation *between*
  the top-level triggers (Toolbar.vue's precedent) plus one behavior neither
  prior component needed: when a submenu is already open, Left/Right closes
  it and opens the adjacent trigger's own submenu instead of just moving
  focus between closed triggers.

  ## Why `menus: Array<{id, label, items}>` instead of composed children

  Same constraint Toolbar.vue's own header comment documents at length for
  every `items`-array component in this package (Accordion.vue, Tabs.vue,
  RadioGroup.vue, ToggleGroup.vue, Toolbar.vue itself, DropdownMenu.vue,
  ContextMenu.vue): there is no mechanism here for a parent component to
  reach into and modify already-rendered slot content the way React's
  context/`asChild`/`cloneElement` can. `menus` is the natural two-level
  generalization of DropdownMenu.vue's single-level `items` prop: "an array
  of dropdown menus", each carrying its own `id`/`label` (for its trigger)
  and its own `items` array (for its content), reusing DropdownMenu.vue's
  own `items[].type` ("item"/"separator") contract unchanged for the inner
  level — see that file's header comment for the full rationale, not
  repeated here.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/menubar/src/menubar.tsx` in the
  read-only clone. Facts this file's design depends on, confirmed there
  rather than assumed:

    1. `Menubar` renders `role="menubar"` on its root, with a
       `RovingFocusGroup.Root orientation="horizontal" loop={true}` one
       level up — confirming both the root's ARIA role and that top-level
       navigation wraps at both ends by default, exactly the `loop`
       default Toolbar.vue's own header comment already documents for its
       own roving-tabindex groups.
    2. `MenubarTrigger` renders `role="menuitem"` (not some
       menubar-specific role) with `aria-haspopup="menu"` — the identical
       ARIA pair DropdownMenu.vue's own trigger already established, now
       reused for a trigger that lives inside `role="menubar"` instead of
       standing alone. Its own `onKeyDown` handles Enter/Space (toggle the
       menu open) and ArrowDown (open the menu), each followed by
       `event.preventDefault()` "to prevent keydown from scrolling window /
       first focused item to execute that keydown" — confirming ArrowDown
       is real, in-scope trigger behavior, not a speculative addition.
    3. `MenubarContent`'s own `onKeyDown` is where the adjacent-menu-switch
       behavior actually lives upstream: on ArrowRight/ArrowLeft, it looks
       up the *next* (or, direction-reversed, previous) non-disabled menu
       value from the shared collection, wrapping via a `wrapArray` helper
       when `loop` is set, and opens that menu — confirming this behavior
       is genuinely keyed off "a menu is currently open and Left/Right was
       pressed while inside it", not off the top-level roving-tabindex
       group at all. This file's own `#onContentKeydown` (below) is the
       fresh-written equivalent of that hook point.
    4. Only one menu is ever open at a time: `Menubar`'s own state is a
       single controlled `value` (the currently-open menu's id, or `''`),
       not N independent booleans — `MenubarMenu`'s `open` is derived as
       `context.value === value`. This file's `<script customelement>`
       mirrors that with a single `#openIndex` field (see below), even
       though the underlying mechanism (N independent `popover="auto"`
       elements) is structurally different from upstream's single
       controlled value.
    5. Upstream also manages its roving `currentTabStopId` by hand
       (`onMenuOpen`/`onMenuToggle` both call `setCurrentTabStopId`), with
       the comment: "in some situations our triggers won't ever be given
       focus (e.g. click to open and then outside to close)" — i.e. the
       roving-tabindex stop tracks *which menu was last interacted with*,
       not literal DOM focus history, and does not reset back to the first
       trigger just because the menu was subsequently closed. This file's
       `#onContentToggle` (below) reproduces that: the roving stop moves to
       a trigger only on that trigger's own menu opening, and is left alone
       on close.
    6. `MenubarSub`/`MenubarSubTrigger`/`MenubarSubContent` (nested
       submenus), `MenubarCheckboxItem`/`MenubarRadioItem` (checkbox/radio
       items), `MenubarLabel`/`MenubarGroup` (grouping), and
       `MenubarArrow`/`MenubarPortal` all exist upstream. None are ported
       here — the identical "small, honest v1" scope cut DropdownMenu.vue's
       own header comment already makes for its own `items[].type`
       contract, now reused unchanged for this file's inner `items`.

  ## The zero-JS baseline, and what genuinely needs script

  Per-menu, the trigger/content pair is wired exactly like
  DropdownMenu.vue's own trigger/content: `popovertarget`/
  `popovertargetaction="toggle"` on the `<button>`, `popover="auto"` on the
  matching `<div id="...">`. This means, with zero JavaScript, every
  individual menu already opens and closes correctly on click, light-
  dismisses on outside click, and closes on Escape — exactly
  DropdownMenu.vue's own zero-JS story, now repeated N times.

  A pleasant, verified-not-assumed consequence of stacking N independent
  `popover="auto"` elements this way: clicking a *second* trigger while a
  different menu is already open does not need any extra script to close
  the first one. Per MDN's "Popover API — Using the Popover API" guide
  (fetched and read before writing this file), in the section on `auto`
  state and light dismissal: "Usually, only one auto popover can be shown
  at a time — showing a second popover when one is already shown will hide
  the first one." Since none of this file's per-menu popovers are nested
  inside one another, every mouse-driven "open a different menu" click
  already gets the correct "close the other one" behavior for free, via
  this native stacking rule — not something this file's own script needs
  to reimplement.

  What genuinely has no native/declarative equivalent, and is therefore
  this script's actual job:
    1. Roving-tabindex across the top-level triggers on Left/Right — a
       plain `<button>` has no native cross-sibling roving-tabindex
       behavior, the same conclusion Toolbar.vue's header comment reaches.
    2. **The Menubar-specific behavior**: when a submenu is already open,
       Left/Right must close it and open the *adjacent* trigger's own
       submenu (focusing its first item), not just move which trigger has
       `tabindex="0"`. This is the one piece of logic in this file with no
       precedent in Toolbar.vue/DropdownMenu.vue/ContextMenu.vue — see the
       hand-traced walkthrough in this file's own commit message and in
       `menubar_test.go`.
    3. ArrowDown on a closed trigger: opens its submenu (mirrors verified
       fact 2 above; Enter/Space need no script at all, since a real
       `<button>`'s native activation already fires the
       `popovertargetaction="toggle"` open/close, exactly the same "a
       disabled `<button>` never dispatches `click`, so no extra guard is
       needed" reasoning Toggle.vue's/DropdownMenu.vue's own header
       comments already establish — only ArrowDown has no native
       button-activation equivalent to piggyback on).
    4. Within an open submenu: Up/Down/Home/End roving-tabindex navigation
       among items (skipping separators/disabled entries) and click-
       delegated activation dispatching a `radix-menubar-select`
       `CustomEvent` then closing that submenu — DropdownMenu.vue's own
       internal item logic, rewritten fresh here for this file's own class
       names, not copy-pasted across files (the two components are
       structurally identical at this level, per DropdownMenu.vue's/
       ContextMenu.vue's own header comments already establishing that
       "menu internals are shared across this whole family").
    5. Focus-on-open: moving focus to the first non-disabled item whenever
       a submenu's content transitions to `"open"` (via its native
       `toggle` event) — DropdownMenu.vue's own "Focus-on-open" convention,
       reused unchanged, including its own documented, deliberate
       simplification of *not* gating on pointer-vs-keyboard modality (see
       that file's header comment; this package still has no cross-
       component modality-tracking primitive to gate it on).
    6. Escape: no explicit handling needed at all — the native `auto`
       popover state already closes on Escape and restores focus, the
       identical "nothing to do here" conclusion DropdownMenu.vue's own
       script reaches for the same reason.

  ## Static roving-tabindex baseline: simpler than Toolbar.vue's/
  DropdownMenu.vue's bounded ternary chain, and why

  Toolbar.vue's/DropdownMenu.vue's own static `:tabindex` baselines need a
  bounded, unrolled ternary chain (see either file's header comment) because
  their item lists are *heterogeneous*: some entries are separators or
  independently `disabled`, so "the first real tab stop" can sit at any of
  several leading positions. This file's top-level `menus` array has no
  such heterogeneity — every entry is a trigger, and this v1 exposes no
  per-menu `disabled` field (matching this component's own design brief) —
  so the true first tab stop is unconditionally `menus[0]`'s trigger,
  exactly like RadioGroup.vue's/ToggleGroup.vue's own homogeneous-list
  baseline. The template below therefore uses a plain
  `index === 0 ? '0' : '-1'` check, not a ternary chain — this is a
  deliberate simplification earned by the data shape, not a missed case.

  Each menu's own *inner* `items` (the submenu content) get a static
  `tabindex="-1"` unconditionally, with **no** baseline "first focusable
  item" computation at all — unlike DropdownMenu.vue's own item baseline.
  This is also a deliberate simplification, not a missed case: DropdownMenu
  ends up rendering standalone (mounted directly at the top level), so its
  static baseline is one honest attempt at "correct before any script
  runs". A Menubar submenu, by contrast, is never independently reachable
  via Tab at all while its popover is closed (a closed popover's
  descendants are removed from the tab order by the UA, the same fact
  DropdownMenu.vue's own baseline note already relies on) — and the very
  first thing this file's `<script customelement>` does whenever a submenu
  opens is recompute the correct tabindex from the live DOM (`#onContentToggle`,
  see below) before the user could possibly reach it by keyboard. So a
  submenu's own static baseline has no observable effect for any real user
  interaction, unlike the top-level trigger baseline (which genuinely is
  the very first thing a keyboard user's Tab key lands on).

  ## Props (REQUIRED — this package has no notion of an optional prop with
  a real default; see Popover.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):

    menus: Array<{ id, label, items }> —
      - id:    shared between this menu's trigger's `popovertarget` and its
               own content element's `id`, exactly DropdownMenu.vue's own
               `id` prop and rationale — must already be correct in the
               very first server-rendered HTML, and must be unique across
               every menu in the same `menus` array (and across any other
               popovertarget/id user on the same page).
      - label: the trigger's visible text.
      - items: Array<{ id, label, disabled, type }> — identical shape and
               contract to DropdownMenu.vue's own `items` prop ("item"/
               "separator"); see that file's header comment.

  Usage:
    <Menubar
      :menus="[
        {
          id: 'file-menu',
          label: 'File',
          items: [
            { type: 'item', id: 'new', label: 'New File' },
            { type: 'item', id: 'open', label: 'Open...' },
            { type: 'separator' },
            { type: 'item', id: 'close', label: 'Close', disabled: true },
          ],
        },
        {
          id: 'edit-menu',
          label: 'Edit',
          items: [
            { type: 'item', id: 'undo', label: 'Undo' },
            { type: 'item', id: 'redo', label: 'Redo' },
          ],
        },
      ]"
    />

  No v-native escape hatch needed: this component's own name ("Menubar")
  auto-registers a lowercase "menubar" alias in the component registry.
  There is a native HTML `<menu>` element, but no bare `<menubar>` element
  (the old HTML5 `<menu type="toolbar">` companion concept, like
  `<menu type="context">`, was dropped from the living standard years ago
  and was never itself a distinct `<menubar>` tag) for that alias to
  collide with — the elements actually used below (`<div>`, `<button>`,
  `<span>`) are unrelated native tags, the same reasoning DropdownMenu.vue's/
  ContextMenu.vue's own header comments already work through for their own
  files.

  ## Custom-element tag name — verified, not guessed

  `component.go`'s `deriveCustomElementTag` splits the file path on "/",
  drops ".vue", and converts each segment from PascalCase/camelCase to
  kebab-case via a lowercase-or-digit-followed-by-uppercase regex
  (`([a-z0-9])([A-Z])` → "$1-$2"), then lowercases. For this file's own
  base name, "Menubar": every letter after the leading "M" is lowercase
  ("enubar"), so the regex finds no lowercase-followed-by-uppercase
  boundary at all — confirmed by actually running this exact algorithm
  against "Menubar.vue" (not eyeballed), which produced the single,
  unhyphenated segment "menubar" (unlike "DropdownMenu" → "dropdown-menu"
  or "ContextMenu" → "context-menu", both of which do have an internal
  case boundary). Per radix.go's own header comment ("Mount prefix and
  custom-element tag names"), every component in this package hardcodes its
  tag assuming Mount{Prefix: "radix", ...}, matching every other file here
  (`radix-toolbar`, `radix-dropdown-menu`, `radix-context-menu`, ...) — so
  the tag registered below is `radix-menubar`.
-->
<template>
  <div class="radix-menubar" role="menubar" :aria-orientation="'horizontal'">
    <template v-for="(menu, index) in menus">
      <span class="radix-menubar-menu">
        <button
          type="button"
          class="radix-menubar-trigger"
          :popovertarget="menu.id"
          popovertargetaction="toggle"
          role="menuitem"
          aria-haspopup="menu"
          :tabindex="index === 0 ? '0' : '-1'"
        >{{ menu.label }}</button>
        <div
          :id="menu.id"
          class="radix-menubar-content"
          popover="auto"
          role="menu"
          aria-orientation="vertical"
        >
          <template v-for="item in menu.items">
            <button
              v-if="item.type === 'item'"
              type="button"
              role="menuitem"
              class="radix-menubar-item radix-menubar-menuitem"
              :data-id="item.id"
              :disabled="item.disabled"
              tabindex="-1"
            >{{ item.label }}</button>
            <div
              v-else-if="item.type === 'separator'"
              class="radix-menubar-item radix-menubar-separator"
              role="separator"
            ></div>
          </template>
        </div>
      </span>
    </template>
  </div>
</template>

<style scoped>
.radix-menubar {
  display: inline-flex;
  flex-direction: row;
  align-items: center;
  gap: 0.15rem;
  padding: 0.25rem;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
}

.radix-menubar-menu {
  display: inline-block;
}

.radix-menubar-trigger {
  font: inherit;
  color: inherit;
  background-color: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  padding: 0.4rem 0.7rem;
  cursor: pointer;
}

.radix-menubar-trigger:hover {
  background-color: #f5f5f5;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-menubar-trigger:focus-visible,
.radix-menubar-menuitem:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 2px;
}

/*
 * Positioning: the same intentional v1 scope cut DropdownMenu.vue's own
 * header comment documents ("Positioning: an intentional v1 scope cut") —
 * this deliberately does NOT override the UA popover stylesheet's default
 * `position: fixed; inset: 0; margin: auto` centering, so a submenu opens
 * centered in the viewport rather than anchored below its trigger. Porting
 * Popover.vue's trigger-rect positioning script onto this component is a
 * believed-straightforward, additive follow-up, not attempted here to keep
 * this file's script scope matched to its own brief (menu keyboard
 * semantics, not floating placement).
 */
.radix-menubar-content {
  flex-direction: column;
  gap: 0.15rem;
  min-width: 10rem;
  padding: 0.35rem;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  background-color: #fff;
  color: #171717;
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
 * unconditional `display: flex` here would therefore make every submenu
 * render permanently visible, even in its closed, zero-JS baseline state
 * — found and fixed via an actual rendered screenshot showing exactly
 * that, not caught by markup-only inspection.
 */
.radix-menubar-content:popover-open {
  display: flex;
}

.radix-menubar-menuitem {
  display: block;
  width: 100%;
  text-align: left;
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  border-radius: 4px;
  padding: 0.4rem 0.6rem;
  cursor: pointer;
}

.radix-menubar-menuitem:hover:not(:disabled) {
  background-color: #f5f5f5;
}

.radix-menubar-menuitem:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-menubar-separator {
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
// above — every individual menu already opens/closes/light-dismisses with
// no script at all (see this file's header comment). What this script adds
// is: roving-tabindex navigation across the top-level triggers, the one
// genuinely new Menubar-specific behavior (an already-open submenu
// following Left/Right to the adjacent trigger instead of just moving
// which trigger is focused), and DropdownMenu.vue-equivalent keyboard
// navigation/activation once a submenu is open.
class RadixMenubar extends HTMLElement {
  #menus = [] // [{ id, trigger, content, items, focusable }], top-level DOM order
  #triggers = [] // #menus[].trigger, kept in sync, in DOM order
  #openIndex = -1 // index into #menus of the currently open submenu, or -1

  connectedCallback() {
    const menubar = this.querySelector('.radix-menubar')
    if (!menubar) return

    const wrappers = Array.from(this.querySelectorAll('.radix-menubar-menu'))
    this.#menus = wrappers
      .map((wrapper) => {
        const trigger = wrapper.querySelector('.radix-menubar-trigger')
        const content = wrapper.querySelector('.radix-menubar-content')
        if (!trigger || !content) return null
        return {
          id: content.id,
          trigger,
          content,
          items: Array.from(content.querySelectorAll('.radix-menubar-item')),
          focusable: [],
        }
      })
      .filter((menu) => menu !== null)
    this.#triggers = this.#menus.map((menu) => menu.trigger)

    // Recompute the top-level tab stop from the live DOM unconditionally —
    // a no-op here (this file's baseline has no bounded-lookahead case to
    // outgrow, see header comment) but kept for the same "never trust the
    // static baseline alone" discipline every roving-tabindex file in this
    // package already follows.
    this.#syncTriggerTabindex(0)

    this.#menus.forEach((menu, index) => {
      menu.content.addEventListener('toggle', (event) => this.#onContentToggle(event, index))
      menu.content.addEventListener('click', (event) => this.#onContentClick(event, index))
      menu.content.addEventListener('keydown', (event) => this.#onContentKeydown(event, index))
    })

    this.addEventListener('keydown', this.#onTriggerKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('keydown', this.#onTriggerKeydown)
    // Per-content listeners were registered as closures (they need each
    // menu's own index), so there is no single function reference left to
    // remove them with individually; they are simply dropped along with
    // the elements themselves when this custom element disconnects, the
    // same "nothing further to do" case Accordion.vue's/Tabs.vue's own
    // per-item closures already accept.
  }

  #refreshFocusable(menu) {
    menu.focusable = menu.items.filter((el) => el.classList.contains('radix-menubar-menuitem') && !el.disabled)
  }

  #syncItemTabindex(menu) {
    menu.focusable.forEach((el, i) => el.setAttribute('tabindex', i === 0 ? '0' : '-1'))
  }

  #syncTriggerTabindex(index) {
    this.#triggers.forEach((el, i) => el.setAttribute('tabindex', i === index ? '0' : '-1'))
  }

  // Top-level Left/Right/ArrowDown, scoped to a keydown whose target is one
  // of #triggers itself (not inside an open submenu's content — that is
  // #onContentKeydown's job, attached directly to each content element
  // below).
  #onTriggerKeydown = (event) => {
    const trigger = event.target.closest('.radix-menubar-trigger')
    if (!trigger || trigger.closest('radix-menubar') !== this) return

    const index = this.#triggers.indexOf(trigger)
    if (index === -1) return

    // ArrowDown has no native <button> activation to piggyback on (unlike
    // Enter/Space, which already fire the trigger's own
    // popovertargetaction="toggle" natively) — this is the one top-level
    // key this file's script must open on directly.
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      this.#openMenu(index)
      return
    }

    let direction
    if (event.key === 'ArrowRight') direction = 1
    else if (event.key === 'ArrowLeft') direction = -1
    else return

    event.preventDefault()
    this.#navigate(index, direction)
  }

  // The Menubar-specific override lives here too, for the (normally brief)
  // window where DOM focus is on a trigger while a *different* menu is
  // still open — #onContentKeydown below covers the far more common case
  // where focus already moved inside the open submenu's own content.
  #onContentKeydown = (event, index) => {
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault()
      this.#navigate(index, event.key === 'ArrowRight' ? 1 : -1)
      return
    }

    const menu = this.#menus[index]
    const item = event.target.closest('.radix-menubar-menuitem')
    if (!item || item.closest('radix-menubar') !== this) return

    const itemIndex = menu.items.indexOf(item)
    if (itemIndex === -1) return

    let target
    switch (event.key) {
      case 'ArrowDown': target = this.#findFocusableItem(menu, itemIndex, 1); break
      case 'ArrowUp': target = this.#findFocusableItem(menu, itemIndex, -1); break
      case 'Home': target = this.#findFocusableItem(menu, -1, 1); break
      case 'End': target = this.#findFocusableItem(menu, menu.items.length, -1); break
      default: return
    }
    if (target === -1) return

    event.preventDefault()
    this.#moveItemFocus(menu, target)
  }

  // Shared by both #onTriggerKeydown and #onContentKeydown: `fromIndex` is
  // always a *top-level* menu index (never an item index) here.
  //
  // If a submenu is currently open (#openIndex !== -1): the
  // Menubar-specific behavior this file exists to add — close the current
  // submenu and open the adjacent trigger's own submenu instead of just
  // moving which trigger has tabindex="0" (the adjacent submenu's first
  // item then receives focus once its content's own `toggle` event fires
  // "open" — see #onContentToggle below, the same toggle-driven
  // focus-on-open convention DropdownMenu.vue's own script already
  // establishes).
  //
  // If no submenu is open: a plain roving-tabindex focus move between
  // closed triggers, matching Toolbar.vue's own Left/Right behavior
  // exactly — nothing opens.
  #navigate(fromIndex, direction) {
    const n = this.#triggers.length
    if (n === 0) return
    const nextIndex = (fromIndex + direction + n) % n

    if (this.#openIndex !== -1) {
      this.#menus[this.#openIndex].content.hidePopover()
      this.#openMenu(nextIndex)
    } else {
      this.#syncTriggerTabindex(nextIndex)
      this.#triggers[nextIndex].focus()
    }
  }

  #openMenu(index) {
    const menu = this.#menus[index]
    if (!menu || menu.content.matches(':popover-open')) return
    menu.content.showPopover()
  }

  // Tracks which submenu is open (mirroring upstream's single controlled
  // `value`, see header comment fact 4) and, on the transition to "open",
  // moves the roving-tabindex stop onto this trigger (fact 5 — the stop is
  // *not* reset back to the first trigger when a menu later closes) and
  // focuses the first non-disabled item (DropdownMenu.vue's own
  // Focus-on-open convention).
  #onContentToggle = (event, index) => {
    const menu = this.#menus[index]
    if (event.newState === 'open') {
      this.#openIndex = index
      this.#syncTriggerTabindex(index)
      this.#refreshFocusable(menu)
      this.#syncItemTabindex(menu)
      if (menu.focusable.length === 0) return
      this.#moveItemFocus(menu, menu.items.indexOf(menu.focusable[0]))
      return
    }
    if (this.#openIndex === index) this.#openIndex = -1
  }

  // Delegated click listener on this menu's own content element: fires for
  // a real mouse click AND for a real <button>'s own native Enter/Space
  // activation (both dispatch 'click' per the HTML spec) — identical
  // reasoning to DropdownMenu.vue's own #onClick, including the same "a
  // disabled <button> never dispatches click, so no extra guard is needed"
  // conclusion.
  #onContentClick = (event, index) => {
    const menu = this.#menus[index]
    const item = event.target.closest('.radix-menubar-menuitem')
    if (!item || item.closest('radix-menubar') !== this) return

    this.dispatchEvent(
      new CustomEvent('radix-menubar-select', {
        detail: { menuId: menu.id, id: item.dataset.id },
        bubbles: true,
      })
    )
    menu.content.hidePopover()
  }

  // Walks a single menu's own #items (menuitems AND separators) one step
  // at a time in `step` direction, wrapping at both ends — DropdownMenu.vue's
  // own #findFocusable, reused unchanged in shape (same wrap formula, same
  // skip-non-focusable-entries logic, same bound of at most items.length
  // steps), just scoped to one menu's own item list instead of a single
  // top-level items array.
  #findFocusableItem(menu, fromIndex, step) {
    const n = menu.items.length
    if (n === 0) return -1
    let index = fromIndex
    for (let i = 0; i < n; i++) {
      index = (index + step + n) % n
      const el = menu.items[index]
      if (el.classList.contains('radix-menubar-menuitem') && !el.disabled) return index
    }
    return -1
  }

  #moveItemFocus(menu, index) {
    menu.focusable.forEach((el) => el.setAttribute('tabindex', '-1'))
    const el = menu.items[index]
    el.setAttribute('tabindex', '0')
    el.focus()
  }
}

customElements.define('radix-menubar', RadixMenubar)
</script>
