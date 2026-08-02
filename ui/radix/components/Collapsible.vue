<!--
  Collapsible — Radix-inspired single expand/collapse disclosure.

  Unlike Accordion.vue (a *group* of items with a shared exclusivity
  constraint — only one open at a time, enforced via the native <details>
  "name" attribute), Collapsible is exactly one such item, standalone: no
  group, no exclusivity, no shared "name". It is the same underlying
  disclosure primitive Accordion.vue's own baseline already builds each
  item out of, used on its own.

  Zero-JS baseline: a single native <details :open="open"><summary>...
  behaves as a complete expand/collapse control with no script at all —
  clicking (or activating via keyboard) the <summary> toggles the
  <details>' open state, which is real, native, spec'd behavior
  (https://html.spec.whatwg.org/multipage/interactive-elements.html#the-details-and-summary-elements),
  not something this component hand-rolls.

  Prop:
    open: boolean (required)
      The server-rendered initial state. Bound as :open="open". Per this
      series' falsy-attribute-omission behavior (see Dialog.vue's
      identical note, commit c1671b8), false/nil/undefined omits the
      `open` attribute entirely (closed — the native <details> default);
      true renders a bare `open` attribute (expanded). After the initial
      render, the *browser* owns the open/closed state from then on (the
      user's own clicks/keypresses flip the real DOM `open` attribute
      directly) — this prop only seeds where the disclosure starts.

  Slots:
    trigger — content rendered inside <summary> (the always-visible,
      clickable header — e.g. a heading, an icon + label, whatever the
      caller composes). Named rather than default because there is a
      second, larger content area below that needs the *default* slot for
      itself (see next). This mirrors the codebase's own named-slot
      convention (README.md, "Named slots": e.g. Layout.vue's #header).
    default — the panel body, revealed/hidden by the native <details>
      toggle. Given the default-slot precedent Dialog.vue already
      establishes for "the one big area of arbitrary caller content" (see
      that file's header comment), and that Collapsible, like Dialog, has
      exactly one such freeform content area (unlike Accordion/Tabs' items
      array, there is no repeated list shape here to justify a v-html
      string prop instead), the panel body uses the default slot rather
      than a second named one.

  Usage:
    <Collapsible :open="false">
      <template #trigger>Show details</template>
      <p>Panel body content, composed freely by the caller.</p>
    </Collapsible>

  On the deliberate absence of a customelement enhancement script:

  Radix's own CollapsibleTrigger manually wires aria-controls/aria-expanded
  onto a <button> and CollapsibleContent onto a separately-positioned <div>
  because, in Radix's model, those are two unrelated DOM nodes that need
  an explicit ARIA relationship stitched between them. That gap does not
  exist here: <summary> and the rest of a <details> element's content are
  *structurally* one disclosure widget as far as the HTML parser and every
  mainstream browser/AT combination that implements <details> are
  concerned — the association is the DOM nesting itself, not an
  aria-controls/aria-labelledby pointer this component would otherwise
  have to hand-roll (and, absent a caller-supplied id, could not even
  build correctly for multiple same-page instances without risking
  duplicate ids).

  This is not assumed by analogy with Switch.vue/RadioGroup.vue's own
  no-script conclusions — it was checked directly against MDN's technical
  summaries for both elements before writing this comment. <details>'
  documented implicit ARIA role is "group"; <summary>'s is explicitly "no
  corresponding role" with MDN's own accessibility note that the exact
  role assistive technology exposes for <summary> varies by browser. So,
  unlike Switch's `role="switch"` + native `:checked` → `aria-checked`
  mapping (a mapping MDN/HTML-AAM affirmatively document), this component
  does not claim any spec-guaranteed aria-expanded auto-sync on <summary>.
  What it does claim, and what holds regardless of that variance, is
  narrower and already true today: <details>/<summary> is universally
  recognized by browsers as *a* native disclosure widget (not a generic
  block of unrelated markup), it is keyboard-operable out of the box, and
  its real DOM `open` attribute/`toggle` event give any interested party
  — CSS via the `:open` pseudo-class, or a caller's own script via
  `addEventListener('toggle', ...)` — a live, accurate signal with zero
  JS contributed by this component. Accordion.vue's own baseline accepts
  that identical trade-off already (it adds a script only for roving
  arrow-key navigation *between* multiple headers, a concern that does
  not exist here since a standalone Collapsible has exactly one
  focusable trigger — nothing to rove between). There is nothing left for
  a script to add.
-->
<template>
  <Tokens></Tokens>
  <details class="radix-collapsible" :open="open">
    <summary class="radix-collapsible-trigger"><slot name="trigger"></slot></summary>
    <div class="radix-collapsible-content"><slot></slot></div>
  </details>
</template>

<style>
.radix-collapsible {
  border: 1px solid var(--radix-sand-6);
  border-radius: var(--radix-radius-3);
}

.radix-collapsible-trigger {
  padding: 0.75rem 1rem;
  font-weight: 600;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-collapsible-trigger:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

.radix-collapsible-content {
  padding: 0 1rem 1rem;
}
</style>
