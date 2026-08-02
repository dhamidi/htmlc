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
        class="radix-tabs-input radix-visually-hidden-input"
        name="radix-tabs"
        :id="'radix-tabs-' + item.id + '-tab'"
        :checked="index === 0"
      />
      <label
        v-native
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

.radix-tabs {
  position: relative; /* containing block for the absolutely-positioned inputs */
  display: flex;
  flex-wrap: wrap;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-2);
  overflow: hidden;
}

.radix-tabs-label {
  order: 1;
  padding: 0.6rem 1rem;
  cursor: pointer;
  background: var(--radix-gray-3);
  border-right: 1px solid var(--radix-gray-6);
  border-bottom: 3px solid transparent;
  font-weight: 500;
  user-select: none;
}

.radix-tabs-input:checked + .radix-tabs-label {
  background: #fff;
  border-bottom-color: var(--radix-blue-9);
  font-weight: 700;
}

/* Never remove the focus outline — keep it visible for keyboard users.
   The input itself is visually hidden, so its focus ring is drawn on the
   paired label instead. */
.radix-tabs-input:focus-visible + .radix-tabs-label {
  outline: 2px solid var(--radix-blue-9);
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
