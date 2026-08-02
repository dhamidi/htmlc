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
     when opened modally.

     The WHATWG HTML spec's "close the dialog" steps say the dialog is
     only actually closed if the "cancel" event it fires on Escape is not
     canceled — which reads like `event.preventDefault()` on that event
     should be enough. It is NOT enough, verified empirically, not
     assumed: a standalone `<dialog>` + `showModal()` +
     `addEventListener('cancel', e => e.preventDefault())` page, driven by
     a real trusted Escape keypress via CDP `Input.dispatchKeyEvent`
     against this repo's own headless Chromium (HeadlessChrome/151.0.7922.71,
     confirmed via CDP `Browser.getVersion`), still closes the dialog —
     because `event.cancelable` on that "cancel" event is `false` in this
     engine, making `preventDefault()` a silent no-op (`defaultPrevented`
     stays `false` after calling it). Whatever the spec text implies in
     the abstract, this is what actually happens here, so this component
     cannot rely on it.

     What does work, verified the same way: intercepting the dialog's
     native "keydown" event for `event.key === 'Escape'` and calling
     event.preventDefault() there. Unlike "cancel", a <dialog>'s keydown
     event is cancelable, and canceling it suppresses the browser's entire
     Escape-to-close default action before the "cancel"/"close" event
     sequence is even queued — confirmed by the same trusted-keypress
     harness: with the keydown intercepted, no "cancel" or "close" event
     fires at all, and the dialog stays open with zero flicker. This is
     the mechanism this component actually uses; see the <script
     customelement> block. The "cancel" listener is still attached too
     (event.preventDefault() there is what the spec's prose describes,
     and costs nothing even though it doesn't do anything in this engine)
     as defense in depth, in case some other code path ever reaches
     "cancel" without going through this dialog's own keydown handler —
     but the keydown interception is the one this component's actual
     Escape-blocking behavior depends on.

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
  <Tokens></Tokens>
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
.radix-alert-dialog {
  padding: 1.5rem;
  max-width: 32rem;
  border: 1px solid var(--radix-brown-6);
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
  outline: 2px solid var(--radix-brown-9);
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the non-modal <dialog open> baseline
// above. Reuses Dialog.vue's modal-promotion and focus-restore behavior
// verbatim, and adds the Escape-blocking behavior documented in the header
// comment above (verified empirically, not assumed from spec text): a
// 'keydown' listener that intercepts Escape and calls
// event.preventDefault() on IT — not on the dialog's 'cancel' event, whose
// event.cancelable is false in this engine, making preventDefault() there a
// silent no-op. Canceling the keydown event suppresses the browser's entire
// Escape-to-close default action before the 'cancel'/'close' event sequence
// is even queued, confirmed by a real trusted-keypress test showing neither
// event fires at all once keydown is intercepted. The 'cancel' listener
// stays attached too, as cheap defense in depth (see header comment), but
// this component's actual Escape-blocking behavior depends on #onKeydown.
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
    // Per spec prose this should be the mechanism that blocks Escape; in
    // this engine event.cancelable is false here, so this call is a
    // verified no-op — kept anyway as free defense in depth (see header
    // comment). #onKeydown below is what actually blocks Escape.
    event.preventDefault()
  }
  #onKeydown = (event) => {
    // Block Escape-to-close at the source: an alert dialog demands an
    // explicit choice via one of its action buttons, not an
    // accidental/impulsive Escape press. Canceling the dialog's own
    // 'keydown' event for Escape suppresses the browser's native
    // close-on-Escape default action outright — verified to work in this
    // engine where canceling the later 'cancel' event does not. See the
    // header comment above for the empirical comparison.
    if (event.key !== 'Escape') return
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
    this.#dialog.addEventListener('keydown', this.#onKeydown)
  }

  disconnectedCallback() {
    if (this.#dialog) {
      this.#dialog.removeEventListener('close', this.#onClose)
      this.#dialog.removeEventListener('cancel', this.#onCancel)
      this.#dialog.removeEventListener('keydown', this.#onKeydown)
    }
  }
}

customElements.define('radix-alert-dialog', RadixAlertDialog)
</script>
