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
-->
<template>
  <dialog v-native role="alertdialog" class="radix-alert-dialog" :open="open">
    <slot></slot>
    <div class="radix-alert-dialog-actions">
      <form method="dialog" class="radix-alert-dialog-cancel-form">
        <button type="submit" class="radix-alert-dialog-cancel">Cancel</button>
      </form>
      <slot name="actions"></slot>
    </div>
  </dialog>
</template>

<style scoped>
.radix-alert-dialog {
  padding: 1.5rem;
  max-width: 32rem;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
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

.radix-alert-dialog-cancel {
  padding: 0.4rem 0.9rem;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-alert-dialog-cancel:focus-visible {
  outline: 2px solid #2563eb;
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
