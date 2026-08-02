<!--
  Separator — Radix-inspired semantic/visual divider.

  A purely static, zero-JS component: `role="separator"` plus
  `aria-orientation` when it is meaningfully part of the page's content
  structure, or `aria-hidden="true"` with no `role` at all when it is
  marked purely decorative. There is no interactive state and nothing to
  progressively enhance, so — like VisuallyHidden.vue — this has no
  custom-element script block; the template below is the whole
  implementation.

  Props (both REQUIRED — see the default-value note below):
    orientation: "horizontal" | "vertical"
    decorative:  boolean

  Semantic/accessibility contract (matches Radix Primitives' own, real
  documented behavior for this component; deliberately does not use
  aria-hidden="true" for the decorative case, which is not what upstream
  Radix actually does):
    - decorative == false (meaningful separator):
        role="separator" is present.
        aria-orientation is present only for "vertical". "horizontal" is
        the ARIA-implicit default value of aria-orientation for
        role="separator", so stating it there would be redundant; it is
        omitted and only added for the "vertical" case.
    - decorative == true (purely visual separator):
        role="none" (the ARIA synonym for role="presentation") replaces
        role="separator", removing the "separator" semantic without
        hiding the element from the accessibility tree outright the way
        aria-hidden="true" would. No aria-orientation is emitted, since
        there is no `role="separator"` left for it to qualify.

  Usage:
    <Separator :orientation="'horizontal'" :decorative="false" />
    <Separator :orientation="'vertical'" :decorative="true" />

  On the "both props required" design (the same required-prop trap
  documented in Accordion.vue's and Tabs.vue's header comments): this
  package has no notion of an optional prop with a real default — any
  identifier referenced in the template becomes a required prop, and an
  unpassed required prop is rendered as a visible, truthy
  "[missing: <name>]" placeholder (see component.go's validateProps). A
  `??`-based default (e.g. `:orientation="orientation ?? 'horizontal'"`)
  would not help here: the placeholder string is itself non-nil/non-false,
  so `??` would never fall through to the literal — the placeholder text
  would leak into the rendered `data-orientation`/`aria-orientation`
  attribute values instead. Unlike Accordion's/Tabs's shared "name" value,
  `orientation` and `decorative` are genuinely meant to vary per call site,
  so they cannot be sidestepped with a hardcoded literal either. The
  chosen, safe pattern is simply to document both as always-required and
  have every call site pass them explicitly (as in the usage examples
  above) — matching Radix's documented defaults of orientation="horizontal"
  and decorative={false} by convention at the call site, not by a
  same-file fallback that this templating engine cannot express safely.
-->
<template>
  <Tokens></Tokens>
  <div
    v-if="decorative"
    class="radix-separator"
    :data-orientation="orientation"
    role="none"
  ></div>
  <div
    v-else
    class="radix-separator"
    :data-orientation="orientation"
    role="separator"
    :aria-orientation="orientation === 'vertical' ? 'vertical' : undefined"
  ></div>
</template>

<style>
/*
 * Default appearance: a thin line, sized along the axis *across* the
 * separator's own orientation (a horizontal separator is a full-width
 * line that's thin top-to-bottom; a vertical separator is a full-height
 * line that's thin side-to-side) — matching the conventional look of a
 * divider like <hr>, which this component intentionally does not use
 * (see below).
 *
 * Gotcha: a vertical separator's "full height" is 100% of its *parent's*
 * height, not an intrinsic size — a plain block/inline element with no
 * explicit height collapses to 0 height on its own. The parent must
 * establish a definite height for `height: 100%` to resolve against,
 * e.g. by being a flex container (`display: flex`) with the separator as
 * a flex item, or by giving the parent an explicit height. Without that,
 * a vertical separator will silently render as an invisible 0x0-height
 * line. This mirrors Radix's own documented caveat for this component.
 *
 * Why not <hr>: <hr> has no vertical rendering mode and no clean way to
 * be marked purely decorative independent of its always-implicit
 * "separator" semantics, so — same as upstream Radix — this uses a plain
 * <div> with explicit ARIA/role attributes instead.
 */
.radix-separator {
  background-color: currentColor;
  opacity: 0.2;
  flex-shrink: 0;
}

.radix-separator[data-orientation='horizontal'] {
  width: 100%;
  height: 1px;
}

.radix-separator[data-orientation='vertical'] {
  width: 1px;
  height: 100%;
}
</style>
