<!--
  Accordion — Radix-inspired exclusive-group disclosure.

  Zero-JS baseline: renders one native <details>/<summary> pair per item.
  Every <details> shares the literal HTML `name="radix-accordion"`
  attribute, so the browser's own native "shared name groups <details>
  elements into one mutually-exclusive set" behavior (a real, spec'd HTML
  feature: https://html.spec.whatwg.org/multipage/interactive-elements.html#the-details-element,
  "name" attribute) makes only one panel open at a time — with zero
  JavaScript. No exclusivity logic is hand-rolled anywhere in this file's
  baseline markup.

  Prop:
    items: Array<{ id: string, title: string, content: string }>
      - id:      stable, unique-within-this-accordion-instance identifier.
                 Used only to build this instance's element ids
                 (e.g. "radix-accordion-<id>-summary"); never sent to the
                 browser as the shared grouping value itself.
      - title:   header text, rendered as plain text inside <summary>.
      - content: panel body, inserted via v-html — callers may pass
                 pre-rendered HTML (e.g. "<p>...</p>"), the same
                 caller-trusted convention used by other components in this
                 codebase (see examples/blog's PostCard.vue, card-excerpt).

  Usage (RFC 014 §6 Example 2):
    <Accordion :items="faqItems" />

  Note on the shared "name" value: it is intentionally a literal string,
  not a prop. This component package (ui/radix) has no built-in notion of
  an *optional* prop with a default — any identifier referenced in a
  template becomes a required prop, and htmlc's default behavior for a
  required-but-unpassed prop is to render a visible "[missing: <name>]"
  placeholder in its place (see component.go's validateProps). Binding
  the shared group name to an optional prop would mean that placeholder
  text could silently end up inside a real HTML attribute value whenever a
  caller forgot to pass it. A hardcoded literal avoids that failure mode
  entirely. Trade-off: multiple <Accordion> instances rendered on the same
  page currently share one native exclusive-group. If per-instance
  grouping is ever needed, that is a good candidate for a small follow-up
  (e.g. an explicit, always-required "name" prop) rather than a change
  made silently here.
-->
<template>
  <div class="radix-accordion">
    <details
      v-for="item in items"
      class="radix-accordion-item"
      name="radix-accordion"
    >
      <summary
        :id="'radix-accordion-' + item.id + '-summary'"
        :aria-controls="'radix-accordion-' + item.id + '-panel'"
      >{{ item.title }}</summary>
      <div
        class="radix-accordion-panel"
        :id="'radix-accordion-' + item.id + '-panel'"
        :aria-labelledby="'radix-accordion-' + item.id + '-summary'"
        v-html="item.content"
      ></div>
    </details>
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

.radix-accordion {
  border-top: 1px solid var(--radix-gray-6);
}

.radix-accordion-item {
  border-bottom: 1px solid var(--radix-gray-6);
}

.radix-accordion-item > summary {
  padding: 0.75rem 0.25rem;
  font-weight: 600;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-accordion-item > summary:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-accordion-panel {
  padding: 0 0.25rem 1rem;
}
</style>

<script customelement>
// Progressive enhancement on top of the native <details>/<summary>
// baseline above. Per RFC 014 §3 Non-Goals, accessibility behavior lives
// here, not in the engine — kept intentionally small.
//
// <details>/<summary> already carry strong native semantics: <summary>
// has an implicit ARIA role of "button" with its aria-expanded state kept
// in sync by the browser automatically, and it is natively focusable and
// keyboard-operable (Enter/Space). This script does not duplicate any of
// that — it adds exactly one thing <details>/<summary> does not provide
// natively: APG-recommended roving arrow-key navigation between an
// accordion's headers (Down/Up move focus to the next/previous header,
// wrapping; Home/End jump to the first/last header). It never touches the
// open/closed state itself, so it cannot fight the native toggle
// behavior or the name-attribute-driven exclusivity from the baseline.
class RadixAccordion extends HTMLElement {
  connectedCallback() {
    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #onKeydown = (event) => {
    const summary = event.target.closest('summary')
    // Ignore keys from anywhere else (e.g. inside a panel's content),
    // and ignore summaries belonging to a nested radix-accordion, if any.
    if (!summary || summary.closest('radix-accordion') !== this) return

    const headers = Array.from(this.querySelectorAll('summary'))
    const index = headers.indexOf(summary)
    if (index === -1) return

    let next
    switch (event.key) {
      case 'ArrowDown': next = (index + 1) % headers.length; break
      case 'ArrowUp':   next = (index - 1 + headers.length) % headers.length; break
      case 'Home':      next = 0; break
      case 'End':       next = headers.length - 1; break
      default: return
    }

    event.preventDefault()
    headers[next].focus()
  }
}

customElements.define('radix-accordion', RadixAccordion)
</script>
