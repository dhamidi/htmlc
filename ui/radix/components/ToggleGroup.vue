<!--
  ToggleGroup — Radix-inspired group of toggle buttons, built on the same
  toggle-button behavior Toggle.vue defines (aria-pressed/data-state, click
  toggling), plus group-level exclusivity (single mode) and roving-tabindex
  arrow-key navigation across the group's buttons (multiple mode and
  single mode both need this — it is a property of the *group*, not of any
  one item's own pressed state).

  Two modes, chosen via the required `type` prop:
    - single:   exactly one item may be pressed at a time (radio-like:
                pressing a new item un-presses whichever item was pressed
                before it). Re-pressing the already-pressed item toggles it
                off, returning the group to "nothing pressed" — this
                mirrors each item still being its own independent toggle
                (per Toggle.vue's own click-flips-its-own-state model) with
                group-level exclusivity layered on top, rather than a
                stricter "can never be empty" radio-input model.
    - multiple: any number of items may be pressed independently
                (checkbox-like); no exclusivity logic at all.

  ## The `value`-shape design decision (read this before changing the props)

  This package's expression language (see expr/doc.go) has no array-
  membership-test operator usable from inside a v-for item's own scope: its
  only membership operator is `in`, and `in` tests *map/struct key*
  membership ("k" in obj), not "is this value present in this array" — see
  expr/doc.go's Binary Operators section and expr/eval.go's inOperator,
  which only implements the Map/Struct cases. There is no `.includes(...)`
  method, no array `find`, and no way to register a project-wide helper
  without expr.RegisterBuiltin global state that this file has no access to
  register safely per-instance. Given that, a Radix-style single
  `value: string[]` prop for multiple mode — checking each item with
  `value.includes(item.id)` — is simply not expressible inside this
  template's v-for scope.

  Resolution, adopted per-mode rather than forcing one shape on both:
    - type="single": the item-membership test is a single scalar equality
      (`item.id === value`), which *is* directly expressible in the
      template with no helper needed — so single mode keeps a familiar
      `value: string` prop, the current pressed item's id (or "" for
      nothing pressed), matching RadioGroup.vue's own `value` prop
      convention for consistency across this package's single-selection
      components.
    - type="multiple": with no in-template way to test "is item.id a member
      of this pressed-set", the membership question is pushed onto the
      caller instead of the template: each entry in `items` carries its own
      `pressed: boolean` field, so the per-item pressed state the template
      needs is already sitting right there in item scope — no membership
      test required at all. This is the documented fallback this file's
      design process considered and adopted (requiring pre-annotated items
      rather than inventing a bespoke set-membership expression feature).

  `items[].pressed` is read only when `type === 'multiple'`; `value` is read
  only when `type === 'single'`. Both remain required props regardless of
  which mode is active — this package has no notion of an optional prop
  with a real default (see Progress.vue's/Avatar.vue's header comments for
  the "[missing: <name>]" placeholder trap) — so a multiple-mode caller must
  still pass some `value` (e.g. "", conventionally unused) and a
  single-mode caller's `items` entries may still carry a `pressed` field
  that this file simply never reads. This mild redundancy is the accepted
  trade-off of a required-props-only prop system applied to two structurally
  different selection models sharing one component; see RadioGroup.vue's
  own header comment for the same kind of accepted trade-off elsewhere in
  this package. Unlike a top-level required prop, `items[].pressed` being
  absent on a given item is not a placeholder-string hazard: per-item field
  access in v-for scope is a plain runtime map/struct member read, not a
  Component.Props()-tracked identifier (see component.go's walkForProps,
  which excludes v-for loop variables from prop collection), so a missing
  `pressed` field simply evaluates to expr's UndefinedValue — falsy, i.e.
  "not pressed" — rather than any "[missing: ...]" text.

  ## Zero-JS baseline

  Each item renders as a plain
    <button type="button" :aria-pressed="..." :data-state="...on/off" :tabindex="...">
  reflecting the correct SSR-computed pressed state per item and mode (see
  above) — but, same honest degradation as Toggle.vue (see that file's
  header comment): clicking does nothing without JS, because a plain
  <button> has no native toggle behavior for the browser to provide for
  free.

  ## Roving-tabindex static starting condition (baseline, before JS runs)

  Per APG's roving-tabindex convention, exactly one item in the group is a
  tab stop (`tabindex="0"`) and every other item is removed from the
  sequential tab order (`tabindex="-1"`) — arrow keys move *within* the
  group once focus lands there, and this needs a real, JS-driven starting
  point since (unlike Tabs.vue's/RadioGroup.vue's native radio-input
  groups) plain <button> elements have no native shared-group tab-stop
  behavior at all.

  Which item starts as the tab stop differs by mode, matching APG's
  established distinction between a single-selection widget (jump straight
  to the current selection, the same real native-radio-group behavior
  Tabs.vue's/RadioGroup.vue's own baselines already lean on) and a
  toolbar-shaped multi-selection widget (no one "current selection" exists
  to jump to, so the first item is the sensible universal entry point):
    - single:   the pressed item (`item.id === value`) gets tabindex="0";
                if `value` is the empty string ("nothing pressed" — the
                same empty-string convention RadioGroup.vue's own header
                comment documents for its analogous case), the first item
                (index 0) gets tabindex="0" instead. This empty-string
                check is deliberately narrower than "no item matched
                value": with no in-template way to ask "did any item
                match?" (the same membership-test gap the value-shape
                section above works around), this file defines the
                fallback trigger as the literal empty string specifically,
                by contract — callers representing "nothing pressed" must
                pass `value=""`, not just any non-matching id, for the
                baseline's tab-stop fallback to engage correctly.
    - multiple: the first item (index 0) always gets tabindex="0",
                regardless of pressed state — this is the toolbar-shaped
                case described above.
  The <script customelement> below then takes over *moving* that single
  tabindex="0" as arrow keys are pressed, the same "static baseline +
  script relocates it" split Tabs.vue's own header comment documents for
  its own roving-tabindex baseline.

  ## <script customelement> (load-bearing, not just polish)

  1. Click handling: flips the clicked button's own aria-pressed/data-state
     (the same self-contained toggle-flip Toggle.vue's own script performs
     — see that file's header comment for why this genuinely needs script,
     not just CSS). In single mode, pressing a previously-unpressed button
     additionally un-presses every other button in the group (mutual
     exclusivity); re-pressing the already-pressed button simply un-presses
     it, with no other button affected, per the "each item is its own
     independent toggle" model documented above.
  2. Roving tabindex + arrow-key navigation: ArrowLeft/ArrowRight (plus
     Home/End) move both DOM focus and the tabindex="0" assignment among
     the group's buttons, *wrapping* at the ends — this deliberately
     matches Tabs.vue's own established wrap-around convention
     (`(index + 1) % length`, `(index - 1 + length) % length`) exactly,
     per this component's own design brief to stay consistent with that
     precedent rather than inventing a different (non-wrapping) convention
     here. Arrow-key navigation only *moves focus*; it never presses or
     un-presses anything — pressing is a separate action (a real click, or
     Enter/Space via the native <button> activation the browser already
     provides for free), matching this file's read of Radix's own
     documented roving-focus-group behavior (moving focus and activating a
     selection are kept as two separate concerns).
  3. Dispatches a group-level `radix-toggle-group-change` CustomEvent
     whenever a click changes the selection, carrying `event.detail.type`
     (echoing this instance's own `type` prop) and `event.detail.value` —
     a single id string (or "") for single mode, an array of pressed ids
     for multiple mode — mirroring Radix's own onValueChange(value)
     callback shape per mode, translated to this library's DOM-event-only
     mechanism the same way Toggle.vue's own radix-toggle-change event
     already does for a single button.

  On disabled needing no script-side guard: identical reasoning to
  Toggle.vue's header comment — a disabled native <button> never dispatches
  'click' at all (HTML spec activation behavior), so the click delegation
  below simply never fires for a disabled item. The `disabled` group prop
  only has to reach the real `disabled` attribute in the baseline below,
  which it does, applied uniformly to every item (no per-item disabled
  override in this first pass, matching RadioGroup.vue's identical
  group-level-only `disabled` scope call).

  On `data-orientation="horizontal"` being a fixed literal, not a prop:
  same deliberate v1 scope call RadioGroup.vue's header comment documents
  for its own layout-direction question — a real per-instance
  `orientation` prop (switching both the CSS layout direction and the
  arrow-key axis together, plus a matching `aria-orientation`) is a
  self-contained, purely additive follow-up rather than something this
  first pass needed to block on. Horizontal-only per this component's own
  design brief.

  On the container being `role="group"`, not `role="toolbar"`/
  `role="radiogroup"`: role="group" is the plain, static ARIA container
  role for "a set of related UI elements", requiring no keyboard-pattern
  contract of its own for this file to satisfy beyond what is already
  implemented below (unlike role="toolbar", which carries its own distinct
  APG keyboard-navigation expectations, or role="radiogroup", which implies
  every child has role="radio" — this file keeps every item's own role as
  the plain implicit "button" role, matching Toggle.vue's aria-pressed
  button model exactly rather than switching to aria-checked/role="radio"
  for single mode). This keeps single and multiple mode's markup uniform
  and keeps every item's accessible semantics identical to using Toggle.vue
  standalone, per this component's own design brief.

  Props (all REQUIRED — this package has no notion of an optional prop with
  a real default; see Progress.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):
    items:    Array<{ id, label, pressed }> — matching Accordion.vue's/
              Tabs.vue's/RadioGroup.vue's own `items` shape convention.
                - id:      stable, unique-within-this-instance identifier;
                           also the value compared against `value` in
                           single mode and dispatched in the group-level
                           CustomEvent.
                - label:   this item's visible button text, rendered as
                           plain text.
                - pressed: boolean — this item's own initial pressed state.
                           Read only when `type === 'multiple'`; see the
                           value-shape design decision above for why.
    type:     "single" | "multiple" — selects the exclusivity model; see
              above.
    value:    string — the currently-pressed item's id in single mode, or
              "" for nothing pressed. Read only when `type === 'single'`
              (see above); a multiple-mode caller must still pass some
              string (conventionally "") to satisfy this required prop.
    disabled: boolean — applied to every item's native `disabled` attribute
              uniformly; no per-item override in this first pass.

  Usage:
    <!-- single mode -->
    <ToggleGroup
      type="single"
      value="bold"
      :disabled="false"
      :items="[
        { id: 'bold', label: 'B', pressed: false },
        { id: 'italic', label: 'I', pressed: false },
        { id: 'underline', label: 'U', pressed: false },
      ]"
    />

    <!-- multiple mode -->
    <ToggleGroup
      type="multiple"
      value=""
      :disabled="false"
      :items="[
        { id: 'bold', label: 'B', pressed: true },
        { id: 'italic', label: 'I', pressed: false },
        { id: 'underline', label: 'U', pressed: true },
      ]"
    />
-->
<template>
  <Tokens></Tokens>
  <div
    class="radix-toggle-group"
    role="group"
    data-orientation="horizontal"
    :data-type="type"
    :data-disabled="disabled ? '' : undefined"
  >
    <button
      v-native
      v-for="(item, index) in items"
      type="button"
      class="radix-toggle-group-item"
      :data-value="item.id"
      :aria-pressed="(type === 'single' ? item.id === value : item.pressed) ? 'true' : 'false'"
      :data-state="(type === 'single' ? item.id === value : item.pressed) ? 'on' : 'off'"
      :tabindex="type === 'single' ? (item.id === value ? '0' : (value === '' && index === 0 ? '0' : '-1')) : (index === 0 ? '0' : '-1')"
      :disabled="disabled"
    >{{ item.label }}</button>
  </div>
</template>

<style>
.radix-toggle-group {
  display: inline-flex;
  flex-direction: row;
  gap: 2px;
  border-radius: var(--radix-radius-3);
}

.radix-toggle-group-item {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.7rem;
  background-color: var(--radix-sand-1);
  border: 1px solid var(--radix-sand-6);
  border-radius: var(--radix-radius-3);
  cursor: pointer;
  line-height: 1;
}

.radix-toggle-group-item[data-state='on'] {
  background-color: var(--radix-brown-4);
  border-color: var(--radix-brown-9);
  color: var(--radix-brown-9);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-toggle-group-item:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

.radix-toggle-group-item:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

<script customelement>
// Progressive enhancement on top of the inert-without-JS <button> group
// baseline above. Per RFC 014 §3 Non-Goals, this reimplements nothing the
// browser already provides for free — but, same conclusion Toggle.vue's
// own header comment reaches, there is no native provider for either
// "pressed" state on a plain <button> or for roving-tabindex behavior
// across a set of plain buttons, so both are genuinely this script's job.
class RadixToggleGroup extends HTMLElement {
  #container = null
  #type = 'single'
  #buttons = []

  connectedCallback() {
    this.#container = this.querySelector('.radix-toggle-group')
    if (!this.#container) return

    this.#type = this.#container.getAttribute('data-type') === 'multiple' ? 'multiple' : 'single'
    this.#buttons = Array.from(this.querySelectorAll('.radix-toggle-group-item'))

    // Delegated listeners on the wrapper, matching Tabs.vue's/Accordion.vue's
    // own established delegation pattern for this library's roving-tabindex
    // components, rather than one listener per button.
    this.addEventListener('click', this.#onClick)
    this.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    this.removeEventListener('click', this.#onClick)
    this.removeEventListener('keydown', this.#onKeydown)
  }

  #setPressed(button, pressed) {
    button.setAttribute('aria-pressed', pressed ? 'true' : 'false')
    button.setAttribute('data-state', pressed ? 'on' : 'off')
  }

  #onClick = (event) => {
    const button = event.target.closest('.radix-toggle-group-item')
    // Ignore clicks from anywhere else, and ignore buttons belonging to a
    // nested radix-toggle-group, if any (mirrors Tabs.vue's own
    // closest(...) !== this guard).
    if (!button || button.closest('radix-toggle-group') !== this) return

    // No disabled-state guard needed here: a disabled <button> never
    // dispatches 'click' in the first place (HTML spec, activation
    // behavior), so this listener simply never runs for a disabled item —
    // identical reasoning to Toggle.vue's own click handler.
    const wasPressed = button.getAttribute('aria-pressed') === 'true'
    const nextPressed = !wasPressed
    this.#setPressed(button, nextPressed)

    if (this.#type === 'single' && nextPressed) {
      // Mutual exclusivity: un-press every other button in the group. Only
      // needed on the "became pressed" branch — re-pressing the already-
      // pressed button (nextPressed === false) leaves every other button
      // untouched, per this file's documented "each item stays its own
      // independent toggle" model.
      for (const other of this.#buttons) {
        if (other !== button && other.getAttribute('aria-pressed') === 'true') {
          this.#setPressed(other, false)
        }
      }
    }

    this.#dispatchChange()
  }

  #dispatchChange() {
    let value
    if (this.#type === 'single') {
      const pressed = this.#buttons.find((b) => b.getAttribute('aria-pressed') === 'true')
      value = pressed ? pressed.getAttribute('data-value') : ''
    } else {
      value = this.#buttons
        .filter((b) => b.getAttribute('aria-pressed') === 'true')
        .map((b) => b.getAttribute('data-value'))
    }
    this.dispatchEvent(
      new CustomEvent('radix-toggle-group-change', {
        detail: { type: this.#type, value },
        bubbles: true,
      })
    )
  }

  // Roving tabindex + arrow-key navigation. Moves focus only — it never
  // presses or un-presses anything, keeping "move focus" and "activate a
  // selection" as two separate concerns (Enter/Space activation is left to
  // the native <button>'s own free click-activation behavior).
  #onKeydown = (event) => {
    const button = event.target.closest('.radix-toggle-group-item')
    if (!button || button.closest('radix-toggle-group') !== this) return

    const index = this.#buttons.indexOf(button)
    if (index === -1) return

    let next
    switch (event.key) {
      // Wraps at both ends — deliberately matching Tabs.vue's own
      // established wrap-around convention for this exact roving-tabindex
      // question, not a different (non-wrapping) one invented here.
      case 'ArrowRight': next = (index + 1) % this.#buttons.length; break
      case 'ArrowLeft':  next = (index - 1 + this.#buttons.length) % this.#buttons.length; break
      case 'Home':       next = 0; break
      case 'End':        next = this.#buttons.length - 1; break
      default: return
    }

    event.preventDefault()
    this.#moveFocus(next)
  }

  #moveFocus(index) {
    this.#buttons.forEach((b, i) => {
      b.setAttribute('tabindex', i === index ? '0' : '-1')
    })
    this.#buttons[index].focus()
  }
}

customElements.define('radix-toggle-group', RadixToggleGroup)
</script>
