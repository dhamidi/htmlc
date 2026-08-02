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
  <details class="radix-collapsible" :open="open">
    <summary class="radix-collapsible-trigger"><slot name="trigger"></slot></summary>
    <div class="radix-collapsible-content"><slot></slot></div>
  </details>
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

.radix-collapsible {
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
}

.radix-collapsible-trigger {
  padding: 0.75rem 1rem;
  font-weight: 600;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-collapsible-trigger:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-collapsible-content {
  padding: 0 1rem 1rem;
}
</style>
