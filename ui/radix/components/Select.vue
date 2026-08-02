<!--
  Select — Radix-inspired custom-styled dropdown listbox layered on top of a
  real, fully-functional native <select>.

  ## Unlike almost every other component in this package, the zero-JS
  baseline here is NOT a degraded experience

  Every other component this package has ported so far (Toggle.vue's
  click-toggle behavior, Tooltip.vue's hover reveal, DropdownMenu.vue's
  roving-tabindex menu) genuinely needs its own <script customelement> to
  reach its real, complete interaction model — without it, those components
  are honestly, deliberately less capable than the finished thing (see each
  file's own header comment for exactly what is lost). A native <select> is
  categorically different: it already IS a fully functional, fully
  accessible form control on its own — real keyboard support (typeahead,
  arrow-key cycling, Home/End), a real accessible name and role from the
  browser's own accessibility mapping, and real, correct form participation,
  all for free, with zero JavaScript. Radix's actual, honest value-add for
  Select is narrower than for any other primitive it ships: browsers only
  expose a handful of CSS properties on a native <select>'s own dropdown
  popup (and historically, none at all in most engines) — so the only real
  gap being filled here is *visual restyling of the popup*, not capability.
  This file's design follows directly from that: the real <select> below
  (visually hidden, never disabled or hidden from assistive tech) remains
  the actual source of truth and the thing a no-JS user directly operates,
  via their browser's own native dropdown UI — full stop, not a fallback.

  ## Baseline: the real, native <select>

  A real <select :name="name" :id="id + '-native'">, with one real <option>
  per item (`:value`, `:selected="item.value === value"`, `:disabled`).
  Visually hidden with the same clip-based technique Checkbox.vue's/
  RadioGroup.vue's own hidden inputs use (position: absolute, 1x1px,
  clip-path: inset(50%) — never display:none, which would drop it from
  focus reach, the tab order, and (depending on the browser/AT pairing) its
  own accessibility exposure). Unlike Checkbox.vue's/RadioGroup.vue's hidden
  inputs, though, this one is not merely a "state holder" behind a purely
  decorative sibling — see "Two real, independently-operable controls"
  below for why that distinction matters here.

  This is the actual form-submitted control: it carries the real `name`
  attribute, and the browser handles its own native selectedness exactly as
  it always does, including a fact worth calling out explicitly because it
  has no counterpart in Checkbox.vue's/RadioGroup.vue's own "unset value"
  handling (see RadioGroup.vue's header comment, "the unset-value case"): if
  a caller's `value` prop does not match any `item.value` (or `items` is
  empty), a real <select> does NOT render with "nothing selected" the way a
  radio group does — the HTML Standard requires the browser to fall back to
  selecting the *first* <option> whenever no option carries `selected` (see
  the "select" element, "the selectedness setting algorithm"). This
  component's contract, matching RadioGroup.vue's own "callers should pass
  a `value` that matches one of `items`" convention, is that `value` is
  expected to match a real item; the browser's own first-option fallback for
  the rare misuse case where it does not is real, spec'd platform behavior,
  not something this file re-implements or should override.

  ## Custom visual layer: shown alongside, kept in sync

  A `<button popovertarget="..." popovertargetaction="toggle">` trigger
  (matching Popover.vue's/DropdownMenu.vue's own zero-JS open/close/
  light-dismiss mechanism — see either file's header comment for the
  verified MDN citations this reuses as-is) displaying the currently
  selected item's label, plus a `<div popover="auto" role="listbox">`
  containing one `<div role="option">` per item — the same per-item
  decomposition DropdownMenu.vue's `<button role="menuitem">` items use,
  translated to APG's listbox vocabulary (`role="listbox"`/`role="option"`/
  `aria-selected`) instead of menu's (`role="menu"`/`role="menuitem"`). No
  `aria-expanded`/`aria-controls`/`aria-details` is hand-set on the trigger,
  for the identical reason Popover.vue's/DropdownMenu.vue's own templates
  omit them: the browser sets up an implicit aria-expanded/aria-details
  relationship for any popovertarget invoker automatically. `aria-haspopup`
  is set to `"listbox"` — a real, spec'd `aria-haspopup` token (WAI-ARIA
  lists `menu`/`listbox`/`tree`/`grid`/`dialog` alongside the boolean
  values), the listbox-flavored counterpart to DropdownMenu.vue's own
  verified `aria-haspopup="menu"`.

  The trigger's initial label is computed entirely server-side from `items`
  and `value` — no placeholder text, no client-side lookup needed for
  correctness on first paint — via a nested `<template v-for><template
  v-if>` pair that emits only the one item whose `value` matches (the same
  `<template v-for>` transparent-fragment mechanism DropdownMenu.vue/
  Tabs.vue/Toolbar.vue already rely on for `v-for`-driven markup; nesting a
  second, condition-only `<template v-if>` inside it costs nothing extra —
  `<template>` without a controlling directive already renders its children
  with no wrapper element, confirmed against renderer.go's own handling of
  bare `<template>` nodes, not assumed).

  ## Two real, independently-operable controls, by design — not an oversight

  A close reading of Radix's own real source (read-only, not transcribed —
  see below) shows upstream Select does NOT keep its own native <select>
  independently operable: their bridging input is `aria-hidden` and
  `tabIndex={-1}` (confirmed in `select.tsx`'s `SelectBubbleInput` — a pure
  form-submission bridge, never meant to be reached by Tab or a screen
  reader; their real interactive surface is `role="combobox"` on the
  trigger instead). This file makes the opposite, deliberate choice, because
  this port's whole premise is that the
  native <select> is NOT a degraded fallback here: it stays exactly as
  reachable and operable as Checkbox.vue's/RadioGroup.vue's own real inputs
  do — no `tabindex="-1"`, no `aria-hidden`, nothing pulling it out of the
  tab order or the accessibility tree (the same "leave the real control
  alone" reasoning Checkbox.vue's header comment already states plainly).
  One direct, honest consequence: a keyboard user tabbing through the page
  will reach *two* separate stops for one logical control — the native
  <select> (fully operable via the browser's own dropdown, exactly as if
  this component did not exist) and, separately, the custom trigger button
  (operable once the enhancement script below has run). This is an
  intentional trade-off, not a bug: it is
  the price of the native control genuinely never being a degradation. It
  also means a keyboard user can legitimately operate the *native* select
  directly instead of the custom trigger at any time — which is exactly why
  the enhancement script listens for the native select's own `change` event
  rather than only reacting to clicks on its own custom options; see
  "`<script customelement>`" below.

  ## `v-native` on the <select> tag — the exact self-reference-cycle trap
  Label.vue's/Dialog.vue's own header comments already document, hitting
  this file directly

  This component's own name, "Select", auto-registers a lowercase "select"
  alias in the component registry the moment ui/radix is mounted (the
  standard `entries[strings.ToLower(name)] = entry` convention — confirmed
  in engine.go's `discoverMountInto`, not assumed). A literal `<select>` tag
  anywhere in THIS component's own template would therefore resolve back to
  this very component instead of being treated as plain HTML (renderer.go
  resolves every implicit, non-native-declared tag against the component
  registry first, unless `v-native` marks it — confirmed at
  renderer.go:1214 area, the identical mechanism Label.vue's/Dialog.vue's/
  Checkbox.vue's own header comments already document for their own
  same-named native tags: <label>/<dialog>/<label> respectively). Unlike
  those files' collisions with an *external* sibling component's alias,
  this is a direct, unavoidable self-collision: "Select" and native
  `<select>` share the exact same lowercase spelling. `v-native` below
  declares the <select> tag a genuine native HTML element up front, the
  same fix every one of those sibling files already applies to their own
  same-named tag.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/select/src/select.tsx` in the
  read-only clone. Facts this file's design depends on, confirmed there
  rather than assumed:

    1. The real content element renders `role="listbox"` with no
       `aria-orientation` at all (select.tsx, `SelectPosition role="listbox"`)
       — unlike DropdownMenu.vue's verified `role="menu"
       aria-orientation="vertical"`, a listbox's ARIA-implicit default
       orientation is already vertical, so this file's own
       `role="listbox"` div likewise carries no `aria-orientation`
       override, the same "omit what is already the implicit default"
       convention DropdownMenu.vue's own separator applies for
       `aria-orientation` on `role="separator"`.
    2. Each real option renders `role="option"` with `aria-selected`
       (select.tsx, `SelectItemImpl`) — confirmed, not guessed, matching
       this file's own `role="option"`/`:aria-selected` items.
    3. The real native-select bridge (`SelectBubbleInput`) is
       `aria-hidden`/`tabIndex={-1}` — confirmed, and deliberately NOT
       replicated here; see "Two real, independently-operable controls"
       above for why this port makes the opposite choice.

  ## `<script customelement>` (this is the one place genuine two-way sync
  matters — get it right)

  1. Selecting a custom option (`click` on a `.radix-select-item`, or
     Enter/Space while one is focused) calls `#activateItem`, which: sets
     the *native* `<select>`'s own `.value` (so form submission stays
     correct — this is the one thing that would silently submit the wrong
     value if it were wrong), then dispatches a real
     `new Event('change', { bubbles: true })` on it so any external
     listener watching the native select still fires correctly, then
     calls `.hidePopover()` to close
     the listbox. It deliberately does NOT poke the trigger label or
     `aria-selected` directly — see point 3 below for why routing through
     the native select's own `change` event instead is a better, more
     honest design than duplicating that logic in two places.
  2. Keyboard nav within the open custom listbox: Up/Down roving via
     `#findFocusable`, reusing the exact wrap-around/skip-disabled formula
     DropdownMenu.vue's own `#findFocusable` established (itself reused
     from Toolbar.vue — `(index + step + n) % n`, bounded to at most
     `#items.length` steps so a listbox with no enabled item terminates
     instead of looping forever) — simplified from DropdownMenu.vue's
     version only in that there is no "item vs. separator" type check to
     skip past, just `aria-disabled`. Home/End jump to the first/last
     enabled option using the same "search from just before index 0 / just
     after the last index" technique. Enter/Space on a focused option
     activates it and closes the listbox, matching DropdownMenu.vue's own
     Enter/Space-selects-and-closes behavior (adapted to an explicit
     keydown check here, since — unlike DropdownMenu.vue's real `<button>`
     items — a plain `<div role="option">` has no native Enter/Space→click
     activation behavior for a delegated `click` listener to piggyback on).
  3. On the native `<select>`'s own `change` event — which fires both for
     this script's own dispatch in point 1 above AND independently for any
     other real change to the native select (a user operating it directly,
     since it deliberately stays fully keyboard/AT-operable per "Two real,
     independently-operable controls" above; a browser extension or
     autofill that properly sets `.value` and dispatches `change`, which
     real autofill implementations do) — `#syncFromNative` re-derives the
     trigger's displayed label and every option's `aria-selected` directly
     from the native select's own current `.value`, and recomputes the
     roving-tabindex baseline to match. Routing BOTH the custom-driven path
     and any externally-driven path through this one function (rather than
     updating the trigger/aria-selected directly inside `#activateItem`)
     is a deliberate design choice, not an accident: it is what gives this
     v1 real (if partial — see below) reverse sync, for free, with no
     duplicated logic to drift out of sync with itself. `#syncFromNative`
     also runs once, unconditionally, in `connectedCallback` — the same
     "recompute from the live DOM on connect" correctness discipline
     DropdownMenu.vue's own `connectedCallback` documents — which is what
     reconciles this component's server-rendered trigger label with the
     browser's own first-option-selectedness fallback (see the baseline
     section above) in the rare case a caller's `value` does not match any
     item.

     What this does NOT cover — documented as a known, deliberate v1 gap
     rather than silently claimed as full bidirectional sync (this is
     treated as "a reasonable, honestly-scoped v1"): a bare, external
     `nativeSelectElement.value = 'x'` assignment with no `change` event
     ever dispatched afterward is genuinely unobservable by any event
     listener in the DOM — there is no platform event for "a script wrote
     to my `.value` property." Detecting that would require polling or a
     `MutationObserver`-style workaround (`.value` is an IDL property, not
     a reflected attribute, so a `MutationObserver` would not see it
     either); no component in this package uses either technique (grepped,
     not assumed), and introducing one here for a single, narrow edge case
     would be a real departure from this package's established "small,
     honest v1" discipline (see DropdownMenu.vue's own "Positioning: an
     intentional v1 scope cut" for the same kind of call). The custom
     layer is therefore the primary interactive path with genuine (if not
     total) reverse sync from the native select, not a one-way mirror onto
     it — a real improvement over "documented as entirely unaddressed,"
     without overreaching into "silently claims full bidirectional sync
     that doesn't actually exist."
  4. The native popovertarget/popover machinery owns opening the listbox
     entirely on its own (no `.showPopover()` call anywhere in this
     script, mirroring Popover.vue's/DropdownMenu.vue's own forbidden-call
     convention). On the content's native `toggle` event transitioning to
     `"open"`, focus moves to whichever option currently carries
     `aria-selected="true"` (falling back to the first enabled option if
     none do) — the ARIA listbox convention of landing focus on the
     selected option, distinct from DropdownMenu.vue's own "always focus
     the first item" menu convention (a menu has no notion of a "currently
     selected" item; a listbox does).

  Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):
    id:    string — shared between the trigger's `popovertarget` and the
           listbox content element's own `id`, exactly Popover.vue's/
           DropdownMenu.vue's own `id` prop and rationale (load-bearing for
           the zero-JS popovertarget baseline itself). Also used to build
           the native <select>'s own id (`id + '-native'`), keeping the two
           real ids distinct so both can independently be referenced
           (e.g. by a `<Label for="...">`) without colliding.
    name:  string — the native `name` attribute on the real <select>;
           needed for it to actually submit as part of a form.
    items: Array<{ value, label, disabled }> — one entry per option.
             - value:    the native `value` attribute for this item's real
                          <option>, and the value compared against this
                          component's own `value` prop / the native
                          select's live `.value` throughout this file.
             - label:    this item's visible option text, rendered as
                          plain text in both the real <option> and the
                          custom `role="option"` div.
             - disabled: applied to the real <option>'s `disabled`
                          attribute and (via `aria-disabled`) to the custom
                          option's own disabled presentation.
    value: string — the currently-selected item's `value`; see the "unset
           value" note in the baseline section above for what happens when
           it does not match any item.

  Usage:
    <Select
      id="fruit"
      name="fruit"
      value="apple"
      :items="[
        { value: 'apple', label: 'Apple' },
        { value: 'banana', label: 'Banana' },
        { value: 'cherry', label: 'Cherry', disabled: true },
      ]"
    />

  ## Custom-element tag name

  This file's own base name, "Select.vue", derives to `radix-select` under
  the standard `Mount{Prefix: "radix"}` this package assumes — see
  radix.go's header comment for the derivation algorithm.
-->
<template>
  <Tokens></Tokens>
  <span class="radix-select">
    <select
      v-native
      class="radix-select-native radix-visually-hidden-input"
      :name="name"
      :id="id + '-native'"
    >
      <option
        v-for="item in items"
        :value="item.value"
        :selected="item.value === value"
        :disabled="item.disabled"
      >{{ item.label }}</option>
    </select>
    <button
      v-native
      type="button"
      class="radix-select-trigger"
      :popovertarget="id"
      popovertargetaction="toggle"
      aria-haspopup="listbox"
    ><template v-for="item in items"><template v-if="item.value === value">{{ item.label }}</template></template></button>
    <div
      :id="id"
      class="radix-select-content"
      popover="auto"
      role="listbox"
    >
      <div
        v-for="(item, index) in items"
        role="option"
        class="radix-select-item"
        :data-value="item.value"
        :aria-selected="item.value === value"
        :aria-disabled="item.disabled ? 'true' : undefined"
        :tabindex="index === (items.length > 0 && !items[0].disabled ? 0 : items.length > 1 && !items[1].disabled ? 1 : items.length > 2 && !items[2].disabled ? 2 : items.length > 3 && !items[3].disabled ? 3 : items.length > 4 && !items[4].disabled ? 4 : items.length > 5 && !items[5].disabled ? 5 : -1) ? '0' : '-1'"
      >{{ item.label }}</div>
    </div>
  </span>
</template>

<style>
.radix-select {
  position: relative; /* containing block for the absolutely-positioned native select */
  display: inline-block;
}

.radix-select-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  min-width: 10rem;
  font: inherit;
  color: inherit;
  background-color: #fff;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
  padding: 0.4rem 0.7rem;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users.
   Applies to both real, independently-operable controls (see header
   comment): the native select's own focus ring is left entirely alone
   (untouched, native browser chrome), while the custom trigger/options get
   an explicit ring here since their focus-visible look is not otherwise
   provided by anything native. */
.radix-select-trigger:focus-visible,
.radix-select-item:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

/*
 * Deliberately no position/inset/margin override here, matching
 * DropdownMenu.vue's own documented "Positioning: an intentional v1 scope
 * cut" — this v1's listbox opens centered in the viewport (the UA popover
 * stylesheet's own default), not anchored below its trigger.
 */
.radix-select-content {
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
 * unconditional `display: flex` here would therefore make this listbox
 * render permanently visible, even in its closed, zero-JS baseline state
 * — found and fixed via an actual rendered screenshot showing exactly
 * that, not caught by markup-only inspection.
 */
.radix-select-content:popover-open {
  display: flex;
}

.radix-select-item {
  display: block;
  width: 100%;
  text-align: left;
  border-radius: var(--radix-radius-2);
  padding: 0.4rem 0.6rem;
  cursor: pointer;
}

.radix-select-item:hover:not([aria-disabled="true"]) {
  background-color: var(--radix-gray-3);
}

.radix-select-item[aria-selected="true"] {
  background-color: var(--radix-blue-4);
  font-weight: 700;
}

.radix-select-item[aria-disabled="true"] {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

<script customelement>
// Progressive enhancement on top of the zero-JS popovertarget baseline
// above (open/close/light-dismiss are all native — see this file's header
// comment). Per RFC 014 §3 Non-Goals, this reimplements nothing the browser
// already provides for free on the real <select> itself — what this script
// actually does is make the *custom* visual layer operable, and keep it
// synced with the real control that remains the actual source of truth.
class RadixSelect extends HTMLElement {
  #nativeSelect = null
  #trigger = null
  #content = null
  #items = [] // every .radix-select-item option div, in DOM order
  #focusable = [] // subset of #items: enabled options only, in DOM order

  connectedCallback() {
    this.#nativeSelect = this.querySelector('.radix-select-native')
    this.#trigger = this.querySelector('.radix-select-trigger')
    this.#content = this.querySelector('.radix-select-content')
    if (!this.#nativeSelect || !this.#trigger || !this.#content) return

    this.#items = Array.from(this.querySelectorAll('.radix-select-item'))

    // Recompute from the live DOM unconditionally on connect — both a
    // defensive correctness fix (mirrors DropdownMenu.vue's own
    // connectedCallback) and the mechanism that reconciles this
    // component's server-rendered trigger label with the browser's own
    // first-option-selectedness fallback (see header comment) whenever a
    // caller's `value` did not match any item.
    this.#syncFromNative()

    this.#nativeSelect.addEventListener('change', this.#onNativeChange)
    this.#content.addEventListener('toggle', this.#onToggle)
    this.#content.addEventListener('click', this.#onClick)
    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    if (this.#nativeSelect) {
      this.#nativeSelect.removeEventListener('change', this.#onNativeChange)
    }
    if (this.#content) {
      this.#content.removeEventListener('toggle', this.#onToggle)
      this.#content.removeEventListener('click', this.#onClick)
    }
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #isDisabled(el) {
    return el.getAttribute('aria-disabled') === 'true'
  }

  #refreshFocusable() {
    this.#focusable = this.#items.filter((el) => !this.#isDisabled(el))
  }

  // Re-derives the trigger's displayed label and every option's
  // aria-selected directly from the native select's own current `.value` —
  // the single function both the custom-driven selection path (via the
  // `change` event #activateItem dispatches) and any externally-driven
  // change to the native select itself route through. See header comment
  // point 3 for why unifying both paths through the native select's own
  // `change` event, rather than updating the trigger/aria-selected
  // directly inside #activateItem, is this file's actual reverse-sync
  // mechanism (and what it does not cover).
  #syncFromNative = () => {
    const value = this.#nativeSelect.value
    let selectedItem = null
    this.#items.forEach((el) => {
      const selected = el.dataset.value === value
      el.setAttribute('aria-selected', selected ? 'true' : 'false')
      if (selected) selectedItem = el
    })
    this.#trigger.textContent = selectedItem ? selectedItem.textContent : ''
    this.#refreshFocusable()
    this.#syncTabindex()
  }

  #onNativeChange = () => {
    this.#syncFromNative()
  }

  // Roving-tabindex baseline: the currently-selected enabled option (or,
  // absent one, the first enabled option) gets tabindex="0"; every other
  // option gets tabindex="-1" — so only one Tab stop exists into the whole
  // listbox, matching DropdownMenu.vue's/Toolbar.vue's own roving-tabindex
  // convention.
  #syncTabindex() {
    const selected = this.#items.find(
      (el) => el.getAttribute('aria-selected') === 'true' && !this.#isDisabled(el)
    )
    const target = selected || this.#focusable[0]
    this.#items.forEach((el) => el.setAttribute('tabindex', el === target ? '0' : '-1'))
  }

  // Moves focus onto the option matching aria-selected="true" whenever the
  // listbox transitions to open — the ARIA listbox convention of landing
  // focus on the currently-selected option (distinct from DropdownMenu.vue's
  // own "always focus the first item" menu convention — a menu has no
  // notion of a currently-selected item; a listbox does). Falls back to the
  // first enabled option when nothing is selected. No action on close: the
  // native popovertarget/popover machinery already owns returning focus to
  // the trigger, the same "nothing to do here" conclusion Popover.vue's/
  // DropdownMenu.vue's own #onToggle reach for their own close case.
  #onToggle = (event) => {
    if (event.newState !== 'open') return
    this.#refreshFocusable()
    this.#syncTabindex()
    if (this.#focusable.length === 0) return
    const selected = this.#items.find((el) => el.getAttribute('aria-selected') === 'true')
    const target = selected && !this.#isDisabled(selected) ? selected : this.#focusable[0]
    this.#moveFocus(this.#items.indexOf(target))
  }

  // Delegated click listener on the content element: activates whichever
  // option was clicked, ignoring clicks on anything else (and on options
  // belonging to a nested radix-select, if any — mirrors DropdownMenu.vue's
  // own closest(...) !== this guard).
  #onClick = (event) => {
    const item = event.target.closest('.radix-select-item')
    if (!item || item.closest('radix-select') !== this) return
    this.#activateItem(item)
  }

  #onKeydown = (event) => {
    const item = event.target.closest('.radix-select-item')
    if (!item || item.closest('radix-select') !== this) return

    const index = this.#items.indexOf(item)
    if (index === -1) return

    // Enter/Space activates the focused option and closes the listbox —
    // matching DropdownMenu.vue's own Enter/Space-selects-and-closes
    // behavior, handled explicitly here (rather than via a delegated click
    // listener piggybacking on native button activation, as
    // DropdownMenu.vue does) because a plain <div role="option"> has no
    // native Enter/Space-triggers-click behavior of its own.
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      this.#activateItem(item)
      return
    }

    // Arrow-key roving + Home/End jump, reusing DropdownMenu.vue's own
    // exact wrap-around/skip-disabled formula (itself reused from
    // Toolbar.vue) — simplified here to skip only aria-disabled options,
    // since (unlike DropdownMenu.vue's mixed menuitem/separator items)
    // every entry in #items is a real, potentially-selectable option.
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

  // Walks #items one step at a time in `step` direction, wrapping at both
  // ends and skipping disabled options — DropdownMenu.vue's own
  // #findFocusable formula (`(index + step + n) % n`, bounded to at most
  // #items.length steps so a listbox with no enabled option terminates
  // instead of looping forever), reused unchanged except for the simpler
  // (no item-type check) skip predicate.
  #findFocusable(fromIndex, step) {
    const n = this.#items.length
    if (n === 0) return -1
    let index = fromIndex
    for (let i = 0; i < n; i++) {
      index = (index + step + n) % n
      const el = this.#items[index]
      if (!this.#isDisabled(el)) return index
    }
    return -1
  }

  #moveFocus(index) {
    this.#items.forEach((el) => el.setAttribute('tabindex', '-1'))
    const el = this.#items[index]
    el.setAttribute('tabindex', '0')
    el.focus()
  }

  // Sets the native <select>'s own real `.value` (the one thing that would
  // silently submit the wrong value if it were wrong — see header comment)
  // and dispatches a real `change` event on it so any external listener
  // still fires correctly, then closes the listbox. Deliberately does NOT
  // touch the trigger label or aria-selected directly here — the `change`
  // event this dispatches is itself what drives #syncFromNative (via the
  // #onNativeChange listener registered in connectedCallback), so this
  // path and any externally-driven change to the native select share the
  // exact same sync logic. See header comment point 3.
  #activateItem(item) {
    if (this.#isDisabled(item)) return
    this.#nativeSelect.value = item.dataset.value
    this.#nativeSelect.dispatchEvent(new Event('change', { bubbles: true }))
    this.#content.hidePopover()
  }
}

customElements.define('radix-select', RadixSelect)
</script>
