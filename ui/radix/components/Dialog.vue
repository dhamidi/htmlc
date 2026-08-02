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

  v-native on the <dialog> tag below: this component's own name (Dialog)
  auto-registers a lowercase "dialog" alias in the component registry (the
  standard entries[lower] = entry convention). Without v-native, the literal
  <dialog> element in this template would resolve right back to this
  component itself, an infinite self-reference reported as "cycle
  detected". v-native declares this tag a genuine native HTML element so
  the component keeps working without every consumer having to add "dialog"
  to their own Options.NativeElements just to use this library.

  v-native on the <form> tag below (added alongside Form.vue): once
  Form.vue exists in this same package, its own name ("Form")
  auto-registers a lowercase "form" alias in the component registry the
  same way — and this file's own close mechanism is a literal <form
  method="dialog">, which would otherwise resolve to Form.vue's component
  instead of staying a plain native form, silently breaking the zero-JS
  close mechanism documented above (a <div>, which is what Form.vue
  renders, does not submit or trigger <form method="dialog">'s native
  close-the-nearest-ancestor-dialog behavior at all). Confirmed empirically
  the same way Checkbox.vue's header comment confirms its own analogous
  <label> finding: rendering this component through a real htmlc.Engine
  with the full ui/radix package mounted, once Form.vue was added, turned
  this <form> into a Form.vue instance with "[missing: id]"/"[missing:
  label]" placeholders and swallowed the Close button into a slot — before
  this v-native was added to fix it.
-->
<template>
  <dialog v-native class="radix-dialog" :open="open">
    <slot></slot>
    <form v-native method="dialog" class="radix-dialog-close-form">
      <Button variant="default" type="submit" :disabled="false">Close</Button>
    </form>
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

.radix-dialog {
  padding: 1.5rem;
  max-width: 32rem;
  border: 1px solid var(--radix-gray-6);
  border-radius: var(--radix-radius-4);
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

/* Never remove the focus outline — keep it visible for keyboard users. */
.radix-dialog-close:focus-visible {
  outline: 2px solid var(--radix-blue-9);
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
