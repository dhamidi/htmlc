<!--
  Checkbox — Radix-inspired tri-state (checked / unchecked / indeterminate
  "mixed") form control.

  Baseline: a real native <input type="checkbox">, visually hidden with the
  same clip-based technique VisuallyHidden.vue/Tabs.vue's own hidden radios
  use (position: absolute, 1x1px, clip-path: inset(50%) — never
  display:none, which would drop it from focus reach and the tab order),
  paired with a sibling <label class="radix-checkbox-box"> that is this
  component's entire custom-styled visual surface. The two are flat,
  adjacent siblings (input immediately followed by label) so the CSS below
  can key off the input's live state with a plain adjacent-sibling
  selector — the exact technique Tabs.vue established for its own
  input+label pairs:

    .radix-checkbox-input:checked + .radix-checkbox-box::after { ... }

  The <label>'s native `for`/`id` pairing with the input is what makes the
  box clickable with zero JS (a real, spec'd browser feature — see
  Label.vue's header comment) and is also why the input itself is left
  fully alone: no tabindex="-1", no aria-hidden, nothing that would pull it
  out of the tab order or the accessibility tree. Unlike Tabs.vue's own
  <script customelement> (which deliberately moves ARIA tab semantics off
  the hidden radio and onto the label, because a tablist's real interactive
  unit is the tab, not the underlying radio), a checkbox's real interactive
  unit already *is* the native <input type="checkbox"> — role="checkbox",
  checked/unchecked state, and Space-key activation all come from the
  browser for free, with nothing to relocate onto the decorative box. The
  box carries aria-hidden="true": it renders no text and no independent
  semantics of its own, so it would otherwise show up as a redundant,
  unlabeled node next to the real control in the accessibility tree.

  No visible text label ships with this component — same decomposition as
  Radix's own real Checkbox (a bare control, with text-label pairing left
  to a separate primitive). Pair with this package's own Label.vue via the
  `for`/`id` convention it already documents:

    <Label for="terms">Accept the terms</Label>
    <Checkbox id="terms" name="terms" value="yes" :checked="false"
              :indeterminate="false" :disabled="false" :required="true" />

  On the required `v-native` on the <label> tag below — this is not the
  usual self-reference-cycle trap documented in Label.vue's/Dialog.vue's
  own header comments (this component's own name, "Checkbox", registers no
  "label" alias — there is no `<checkbox>` native tag for a self-cycle to
  even be possible). It is a *cross-component* collision instead: this
  package's own Label.vue registers a lowercase "label" alias engine-wide
  once the whole ui/radix package is mounted, and a literal <label> tag
  anywhere else in the same mount resolves against that alias before it
  ever falls through to being treated as plain HTML (renderer.go resolves
  every implicit, non-native-declared tag against the component registry
  first — resolveComponent runs unconditionally unless isNative is true).
  This was confirmed empirically while building this component, by
  mounting the full ui/radix package (the same shape examples/radix-demo
  uses) and rendering Tabs.vue's own <label class="radix-tabs-label">
  through it: it silently rendered as `<label class="radix-label" ...>`
  instead — Label.vue's component, not a plain native tag — dropping
  Tabs.vue's own class, id, and aria-controls attributes in the process.
  (That is a pre-existing issue in Tabs.vue, out of scope to fix here, but
  a real trap worth documenting loudly so this file doesn't repeat it.)
  v-native declares the <label> below a genuine native HTML element,
  matching the exact fix Label.vue/Dialog.vue already apply to their own
  same-named tags, so this component's box renders correctly regardless of
  which other ui/radix components happen to be mounted alongside it.

  On HTML having no `indeterminate` *attribute* (the core design
  constraint this component is built around): every prior state this
  library has ported — Tabs' selected tab, Accordion's open panel,
  Dialog's open/closed, Progress's determinate/indeterminate — has some
  real HTML attribute or native element state backing its zero-JS
  baseline. A checkbox's "mixed" state does not. The DOM only exposes
  indeterminate-ness through the `HTMLInputElement.indeterminate` IDL
  property (https://html.spec.whatwg.org/multipage/input.html#dom-input-indeterminate)
  — settable only from script, never reflected to or from any markup
  attribute, and reset to false by the browser itself the moment a user
  interacts with the checkbox. There is no declarative spelling of it: no
  `indeterminate` attribute exists to write in server-rendered HTML, so a
  page that never runs this component's <script customelement> — JS
  disabled, the script blocked, or simply not loaded yet — can only ever
  show a plain checked or unchecked box. This is not a corner this port
  cut; it is a genuine, unavoidable platform limitation, and it applies to
  *this component's own custom box* just as much as it would to the bare
  native input: nothing stops this file from painting a static "mixed
  dash" glyph on the box the moment `indeterminate` is true, with zero JS
  involved (the box is just a styled <label>, not the real input) — but
  doing so would be actively misleading rather than a neutral degradation.
  A sighted zero-JS visitor would see a dash that means "mixed", while a
  screen-reader user landing on the same real <input> would hear "not
  checked" (its only two native states), because nothing has actually told
  the browser or any assistive technology that this checkbox is anything
  other than unchecked. That visual/semantic mismatch would be a strictly
  worse outcome than simply not showing a mixed look at all. This
  component instead keeps the visual and the semantic layers honest with
  each other in every case, including the zero-JS one: the CSS rule that
  paints the "mixed" dash (see <style scoped> below) is keyed off the real
  `:indeterminate` CSS pseudo-class
  (https://developer.mozilla.org/en-US/docs/Web/CSS/:indeterminate), which
  can only ever match once script has actually set
  `input.indeterminate = true` — so the exact same platform fact that makes
  indeterminate unreachable from markup also guarantees the baseline can
  never show a mixed look it hasn't earned. Indeterminate therefore
  degrades to plain unchecked without JS: a defensible, common choice (the
  same one most headless-UI checkbox implementations make), and the
  *only* choice that keeps what a sighted zero-JS user sees consistent
  with what a screen-reader zero-JS user hears.

  Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):
    id:            string  — required for the <label>'s `for` pairing
                              above (see the v-native note) and so external
                              callers can pair their own <Label for="...">
                              with this control, the same `for`/`id`
                              contract Label.vue already documents.
    name:          string  — the native `name` attribute; needed for this
                              control to actually submit as part of a form.
    value:         string  — the native `value` attribute. Radix's own
                              documented default is `value="on"` (the same
                              value a bare native checkbox submits when no
                              `value` attribute is given at all) — this
                              port's call sites should pass `"on"`
                              explicitly by convention when no more
                              specific value is needed, matching
                              Progress.vue's identical `max={100}`-by-
                              convention note.
    checked:       boolean — the control's checked state. Ignored when
                              `indeterminate` is true — by convention
                              *and* in code: the native `checked` attribute
                              below is bound to `checked && !indeterminate`,
                              never letting a caller-supplied `checked=true`
                              leak into the baseline while `indeterminate`
                              is also true (see the review note in this
                              commit's process — this is a deliberate,
                              defensive belt-and-suspenders guard, not
                              accidental).
    indeterminate: boolean — true renders the "mixed" state. As documented
                              above, this can only ever take visible/ARIA
                              effect once the <script customelement>
                              enhancement below has run; the zero-JS
                              baseline always shows plain unchecked for
                              this case (checked is forced false, per the
                              rule above).
    disabled:      boolean — the native `disabled` attribute.
    required:      boolean — the native `required` attribute (a real HTML
                              attribute name — coincidentally the same
                              English word this whole file's props list
                              already uses to describe itself; the two
                              "required"s are unrelated homonyms, not a
                              conflict).

  Usage:
    <Checkbox id="terms" name="terms" value="on" :checked="false"
              :indeterminate="false" :disabled="false" :required="false" />

    <!-- a real "select all" mixed-state example -->
    <Checkbox id="select-all" name="select-all" value="on"
              :checked="false" :indeterminate="true" :disabled="false"
              :required="false" />
-->
<template>
  <span
    class="radix-checkbox"
    :data-state="indeterminate ? 'indeterminate' : (checked ? 'checked' : 'unchecked')"
    :data-disabled="disabled ? '' : undefined"
  >
    <input
      type="checkbox"
      class="radix-checkbox-input radix-visually-hidden-input"
      :id="id"
      :name="name"
      :value="value"
      :checked="checked && !indeterminate"
      :disabled="disabled"
      :required="required"
      :data-indeterminate="indeterminate"
    />
    <label
      v-native
      class="radix-checkbox-box"
      :for="id"
      aria-hidden="true"
      :data-state="indeterminate ? 'indeterminate' : (checked ? 'checked' : 'unchecked')"
    ></label>
  </span>
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

.radix-checkbox {
  position: relative; /* containing block for the absolutely-positioned input */
  display: inline-flex;
  align-items: center;
}

.radix-checkbox-box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 1.25rem;
  height: 1.25rem;
  background-color: #fff;
  border: 1px solid var(--radix-gray-8);
  border-radius: var(--radix-radius-2);
  cursor: pointer;
}

/*
 * The check glyph and the mixed-state dash are both drawn with a single
 * ::after pseudo-element on the box — no inner markup needed at all, and
 * no inline SVG (this port avoids introducing untested foreign-content
 * (SVG) parsing into this codebase when a couple of CSS shapes do the same
 * job). Hidden by default; the two rules below turn it into either a
 * checkmark or a dash depending on which real input state matches.
 */
.radix-checkbox-box::after {
  content: '';
  display: none;
  box-sizing: border-box;
}

/* Checked: a classic CSS-only checkmark (a rotated bottom+right border
   corner), driven by the input's real, native :checked pseudo-class. */
.radix-checkbox-input:checked + .radix-checkbox-box {
  background-color: var(--radix-blue-9);
  border-color: var(--radix-blue-9);
}

.radix-checkbox-input:checked + .radix-checkbox-box::after {
  display: block;
  width: 0.3rem;
  height: 0.55rem;
  border: solid #fff;
  border-width: 0 2px 2px 0;
  transform: translateY(-1px) rotate(45deg);
}

/*
 * Indeterminate ("mixed"): a plain horizontal dash. Deliberately keyed off
 * the real `:indeterminate` CSS pseudo-class — not the `data-state`
 * attribute rendered above, and not a class the template could set
 * unconditionally — because :indeterminate can only ever match once the
 * <script customelement> enhancement below has actually set
 * `input.indeterminate = true`. That is precisely what keeps the zero-JS
 * baseline honest: see this file's header comment for why painting this
 * dash off of anything else (like the always-present `data-state`
 * attribute) would let a sighted zero-JS visitor see "mixed" while a
 * screen-reader user, on the same unenhanced input, still hears "not
 * checked".
 */
.radix-checkbox-input:indeterminate + .radix-checkbox-box {
  background-color: var(--radix-blue-9);
  border-color: var(--radix-blue-9);
}

.radix-checkbox-input:indeterminate + .radix-checkbox-box::after {
  display: block;
  width: 0.65rem;
  height: 2px;
  background-color: #fff;
}

/* Never remove the focus outline — keep it visible for keyboard users.
   The input itself is visually hidden, so its focus ring is drawn on the
   paired box instead, matching Tabs.vue's identical treatment of its own
   hidden radio + label pair. */
.radix-checkbox-input:focus-visible + .radix-checkbox-box {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-checkbox-input:disabled + .radix-checkbox-box {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>

<script customelement>
// Progressive enhancement on top of the checked/unchecked-only zero-JS
// baseline above. Per RFC 014 §3 Non-Goals, this does the one thing that
// is genuinely impossible any other way (per this file's header comment):
// set the native `indeterminate` IDL property, which has no markup
// equivalent at all. Nothing here reimplements checked/unchecked toggling,
// keyboard activation, or form participation — the browser already
// provides all of that for free on the real <input type="checkbox">, and
// this script never touches it.
//
// On the explicit aria-checked="mixed" override: native <input
// type="checkbox"> does map its `indeterminate` IDL property to the
// accessibility tree's mixed checked state on its own in evergreen
// browsers per the HTML-AAM spec, so this override is not filling a gap
// that exists everywhere. It is added anyway because that mapping has
// real, documented history of being inconsistent across specific
// browser/assistive-technology pairings (screen-reader compatibility
// testing of native-indeterminate-only checkboxes has turned up gaps that
// pairing the same input with an explicit aria-checked="mixed" closes) —
// unlike, say, Progress.vue's aria-valuenow omission, which this port
// confirmed needs no such belt-and-suspenders override. The two are kept
// in sync explicitly rather than left to chance: aria-checked is set to
// "mixed" exactly when indeterminate is turned on, and explicitly removed
// (falling back to the browser's own implicit computation from `checked`)
// the moment indeterminate is no longer true — including when the browser
// itself auto-clears `indeterminate` after a user's own click, which is
// why the change listener below re-checks and clears the override rather
// than setting it once and forgetting it.
class RadixCheckbox extends HTMLElement {
  #input = null

  connectedCallback() {
    this.#input = this.querySelector('.radix-checkbox-input')
    if (!this.#input) return

    this.#syncIndeterminate()
    this.#input.addEventListener('change', this.#onChange)
  }

  disconnectedCallback() {
    if (this.#input) {
      this.#input.removeEventListener('change', this.#onChange)
    }
  }

  // The only piece of behavior that is genuinely impossible without JS:
  // HTML has no `indeterminate` attribute, so this JS-only IDL property
  // assignment is the sole way to ever reach the mixed state at all.
  #syncIndeterminate() {
    const wantsIndeterminate = this.#input.dataset.indeterminate === 'true'
    if (wantsIndeterminate) {
      this.#input.indeterminate = true
      this.#input.setAttribute('aria-checked', 'mixed')
    } else {
      this.#input.indeterminate = false
      this.#input.removeAttribute('aria-checked')
    }
  }

  // The browser itself clears `indeterminate` to false the moment a user
  // interacts with the checkbox (a native behavior this script does not
  // reimplement). When that happens, the aria-checked="mixed" override
  // set above would otherwise stay stuck, contradicting the now-plain
  // checked/unchecked state the box and the input itself both correctly
  // moved to. Removing the override here lets the browser's own implicit
  // aria-checked computation (from the real `checked` property) take back
  // over.
  #onChange = () => {
    if (!this.#input.indeterminate) {
      this.#input.removeAttribute('aria-checked')
    }
  }
}

customElements.define('radix-checkbox', RadixCheckbox)
</script>
