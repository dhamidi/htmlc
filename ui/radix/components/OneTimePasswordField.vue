<!--
  OneTimePasswordField — Radix-inspired segmented one-time-code entry field.

  ## Baseline: the real, single, native <input> — NOT a degraded experience

  Same framing as Select.vue's/ScrollArea.vue's own header comments (see
  either for the fuller argument this port's design follows): a single
  native <input type="text" inputmode="numeric" pattern="[0-9]*"
  autocomplete="one-time-code"> is already a complete, fully-accessible,
  fully-functional OTP entry control on its own — a user can type or paste
  the whole code into it, `maxlength` caps it at the right length, and
  `pattern="[0-9]*"` gives real client-side validation feedback (`:invalid`)
  with zero JavaScript. Critically, `autocomplete="one-time-code"` is not
  this port's own invention or a cosmetic attribute value — it is the real,
  standard token (WHATWG HTML's Autofill spec) browsers and password
  managers already key off of to recognize an OTP field and offer to fill
  it from an SMS/email code, exactly the behavior real users expect from
  "the code field" on a 2FA screen. A hand-rolled N-box implementation that
  skipped this single real input entirely (e.g. wiring autocomplete only
  onto one of several small per-character inputs, or onto none of them)
  would have to work much harder to participate in that same autofill path
  correctly. This baseline isn't a fallback to apologize for; it is the
  actual, correct, standards-based way to build this control, and it is
  what a no-JS request/browser/AT combination gets in full.

  ## The `length`-as-array decision (read this before the props section)

  Radix's own real OneTimePasswordField takes a `maxLength` implied by
  however many <OneTimePasswordFieldInput> children a caller composes by
  hand (see the read-only source referenced below) — this port instead
  needs to render N *visual* boxes from a single `length`-shaped prop, via
  this template language's `v-for`. That immediately runs into the same
  wall Slider.vue's/ScrollArea.vue's own header comments already document
  for their own "generate N of something from a bare count" needs: this
  codebase's expression language (see expr/doc.go's own "Unsupported
  Constructs" list) has no range-generation construct — no `range(n)`
  built-in (none are pre-registered; see doc.go's "Built-in Functions"),
  no spread operator, nothing that turns a bare integer into N loop
  iterations. `v-for` only ever iterates a real array or object already
  present in scope (confirmed against every existing `v-for` in this
  package — Slider.vue's `v-for="(v, index) in values"`, RadioGroup.vue's/
  ToggleGroup.vue's own `v-for="item in items"` — none of them loop over a
  bare number). Rather than invent a template-side workaround this
  language does not support, this file follows Slider.vue's own precedent
  exactly: `length` is not a bare number here — it is an **array** prop
  (e.g. `Array(6).fill(null)`, constructed once by the caller/Go side, the
  same way a caller already builds `items`/`values` arrays for
  RadioGroup.vue/Slider.vue), and `v-for="(digit, index) in length"` below
  iterates it to render one visual box per entry. The array's own contents
  are never read (each entry is a placeholder, `null` or otherwise) — only
  its *length* matters, read via this language's built-in `.length` member
  property (doc.go, "the common case of measuring collection sizes") for
  both the box count and the real input's own `:maxlength="length.length"`.
  This keeps the prop's name ("length") intuitive for callers while working
  within the one real constraint this template language imposes — the same
  trade this package already made for Slider.vue's `values: number[]`.

  ## Two-layer design: one real input, N synced visual boxes

  Structure:

    <span class="radix-otp-field">
      <input class="radix-otp-field-input" type="text" inputmode="numeric"
             pattern="[0-9]*" autocomplete="one-time-code" :maxlength="..."
             :name="name" />
      <span class="radix-otp-field-boxes" data-state="hidden">
        <span class="radix-otp-field-box" role="textbox" :aria-label="..."
              tabindex="-1"></span>   (one per entry in `length`)
        ...
      </span>
    </span>

  Pre-JS: `.radix-otp-field-boxes` renders with `data-state="hidden"`
  (`display: none` — see <style scoped> below), so the *only* visible,
  operable control on the page is the real input above, exactly the
  baseline described above — a sighted, zero-JS user sees one plain,
  fully-usable text field, never an inert row of empty decorative boxes
  masquerading as something typable. This is the "keep it visible" half of
  this commit's own brief: unlike Select.vue's real <select>, which is
  *always* clip-hidden because a genuine zero-JS mechanism
  (`popovertarget`) already powers its visible replacement, there is no
  equivalent zero-JS way to make N separate boxes independently focusable
  and typable — so, unlike Select.vue, this file cannot afford to hide the
  one thing that actually works before JS runs.

  Post-JS (`<script customelement>`'s `connectedCallback`): the boxes'
  `data-state` flips to `"visible"` and the real input's own `data-state`
  flips to `"hidden"` (a *different* attribute/rule pair, `.radix-otp-
  field-input[data-state='hidden']`, also `display: none` — see <style
  scoped>). This is a deliberate, documented departure from Select.vue's
  own choice to keep its native control simultaneously, independently
  reachable alongside its custom layer ("Two real, independently-operable
  controls" in that file's header comment): here, once boxes are live they
  are a strict, full replacement for the plain input as the interactive
  and accessible surface, not a second parallel stop for the same value —
  Select.vue's two-stops design was justified by the native <select> and
  the custom trigger being two *genuinely different, both-useful*
  affordances (native dropdown UI vs. custom-styled listbox); here, both
  "the plain input" and "the box row" are the exact same affordance (typing
  a code) wearing two different outfits, so leaving both simultaneously
  focusable would just be two redundant Tab stops for identical capability,
  not two meaningfully different ones. `display: none` (not Select.vue's
  clip-path) is the correct tool for that full swap: unlike Checkbox.vue's/
  RadioGroup.vue's/Select.vue's own hidden inputs — which all need to
  *stay* keyboard/AT-reachable, hence clip-path — this input's whole point
  once hidden is to stop being an independent interaction target;
  `display: none` does not remove it from form submission (only a real
  `disabled` attribute would do that — untouched here), so it remains the
  actual value the form submits either way. The data-state-attribute-driven
  CSS toggle itself (rather than a raw DOM `.hidden` property flip or an
  inline style) directly reuses Avatar.vue's own established
  `data-state="hidden"`/`data-state="visible"` pattern for exactly this
  kind of script-driven visibility swap.

  On the boxes' `role="textbox"` — an honest, documented v1 approximation,
  not a claim of full native-input parity: WAI-ARIA's textbox role is
  formally meant for a genuinely editable widget (a real `<input>`/
  `<textarea>`, or a `contenteditable` element), and a plain, non-
  `contenteditable` `<span>` whose content this script rewrites via
  `textContent` is not that. It is, however, the closest real ARIA role
  available for "a small, independently focusable, keyboard-operable
  single-character field" with no dedicated widget-role of its own in the
  ARIA taxonomy — the same "reasonable, honestly-scoped v1" latitude this
  commit's own brief and Select.vue's own reverse-sync section both invoke
  for an analogous imperfect-but-documented choice. Each box's `aria-label`
  ("Character N of <length>") is computed entirely server-side from the
  `length` array/`index`, the same "no client-side lookup needed for
  correctness on first paint" discipline Select.vue's own trigger-label
  computation already established, reusing the exact
  `'literal ' + (index + 1) + ' of ' + n` string-concatenation shape
  Slider.vue's own per-thumb `aria-label` binding already uses.

  ## `<script customelement>` (the load-bearing part)

  1. **Sync**: `#syncFromInput` re-paints every box's `textContent` from
     the real input's own live `.value` (never the other way around — the
     input is the single source of truth throughout, the same "native
     element stays authoritative" design Select.vue's own header comment
     states plainly), run once unconditionally on `connectedCallback` (so
     a value already present on the input — e.g. from a password manager's
     autofill that ran before this script attached — is reflected in the
     boxes immediately) and again after every mutation this script itself
     makes to `.value`.
  2. **Typing a digit** (`#onKeydown`, a single-digit `event.key`):
     overwrites the input's value at the focused box's own index (via
     plain string slicing — `value.slice(0, index) + digit +
     value.slice(index + 1)`), dispatches real `input`/`change` events on
     the input (matching Select.vue's own precedent for keeping external
     listeners correctly informed), re-syncs the boxes, and moves focus to
     the next box — or, if the digit just filled the *last* box, blurs the
     current one instead (there being no next box to advance into).
  3. **Backspace** (`#onKeydown`): two cases —
       - the focused box already holds a character: that character is
         removed (the value past it shifts left by one, so no input value
         ever contains an internal gap a plain string cannot represent),
         focus stays on the same box.
       - the focused box is already empty (nothing to remove there): focus
         moves to the *previous* box and clears *that* box's character.
  4. **Paste** (`#onPaste`, on the boxes container — pasting only makes
     sense once the box row is the live interactive surface): reads
     `event.clipboardData`, strips every non-digit character (`/[^0-9]/g`
     — a real, common OTP-paste robustness concern: a code copied with
     spaces or dashes, e.g. "12 34-56", must not be rejected wholesale),
     takes at most `length.length` digits, writes the result straight into
     the input's `.value`, dispatches `input`/`change`, re-syncs the
     boxes, and focuses the last filled box if the paste supplied a full
     code, or the first still-empty box otherwise.

  Every one of points 2-4 above funnels its own value mutation through one
  private helper, `#setValue`, which writes `.value`, dispatches both real
  events, and re-runs `#syncFromInput` — the same "one function, one path,
  nothing to drift out of sync with itself" discipline Select.vue's own
  `#syncFromNative`/ScrollArea.vue's own `#syncThumb` both already
  establish for their own analogous single-source-of-truth reconciliation.
  Only "frontier and filled" boxes (index <= the value's current length)
  ever carry `tabindex="0"` — the rest stay at the template's own static
  `tabindex="-1"` default — the identical roving-tabindex convention
  Select.vue's/DropdownMenu.vue's own `#syncTabindex` already establish,
  reused here to stop a user from tabbing/clicking directly into a box
  past the next fillable position and creating a value with an
  unrepresentable internal gap.

  Out of scope for this v1 (documented, not silently omitted, matching
  this package's established "small, honest v1" discipline — see
  DropdownMenu.vue's own "Positioning" section or ScrollArea.vue's own
  "Vertical-only" section for the same kind of call): arrow-key navigation
  between boxes, `Delete` (forward-delete), IME/composition input, and an
  `autoSubmit`-style callback on completion. None of these are needed for
  the documented auto-advance/backspace/paste-splitting contract this
  commit's brief actually asks for, and each would be a small, additive,
  mechanically-similar follow-up rather than a redesign if a future caller
  needs it.

  ## Props (both REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):
    length: array  — one entry per box, e.g. `Array(6).fill(null)` for a
                      6-digit code (see "The `length`-as-array decision"
                      above for why this is an array, not a bare number).
                      Only `length.length` is ever read; entries themselves
                      are placeholders.
    name:   string — the native `name` attribute on the real <input>;
                      needed for it to actually submit as part of a form.

  Usage — `Array(6).fill(null)` above (and in "The `length`-as-array
  decision" section) describes how the *caller* builds the value in Go
  (e.g. `make([]any, 6)`), not template syntax: this template language has
  no `Array(...)` global and no `.fill(...)` array method (confirmed
  against expr/doc.go's own "Function Calls"/"Built-in Functions" sections
  — no built-in functions ship pre-registered, and there is no method-call
  syntax on arrays beyond the `.length` member property), so a caller
  either binds to an already-built Go-side array already in scope, or uses
  this language's own array-literal syntax (doc.go, "Array and Object
  Literals") directly, the same literal-array shape Slider.vue's own
  `:values="[25, 75]"` usage example already demonstrates:

    <OneTimePasswordField :length="[null, null, null, null, null, null]" name="otp" />

  or, referencing a Go-side value already in scope (plain prose, not an
  inline HTML comment, below — a literal comment-close sequence inside
  this outer header comment would truncate it early, the exact trap
  Slider.vue's own header comment already hit and documents fixing once):
  the page's own Go code passed otpLength as page/scope data, built as
  make([]any, 6) or the moral equivalent of Array(6).fill(null):

    <OneTimePasswordField :length="otpLength" name="otp" />

  ## Self-reference-cycle check (the trap Label.vue's/Dialog.vue's/
  Select.vue's own header comments document for their own same-named native
  tags): this component's own name, "OneTimePasswordField", auto-registers
  a lowercase "onetimepasswordfield" alias in the component registry — but
  "onetimepasswordfield" is not a native HTML element name, so there is no
  literal tag anywhere in this template for that alias to collide with
  (the elements actually used below — `<span>`/`<input>` — are unrelated
  native tags with their own names). No `v-native` escape hatch is needed
  here, the same conclusion Avatar.vue's own header comment reaches for the
  identical reason.

  ## Verified against Radix's real source before writing this file (read
  only, never transcribed): `packages/react/one-time-password-field/src/
  one-time-password-field.tsx` in the read-only clone. Facts this file's
  design depends on, confirmed there rather than assumed: the real
  `autoComplete` default is `"one-time-code"` (`AutoComplete.OneTimeCode`),
  the real numeric-validation `pattern` is per-character (`'\\d{1}'`, the
  single-input analogue this file's own whole-value `pattern="[0-9]*"`
  adapts to a single multi-character baseline), each real input's
  `aria-label` is `Character ${index + 1} of ${collection.size}` (the exact
  phrasing this file's own per-box `aria-label` reuses), and paste handling
  strips non-digit characters before splitting across inputs (the same
  robustness concern this file's own `/[^0-9]/g` strip implements,
  independently, for its own single-real-input/N-visual-box design rather
  than Radix's own N-real-inputs design).

  ## Custom-element tag name

  This file's own base name, "OneTimePasswordField.vue", derives to
  `radix-one-time-password-field` under the standard `Mount{Prefix:
  "radix"}` this package assumes — see radix.go's header comment for the
  derivation algorithm.
-->
<template>
  <Tokens></Tokens>
  <span class="radix-otp-field">
    <input
      type="text"
      inputmode="numeric"
      pattern="[0-9]*"
      autocomplete="one-time-code"
      class="radix-otp-field-input"
      :maxlength="length.length"
      :name="name"
    />
    <span class="radix-otp-field-boxes" data-state="hidden">
      <span
        v-for="(digit, index) in length"
        class="radix-otp-field-box"
        role="textbox"
        :aria-label="'Character ' + (index + 1) + ' of ' + length.length"
        tabindex="-1"
      ></span>
    </span>
  </span>
</template>

<style>
.radix-otp-field {
  position: relative;
  display: inline-flex;
  align-items: center;
}

/*
 * The real, fully-functional zero-JS baseline input — see this file's
 * header comment for why staying visible by default (rather than
 * Select.vue's always-clip-hidden native <select>) is load-bearing here:
 * there is no zero-JS mechanism that makes the box row below independently
 * operable, so this input must remain the thing a no-JS user actually
 * sees and types into.
 */
.radix-otp-field-input {
  font: inherit;
  text-align: center;
  letter-spacing: 0.3em;
  padding: 0.4rem 0.6rem;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
}

/*
 * Post-JS full swap (see header comment): once the box row below has been
 * revealed, this input is no longer an independent interaction target —
 * display:none removes it from layout/focus/AT reach without touching its
 * real form-submission value (only a real `disabled` attribute — never
 * set here — would do that).
 */
.radix-otp-field-input[data-state='hidden'] {
  display: none;
}

.radix-otp-field-boxes {
  display: inline-flex;
  gap: 0.5rem;
}

/* Zero-JS default: no script has run to make these boxes operable, so they
   stay entirely out of layout/focus/AT reach — see header comment. */
.radix-otp-field-boxes[data-state='hidden'] {
  display: none;
}

.radix-otp-field-box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.75rem;
  font-size: 1.125rem;
  font-family: ui-monospace, 'SFMono-Regular', Consolas, monospace;
  color: var(--radix-gray-12);
  background-color: #fff;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-3);
  cursor: text;
  user-select: none;
}

/* A box currently holding a character gets a subtle highlight — purely
   cosmetic, keyed off real content rather than a separate tracked
   attribute. */
.radix-otp-field-box:not(:empty) {
  border-color: var(--radix-blue-9);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-otp-field-box:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the zero-JS single-<input> baseline
// above (typing/pasting the whole code, native validation via `pattern`,
// and one-time-code autofill are all real, native, zero-JS behavior — see
// this file's header comment). This script's job is exactly the two
// things a plain <input> cannot do on its own: paint a segmented,
// per-character box UI, and make that box UI itself typable while keeping
// the real input the one and only source of truth underneath it. Per
// RFC 014 §3 Non-Goals, this reimplements nothing the browser already
// provides for free on the input itself.
class RadixOneTimePasswordField extends HTMLElement {
  #input = null
  #boxesContainer = null
  #boxes = [] // every .radix-otp-field-box span, in DOM order

  connectedCallback() {
    this.#input = this.querySelector('.radix-otp-field-input')
    this.#boxesContainer = this.querySelector('.radix-otp-field-boxes')
    if (!this.#input || !this.#boxesContainer) return

    this.#boxes = Array.from(this.querySelectorAll('.radix-otp-field-box'))
    if (this.#boxes.length === 0) return

    // The full visibility swap described in this file's header comment:
    // the box row becomes the live interactive surface, the plain input
    // steps back to being a pure value-holder.
    this.#boxesContainer.setAttribute('data-state', 'visible')
    this.#input.setAttribute('data-state', 'hidden')

    // Recompute from the live DOM unconditionally on connect — the same
    // "don't trust server-rendered state to already match the live
    // control" discipline Select.vue's/ScrollArea.vue's own
    // connectedCallback document for their own analogous initial sync.
    // This also picks up any value a password manager/autofill wrote to
    // the input before this script ever attached.
    this.#syncFromInput()
    this.#syncTabindex()

    this.#boxesContainer.addEventListener('keydown', this.#onKeydown)
    this.#boxesContainer.addEventListener('paste', this.#onPaste)
    this.#boxesContainer.addEventListener('click', this.#onClick)
  }

  disconnectedCallback() {
    if (this.#boxesContainer) {
      this.#boxesContainer.removeEventListener('keydown', this.#onKeydown)
      this.#boxesContainer.removeEventListener('paste', this.#onPaste)
      this.#boxesContainer.removeEventListener('click', this.#onClick)
    }
  }

  // Re-paints every box's displayed character from the real input's own
  // live .value — the single source of truth every other method in this
  // class only ever writes to indirectly, via #setValue. Never the other
  // direction: a box's own textContent is never read back into anything.
  #syncFromInput = () => {
    const value = this.#input.value
    this.#boxes.forEach((box, index) => {
      box.textContent = value[index] || ''
    })
  }

  // Roving tabindex: only the "frontier" box (the first still-empty one,
  // i.e. index === value.length) — or the last box, once the value is
  // already full — is a real Tab stop, matching Select.vue's/
  // DropdownMenu.vue's own roving-tabindex convention. This also keeps a
  // user from tabbing/clicking past the frontier into a box whose typing
  // would otherwise require representing an internal gap the real input's
  // plain string value cannot hold.
  #syncTabindex() {
    const value = this.#input.value
    const target = Math.min(value.length, this.#boxes.length - 1)
    this.#boxes.forEach((box, index) => {
      box.setAttribute('tabindex', index === target ? '0' : '-1')
    })
  }

  // The one path every value mutation in this class goes through: write
  // the real input's .value, dispatch real input/change events on it (so
  // any external listener — form validation, a controlled-value caller —
  // fires exactly as it would for a real keystroke on the input itself,
  // matching Select.vue's own precedent for keeping external listeners
  // informed), then re-derive the boxes and roving-tabindex baseline from
  // that same new value. One function, one path — nothing here can drift
  // out of sync with itself.
  #setValue(next) {
    this.#input.value = next
    this.#input.dispatchEvent(new Event('input', { bubbles: true }))
    this.#input.dispatchEvent(new Event('change', { bubbles: true }))
    this.#syncFromInput()
    this.#syncTabindex()
  }

  #focusBox(index) {
    const box = this.#boxes[index]
    if (box) box.focus()
  }

  #boxIndex(event) {
    const box = event.target.closest('.radix-otp-field-box')
    if (!box || box.closest('radix-one-time-password-field') !== this) return -1
    return this.#boxes.indexOf(box)
  }

  // Clicking a box past the current frontier redirects focus to the
  // frontier box instead of the clicked one — the same "don't let a click
  // create an unrepresentable gap" reasoning #syncTabindex's own roving
  // baseline already encodes for keyboard/Tab navigation, applied here to
  // pointer input too.
  #onClick = (event) => {
    const index = this.#boxIndex(event)
    if (index === -1) return
    const value = this.#input.value
    const frontier = Math.min(value.length, this.#boxes.length - 1)
    this.#focusBox(Math.min(index, frontier))
  }

  #onKeydown = (event) => {
    const index = this.#boxIndex(event)
    if (index === -1) return

    // Backspace: two distinct cases, matching this file's header comment
    // (point 3) precisely.
    if (event.key === 'Backspace') {
      event.preventDefault()
      const value = this.#input.value
      if (value[index]) {
        // The focused box already holds a character: remove just that
        // character (shifting anything after it left by one, so the
        // input's own plain string value never needs to represent an
        // internal gap) and stay on the same box.
        this.#setValue(value.slice(0, index) + value.slice(index + 1))
        this.#focusBox(index)
      } else if (index > 0) {
        // The focused box is already empty: move to the previous box and
        // clear *that* box's character instead.
        this.#setValue(value.slice(0, index - 1) + value.slice(index))
        this.#focusBox(index - 1)
      }
      return
    }

    // A single digit key: overwrite the value at this box's own position,
    // then auto-advance — to the next box, or blur (there being no next
    // box) once the last one has just been filled.
    if (/^[0-9]$/.test(event.key)) {
      event.preventDefault()
      const value = this.#input.value
      this.#setValue(value.slice(0, index) + event.key + value.slice(index + 1))
      if (index < this.#boxes.length - 1) {
        this.#focusBox(index + 1)
      } else {
        event.target.blur()
      }
      return
    }
  }

  // Paste-splitting: strip everything but digits (a real, common OTP-paste
  // robustness concern — a code copied with spaces or dashes, e.g.
  // "12 34-56", must not be rejected wholesale), cap at the number of
  // boxes, write the result straight into the real input, then focus the
  // last filled box (a full-length paste) or the first still-empty one (a
  // short paste) — this file's header comment, point 4, verbatim.
  #onPaste = (event) => {
    event.preventDefault()
    const clipboard = event.clipboardData || window.clipboardData
    const text = clipboard ? clipboard.getData('text/plain') : ''
    const digits = text.replace(/[^0-9]/g, '').slice(0, this.#boxes.length)
    if (!digits) return // nothing digit-like was pasted; leave the existing value untouched

    this.#setValue(digits)

    const target =
      digits.length >= this.#boxes.length ? this.#boxes.length - 1 : digits.length
    this.#focusBox(target)
  }
}

customElements.define('radix-one-time-password-field', RadixOneTimePasswordField)
</script>
