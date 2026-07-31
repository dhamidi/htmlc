<!--
  Progress — Radix-inspired determinate/indeterminate progress indicator.

  `role="progressbar"` with `aria-valuemin`/`aria-valuemax`/`aria-valuenow`
  for the determinate case. Per the ARIA spec, a progressbar communicating
  "in progress, percentage unknown" omits `aria-valuenow` entirely rather
  than supplying any numeric placeholder for it — assistive tech treats a
  progressbar with no `aria-valuenow` as indeterminate, the same way the
  native HTML `<progress>` element with no `value` attribute renders as an
  indeterminate throbber. This component's indeterminate branch genuinely
  omits the attribute (see the props note below and progress_test.go's
  TestProgress_IndeterminateOmitsAriaValuenow) rather than emitting
  `aria-valuenow="0"`, which would incorrectly announce "0% complete"
  instead of "progress is happening, amount unknown".

  Implementation choice — a custom `<div role="progressbar">` with an
  inner fill element, not the native `<progress>` element:

  Native `<progress>` gets `role="progressbar"` and value semantics for
  free from the browser (an implicit ARIA role, no attributes to hand-roll)
  and needs zero JS — the same "native element, correct behavior for free"
  trade-off this directory already takes with Accordion.vue's
  `<details>`/`<summary>` and Dialog.vue's `<dialog>`. But `<progress>`'s
  own fill is rendered by browser-internal pseudo-elements
  (`::-webkit-progress-value`, `::-moz-progress-bar`, and a *different*
  subset of the box model is stylable in each browser engine), which is
  exactly why cross-browser-consistent restyling of `<progress>` has long
  needed vendor-prefixed pseudo-element overrides with diverging behavior
  per engine, rather than one shared set of ordinary CSS properties — this
  is the well-known reason component libraries (Radix's own real
  implementation included) build progress bars out of plain elements
  instead of the native one. A progress bar's fill color, shape, corner
  radius, and (for the indeterminate case) sweep animation are typically a
  core, load-bearing piece of a design system's visual identity — not an
  incidental detail a caller might reasonably accept the browser's default
  for, the way AspectRatio.vue's ratio-box or Separator.vue's divider line
  might. That tips this component to the opposite side of the trade-off
  AspectRatio.vue documents: full styling control (an inner div whose
  width is driven by the percentage in plain, portable CSS, with the
  indeterminate sweep as an ordinary `@keyframes` animation — see
  <style scoped> below) at the cost of hand-writing the ARIA attributes
  that `<progress>` would have supplied automatically. This mirrors
  Tabs.vue's own precedent of choosing custom markup over a
  simpler-but-harder-to-restyle native building block when styling control
  is the thing that actually matters for the component in question.

  No custom-element enhancement script: value/percentage rendering is pure
  server-side data binding (an aria-valuenow attribute and a fill-width
  percentage, both plain arithmetic on props already known at render time)
  with no interactive state and nothing that depends on client-side
  events — same reasoning as AspectRatio.vue/Separator.vue/Label.vue in
  this directory. Confirmed: unlike Avatar.vue (which needs a real
  `error` event listener to know whether an <img> actually loaded, a fact
  genuinely unknowable at render time) or Tabs.vue/Accordion.vue (which
  layer roving-tabindex keyboard behavior on top of native
  radio/`<details>` interaction), nothing about this component depends on
  anything that happens after the page has already rendered.

  Props (all REQUIRED — see the default-value note below):
    value:         number  — current progress amount, expected to satisfy
                              0 <= value <= max. Ignored when indeterminate
                              is true (pass 0 by convention, matching
                              Avatar.vue's `delayMs={0}`-for-"not
                              applicable" convention for its own
                              optional-feeling required prop).
    max:           number  — the value that represents 100% complete.
                              This package has no notion of an optional
                              prop with a real default (see below), so
                              there is no same-file `max={100}` fallback;
                              Radix's own documented default is `max={100}`
                              by convention at the call site, and that is
                              the convention this port documents too,
                              matching AspectRatio.vue's identical
                              ratio-default note.
    indeterminate: boolean — true renders the indeterminate state
                              (aria-valuenow omitted, sweep animation);
                              false renders the determinate state
                              (aria-valuenow bound to value, fill width
                              bound to the value/max percentage).

  On signalling "indeterminate" given the required-prop trap (the same
  trap documented in Separator.vue's/AspectRatio.vue's/Label.vue's/
  Avatar.vue's header comments: any identifier referenced anywhere in the
  template becomes a required prop, and an unpassed required prop renders
  a visible, truthy "[missing: <name>]" placeholder rather than falling
  through to any same-file default — see component.go's validateProps):
  Radix's own real prop is `value: number | null | undefined`, with `null`
  meaning "indeterminate". Mirroring that directly here would mean passing
  a bare JS `null`/`undefined` *literal* is the only way to opt into
  indeterminate — expr's grammar does support `null`/`undefined` as real
  literals (see expr/doc.go's Literals section), and validateProps' missing-
  prop check only fires when a prop key is absent from the call site's scope
  *entirely*, not when it is present but bound to `null` — so `:value="null"`
  would technically thread through without tripping the placeholder trap.
  But relying on that is fragile in a way this component doesn't need to
  risk: it depends on every caller remembering to write out the `null`
  literal explicitly (an easy step to just forget, at which point `value`
  goes missing entirely and "[missing: value]" — a non-numeric, always-
  truthy string — leaks straight into the `aria-valuenow`/fill-width
  arithmetic, corrupting both), and it overloads one prop with two
  unrelated concerns (the numeric amount, and the determinate/indeterminate
  mode switch) for no real benefit in a template language that has no
  optional-prop defaults to make the overload pay for itself. A separate,
  always-required `indeterminate: boolean` prop is cleaner: it makes the
  mode switch an explicit, self-documenting value at every call site (the
  same shape as Separator.vue's `decorative` boolean), is immune to the
  missing-vs-null distinction entirely, and keeps `value` doing exactly one
  job. `max`/`value`/`indeterminate` are therefore all required, with every
  call site passing them explicitly — the same trade-off, and the same
  resolution, as Separator.vue's orientation/decorative and AspectRatio.vue's
  ratio.

  Usage:
    <Progress :value="80" :max="100" :indeterminate="false" />
    <Progress :value="0" :max="100" :indeterminate="true" />

  On self-reference-cycle risk (the trap documented in Label.vue's/
  Dialog.vue's header comments, where a component's own name auto-
  registers a lowercase alias in the component registry that can collide
  with a literal native tag of the same name inside its own template):
  this component's auto-registered lowercase alias is "progress", which
  *is* a real native HTML tag name (`<progress>`) — but this file's
  template never contains a literal `<progress>` element (see the
  implementation-choice note above: this port deliberately uses a plain
  `<div role="progressbar">`, not the native element), so there is nothing
  for the "progress" alias to collide with and no `v-native` escape hatch
  is needed here, unlike Label.vue's `<label>` or Dialog.vue's `<dialog>`.
-->
<template>
  <div
    class="radix-progress"
    role="progressbar"
    aria-valuemin="0"
    :aria-valuemax="max"
    :aria-valuenow="indeterminate ? undefined : value"
    :data-state="indeterminate ? 'indeterminate' : (value >= max ? 'complete' : 'loading')"
    :data-max="max"
  >
    <div
      class="radix-progress-indicator"
      :data-state="indeterminate ? 'indeterminate' : (value >= max ? 'complete' : 'loading')"
      :style="indeterminate ? {} : { width: (value / max * 100) + '%' }"
    ></div>
  </div>
</template>

<style scoped>
/*
 * Track: a fixed-height, fully-rounded, clipped bar. `overflow: hidden`
 * keeps the inner indicator's own rounded corners and (for the
 * indeterminate case) its animated sweep clipped to the track's bounds,
 * rather than spilling a square corner or an off-track sweep past the
 * rounded ends.
 */
.radix-progress {
  position: relative;
  overflow: hidden;
  width: 100%;
  height: 0.5rem;
  border-radius: 9999px;
  background-color: #e5e5e5;
}

/*
 * Determinate fill: width is set inline per-render from the value/max
 * percentage (see :style binding above); the transition here only smooths
 * *subsequent* width changes (e.g. a caller re-rendering this component
 * with a larger `value` as real progress advances), not the initial paint.
 */
.radix-progress-indicator {
  height: 100%;
  width: 0;
  background-color: #2563eb;
  border-radius: 9999px;
  transition: width 660ms cubic-bezier(0.65, 0, 0.35, 1);
}

/*
 * Indeterminate fill: the template's :style binding evaluates to an empty
 * object for this state (see header comment), so no inline `width` is
 * rendered at all and this class rule's `width: 40%` applies with no
 * inline-style specificity to fight — a fixed-width band that sweeps left
 * to right on loop via `transform`, communicating "something is
 * happening" without claiming any particular completion percentage.
 */
.radix-progress-indicator[data-state='indeterminate'] {
  width: 40%;
  animation: radix-progress-indeterminate-sweep 1.5s ease-in-out infinite;
}

@keyframes radix-progress-indeterminate-sweep {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(350%);
  }
}

/*
 * Respect a user's reduced-motion preference by freezing the sweep instead
 * of looping it indefinitely; the fixed-width band still communicates
 * "in progress" without the motion.
 */
@media (prefers-reduced-motion: reduce) {
  .radix-progress-indicator[data-state='indeterminate'] {
    animation: none;
  }
}
</style>
