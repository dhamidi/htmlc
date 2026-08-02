<!--
  AspectRatio — Radix-inspired fixed width:height ratio container.

  Keeps its slotted content proportioned to a fixed width-to-height ratio
  regardless of the content's own intrinsic size (the classic use case is a
  16:9 video/image embed that must stay 16:9 as the surrounding layout
  resizes), by constraining the *container* rather than the content itself.

  Prop (REQUIRED — see the default-value note below):
    ratio: number
      Expected as "width divided by height", e.g. `16/9` for a widescreen
      video, `1` (or `1/1`) for a square, `4/3` for a classic-TV frame.
      This templating engine's expr evaluator performs real floating-point
      division for `/` (see numericBinOp in expr/eval.go), so an
      expression like `:ratio="16/9"` evaluates to the expected ~1.778
      rather than truncating to an integer — passing a literal fraction
      directly at the call site works as expected, no pre-division needed:

        <AspectRatio :ratio="16/9">
          <img src="..." style="width: 100%; height: 100%; object-fit: cover;" />
        </AspectRatio>

  Implementation choice — modern CSS `aspect-ratio`, not the classic
  "padding-bottom percentage trick":

  Radix Primitives' own upstream implementation uses the padding-bottom
  trick (an outer relatively-positioned, zero-height element with
  `padding-bottom: calc(100% / ratio)`, plus an inner absolutely-positioned
  `inset: 0` element carrying the actual content) specifically because it
  needs to support browsers old enough to lack the CSS `aspect-ratio`
  property outright. This port targets a different, already-established
  baseline: Dialog.vue in this same directory already assumes a
  <dialog>-with-showModal()-capable browser as its zero-JS baseline (see
  that file's header comment), and Separator.vue/VisuallyHidden.vue carry
  no such legacy-compatibility concessions either. Matching that existing
  modernity bar, this component uses the CSS `aspect-ratio` property
  directly: one declaration on the container, no extra wrapper element, no
  positioning trickery required to keep an inner element in sync with an
  outer one. `aspect-ratio` has been supported in all major evergreen
  browsers since 2021, well within the bar this library's other components
  already assume.

  Unlike upstream's two-element structure (outer sizing wrapper + inner
  absolutely-positioned content holder), the modern property needs only
  one element: the ratio is enforced on the container itself, and slotted
  content can size itself to fill that container with ordinary CSS (e.g.
  `width: 100%; height: 100%; object-fit: cover` on a slotted <img>/<video>,
  same as callers would do for the inner element in upstream's version).

  No custom-element enhancement script: this is a pure CSS technique with
  no interactive state and nothing to progressively enhance — same
  reasoning as VisuallyHidden.vue and Separator.vue.

  On "ratio" being a required prop with no same-file default: this package
  has no notion of an optional prop with a real default — any identifier
  referenced in the template becomes a required prop, and an unpassed
  required prop renders as a visible, truthy "[missing: ratio]" placeholder
  (see component.go's validateProps), so a `??`-based fallback would leak
  that placeholder text into the rendered `aspect-ratio` CSS value instead
  of falling through to a literal. Radix's own documented default is
  `ratio={1}` (a square) by convention at the call site; this port
  documents the same expectation here rather than attempting a same-file
  fallback this templating engine cannot express safely (see Separator.vue's
  header comment for the identical trade-off on its own required props).
-->
<template>
  <div class="radix-aspect-ratio" data-radix-aspect-ratio-wrapper="" :style="{ aspectRatio: ratio }">
    <slot></slot>
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

/*
 * `aspect-ratio` alone establishes the box's width-to-height relationship;
 * `width: 100%` makes the container actually fill its parent so that
 * relationship has a real width to compute a height from (an
 * `aspect-ratio` on an element with no definite width/height in either
 * axis still needs *one* axis pinned down — width is the conventional
 * choice here, matching how this ratio is meant to be used: a
 * layout-width-driven box whose height then follows from `ratio`).
 * `overflow: hidden` keeps slotted content (e.g. an oversized image) from
 * spilling outside the ratio-constrained box; content is expected to size
 * itself to fill the container (see header comment) rather than this
 * component clipping arbitrary content by accident.
 */
.radix-aspect-ratio {
  width: 100%;
  overflow: hidden;
}
</style>
