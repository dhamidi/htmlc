<!--
  Label — Radix-inspired accessible wrapper around the native <label>.

  Native <label> already does the real work for free: clicking anywhere on
  a <label>'s text focuses/activates the form control it is associated
  with, either through a `for`/`id` pairing or by the control being nested
  inside the <label> element itself (HTML Standard, "the label element" —
  https://html.spec.whatwg.org/multipage/forms.html#the-label-element).
  That click-to-focus behavior is a real, spec'd browser feature and needs
  zero JavaScript to work; nothing in this file re-implements it.

  Radix's own actual value-add on top of the bare native element is one
  narrow UX fix: double-clicking label text selects it like plain text,
  which is visually distracting and not what a user wants when interacting
  with a form. Upstream Radix fixes this with a JS `onMouseDown` handler
  that calls `preventDefault()` on multi-click mousedowns (while carefully
  excluding clicks that land on a nested `button`/`input`/`select`/
  `textarea`, so double-clicking *inside* a nested control still works
  normally). That handler is Radix's own source and is intentionally not
  transcribed here (this port does not copy Radix source — see below). But
  the *effect* it produces is fully achievable with plain CSS:
  `user-select: none` (see <style scoped> below) prevents the text from
  being selectable in the first place, which sidesteps the double-click
  papercut entirely without needing to distinguish click targets or count
  clicks in JavaScript at all. Native text-selection handling already
  respects `user-select: none` on descendant interactive controls
  correctly (a nested <input>'s own editable text remains selectable
  regardless of the label's `user-select`, since form control text editing
  has its own selection model independent of CSS user-select on an
  ancestor) — so the "don't break selection inside nested controls" carve-out
  that upstream's JS handler exists for is not needed here; CSS alone gets
  the same net result. That is why this component has no custom-element
  enhancement script block: there is no interactive state, no event
  handling, and nothing left to progressively enhance once the CSS fix is
  in place — same reasoning as VisuallyHidden.vue/Separator.vue/
  AspectRatio.vue in this directory.

  Prop (REQUIRED — see the default-value note below):
    for: string
      Maps to the native `for` attribute, pairing this label with a form
      control elsewhere in the document by id (`<Label for="email">` +
      `<input id="email">`).

  On the prop name "for" (not "htmlFor"): React/JSX uses `htmlFor` instead
  of `for` purely as a JSX/DOM-props naming convention (avoiding `class`/
  `for` in favor of `className`/`htmlFor`), not because `for` is unusable
  there. This templating engine has no such convention to avoid: `for` is
  not a keyword in expr's grammar (see expr/lexer.go's `keywords` table —
  it lists true/false/null/undefined/typeof/void/in/instanceof/new, and
  "for" is not among them, so `for` tokenizes as an ordinary identifier
  and works anywhere an identifier does, including in a ternary like
  `for ? for : undefined` below), and Tabs.vue in this same directory
  already binds a native `<label>`'s `for` attribute via `:for="..."`
  without incident. Using `for` here both matches the native HTML
  attribute name exactly (so `<Label for="email">` reads exactly like the
  bare `<label for="email">` a reader already knows) and matches this
  package's existing intra-file precedent, so it was chosen over
  `htmlFor`.

  On making "for" optional (nesting the control instead): Radix's own
  `for` prop is optional — pairing by nesting the control inside <label>
  instead of by `for`/`id` also works natively, with no prop needed at
  all. This package has no notion of an optional prop with a real default,
  though: any identifier referenced anywhere in the template becomes a
  required prop, and an unpassed required prop renders as a visible, truthy
  "[missing: for]" placeholder (see renderer.go's validateProps) — matching
  the same trap documented in Accordion.vue's/Separator.vue's/
  AspectRatio.vue's header comments. Critically, that placeholder text is
  *truthy*, so it cannot be used as a sentinel for "not passed": a naive
  `for ?? undefined` fallback would never trigger, and the placeholder
  string would leak straight into the rendered `for="[missing: for]"`
  attribute — which is worse than simply omitting `for`, because a stray
  `for` attribute pointing at a nonexistent id actively suppresses the
  browser's native nested-control fallback (per the HTML Standard: if
  `for` is specified, the labeled control is *only* the element with that
  id, even if none exists — the algorithm does not fall through to a
  nested descendant in that case). So this component keeps `for` a
  required prop, matching Separator.vue's/AspectRatio.vue's own
  required-but-documented props, but adds one deliberate, documented
  escape hatch: pass an empty string (`for=""`) to opt into nesting-only
  mode. Empty string is falsy under this engine's truthiness rules (see
  expr/doc.go's Truthiness section), so `for ? for : undefined` evaluates
  to `undefined` for an empty string, and a `:for` binding that evaluates
  to `undefined` omits the attribute entirely instead of rendering it
  (renderer.go's attrShouldOmit) — leaving a bare `<label>` free to use the
  native nested-control association with zero `for` attribute in the
  output. Callers who genuinely need the `for`/`id` pairing pass the real
  id string as usual; callers who want to nest the control instead pass
  `for=""` explicitly, documenting their choice at the call site the same
  way Separator.vue's/AspectRatio.vue's callers document their required
  props' conventional defaults.

  Usage:
    <Label for="email">Email</Label>
    <input id="email" type="email" />

    <Label for="">
      Email
      <input type="email" />
    </Label>

  v-native on the <label> tag below: this component's own name (Label)
  auto-registers a lowercase "label" alias in the component registry (the
  standard entries[lower] = entry convention — see Dialog.vue's header
  comment for the identical trade-off with its own native <dialog> tag).
  Without v-native, the literal <label> element in this template would
  resolve right back to this component itself: HTML parsers lowercase tag
  names, so the author-facing `<Label>` and the native `<label>` element
  collide on the exact same lowercase name, and the component would try to
  render itself, forever, reported as "cycle detected" (confirmed by
  actually rendering this component through a real htmlc.Engine during
  development — this isn't a hypothetical). v-native declares this tag a
  genuine native HTML element so the component works without every
  consumer having to add "label" to their own Options.NativeElements.
-->
<template>
  <label v-native class="radix-label" :for="for ? for : undefined"><slot></slot></label>
</template>

<style scoped>
/*
 * The one real fix this component adds over a bare native <label>: without
 * it, double-clicking the label's text selects that text the way it would
 * select any other plain text on the page — a visually distracting papercut
 * that has nothing to do with the label's actual job of activating its
 * associated control. `user-select: none` (plus the vendor-prefixed forms
 * some browsers still expect) makes the label's own text unselectable,
 * which sidesteps the papercut with zero JavaScript: native <label>
 * click-to-focus behavior does not depend on the text being selectable, so
 * removing selectability here costs nothing functionally.
 */
.radix-label {
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
}
</style>
