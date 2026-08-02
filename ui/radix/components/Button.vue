<!--
  Button — shared, generic clickable-button primitive, used across this
  package (and by consumers) wherever a plain labeled action button is
  needed: Dialog's/AlertDialog's own Close/Cancel buttons, and any
  consumer-authored button in slotted content (a form's submit button, a
  destructive confirm action).

  Deliberately NOT used for every literal <button> in this package.
  Several components render a <button> carrying highly specific ARIA/
  state semantics of their own — a toggle's aria-pressed state
  (Toggle.vue/ToggleGroup.vue), a menu/listbox trigger's popovertarget
  wiring (Popover.vue/DropdownMenu.vue/Menubar.vue/NavigationMenu.vue/
  Select.vue), a roving-tabindex menu item (ContextMenu.vue/
  DropdownMenu.vue/Menubar.vue/Toolbar.vue), or an icon-only control with
  no visible label (Toast.vue's "×" dismiss, PasswordToggleField.vue's
  show/hide toggle). Routing those through this component would mean
  either dropping their specific behavior or growing this file's own
  required-prop surface to cover every one of those distinct shapes —
  neither improves on each of those files already owning its own correct,
  specific markup. This component covers the remaining, genuinely generic
  case: a labeled button that does nothing but look like part of this
  design system and click/submit natively.

  v-native on the <button> below: this component's own name ("Button")
  auto-registers a lowercase "button" alias in the component registry —
  the same self-reference-cycle trap Label.vue's/Dialog.vue's/Form.vue's
  own header comments already document for their own same-named native
  tags. Every *other* literal <button> tag across this package was
  updated, in the commit immediately before this one, to carry v-native
  for the identical reason — done first, so this file introduces no
  collision the moment it lands.

  Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the "[missing: <name>]" placeholder trap this convention avoids):
    variant:  string  — "default" | "primary" | "destructive". "default"
                         is a bordered, neutral-background button (a
                         Cancel/Close-style secondary action); "primary"
                         is a solid accent-color button (the one primary
                         call-to-action on a given screen, e.g. a form's
                         submit button); "destructive" is a solid
                         danger-color button (a delete/irreversible
                         confirm action).
    type:     string  — the native `type` attribute ("button" | "submit"
                         | "reset"). No default of its own is rendered
                         when omitted, because the browser's own implicit
                         default ("submit") has real, surprising
                         consequences for a button placed inside a form —
                         better to require callers to say what they mean.
    disabled: boolean — the native `disabled` attribute.

  Usage:
    <Button variant="default" type="button" :disabled="false">Cancel</Button>
    <Button variant="primary" type="submit" :disabled="false">Sign up</Button>
    <Button variant="destructive" type="button" :disabled="false">Delete</Button>
-->
<template>
  <Tokens></Tokens>
  <button v-native :type="type" :disabled="disabled" class="radix-button" :data-variant="variant">
    <slot></slot>
  </button>
</template>

<style>
.radix-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--radix-sand-6);
  border-radius: var(--radix-radius-2);
  background-color: var(--radix-sand-1);
  color: var(--radix-sand-12);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

.radix-button:hover {
  background-color: var(--radix-sand-3);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-button:focus-visible {
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}

.radix-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-button[data-variant='primary'] {
  border-color: var(--radix-brown-9);
  background-color: var(--radix-brown-9);
  color: #fff;
}

.radix-button[data-variant='primary']:hover {
  border-color: var(--radix-brown-10);
  background-color: var(--radix-brown-10);
}

.radix-button[data-variant='destructive'] {
  border-color: var(--radix-ruby-9);
  background-color: var(--radix-ruby-9);
  color: #fff;
}

.radix-button[data-variant='destructive']:hover {
  border-color: var(--radix-ruby-10);
  background-color: var(--radix-ruby-10);
}
</style>
