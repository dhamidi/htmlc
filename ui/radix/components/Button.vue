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
  <button v-native :type="type" :disabled="disabled" class="radix-button" :data-variant="variant">
    <slot></slot>
  </button>
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

.radix-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-2);
  background-color: #fff;
  color: var(--radix-gray-12);
  font: inherit;
  font-weight: 500;
  cursor: pointer;
}

.radix-button:hover {
  background-color: var(--radix-gray-3);
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-button:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.radix-button[data-variant='primary'] {
  border-color: var(--radix-blue-9);
  background-color: var(--radix-blue-9);
  color: #fff;
}

.radix-button[data-variant='primary']:hover {
  border-color: var(--radix-blue-10);
  background-color: var(--radix-blue-10);
}

.radix-button[data-variant='destructive'] {
  border-color: var(--radix-red-9);
  background-color: var(--radix-red-9);
  color: #fff;
}

.radix-button[data-variant='destructive']:hover {
  border-color: var(--radix-red-10);
  background-color: var(--radix-red-10);
}
</style>
