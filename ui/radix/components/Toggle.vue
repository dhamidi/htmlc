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

<style scoped>
.radix-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.7rem;
  background-color: #fff;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  cursor: pointer;
  line-height: 1;
}

.radix-toggle[data-state='on'] {
  background-color: #e0e7ff;
  border-color: #2563eb;
  color: #2563eb;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-toggle:focus-visible {
  outline: 2px solid #2563eb;
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
