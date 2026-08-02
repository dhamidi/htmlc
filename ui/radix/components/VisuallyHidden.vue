<!--
  VisuallyHidden — Radix-inspired sr-only wrapper.

  Hides its default-slot content visually while keeping it available to
  assistive technology (it stays in the accessibility tree) and, if the
  content contains a focusable element, in the tab order. This is the
  well-known "visually-hidden" / "sr-only" CSS clipping technique used
  across the accessibility community (Bootstrap's .visually-hidden, WordPress's
  .screen-reader-text, the older "CSS clip" pattern, etc.) — not something
  unique to Radix.

  Deliberately NOT the `display` or `visibility` CSS properties (setting
  either to their "hidden" keyword removes an element, and its
  descendants, from the accessibility tree, which would defeat the entire
  purpose — screen readers would skip the content too). Instead this pins
  the element to a 1x1px box, clips its
  overflow, and pulls it fully out of the visual flow while leaving it
  genuinely rendered, so it stays exposed to assistive tech and reachable
  by keyboard (Tab) if it contains focusable content.

  No prop beyond the slot content — matches Dialog.vue's default-slot
  pattern (README.md, "Default slot"):

    <VisuallyHidden>Extra context for screen reader users only</VisuallyHidden>

  A common real-world use: pairing an icon-only button with a text label
  that sighted users don't need to see but screen reader users do, e.g.
  <button><XIcon /><VisuallyHidden>Close</VisuallyHidden></button>.

  No custom-element enhancement script: this is a pure CSS technique with no
  interactive state, no event handling, and nothing to progressively
  enhance — unlike this library's other components (Accordion, Tabs,
  Dialog), which all layer JS behavior on top of a zero-JS baseline. There
  is no baseline/enhancement split here because there is no JS side at
  all; the <style scoped> block below is the entire implementation.
-->
<template>
  <span class="radix-visually-hidden"><slot></slot></span>
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

/*
 * The classic sr-only/visually-hidden clip technique. Every declaration
 * below is load-bearing for keeping the element out of the visual flow
 * while remaining part of the accessibility tree and the tab order:
 *
 *   position: absolute   Removes the element from normal document flow
 *                         (so it doesn't affect surrounding layout) without
 *                         removing it from the accessibility tree the way
 *                         the display/visibility "hidden" properties would.
 *   width/height: 1px     Collapses the box to the smallest non-zero size.
 *                         Deliberately NOT 0x0 — some browsers/AT
 *                         implementations treat a genuinely zero-area box
 *                         as invisible/unreachable; 1px avoids that.
 *   padding: 0 / border: 0
 *                         Prevents padding or a border from inflating the
 *                         1px box back to something visible.
 *   margin: -1px          Cancels the 1px box out of the layout so it
 *                         still doesn't reserve visible space.
 *   overflow: hidden       Clips any content larger than the 1x1px box so
 *                         nothing visibly spills out.
 *   clip: rect(0 0 0 0) + clip-path: inset(50%)
 *                         Belt-and-suspenders visual clipping: `clip` is
 *                         the long-established (if deprecated) property
 *                         still relied on by some AT/browser
 *                         combinations; `clip-path: inset(50%)` is the
 *                         modern equivalent. Neither one touches the
 *                         accessibility tree or focusability the way
 *                         display/visibility would.
 *   white-space: nowrap    Without this, a collapsed 1px-wide box can wrap
 *                         text onto many lines, some of which may render
 *                         outside the clipped region in certain browsers.
 */
.radix-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}
</style>
