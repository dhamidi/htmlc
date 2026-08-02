<!--
  AlertDialog — modal, non-light-dismissable confirmation dialog built on
  the native <dialog> element. Nearly identical to Dialog.vue; read that
  file's header first, this comment only calls out what differs.

  ## Why this needs its own component instead of just "Dialog with a prop"

  An alert dialog interrupts the user to demand an explicit choice — e.g.
  "Delete this file? [Cancel] [Delete]" — and must not be dismissable by
  accident. That changes two things relative to Dialog.vue:

  1. Role. Native <dialog> shown via showModal() already carries an
     implicit ARIA role of "dialog" — which is exactly why Dialog.vue
     never sets role="dialog" explicitly; the implicit role already
     matches. There is no such implicit "alertdialog" role for <dialog>,
     so it must be set explicitly here, or assistive tech announces this
     as a plain dialog and the user loses the "this demands a decision"
     semantic.

  2. Escape must not dismiss it. It's tempting to assume this component
     needs to prevent light-dismiss-on-outside-click too, but that turns
     out to be a non-issue: a <dialog> opened via showModal() has no
     native outside-click dismissal to begin with (that's a
     popover="auto"-API behavior, not a <dialog> one), so Dialog.vue never
     had to suppress it and neither does this component. Escape is the
     real gap: native <dialog> *does* close on Escape by default even
     when opened modally. The fix is to listen for the dialog's native
     "cancel" event — fired when Escape is pressed on an open <dialog>,
     before it closes — and call event.preventDefault() on it. Per the
     WHATWG HTML spec's "close the dialog" steps, the dialog is only
     actually closed if that cancel event is *not* canceled; canceling it
     leaves the dialog open, forcing the user back to one of the explicit
     action buttons below. See the <script customelement> block.

  Everything else — the `open`-attribute zero-JS baseline, the
  removeAttribute('open') -> showModal() sequencing fix for the
  InvalidStateError gotcha, and the <form method="dialog"> zero-JS close
  mechanism — is reused verbatim from Dialog.vue; see that file for the
  full rationale.

  Prop:
    open: boolean
      Same contract as Dialog.vue's `open` prop: false/nil/undefined omits
      the `open` attribute (closed); true renders a bare `open` attribute
      (the non-modal zero-JS baseline, upgraded to a true modal on connect
      by the <script customelement> block below).

  Slots:
    default — title/description/body content, e.g.:
      <AlertDialog :open="confirmingDelete">
        <h2>Delete this file?</h2>
        <p>This action cannot be undone.</p>
      </AlertDialog>

    actions — the caller-provided confirm/destructive action button(s).
      This component always renders its own "Cancel" button (closes via
      the same zero-JS <form method="dialog"> mechanism Dialog.vue uses,
      no script required) alongside whatever is placed in this slot, e.g.:
        <AlertDialog :open="confirmingDelete">
          <h2>Delete this file?</h2>
          <p>This action cannot be undone.</p>
          <template #actions>
            <button type="button" onclick="deleteFile()">Delete</button>
          </template>
        </AlertDialog>
      What the action button actually does (the delete call, a form
      submission, etc.) is entirely the caller's concern — this
      component's job is only the correctly-marked-up, correctly-behaving
      dialog shell, not the semantics of the destructive action itself.

  v-native on the <dialog> tag below: same self-reference-cycle fix as
  Dialog.vue. This component's own name (AlertDialog) auto-registers a
  lowercase "alertdialog" alias in the component registry, but the literal
  tag here is <dialog>, a genuine native element, not that alias — so this
  guards against the same class of collision Dialog.vue documents, even
  though the literal tag names themselves differ.

  v-native on the <form> tag below: same fix as Dialog.vue's own
  <form v-native method="dialog"> — see that file's header comment for the
  full empirically-confirmed finding. Once Form.vue exists in this same
  package, its name auto-registers a lowercase "form" alias that this
  literal <form method="dialog"> would otherwise resolve to instead of
  staying a plain native form, breaking the zero-JS Cancel mechanism.
-->
<template>
  <dialog v-native role="alertdialog" class="radix-alert-dialog" :open="open">
    <slot></slot>
    <div class="radix-alert-dialog-actions">
      <form v-native method="dialog" class="radix-alert-dialog-cancel-form">
        <Button variant="default" type="submit" :disabled="false">Cancel</Button>
      </form>
      <slot name="actions"></slot>
    </div>
  </dialog>
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

.radix-alert-dialog {
  padding: 1.5rem;
  max-width: 32rem;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-4);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
}

/* Only takes visible effect once the dialog is actually opened via
   showModal() — the non-modal `open`-attribute baseline state never
   enters the top layer, so it never has a ::backdrop to style. Same
   documented degradation as Dialog.vue. */
.radix-alert-dialog::backdrop {
  background: rgba(0, 0, 0, 0.5);
}

.radix-alert-dialog-actions {
  margin: 1rem 0 0;
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.radix-alert-dialog-cancel-form {
  margin: 0;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-alert-dialog-cancel:focus-visible {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the non-modal <dialog open> baseline
// above. Reuses Dialog.vue's modal-promotion and focus-restore behavior
// verbatim, and adds exactly one new piece: intercepting the dialog's
// native "cancel" event (fired when Escape is pressed on an open modal
// <dialog>, before it closes) and calling event.preventDefault() on it so
// Escape cannot dismiss this alert dialog. Per spec, the dialog's "close
// the dialog" steps only run if that cancel event was not canceled, so
// preventDefault() here genuinely blocks the close, not just the event's
// default in some cosmetic sense.
class RadixAlertDialog extends HTMLElement {
  #dialog = null
  #triggeringElement = null
  #onClose = () => {
    // Restore focus to whatever had focus before the dialog opened (e.g.
    // the button that triggered it), if it's still attached to the
    // document. Native <dialog> does not do this for us.
    if (this.#triggeringElement && this.#triggeringElement.isConnected) {
      this.#triggeringElement.focus()
    }
    this.#triggeringElement = null
  }
  #onCancel = (event) => {
    // Block Escape-to-close: an alert dialog demands an explicit choice
    // via one of its action buttons, not an accidental/impulsive Escape
    // press. Canceling this event is what stops the dialog from closing;
    // see the header comment above for the spec citation.
    event.preventDefault()
  }

  connectedCallback() {
    this.#dialog = this.querySelector('dialog')
    if (!this.#dialog) return

    this.#triggeringElement = document.activeElement

    // The server-rendered zero-JS baseline may already have the `open`
    // *attribute* set (non-modal state). Calling showModal() on a
    // <dialog> that already carries the open attribute throws
    // InvalidStateError ("already has an 'open' attribute, and therefore
    // cannot be opened modally") per spec, so the attribute must be
    // removed first — showModal() itself re-adds it as part of entering
    // the modal/top-layer state.
    if (this.#dialog.hasAttribute('open')) {
      this.#dialog.removeAttribute('open')
      this.#dialog.showModal()
    }

    this.#dialog.addEventListener('close', this.#onClose)
    this.#dialog.addEventListener('cancel', this.#onCancel)
  }

  disconnectedCallback() {
    if (this.#dialog) {
      this.#dialog.removeEventListener('close', this.#onClose)
      this.#dialog.removeEventListener('cancel', this.#onCancel)
    }
  }
}

customElements.define('radix-alert-dialog', RadixAlertDialog)
</script>
