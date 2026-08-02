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
