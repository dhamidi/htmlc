<!--
  Toggle — Radix-inspired two-state pressed/unpressed button (e.g. a
  bold/italic formatting toggle in a toolbar).

  Unlike Checkbox.vue/Switch.vue/RadioGroup.vue, this component is NOT
  built on a native form-control element with its own built-in
  checked/unchecked state. It is a plain <button>, and HTML has no native
  notion of a "pressed" toggle button — `aria-pressed` is purely an ARIA
  attribute a script must maintain; there is no equivalent to a checkbox's
  real `checked` IDL property that the browser flips for free on click.
  That is the genuine, structural difference from this series' other
  two-state controls, and it is why this file — alone among them —
  actually needs its <script customelement> block to do real, load-bearing
  work rather than mere polish.

  Baseline (zero-JS): a native
    <button type="button" :aria-pressed="pressed ? 'true' : 'false'" :data-state="pressed ? 'on' : 'off'">
  This starts in the correct visual/ARIA state for whatever `pressed` value
  was rendered server-side — a reader/screen-reader user loading the page
  sees accurate state immediately. But clicking it does *nothing* without
  JS: a plain <button> has no native toggle behavior the way a checkbox or
  radio input does, so with script disabled this baseline is genuinely
  inert for the one interaction that defines a toggle button. This is a
  real, documented degradation (unlike Checkbox's/Switch's baselines,
  which remain fully operable with zero JS) — not a bug to work around,
  just an honest limit of what static HTML can express for this control
  shape.

  On the `pressed ? 'true' : 'false'` ternary (rather than the shorter
  `:aria-pressed="pressed"`): this codebase's dynamic-attribute binding
  omits an attribute entirely whenever its bound expression evaluates to
  Go's bare `false` (the same falsy-attribute-omission convention
  Dialog.vue's/RadioGroup.vue's header comments describe, matching Vue.js's
  own v-bind semantics — it is what correctly makes `:disabled="disabled"`
  below drop the attribute when not disabled). That convention is exactly
  right for genuine HTML boolean/presence attributes like `disabled`, but
  `aria-pressed` is not one: it is a tristate ARIA attribute
  (true/false/mixed) whose *absence* means "this is not a toggle button at
  all" to assistive technology, not "false". Binding it directly to the
  boolean prop would therefore silently vanish the attribute whenever
  `pressed` is false — the exact failure this port's own scratch-render
  self-check (RenderFragmentString with pressed=false) caught before this
  commit. Coercing to the literal strings `'true'`/`'false'` first sidesteps
  the omission rule entirely, the same technique `:data-state` already uses
  next to it.

  <script customelement> (load-bearing, not just polish): on click, flips
  the button's own `aria-pressed`/`data-state` between the two values and
  dispatches a `radix-toggle-change` CustomEvent from the button, carrying
  the new pressed state as `event.detail.pressed` (a plain boolean — the
  minimal, unambiguous shape a listener needs; matches Radix's own
  onPressedChange(pressed) callback's single-argument shape in spirit,
  translated to this library's DOM-event-only mechanism since ui/radix has
  no client-side state/data-binding system of its own). The event bubbles,
  so a consuming page's own script — or a future composite component like
  ToggleGroup built on top of this one — can listen for it either on the
  button itself or on any ancestor, including this component's own
  <radix-toggle> wrapper tag.

  Respecting `disabled`: the native `disabled` attribute on <button> is
  what makes a disabled toggle inert, not any script-side check. Per the
  HTML spec, a disabled button never dispatches `click` at all — the
  browser suppresses the activation behavior before any listener (bubbling
  or otherwise) ever runs — so this component needs no
  `if (this.#button.disabled) return` guard in the click handler below.
  Adding one would be redundant at best, and at worst a second source of
  truth that could silently drift from the real `disabled` attribute (e.g.
  if a caller ever mutates the attribute directly). The only thing that
  actually has to be correct is that the baseline template below renders
  the real `disabled` attribute from the prop, which it does.

  Props (both REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the "[missing: <name>]" placeholder trap this convention avoids):
    pressed:  boolean — the server-rendered initial pressed state.
    disabled: boolean — the native `disabled` attribute.

  Slot:
    default — the toggle's content (icon/text), e.g. <Toggle :pressed="isBold" :disabled="false">B</Toggle>.

  Kept deliberately composable/reusable rather than bespoke: this is the
  building block a later ToggleGroup.vue is expected to build on, so this
  file has no toolbar/group-specific assumptions baked in (no shared name,
  no roving tabindex, no group-exclusivity logic) — just one button that
  knows how to flip its own two states and announce that it did.

  Usage:
    <Toggle :pressed="false" :disabled="false">
      <strong>B</strong>
    </Toggle>
-->
<template>
  <button
    v-native
    type="button"
    class="radix-toggle"
    :aria-pressed="pressed ? 'true' : 'false'"
    :data-state="pressed ? 'on' : 'off'"
    :data-disabled="disabled ? '' : undefined"
    :disabled="disabled"
  ><slot></slot></button>
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

.radix-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.7rem;
  background-color: #fff;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
  cursor: pointer;
  line-height: 1;
}

.radix-toggle[data-state='on'] {
  background-color: var(--radix-blue-4);
  border-color: var(--radix-blue-9);
  color: var(--radix-blue-9);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-toggle:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-toggle:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

<script customelement>
// Progressive enhancement on top of the inert-without-JS <button> baseline
// above. Per RFC 014 §3 Non-Goals, this reimplements nothing the browser
// already provides — but unlike this series' other two-state controls,
// there genuinely is no native provider for "pressed" state on a plain
// <button>, so unlike Switch.vue's file (which has no customelement block
// at all) or Accordion.vue's/Tabs.vue's (which only layer keyboard
// navigation on top of already-working native controls), this one
// performs the toggle action itself.
class RadixToggle extends HTMLElement {
  #button = null

  #onClick = () => {
    const next = this.#button.getAttribute('aria-pressed') !== 'true'
    this.#button.setAttribute('aria-pressed', String(next))
    this.#button.setAttribute('data-state', next ? 'on' : 'off')
    this.#button.dispatchEvent(
      new CustomEvent('radix-toggle-change', {
        detail: { pressed: next },
        bubbles: true,
      })
    )
  }

  connectedCallback() {
    this.#button = this.querySelector('button')
    if (!this.#button) return

    // No disabled-state guard needed here: a disabled <button> never
    // dispatches 'click' in the first place (HTML spec, activation
    // behavior), so this listener simply never runs for a disabled
    // toggle. See this file's header comment for why an explicit
    // script-side disabled check would be redundant.
    this.#button.addEventListener('click', this.#onClick)
  }

  disconnectedCallback() {
    if (this.#button) {
      this.#button.removeEventListener('click', this.#onClick)
    }
  }
}

customElements.define('radix-toggle', RadixToggle)
</script>
