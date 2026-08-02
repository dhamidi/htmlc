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
  <Tokens></Tokens>
  <span class="radix-visually-hidden"><slot></slot></span>
</template>

<style>
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
