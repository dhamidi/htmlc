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
