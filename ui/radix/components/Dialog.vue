<!--
  Dialog — Radix-inspired modal dialog built on the native <dialog> element.

  Zero-JS baseline: <dialog>'s genuinely modal behavior — top-layer
  stacking, the ::backdrop pseudo-element, focus containment, light
  dismiss, and Escape-to-close — is only available when the element is
  opened via its .showModal() *method*. There is no attribute that
  produces that behavior; the only zero-JS lever is the `open` attribute,
  which renders <dialog> non-modally, inline in normal document flow: no
  backdrop, no top-layer, no focus containment, but visible and readable.
  Per RFC 014 §4.1, that degraded-but-functional state is the intended
  zero-JS baseline for this component, not a bug to work around.

  Prop:
    open: boolean
      Bound as :open="open". Per this series' falsy-attribute-omission
      behavior (commit c1671b8), false/nil/undefined omits the `open`
      attribute entirely (dialog closed — the native <dialog> default);
      true renders a bare `open` attribute (dialog visible, non-modally —
      the zero-JS baseline described above). The <script customelement>
      block below upgrades a baseline-open dialog to a true modal on
      connect; see that block for the required attribute-removal-before-
      showModal() sequencing.

  Slot:
    default — arbitrary caller-composed content (heading, body, form
    fields, footer, etc.), rendered inside the <dialog>. Unlike
    Accordion/Tabs' `items` array prop, dialog content has no repeated
    list shape, so it is composed via <slot/>, matching this codebase's
    documented default-slot pattern (README.md, "Default slot"):

      <Dialog :open="showDialog">
        <h2>Title</h2>
        <p>Body copy.</p>
      </Dialog>

  Native, zero-JS close mechanism: the <form method="dialog"> button
  below submits with zero JavaScript and natively closes the *nearest
  ancestor* <dialog> — a real, spec'd HTML feature
  (https://html.spec.whatwg.org/multipage/interactive-elements.html#attr-fs-formmethod-dialog),
  not something this component hand-rolls. It works identically in both
  the non-modal baseline (open attribute) and the enhanced modal
  (showModal()) state.

  Declarative trigger pattern (not part of this component): callers on
  browsers that support the HTML Invoker Commands API can open this
  dialog with zero JS from their *own* template, without an :open prop
  at all, e.g.:

    <button commandfor="my-dialog-id" command="show-modal">Open</button>
    <Dialog id="my-dialog-id">...</Dialog>

  That trigger button is orthogonal to Dialog.vue itself — it lives in
  the caller's markup, not in this file — so it is documented here only
  as a pattern, not wired up as part of this component's own template.

  Usage (RFC 014 §6):
    <Dialog :open="showDialog">
      <h2>Title</h2>
      <p>Body copy.</p>
    </Dialog>
-->
<template>
  <dialog class="radix-dialog" :open="open">
    <slot></slot>
    <form method="dialog" class="radix-dialog-close-form">
      <button type="submit" class="radix-dialog-close">Close</button>
    </form>
  </dialog>
</template>

<style scoped>
.radix-dialog {
  padding: 1.5rem;
  max-width: 32rem;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
}

/* Only takes visible effect once the dialog is actually opened via
   showModal() — the non-modal `open`-attribute baseline state never
   enters the top layer, so it never has a ::backdrop to style. That is
   expected: it's the zero-JS baseline's documented degradation, not a
   bug in this rule. */
.radix-dialog::backdrop {
  background: rgba(0, 0, 0, 0.5);
}

.radix-dialog-close-form {
  margin: 1rem 0 0;
  text-align: right;
}

.radix-dialog-close {
  padding: 0.4rem 0.9rem;
  cursor: pointer;
}

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-dialog-close:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 2px;
}
</style>

<script customelement>
// Progressive enhancement on top of the non-modal <dialog open> baseline
// above. Per RFC 014 §3 Non-Goals, this does not reimplement anything the
// browser already provides once showModal() is in play (top-layer,
// ::backdrop, initial focus, Escape-to-close, inert background) — it only
// (a) promotes a server-rendered baseline-open dialog to a true modal, and
// (b) restores focus to whatever triggered the dialog when it closes,
// which native <dialog> does not do on its own.
class RadixDialog extends HTMLElement {
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
  }

  disconnectedCallback() {
    if (this.#dialog) {
      this.#dialog.removeEventListener('close', this.#onClose)
    }
  }
}

customElements.define('radix-dialog', RadixDialog)
</script>
