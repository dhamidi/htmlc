<!--
  Toolbar — Radix-inspired container that groups a row (or column) of
  related controls under one keyboard-navigable roving-tabindex region,
  per APG's toolbar pattern.

  ## Why `items: Array<{type, ...}>` instead of composed child components

  Radix's real Toolbar is a composable container: a `<Toolbar.Root>`
  provides roving-tabindex/keyboard-navigation *context* to arbitrary
  children (`Toolbar.Button`, `Toolbar.Link`, `Toolbar.ToggleGroup`,
  `Toolbar.Separator`) via React context and `asChild`/`cloneElement`. That
  composition pattern does not translate to htmlc's server-rendered
  templating model: there is no mechanism here for a parent component to
  reach into and modify already-rendered slot content the way React's
  context/cloneElement can — a `<slot>` inserts inert markup, and this
  component has no way to walk it back out and prop-inject
  roving-tabindex/keydown wiring onto whatever a caller happened to put
  inside. Every prior ui/radix component that needed to coordinate a set of
  related sub-items (Accordion.vue, Tabs.vue, RadioGroup.vue,
  ToggleGroup.vue) sidestepped this exact problem the same way: an
  `items: Array<{...}>` prop, with the parent component itself rendering
  the appropriate markup for each item internally, rather than accepting
  arbitrary composed child components via slots. This file follows that
  same established convention.

  ## `items[].type` scope: "button" and "separator" in this v1

  `type` discriminates what each entry renders as:
    - "button":    a plain action control — `id`, `label`, `disabled`.
    - "separator": a visual divider between groups of controls — same
                    role="separator"/aria-orientation ARIA contract
                    Separator.vue already established (matched inline
                    below, rather than literally invoking that component,
                    since these separators need a specific perpendicular
                    orientation relationship with the toolbar's own
                    `orientation` — see below).

  Radix's own Toolbar also offers `Toolbar.Link` and `Toolbar.ToggleGroup`.
  This first pass deliberately does not reimplement either:
    - A link-styled item is a reasonable, self-contained, purely additive
      follow-up (a third `type: "link"` branch with `href`/`label`) — left
      out here to keep this pass reasonably scoped, not because it is hard.
    - Toggle-button behavior (pressed/unpressed, single- vs multi-select
      exclusivity) is *already* ToggleGroup.vue's entire job, and
      reimplementing that state machine a second time inside this file's
      own item-type switch — just to let one `items` entry claim to behave
      like a toggle button — would duplicate real, non-trivial logic this
      package already has one correct copy of. The honest way to combine
      the two is for a *caller's own template* to compose a `<ToggleGroup>`
      instance as one of the toolbar's visual sections (e.g. rendered
      alongside `<Toolbar>` inside a caller-authored flex row, or via
      `<ToggleGroup>` embedded in a slot this file does not currently even
      expose) rather than Toolbar's own `items` loop pretending to be
      ToggleGroup internally. Both directions — adding "link" and composing
      with ToggleGroup — are believed to be self-contained, additive
      follow-ups on top of what is implemented here, not blocked by
      anything in this file's structure.

  ## `orientation`: layout axis, arrow-key axis, and separator axis

  `orientation` is `"horizontal"` | `"vertical"`, matching Tabs.vue's/
  ToggleGroup.vue's own horizontal-vs-vertical vocabulary. It drives three
  things at once, mirroring Radix's own documented Toolbar behavior:
    1. `aria-orientation` on the root — set unconditionally to `orientation`
       (unlike Separator.vue's own conditional-omit-when-horizontal
       convention, which applies to role="separator" specifically; Radix's
       real Toolbar always emits `aria-orientation` on the toolbar root
       itself, so this file matches that instead).
    2. Which arrow keys move roving focus in the `<script customelement>`
       below: ArrowLeft/ArrowRight for "horizontal" (the default toolbar
       shape — a horizontal row of controls), ArrowUp/ArrowDown for
       "vertical". Always wraps at both ends, matching Tabs.vue's/
       ToggleGroup.vue's own established wrap-around convention — Radix's
       real RovingFocusGroup also defaults `loop` to `true`, so this is not
       a new choice, just the same one already made twice in this package.
    3. Each separator item's own orientation: perpendicular to the
       toolbar's, exactly matching Radix's own documented
       ToolbarSeparator behavior (a horizontal toolbar's dividers are
       drawn as short *vertical* lines between button groups, and vice
       versa) — see the template below.

  ## Static roving-tabindex baseline, and why it is bounded

  Per APG's roving-tabindex convention, exactly one item is a tab stop
  (`tabindex="0"`) and every other interactive item is removed from the
  sequential tab order (`tabindex="-1"`) — separators are never part of the
  tab order at all (no `tabindex` attribute on them, matching
  Separator.vue). "Which one" needs to be the *first* item that is both
  `type === 'button'` and not `disabled`, per this component's own design
  brief — unlike ToggleGroup.vue's homogeneous item list (where a plain
  `index === 0` check was always correct, because ToggleGroup's disabled
  state is scoped to the whole group, never per item), Toolbar's items are
  heterogeneous and each button carries its own independent `disabled`, so
  the true first tab stop can sit at any position depending on how many
  leading separators/disabled buttons precede it.

  This package's expression language (expr) has no loop, no `.find`/
  `.filter`/reduce, and no way to register a per-file helper function (see
  ToggleGroup.vue's own header comment, "the value-shape design decision",
  for the identical constraint applied to a different problem) — so a
  single expr expression genuinely cannot search an arbitrary-length array
  for "the first index satisfying some predicate". What expr *can* do is a
  right-associative ternary chain, which is exactly a bounded, unrolled
  version of that search: checking `items[0]`, then `items[1]`, ... up to a
  fixed depth, each guarded by `items.length > i &&` so indexing never runs
  past the end of a shorter array (out-of-bounds indexing is a hard runtime
  error in this engine, not `undefined` — see expr/eval.go's slice/array
  case). The `:tabindex` expression below unrolls this check for the first
  6 positions (`items[0]` through `items[5]`) — comfortably more leading
  separators/disabled buttons than any real toolbar would plausibly stack
  before its first real control — and falls back to `-1` for every item if
  none of the first 6 positions qualifies.

  This bound is a deliberate, honest v1 scope call, not a silent gap: if a
  caller's `items` array has more than 6 leading non-interactive entries,
  the static (pre-JS) baseline degrades safely to "no tab stop" rather than
  picking a wrong one — and the `<script customelement>` below
  unconditionally recomputes the correct tab stop from the live DOM on
  `connectedCallback` regardless of array length (real JS has no such
  bound; it just walks the actual rendered children), so the enhanced,
  scripted toolbar is always exactly right. The bound only affects the
  degraded, no-JS-yet render of an unusually front-loaded item list, for an
  instant before the script runs.

  ## `<script customelement>` (load-bearing, not just polish)

  1. On connect, recomputes and re-applies the roving-tabindex baseline
     from the live DOM (see above) — a real correctness fix for arrays
     longer than the static baseline's bound, and a no-op (same result) for
     everything within it.
  2. Arrow-key roving focus: `#findFocusable` walks the full item list
     (buttons *and* separators, in DOM order) one step at a time in the
     requested direction, wrapping at both ends, skipping any element that
     is not an enabled button — this is the one genuinely new piece of
     logic this component needed beyond what Tabs.vue/ToggleGroup.vue
     already established: a plain modulo index walk
     (`(index + 1) % length`) is not enough once some entries in the list
     are not focusable at all. Hand-traced for `[button, separator,
     button, button]`, "next" from index 0: step lands on index 1 first
     (the separator), which fails the focusable check, so the walk takes a
     second step to index 2 (a button), which passes — matching the
     expected "skip the separator, land on the next real button" behavior.
     Home/End reuse the same walk, started just before index 0 / just
     after the last index, so they naturally resolve to the first/last
     *focusable* item rather than the first/last *item*.
  3. No click handling: a real `<button>` already provides free, correct
     click/Enter/Space activation, and a plain action button (unlike
     ToggleGroup's pressed/unpressed items) has no additional state for
     this component to own or synchronize — a caller that wants to react to
     a toolbar button's activation can listen for the native, bubbling
     `click` event on this element and read the clicked button's
     `data-id` (see below), the same way any plain `<button>` click is
     normally observed; no custom event is invented here since nothing
     needs to travel through one.

  Props (both REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):
    items:       Array<{ type, id, label, disabled }> — matching
                 Accordion.vue's/Tabs.vue's/RadioGroup.vue's/
                 ToggleGroup.vue's own `items` shape convention.
                   - type:     "button" | "separator" (see above).
                   - id:       stable, unique-within-this-instance
                               identifier for a "button" item, rendered as
                               `data-id` so a caller's own click listener
                               can identify which button was activated.
                               Not read for a "separator" item.
                   - label:    a "button" item's visible text, rendered as
                               plain text. Not read for a "separator" item.
                   - disabled: boolean — a "button" item's own independent
                               disabled state. Like ToggleGroup.vue's own
                               per-item `pressed` field (see that file's
                               header comment), this is read via plain
                               v-for item-scope member access, not a
                               Component.Props()-tracked identifier — a
                               "button" item that omits `disabled` simply
                               evaluates it as expr's falsy UndefinedValue
                               ("not disabled"), never a "[missing: ...]"
                               placeholder. Not read for a "separator" item.
    orientation: "horizontal" | "vertical" — see above.

  Usage:
    <Toolbar
      orientation="horizontal"
      :items="[
        { type: 'button', id: 'bold', label: 'B' },
        { type: 'button', id: 'italic', label: 'I' },
        { type: 'separator' },
        { type: 'button', id: 'link', label: 'Link', disabled: true },
        { type: 'button', id: 'more', label: 'More' },
      ]"
    />
-->
<template>
  <div
    class="radix-toolbar"
    role="toolbar"
    :aria-orientation="orientation"
    :data-orientation="orientation"
  >
    <template v-for="(item, index) in items">
      <button
        v-if="item.type === 'button'"
        type="button"
        class="radix-toolbar-item radix-toolbar-button"
        :data-id="item.id"
        :disabled="item.disabled"
        :tabindex="index === (items.length > 0 && items[0].type === 'button' && !items[0].disabled ? 0 : items.length > 1 && items[1].type === 'button' && !items[1].disabled ? 1 : items.length > 2 && items[2].type === 'button' && !items[2].disabled ? 2 : items.length > 3 && items[3].type === 'button' && !items[3].disabled ? 3 : items.length > 4 && items[4].type === 'button' && !items[4].disabled ? 4 : items.length > 5 && items[5].type === 'button' && !items[5].disabled ? 5 : -1) ? '0' : '-1'"
      >{{ item.label }}</button>
      <div
        v-else-if="item.type === 'separator'"
        class="radix-toolbar-item radix-toolbar-separator"
        role="separator"
        :data-orientation="orientation === 'vertical' ? 'horizontal' : 'vertical'"
        :aria-orientation="orientation === 'vertical' ? undefined : 'vertical'"
      ></div>
    </template>
  </div>
</template>

<style scoped>
.radix-toolbar {
  display: inline-flex;
  flex-direction: row;
  align-items: center;
  gap: 0.35rem;
  padding: 0.35rem;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
}

.radix-toolbar[data-orientation='vertical'] {
  flex-direction: column;
  align-items: stretch;
}

.radix-toolbar-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.7rem;
  background-color: #fff;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  line-height: 1;
}

.radix-toolbar-button:hover:not(:disabled) {
  background-color: #f5f5f5;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-toolbar-button:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 2px;
}

.radix-toolbar-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

/*
 * Drawn perpendicular to the toolbar's own axis (see the header comment's
 * `orientation` section): a horizontal toolbar's separators are short
 * vertical dividers that should span the row's height, sized along the
 * same width/height contract Separator.vue itself uses; a vertical
 * toolbar's separators are horizontal dividers spanning the column's
 * width instead.
 */
.radix-toolbar-separator {
  background-color: currentColor;
  opacity: 0.2;
  flex-shrink: 0;
}

.radix-toolbar[data-orientation='vertical'] .radix-toolbar-separator {
  width: 100%;
  height: 1px;
}

.radix-toolbar:not([data-orientation='vertical']) .radix-toolbar-separator {
  width: 1px;
  align-self: stretch;
}
</style>

<script customelement>
// Progressive enhancement on top of the bounded-but-honest zero-JS
// baseline above (see the header comment's "Static roving-tabindex
// baseline" section for why the pre-JS render is only *reliably* correct
// up to 6 leading non-interactive items). Per RFC 014 §3 Non-Goals, this
// reimplements nothing the browser already provides for free — a plain
// <button> and a plain <div role="separator"> have no native
// roving-tabindex behavior across a set of sibling elements for the
// browser to supply, so that part is genuinely this script's job, the
// same conclusion Tabs.vue's/ToggleGroup.vue's own header comments reach
// for their own roving-tabindex groups.
class RadixToolbar extends HTMLElement {
  #container = null
  #orientation = 'horizontal'
  #items = [] // every item element (buttons AND separators), in DOM order
  #focusable = [] // subset of #items: enabled buttons only, in DOM order

  connectedCallback() {
    this.#container = this.querySelector('.radix-toolbar')
    if (!this.#container) return

    this.#orientation = this.#container.getAttribute('data-orientation') === 'vertical' ? 'vertical' : 'horizontal'
    this.#items = Array.from(this.querySelectorAll('.radix-toolbar-item'))
    this.#refreshFocusable()

    // Recompute the tab stop from the live DOM unconditionally. This is a
    // genuine correctness fix (not just a style refresh) for any `items`
    // array longer than the static baseline's bound documented above —
    // real JS has no such bound, it just walks the elements that actually
    // rendered — and is a harmless no-op when the static baseline already
    // got it right.
    this.#syncTabindex()

    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #refreshFocusable() {
    this.#focusable = this.#items.filter((el) => el.classList.contains('radix-toolbar-button') && !el.disabled)
  }

  #syncTabindex() {
    this.#focusable.forEach((el, i) => el.setAttribute('tabindex', i === 0 ? '0' : '-1'))
  }

  #onKeydown = (event) => {
    const item = event.target.closest('.radix-toolbar-button')
    // Ignore keys from anywhere else, and ignore buttons belonging to a
    // nested radix-toolbar, if any (mirrors Tabs.vue's/ToggleGroup.vue's
    // own closest(...) !== this guard).
    if (!item || item.closest('radix-toolbar') !== this) return

    const index = this.#items.indexOf(item)
    if (index === -1) return

    const nextKey = this.#orientation === 'vertical' ? 'ArrowDown' : 'ArrowRight'
    const prevKey = this.#orientation === 'vertical' ? 'ArrowUp' : 'ArrowLeft'

    let target
    switch (event.key) {
      case nextKey: target = this.#findFocusable(index, 1); break
      case prevKey: target = this.#findFocusable(index, -1); break
      // Search forward from just before the first item / backward from
      // just after the last item, so these naturally resolve to the
      // first/last *focusable* item rather than the first/last item.
      case 'Home': target = this.#findFocusable(-1, 1); break
      case 'End': target = this.#findFocusable(this.#items.length, -1); break
      default: return
    }
    if (target === -1) return

    event.preventDefault()
    this.#moveFocus(target)
  }

  // Walks #items (buttons AND separators) one step at a time in `step`
  // direction, wrapping at both ends — deliberately matching Tabs.vue's/
  // ToggleGroup.vue's own established wrap-around convention
  // ((index + step + length) % length) — but, unlike either of those
  // homogeneous item lists, keeps stepping past any element that is not
  // an enabled button instead of stopping on whatever index is next. A
  // plain modulo index walk on its own is not enough once some entries in
  // the list (separators, disabled buttons) are not focusable at all.
  // Bounded to at most #items.length steps, so a toolbar with no
  // focusable item at all terminates instead of looping forever.
  #findFocusable(fromIndex, step) {
    const n = this.#items.length
    if (n === 0) return -1
    let index = fromIndex
    for (let i = 0; i < n; i++) {
      index = (index + step + n) % n
      const el = this.#items[index]
      if (el.classList.contains('radix-toolbar-button') && !el.disabled) return index
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

customElements.define('radix-toolbar', RadixToolbar)
</script>
