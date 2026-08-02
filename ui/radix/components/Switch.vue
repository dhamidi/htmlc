<!--
  Switch — Radix-inspired on/off toggle. Semantically similar to Checkbox
  (a two-state, boolean form control), but announced to assistive
  technology as a switch (`role="switch"`) rather than a checkbox — the
  distinction matters to screen-reader users because a switch communicates
  "on/off, takes effect immediately" where a checkbox communicates
  "selected/unselected, part of a set", even though both happen to share
  the same underlying true/false model.

  Baseline: same decomposition as Checkbox.vue — a real native
  <input type="checkbox">, visually hidden with the identical clip-based
  technique (position: absolute, 1x1px, clip-path: inset(50%); never
  display:none, which would drop it from focus reach and the tab order),
  paired with an adjacent <label class="radix-switch-track"> that is this
  component's entire custom-styled visual surface (a pill-shaped track
  with a sliding thumb). The two are flat, adjacent siblings so the CSS
  below can key off the input's live state with a plain adjacent-sibling
  selector:

    .radix-switch-input:checked + .radix-switch-track::after { ... }

  On `role="switch"` needing no JS at all (the core design question this
  component is built around, and the one thing that sets it apart from
  every other component in this library so far): `role="switch"` is a
  real, valid, static HTML attribute on a native <input type="checkbox">.
  It is not a made-up or unsupported combination — the "ARIA in HTML"
  specification's allowed-roles table explicitly permits `switch` as a
  role override on `input[type=checkbox]`
  (https://www.w3.org/TR/html-aria/#el-input-checkbox), and HTML-AAM
  documents the resulting accessible-state mapping: the input's native
  `checked` IDL property/attribute is what supplies the accessible
  on/off state, regardless of whether the role is left as the implicit
  `checkbox` or overridden to `switch`
  (https://www.w3.org/TR/html-aam-1.0/#input-type-checkbox). Concretely,
  this means the browser computes `aria-checked` from the real `checked`
  state for us — exactly the same free mapping Checkbox.vue already
  relies on for its own checked/unchecked states, just surfaced to
  assistive technology as a switch's on/off language instead of a
  checkbox's selected language. Nothing about this mapping depends on
  script running: it is computed from the DOM/accessibility-tree
  integration browsers already ship, the same way `:checked` is
  available to CSS with zero JS. This was verified against the spec
  references above rather than assumed; it is standard, well-established
  practice (the same technique widely used to build accessible toggle
  switches with a bare native checkbox), not something novel to this
  port.

  The practical consequence: unlike Checkbox.vue's `indeterminate` state
  (genuinely impossible to reach without JS, because HTML has no
  `indeterminate` *attribute* — see that file's header comment), Switch
  has no state that is unreachable from markup. `checked`/`unchecked` are
  both real, declarative, native-attribute-backed states from the very
  first paint, and `role="switch"` is a plain static attribute requiring
  no runtime involvement whatsoever. There is therefore **no
  customelement enhancement script block in this file** — not because a
  progressive-enhancement script was skipped by oversight, but because
  there is nothing left for one to do that the browser does not already
  do for free. Adding a script that toggles a class on `change` purely to
  drive the thumb-slide animation was considered and rejected too: the
  thumb's position is already keyed off the real `:checked` pseudo-class
  via the adjacent-sibling selector below, and a plain CSS `transition`
  on the animated property (see `.radix-switch-track::after` in
  <style scoped>) fires automatically whenever that selector's match
  state changes — including when it changes because a *user* toggles the
  real input, which is a genuine DOM state change, not a synthetic one.
  CSS transitions do not care what caused the recalculation (a class
  toggle from JS and a `:checked` pseudo-class flip from user input are
  equally "real" style changes to the transition engine); a `change`-
  triggered script would only be reproducing motion the browser already
  provides, at the cost of extra runtime code for zero visible benefit.

  On the required `v-native` on the <label> tag below: this package's own
  Label.vue registers a lowercase "label" alias engine-wide once ui/radix
  is mounted as a whole package, and a literal <label> tag anywhere else
  in the same mount resolves against that alias before ever being treated
  as plain HTML (renderer.go resolves every implicit, non-native-declared
  tag against the component registry first). Checkbox.vue's own commit
  hit this exact collision after the fact, empirically, by rendering
  Tabs.vue's <label> through a mounted ui/radix package and watching it
  silently render as Label.vue's markup instead, dropping Tabs.vue's own
  attributes. This file applies the identical, by-now-established fix
  proactively instead of waiting to rediscover the same bug a third time:
  v-native below declares this <label> a genuine native HTML element, so
  this component's track renders correctly regardless of which other
  ui/radix components happen to be mounted alongside it — including
  Label.vue itself.

  Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the "[missing: <name>]" placeholder trap this convention avoids):
    id:        string  — required for the <label>'s `for` pairing above
                          (see the v-native note) and so external callers
                          can pair their own <Label for="..."> with this
                          control, the same `for`/`id` contract Label.vue
                          already documents.
    name:      string  — the native `name` attribute; needed for this
                          control to actually submit as part of a form.
    value:     string  — the native `value` attribute. Radix's own
                          documented default is `value="on"` (the same
                          value a bare native checkbox submits when no
                          `value` attribute is given at all) — this port's
                          call sites should pass `"on"` explicitly by
                          convention when no more specific value is
                          needed, matching Checkbox.vue's identical note.
    checked:   boolean — the control's on/off state.
    disabled:  boolean — the native `disabled` attribute.
    required:  boolean — the native `required` attribute.

  Usage:
    <Switch id="airplane-mode" name="airplane-mode" value="on"
            :checked="false" :disabled="false" :required="false" />
-->
<template>
  <span
    class="radix-switch"
    :data-state="checked ? 'checked' : 'unchecked'"
    :data-disabled="disabled ? '' : undefined"
  >
    <input
      type="checkbox"
      role="switch"
      class="radix-switch-input radix-visually-hidden-input"
      :id="id"
      :name="name"
      :value="value"
      :checked="checked"
      :disabled="disabled"
      :required="required"
    />
    <label
      v-native
      class="radix-switch-track"
      :for="id"
      aria-hidden="true"
      :data-state="checked ? 'checked' : 'unchecked'"
    ></label>
  </span>
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

.radix-switch {
  position: relative; /* containing block for the absolutely-positioned input */
  display: inline-flex;
  align-items: center;
}

.radix-switch-track {
  position: relative;
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  width: 2.75rem;
  height: 1.5rem;
  background-color: var(--radix-gray-6);
  border-radius: var(--radix-radius-thumb);
  cursor: pointer;
  transition: background-color 150ms ease;
  box-sizing: border-box;
}

/*
 * The thumb is a single ::after pseudo-element on the track — no inner
 * markup, no inline SVG. Positioned at the track's left edge by default;
 * the :checked sibling-selector rule below slides it to the right edge
 * via `transform: translateX(...)`, which is what the `transition`
 * property here animates. This transition fires off of the real
 * `:checked` pseudo-class match changing on the paired input (see this
 * file's header comment) — no JS involved.
 */
.radix-switch-track::after {
  content: '';
  position: absolute;
  top: 0.125rem;
  left: 0.125rem;
  width: 1.25rem;
  height: 1.25rem;
  background-color: #fff;
  border-radius: var(--radix-radius-thumb);
  transform: translateX(0);
  transition: transform 150ms ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* Checked: track turns "on" color, thumb slides to the right edge. Both
   driven by the input's real, native :checked pseudo-class. */
.radix-switch-input:checked + .radix-switch-track {
  background-color: var(--radix-blue-9);
}

.radix-switch-input:checked + .radix-switch-track::after {
  transform: translateX(1.25rem);
}

/* Never remove the focus outline — keep it visible for keyboard users.
   The input itself is visually hidden, so its focus ring is drawn on the
   paired track instead, matching Checkbox.vue's identical treatment. */
.radix-switch-input:focus-visible + .radix-switch-track {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-switch-input:disabled + .radix-switch-track {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
