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
  <Tokens></Tokens>
  <div class="radix-aspect-ratio" data-radix-aspect-ratio-wrapper="" :style="{ aspectRatio: ratio }">
    <slot></slot>
  </div>
</template>

<style>
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
