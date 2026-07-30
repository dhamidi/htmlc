<!--
  Tabs — Radix-inspired tabbed interface.

  Zero-JS baseline: the CSS-only radio-input tabs technique. Each item
  renders three *flat* siblings, in this exact repeating order:

    <input type="radio">  (visually hidden, but focusable/keyboard-operable)
    <label>                (the visible tab button, via native for/id)
    <div class="...panel"> (the tab's content panel)

  repeated once per item, all as direct children of the wrapping
  .radix-tabs div — never nested per-item, because the CSS below depends on
  the panel being the input's *next-next* adjacent sibling:

    .radix-tabs-input:checked + .radix-tabs-label + .radix-tabs-panel {
      display: block;
    }

  All radios share the literal HTML `name="radix-tabs"` attribute, so the
  browser groups them into one native mutually-exclusive radio group (real,
  spec'd HTML form behavior — no exclusivity logic hand-rolled here). A
  native radio group already gives arrow-key roving between options and a
  single Tab-key stop for the whole group, which is a close starting point
  for APG tab roving-tabindex behavior; the first radio is checked by
  default so there is a sensible initial selection with zero JS and no
  URL/fragment dependency.

  Visually, tab buttons need to read as a horizontal strip with one shared
  content panel below — but that only works if the (single) visible panel
  always renders *after every* label, and interleaved flat siblings alone
  would place it right after only its own label instead. `.radix-tabs` is a
  flex container that reconciles this: labels get `order: 1`, panels get
  `order: 2` and `width: 100%`. Flex `order` reflows *visual* position
  without touching DOM order, so the `:checked + label + panel` adjacency
  the CSS-only selector needs keeps working while the layout still reads as
  "row of tabs, then the active panel below". The hidden radios use
  `position: absolute` so they never occupy a flex slot themselves.

  Props:
    items: Array<{ id: string, label: string, content: string }>
      - id:      stable, unique-within-this-Tabs-instance identifier. Used
                 only to build this instance's element ids
                 (e.g. "radix-tabs-<id>-tab"); never sent to the browser as
                 the shared radio-group name itself.
      - label:   tab button text, rendered as plain text inside <label>.
      - content: panel body, inserted via v-html — callers may pass
                 pre-rendered HTML, the same caller-trusted convention used
                 elsewhere in this codebase (see Accordion.vue, and
                 examples/blog's PostCard.vue card-excerpt).

  Usage:
    <Tabs :items="tabItems" />

  Note on the shared "name" value: it is a hardcoded literal, not a prop —
  same rationale and the same accepted trade-off as Accordion.vue's
  name="radix-accordion". This package has no notion of an optional prop
  with a real default: any identifier referenced in a template becomes a
  required prop, and an unpassed required prop renders a visible, truthy
  "[missing: <name>]" placeholder in its place (see component.go's
  validateProps) — a pattern like `:name="groupName ?? 'default'"` would
  never fall through to the literal, because the placeholder string itself
  is truthy. A hardcoded literal sidesteps that failure mode entirely.
  Trade-off: multiple <Tabs> instances rendered on the same page currently
  share one native radio group, so selecting a tab in one instance would
  also uncheck the selected tab in every other instance on the page (no
  tab left checked there). If per-instance grouping is ever needed, a
  small, explicit, always-required "name" prop is a reasonable follow-up
  rather than a change made silently here.
-->
<template>
  <div class="radix-tabs">
    <template v-for="(item, index) in items">
      <input
        type="radio"
        class="radix-tabs-input"
        name="radix-tabs"
        :id="'radix-tabs-' + item.id + '-tab'"
        :checked="index === 0"
      />
      <label
        class="radix-tabs-label"
        :id="'radix-tabs-' + item.id + '-tab-label'"
        :for="'radix-tabs-' + item.id + '-tab'"
        :aria-controls="'radix-tabs-' + item.id + '-panel'"
      >{{ item.label }}</label>
      <div
        class="radix-tabs-panel"
        :id="'radix-tabs-' + item.id + '-panel'"
        :aria-labelledby="'radix-tabs-' + item.id + '-tab-label'"
        v-html="item.content"
      ></div>
    </template>
  </div>
</template>

<style scoped>
.radix-tabs {
  position: relative; /* containing block for the absolutely-positioned inputs */
  display: flex;
  flex-wrap: wrap;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  overflow: hidden;
}

/* Visually hidden — but never display:none, which would drop the radio
   from focus/keyboard reach entirely and break the zero-JS keyboard story.
   Taken out of flow so it never claims a flex slot of its own; the
   checked/focus state is read via the sibling selectors below. */
.radix-tabs-input {
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

.radix-tabs-label {
  order: 1;
  padding: 0.6rem 1rem;
  cursor: pointer;
  background: #f5f5f5;
  border-right: 1px solid #d9d9d9;
  border-bottom: 3px solid transparent;
  font-weight: 500;
  user-select: none;
}

.radix-tabs-input:checked + .radix-tabs-label {
  background: #fff;
  border-bottom-color: #2563eb;
  font-weight: 700;
}

/* Never remove the focus outline — keep it visible for keyboard users.
   The input itself is visually hidden, so its focus ring is drawn on the
   paired label instead. */
.radix-tabs-input:focus-visible + .radix-tabs-label {
  outline: 2px solid #2563eb;
  outline-offset: -2px;
}

.radix-tabs-panel {
  order: 2;
  width: 100%;
  display: none;
  padding: 1rem;
}

.radix-tabs-input:checked + .radix-tabs-label + .radix-tabs-panel {
  display: block;
}
</style>

<script customelement>
// Progressive enhancement on top of the CSS-only radio-input baseline
// above. Per RFC 014 §3 Non-Goals, accessibility behavior lives here, not
// in the engine — kept intentionally small.
//
// Design choice: layer ARIA tablist/tab/tabpanel semantics on top of the
// existing radio/label/panel structure rather than replacing the radio
// inputs outright. The inputs keep doing exactly what they already do in
// the baseline (holding the "which tab is selected" state, and driving
// panel visibility via the :checked CSS chain) — this script never
// reimplements that. What it changes is *which element assistive tech and
// the keyboard operate on*: the <label> becomes the real "tab" (role="tab",
// roving tabindex, aria-selected), while the underlying <input> is taken
// out of the accessibility tree and the sequential tab order
// (aria-hidden="true", tabindex="-1") so a screen reader announces one
// coherent tablist instead of both a tab and a redundant radio button for
// the same control. This mirrors well-established practice for CSS-only
// radio tabs (e.g. the pattern described by Sara Soueidan / Scott O'Hara)
// and follows APG's "automatic activation" tabs model: arrow keys move
// focus *and* selection together, so no separate Enter/Space activation
// step is needed.
class RadixTabs extends HTMLElement {
  #inputs = []
  #labels = []
  #panels = []

  connectedCallback() {
    const container = this.querySelector('.radix-tabs')
    if (!container) return
    // The panels are structurally nested inside the same flat container as
    // the tabs (a requirement of the CSS-only technique above), so
    // role="tablist" ends up on their shared ancestor rather than on a
    // tabs-only wrapper. This is a known, pragmatic simplification for
    // CSS-only tab implementations; it does not affect the individual
    // role="tab"/role="tabpanel" semantics set below.
    container.setAttribute('role', 'tablist')

    this.#inputs = Array.from(this.querySelectorAll('.radix-tabs-input'))
    this.#labels = Array.from(this.querySelectorAll('.radix-tabs-label'))
    this.#panels = Array.from(this.querySelectorAll('.radix-tabs-panel'))

    this.#labels.forEach((label, i) => {
      const input = this.#inputs[i]
      const panel = this.#panels[i]
      if (!input || !panel) return

      label.setAttribute('role', 'tab')
      panel.setAttribute('role', 'tabpanel')

      input.setAttribute('tabindex', '-1')
      input.setAttribute('aria-hidden', 'true')
      // Native label-click focus behavior would otherwise still be able to
      // move focus onto the now aria-hidden input; redirect it back to the
      // label, which is the element real keyboard/AT interaction targets.
      input.addEventListener('focus', () => label.focus())

      label.tabIndex = input.checked ? 0 : -1
      label.setAttribute('aria-selected', String(input.checked))

      input.addEventListener('change', () => this.#syncSelection(i))
    })

    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #syncSelection(selectedIndex) {
    this.#labels.forEach((label, i) => {
      const selected = i === selectedIndex
      label.setAttribute('aria-selected', String(selected))
      label.tabIndex = selected ? 0 : -1
    })
  }

  #onKeydown = (event) => {
    const label = event.target.closest('.radix-tabs-label')
    // Ignore keys from anywhere else (e.g. inside a panel's content), and
    // ignore labels belonging to a nested radix-tabs, if any.
    if (!label || label.closest('radix-tabs') !== this) return

    const index = this.#labels.indexOf(label)
    if (index === -1) return

    let next
    switch (event.key) {
      case 'ArrowRight': next = (index + 1) % this.#labels.length; break
      case 'ArrowLeft':  next = (index - 1 + this.#labels.length) % this.#labels.length; break
      case 'Home':       next = 0; break
      case 'End':        next = this.#labels.length - 1; break
      default: return
    }

    event.preventDefault()
    this.#selectAndFocus(next)
  }

  #selectAndFocus(index) {
    const input = this.#inputs[index]
    if (!input) return
    input.checked = true
    input.dispatchEvent(new Event('change', { bubbles: true }))
    this.#labels[index].focus()
  }
}

customElements.define('radix-tabs', RadixTabs)
</script>
