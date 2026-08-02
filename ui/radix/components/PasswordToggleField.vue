<!--
  PasswordToggleField — Radix-inspired password input paired with a
  show/hide toggle button.

  Baseline (zero-JS): a real native <input type="password" :id="id"
  :name="name">. Password masking is a genuine native browser feature —
  the input works exactly as any password field does, with zero JS
  involved, including full form participation (name/value submission,
  required validation, disabled state). Nothing about the input's core
  job depends on this file's <script customelement> block.

  The show/hide *toggle button*, however, is a plain <button> with no
  native "flip my sibling input's type" behavior — HTML has no
  declarative way to make one element's `type` attribute react to
  another element's click (the same structural gap Toggle.vue's own
  header comment documents for its own pressed/unpressed state, except
  here the thing being toggled isn't even the button's own attribute,
  it's the *input's*). So, same honest framing as Toggle.vue: this
  button renders in the correct initial state (masked/hidden, "Show
  password") with zero JS, but clicking it does nothing at all without
  JS. That is a real, documented degradation, not a bug to route around —
  there is no HTML-only way to reveal a password's characters.

  On the baseline always starting "hidden":
  unlike Toggle.vue's `pressed` prop, this component takes no prop for
  its initial visibility state. Radix's own real implementation *does*
  support server/caller-controlled initial visibility, but that only
  matters once the reveal is real — and revealing a password's plaintext
  is exactly the one thing this zero-JS baseline structurally cannot do
  (see above). A prop that claimed to control "initial visibility" would
  therefore be misleading here: with JS absent, every value of such a
  prop would render identically masked (the button doing nothing to
  change it), and with JS present the reveal is a script-owned runtime
  event, not a page-load-time server decision, so a prop for it would
  only ever describe a state this component can't actually start in.
  Baseline is always masked/hidden; that is the only honest server-
  rendered starting point.

  Props (both REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments
  for the "[missing: <name>]" placeholder trap this convention avoids):
    id:       string  — the input's `id`, so an external <Label for="...">
                         (this package's own Label.vue) can be paired with
                         it, matching Checkbox.vue's identical `id` contract.
                         Also used verbatim as the toggle button's
                         `aria-controls`, associating the two per WAI-ARIA
                         guidance for a control that affects another
                         element's presentation.
    name:     string  — the native `name` attribute, needed for this field
                         to actually submit as part of a form.
    required: boolean — the native `required` attribute.
    disabled: boolean — the native `disabled` attribute, applied to the
                         input only (see below).

  On NOT propagating `disabled` to the toggle button: Radix's own real
  PasswordToggleFieldToggle is a fully independent, composable primitive
  with no automatic coupling to the input's disabled state at all — a
  caller wires that up themselves if they want it. This port keeps that
  same decoupling rather than inventing a coupling Radix itself doesn't
  have: a disabled input's existing value can still be legitimately
  worth revealing for visual review (e.g. confirming what was
  autofilled) even while editing is blocked, so forcibly disabling the
  toggle alongside the input would remove a capability without a
  documented reason to.

  On the ARIA-boolean attributes (`aria-pressed`, `aria-label`) on the
  toggle button, and the falsy-attribute-omission audit they require (see
  Toggle.vue's header comment for the general pitfall: this
  codebase's dynamic-attribute binding omits an attribute entirely
  whenever its bound expression evaluates to Go's bare `false`, which is
  correct for real HTML boolean/presence attributes like `disabled` but
  wrong for a tristate ARIA attribute like `aria-pressed`, whose very
  *absence* means something different from "false" to assistive
  technology): unlike Toggle.vue, this component has no `pressed`-style
  boolean *prop* feeding `aria-pressed` at all — per the "always starts
  hidden" design above, the pre-JS baseline has exactly one possible
  state, full stop. There is therefore no dynamic Go boolean expression
  bound to `aria-pressed`/`aria-label` here for the omission rule to ever
  mis-fire on; both are plain literal template text —
  `aria-pressed="false"` and `aria-label="Show password"` — the same
  "constant ARIA attribute rendered as a static literal, not a `:`
  binding" pattern Checkbox.vue's own box already establishes for its
  static `aria-hidden="true"`. A static literal can never be silently
  dropped by the falsy-omission rule, because that rule only ever
  triggers on a *bound* expression evaluating to bare `false` — so this
  sidesteps the pitfall entirely rather than needing the ternary
  coercion fix Toggle.vue's `:aria-pressed="pressed ? 'true' : 'false'"`
  applies. (Should a future revision add a caller-facing prop that drives
  this button's initial state, that binding would need the exact same
  `propName ? 'true' : 'false'` coercion Toggle.vue already established —
  documented here so that trap doesn't get reintroduced silently.) The
  *runtime* flip after a click is handled entirely in
  <script customelement> below via `setAttribute`, which needs no such
  coercion since it always writes an explicit string.

  <script customelement> (load-bearing, not just polish): on toggle
  click, flips the sibling <input>'s own `type` attribute between
  `"password"` and `"text"` — the actual, load-bearing behavior; nothing
  else in this browser would ever do this for us — and updates the
  button's `aria-pressed`/`aria-label` and visible icon text to match,
  using the same `setAttribute` approach Toggle.vue's script already
  established for its own two-state flip.

  On focus management after toggling (a deliberate choice, not a
  default): Radix's own real implementation moves focus back to the
  input after a *pointer*-triggered toggle (restoring the prior cursor/
  selection position) while leaving focus on the button after a
  *keyboard*-triggered toggle (Enter/Space activation already leaves
  focus on the button, which is also where a keyboard user's attention
  already is) — a real, deliberately nuanced distinction, backed by
  pointer/selection-tracking state this port does not reimplement. This
  component instead makes one simpler, deliberate choice for both input
  modalities: after every toggle, focus moves back to the password
  input. Rationale: the toggle button's entire purpose is to let someone
  keep looking at (and typically keep typing/editing) the password field
  underneath it — once visibility has changed, the input is almost
  always where the user's next action belongs, and leaving focus
  stranded on the button would force an extra Tab/Shift+Tab just to
  resume typing. This is a real, deliberate trade-off against Radix's
  finer-grained behavior (a keyboard user who fires the toggle with
  Space and immediately presses Space again, expecting to toggle a
  second time, will instead land back in the input) — judged worth it for
  the more common "toggle, then keep typing" flow, and it keeps this
  component's script meaningfully simpler than reimplementing Radix's
  pointer-vs-keyboard heuristic and cursor-position bookkeeping.

  Usage:
    <Label for="pw">Password</Label>
    <PasswordToggleField id="pw" name="password" :required="true" :disabled="false" />
-->
<template>
  <Tokens></Tokens>
  <span class="radix-password-toggle-field">
    <input
      type="password"
      class="radix-password-toggle-field-input"
      :id="id"
      :name="name"
      :required="required"
      :disabled="disabled"
      autocomplete="current-password"
      autocapitalize="off"
      spellcheck="false"
    />
    <button
      v-native
      type="button"
      class="radix-password-toggle-field-toggle"
      :aria-controls="id"
      aria-pressed="false"
      aria-label="Show password"
    ><span class="radix-password-toggle-field-icon" aria-hidden="true">Show</span></button>
  </span>
</template>

<style>
.radix-password-toggle-field {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.radix-password-toggle-field-input {
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--radix-brown-6);
  border-radius: var(--radix-radius-3);
  font: inherit;
}

.radix-password-toggle-field-input:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-password-toggle-field-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.6rem;
  background-color: var(--radix-brown-1);
  border: 1px solid var(--radix-brown-6);
  border-radius: var(--radix-radius-3);
  cursor: pointer;
  line-height: 1;
  font-size: 0.85rem;
}

.radix-password-toggle-field-toggle[aria-pressed='true'] {
  background-color: var(--radix-brown-4);
  border-color: var(--radix-brown-9);
  color: var(--radix-brown-9);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-password-toggle-field-toggle:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the always-masked, zero-JS baseline
// above. Per RFC 014 §3 Non-Goals, this reimplements nothing the browser
// already provides for the password input itself (masking, form
// submission, required validation) — it only does the one thing that is
// genuinely impossible any other way: flip the input's own `type`
// attribute, which is the real, load-bearing behavior behind "show/hide
// password". Toggling a class or data-attribute alone would be pure
// theater — the browser only actually unmasks characters when the real
// `type` attribute itself is `"text"`, so this handler mutates that
// attribute directly, not some cosmetic stand-in for it.
class RadixPasswordToggleField extends HTMLElement {
  #input = null
  #button = null
  #icon = null

  #onClick = () => {
    const nextVisible = this.#input.getAttribute('type') !== 'text'

    // The actual behavior: flip the real `type` attribute so the browser
    // itself starts rendering plaintext instead of masked dots. Every
    // other line below is bookkeeping to keep the button's ARIA state and
    // label in sync with this one source of truth.
    this.#input.setAttribute('type', nextVisible ? 'text' : 'password')

    this.#button.setAttribute('aria-pressed', String(nextVisible))
    this.#button.setAttribute('aria-label', nextVisible ? 'Hide password' : 'Show password')
    this.#icon.textContent = nextVisible ? 'Hide' : 'Show'

    // Deliberate choice (see this file's header comment): always return
    // focus to the password input after toggling, regardless of whether
    // the click came from a pointer or the keyboard, so the user can
    // resume typing/editing immediately.
    this.#input.focus()
  }

  connectedCallback() {
    this.#input = this.querySelector('.radix-password-toggle-field-input')
    this.#button = this.querySelector('.radix-password-toggle-field-toggle')
    this.#icon = this.querySelector('.radix-password-toggle-field-icon')
    if (!this.#input || !this.#button || !this.#icon) return

    this.#button.addEventListener('click', this.#onClick)
  }

  disconnectedCallback() {
    if (this.#button) {
      this.#button.removeEventListener('click', this.#onClick)
    }
  }
}

customElements.define('radix-password-toggle-field', RadixPasswordToggleField)
</script>
