<!--
  Slider — Radix-inspired draggable value slider, generalized to support any
  number of thumbs (a plain single-value slider, a two-thumb min/max range
  slider, or more) from one shared `values: number[]` prop.

  ## The multi-thumb design problem

  Native HTML has exactly one range-input widget, <input type="range">, and
  it only ever has one thumb — there is no native multi-thumb range control
  to reach for the way Checkbox.vue/Switch.vue/RadioGroup.vue lean on a
  single native form element each. Radix's own real Slider solves this by
  hand-rolling pointer-drag math and keyboard value-stepping entirely in
  script, driving N absolutely-positioned custom `role="slider"` spans
  itself. That is real, necessary complexity for a *from-scratch* slider,
  but this port does not need to pay that cost: this file instead stacks N
  real, separate native <input type="range"> elements, one per entry in
  `values`, all absolutely positioned on top of one shared visual track —
  each one is a genuine native range input, so dragging, every native
  keyboard interaction (arrow keys, Home/End, Page Up/Down), focus handling,
  and the implicit `role="slider"`/`aria-valuemin`/`aria-valuenow`/
  `aria-valuemax` accessible contract all come from the browser for free, on
  every single thumb, with zero JS. This is a real, established technique
  (not invented for this port) for building a multi-thumb range control out
  of ordinary native form pieces; the one genuine problem it introduces —
  overlapping thumbs need to each still be independently grabbable — is
  solved below with CSS alone (see the "Thumb-overlap disambiguation"
  section).

  `values: number[]` is a single prop that scales cleanly with thumb count:
  a one-element array (`[50]`) renders a plain single-thumb slider; a
  two-element array (`[25, 75]`) renders a min/max range slider; there is no
  separate single-vs-multi-thumb component variant to maintain, matching
  RadioGroup.vue's/ToggleGroup.vue's own `items`-array-driven approach to
  generalizing across a variable number of rendered controls.

  ## Structure

    <span class="radix-slider">
      <span class="radix-slider-track">
        <span class="radix-slider-range" :style="left/width percentages">
      </span>
      <input type="range" class="radix-slider-input" ... />   (one per value)
      <input type="range" class="radix-slider-input" ... />
      ...
    </span>

  `.radix-slider-track`/`.radix-slider-range` are purely decorative — a
  static track background plus a filled "selected range" bar, both `<span>`s
  with no interactive role of their own (this is exactly the same
  decorative-div-plus-percentage-math shape Progress.vue's own fill
  indicator already uses, applied here to a start/end range instead of a
  single 0%-to-value% fill). Every real input/keyboard/pointer/ARIA
  interaction instead lives on the stacked native <input type="range">
  elements layered on top, one per thumb.

  ## Thumb-overlap disambiguation (the one real implementation challenge)

  With N native inputs stacked on the exact same rectangle, the naive
  version of this technique has a real bug: each input's own full-width
  track area is, by default, just as clickable/draggable as its thumb, so
  whichever input is *later in DOM order* (topmost in paint order) has its
  entire invisible track sitting on top of every earlier input's own thumb,
  making that earlier thumb ungrabbable wherever the two inputs overlap.

  The fix used here is pure CSS, verified by hand-tracing the cascade before
  committing to it (no enhancement script was needed):

    1. `pointer-events: none` on `.radix-slider-input` itself turns off hit
       testing for the *entire* input box — including its own track — so an
       input's inert, full-width box can no longer steal clicks meant for
       whatever is underneath it.
    2. `pointer-events: auto` is then set explicitly back on
       `::-webkit-slider-thumb`/`::-moz-range-thumb` — the browser-generated
       pseudo-elements that render each input's own actual draggable thumb
       handle. This is the load-bearing counter-rule: without it, step (1)
       alone would make the *whole* input inert, thumb included, which was
       explicitly checked for and avoided here (a common, easy mistake with
       this technique — see this component's commit message for the
       self-adversarial review that confirmed this rule is present and
       correctly scoped). CSS lets a pseudo-element's own `pointer-events`
       value override what its host's `pointer-events: none` would otherwise
       imply, because pointer-events (like any other inheritable property)
       can be re-declared per-selector; it is not an all-or-nothing switch
       on the host box. With this pair of rules, only the small thumb-shaped
       hit areas compete for clicks — never the full-width invisible track —
       so every thumb, regardless of which input is on top in DOM order, is
       independently reachable.
    3. Two thumbs can still end up rendering at (or very near) the exact
       same pixel position when their values are equal or close, at which
       point only the input currently on top in *paint* order is reachable
       by a fresh click, same as native `<input type="range">` offers no
       different affordance for this either. This is mitigated, still with
       zero JS, by bumping the host input's `z-index` on
       `:hover`/`:focus-visible`/`:active`:
       hovering (or focusing, or actively dragging) an input's own thumb
       pseudo-element counts as hovering/focusing the host input itself —
       generated pseudo-element content is treated as part of its host's box
       for hit-testing and state-matching purposes, even though the rest of
       that host's box has `pointer-events: none` everywhere else — so the
       thumb currently under the pointer (or holding focus, or mid-drag)
       is always the one raised to the front, letting a user "dig out" a
       thumb stacked underneath another one. Once a drag has actually begun,
       native `<input type="range">` drag-tracking itself continues
       regardless of z-index or literal cursor position (the browser keeps
       the drag bound to the input that started it, not to whatever element
       currently sits under the pointer), so z-index only has to get a drag
       *started* correctly, not stay correct for its whole duration.
    4. Clicking directly on empty track space (not on any thumb) does
       nothing, for any thumb, in this baseline: because step (1) makes
       every input's own track area inert, there is no native "click to jump
       the nearest thumb here" affordance the way a single plain
       <input type="range"> would offer on its own track. This is an
       accepted, documented trade-off of the overlap fix above — the
       alternative (leaving track areas clickable) is exactly the bug this
       technique exists to avoid, so it is not a viable alternative, and
       every thumb remains fully keyboard- and drag-operable regardless.

  There is no custom-element enhancement script block anywhere in this file:
  per RFC 014 §3 Non-Goals, a script is only warranted when something is
  genuinely unreachable without one, and — per the reasoning above — the
  pointer-events layering trick was confirmed (not merely assumed) to make
  the thumb-overlap problem fully solvable in CSS alone.

  ## The visual range-fill bar is static, not drag-synced

  `.radix-slider-range`'s left/width percentages (see <style scoped>/the
  :style binding below) are computed once, at render time, from the
  `values`/`min`/`max` props — the same "plain arithmetic on props already
  known at render time, no client-side event dependency" shape Progress.vue's
  own fill-width binding already uses. Dragging one of the native inputs
  above updates that *input's own* native thumb rendering for free (the
  browser repaints it locally, no JS involved) but does **not** re-run this
  template's percentage math, because nothing here is wired to listen for a
  live `input`/`change` event and recompute it — this is genuine, real SSR
  output, not a client-reactive framework. Concretely: dragging a thumb
  moves that thumb correctly, but the separate visual fill bar behind it
  will not visually follow along until the next full server render. This is
  an accepted baseline limitation matching this component's documented
  design (a static, props-driven fill layer, not a live-synced one) rather
  than an oversight; wiring the fill bar to track live drags would be a
  small, self-contained, purely additive progressive-enhancement script
  follow-up (listening for `input` events on each `.radix-slider-input` and
  recomputing the same left/width formula client-side) if a future caller
  needs it, not something this first pass needed to block on.

  ## `values` ordering contract

  The fill-bar math below reads `values[0]` as the range's low end and
  `values[values.length - 1]` as its high end — i.e. it assumes `values` is
  passed pre-sorted ascending, matching how every Radix Slider example
  already passes a range's `[min, max]` pair. This port does not re-sort or
  validate `values` at render time (there is no array-sort operation
  available in this template's expression language — see expr/doc.go's
  Unsupported Constructs list, which has no spread/mutation/method-call
  surface for that), so an out-of-order `values` array will render each
  native input at its own correct native position (every input is
  independently correct regardless of array order) but will produce an
  incorrect fill bar. Callers are expected to keep `values` sorted, the same
  documented-contract shape Radix's own examples already follow by
  convention.

  ## Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):
    values: number[] — one entry per thumb, ascending (see the ordering
                        contract above). `[50]` renders one thumb; `[25, 75]`
                        renders a two-thumb min/max range slider; longer
                        arrays render additional thumbs the same way.
    min:    number    — shared lower bound for every thumb.
    max:    number    — shared upper bound for every thumb.
    step:   number    — shared native step granularity for every thumb.

  `disabled`/`name`/`orientation` are out of scope for this first pass —
  the same kind of deliberate v1 scope call RadioGroup.vue's/ToggleGroup.vue's
  own header comments document for their analogous omitted props — rather
  than something this pass needed to block on.

  Usage (plain prose labels below, not inline HTML comments: a literal
  comment-close sequence — hyphen, hyphen, greater-than — appearing inside
  this outer header comment would truncate it early, a real trap this file
  hit and fixed once already while drafting the two labeled examples below):

    single-thumb slider:
      <Slider :values="[50]" :min="0" :max="100" :step="1" />

    two-thumb min/max range slider:
      <Slider :values="[25, 75]" :min="0" :max="100" :step="1" />

  On aria-label per thumb: mirrors Radix's own documented convention —
  omitted entirely for a single-thumb slider (nothing to disambiguate),
  "Minimum"/"Maximum" for an exactly-two-thumb range slider, and
  "Value <n> of <total>" for three or more thumbs. Binding to `undefined`
  (the single-thumb case) omits the attribute entirely rather than rendering
  a literal "undefined" string, the same omission behavior Progress.vue's
  own `:aria-valuenow="indeterminate ? undefined : value"` binding already
  relies on.
-->
<template>
  <Tokens></Tokens>
  <span class="radix-slider" data-orientation="horizontal">
    <span class="radix-slider-track">
      <span
        class="radix-slider-range"
        :style="{
          left: (values.length > 1 ? ((values[0] - min) / (max - min) * 100) : 0) + '%',
          width: (((values[values.length - 1] - min) / (max - min) * 100) - (values.length > 1 ? ((values[0] - min) / (max - min) * 100) : 0)) + '%'
        }"
      ></span>
    </span>
    <input
      v-for="(v, index) in values"
      type="range"
      class="radix-slider-input"
      :min="min"
      :max="max"
      :step="step"
      :value="v"
      :aria-label="values.length > 2 ? ('Value ' + (index + 1) + ' of ' + values.length) : (values.length === 2 ? (index === 0 ? 'Minimum' : 'Maximum') : undefined)"
    />
  </span>
</template>

<style>
.radix-slider {
  position: relative; /* containing block for the absolutely-positioned inputs */
  display: inline-flex;
  align-items: center;
  width: 100%;
  min-width: 10rem;
  height: 1.25rem;
  touch-action: none;
}

.radix-slider-track {
  position: relative;
  width: 100%;
  height: 0.3rem;
  border-radius: var(--radix-radius-thumb);
  background-color: var(--radix-brown-4);
}

/*
 * The selected-range fill. See this file's header comment ("The visual
 * range-fill bar is static, not drag-synced") for why its left/width
 * percentages are computed once from props rather than kept in sync with
 * live dragging.
 */
.radix-slider-range {
  position: absolute;
  top: 0;
  bottom: 0;
  border-radius: var(--radix-radius-thumb);
  background-color: var(--radix-brown-9);
}

/*
 * One real native <input type="range"> per thumb value, stacked exactly on
 * top of the track above via absolute positioning against the shared
 * `.radix-slider` containing block. `pointer-events: none` here is step 1
 * of the thumb-overlap fix documented in the header comment above — the
 * counteracting `pointer-events: auto` rule that keeps each input's own
 * thumb actually grabbable lives on the ::-webkit-slider-thumb/
 * ::-moz-range-thumb rules further down; the two must be read together
 * (this rule alone would make the whole input inert, thumb included).
 */
.radix-slider-input {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 100%;
  height: 1.1rem;
  margin: 0;
  background: transparent;
  -webkit-appearance: none;
  -moz-appearance: none;
  appearance: none;
  pointer-events: none;
  z-index: 1;
}

/* Kill each engine's own default track paint; the shared
   .radix-slider-track behind every input is the only visible track. Also
   explicitly inert (redundant with the input-level pointer-events: none
   above, but written out so this rule stays correct even if the two are
   ever split apart). */
.radix-slider-input::-webkit-slider-runnable-track {
  height: 1.1rem;
  background: transparent;
  border: none;
  pointer-events: none;
}

.radix-slider-input::-moz-range-track {
  height: 1.1rem;
  background: transparent;
  border: none;
  pointer-events: none;
}

/*
 * The actual draggable thumb handle. `pointer-events: auto` here is step 2
 * of the thumb-overlap fix (see header comment): it re-enables hit-testing
 * specifically on this pseudo-element after the host input turned it off
 * for its whole box above, so only the small thumb-shaped area — never the
 * input's full-width inert track — can intercept a click or drag.
 */
.radix-slider-input::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  pointer-events: auto;
  width: 1.1rem;
  height: 1.1rem;
  margin-top: 0; /* runnable-track height above is set equal to thumb height, so no vertical offset is needed */
  border-radius: var(--radix-radius-thumb);
  background-color: var(--radix-brown-1);
  border: 1px solid var(--radix-brown-8);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  cursor: pointer;
}

.radix-slider-input::-moz-range-thumb {
  pointer-events: auto;
  width: 1.1rem;
  height: 1.1rem;
  border-radius: var(--radix-radius-thumb);
  background-color: var(--radix-brown-1);
  border: 1px solid var(--radix-brown-8);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  cursor: pointer;
  box-sizing: border-box;
}

/*
 * Step 3 of the thumb-overlap fix (see header comment): raise whichever
 * thumb is currently hovered, focused, or mid-drag above every other
 * stacked thumb, so an overlapping/near-overlapping thumb can still be
 * "dug out" from underneath another one with zero JS.
 */
.radix-slider-input:hover,
.radix-slider-input:focus-visible,
.radix-slider-input:active {
  z-index: 2;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-slider-input:focus-visible::-webkit-slider-thumb {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

.radix-slider-input:focus-visible::-moz-range-thumb {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

/*
 * This component has no `disabled` prop of its own in this first pass (see
 * the header comment), but a native <input type="range"> can still end up
 * matching :disabled "for free" when a caller nests this component inside
 * a real <fieldset disabled> ancestor — a genuine native HTML behavior
 * (disabling a fieldset disables every descendant form control) this rule
 * simply styles for, at no extra cost, the same way the rest of this
 * component leans on native behavior wherever it already exists.
 */
.radix-slider-input:disabled::-webkit-slider-thumb {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-slider-input:disabled::-moz-range-thumb {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
