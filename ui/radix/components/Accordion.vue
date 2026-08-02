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
  <Tokens></Tokens>
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
