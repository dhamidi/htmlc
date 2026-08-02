<!--
  RadioGroup — Radix-inspired set of mutually-exclusive radio options.

  Baseline: one real native <input type="radio"> per item, visually hidden
  with the same clip-based technique Checkbox.vue's/Switch.vue's own hidden
  inputs use (position: absolute, 1x1px, clip-path: inset(50%) — never
  display:none, which would drop it from focus reach and the tab order),
  paired with an adjacent <label v-native class="radix-radiogroup-item-label">
  that is this item's entire custom-styled visual surface: a CSS-only dot
  indicator (painted on the label's own ::before pseudo-element, no inner
  markup and no inline SVG, same "a couple of CSS shapes do the same job"
  call Checkbox.vue's own box already made) immediately followed by the
  item's visible text. The two are flat, adjacent siblings within each
  item's own wrapper (input immediately followed by label) so the CSS below
  can key off the input's live state with a plain adjacent-sibling
  selector — the exact technique family Checkbox.vue/Switch.vue/Tabs.vue all
  already use:

    .radix-radiogroup-input:checked + .radix-radiogroup-item-label::before { ... }

  All items in one instance share the real, literal HTML `name` attribute
  passed via this component's own `name` prop, so the browser's own native
  "shared name groups radio inputs into one mutually-exclusive set" behavior
  (a real, spec'd HTML feature) makes selecting one item automatically
  uncheck every other item in the same group — no exclusivity logic
  hand-rolled here.

  On `name` being a real, per-instance, REQUIRED prop — unlike
  Accordion.vue's/Tabs.vue's own hardcoded shared name="radix-accordion"/
  name="radix-tabs" literals: those two components made a deliberate,
  documented trade-off (see their own header comments) accepting that
  multiple instances on one page would share native grouping state, because
  neither component's design called for per-instance identity. A page
  legitimately wants multiple *independent* radio groups at once (e.g. two
  unrelated multiple-choice questions), so silently sharing one native group
  across every <RadioGroup> instance on a page would be a real correctness
  bug here, not just an accepted simplification — every item in one instance
  would stay mutually exclusive with items in a *different* instance too,
  and selecting an option in one group could uncheck a same-valued option in
  another. Making `name` a required prop (matching Checkbox.vue's/
  Switch.vue's own `id`/`name` prop pattern) puts the choice of grouping
  boundary in the caller's hands, the same way plain HTML forms already
  require of any native radio group.

  On the container being a <div role="radiogroup">, not a <fieldset>: this
  matches Radix's own documented choice for RadioGroupPrimitive.Root — a
  <fieldset> carries its own, harder-to-override default browser chrome
  (a border, padding, and a <legend>-shaped content model) that a
  role="radiogroup" <div> simply does not have to fight. role="radiogroup"
  is a real, static ARIA role — WAI-ARIA maps it to the same "exactly one
  radio among children may be checked" semantics as a native <fieldset>
  around <input type="radio">s, so nothing is lost by not using <fieldset>
  here.

  On keyboard behavior needing no JS at all (the load-bearing fact this
  component leans on, same conclusion Switch.vue's header comment reaches
  for its own role="switch" question): a native <input type="radio">
  group — several radios sharing one `name` — already gives arrow-key
  roving selection among the group's own options and a single Tab stop
  into/out of the whole group, for free, as real browser behavior. This is
  the exact same free correctness Tabs.vue's own baseline already leaned on
  for its analogous CSS-only radio-tabs technique (see that file's header
  comment). Nothing about that native roving/tab-stop behavior depends on
  `role="radiogroup"` being present on some ancestor, or on script running
  at all — it is intrinsic to how the browser handles a same-`name` group of
  radio inputs.

  On `aria-orientation` / layout direction: this first pass defaults to a
  single, fixed vertical layout (a plain flex column — see <style scoped>
  below) and does not expose an `orientation` prop or render
  `aria-orientation`. This was a deliberate v1 scope call, not an oversight:
  Radix's own RadioGroup supports both, but a real per-instance
  `orientation` prop that switches the CSS layout direction *and* keeps a
  matching `aria-orientation` attribute in sync is a self-contained, purely
  additive follow-up (it changes nothing about the baseline decomposition
  documented above) rather than something this port's first pass needed to
  block on.

  On the required `v-native` on the <label> tag below — applied proactively
  here, the same "don't wait to rediscover the same bug a third time"
  reasoning Switch.vue's own header comment already documents: this
  package's own Label.vue registers a lowercase "label" alias engine-wide
  once ui/radix is mounted as a whole package, and a literal <label> tag
  anywhere else in the same mount resolves against that alias before ever
  being treated as plain HTML (renderer.go resolves every implicit,
  non-native-declared tag against the component registry first —
  resolveComponent runs unconditionally unless isNative is true).
  Checkbox.vue's own commit hit this collision empirically the hard way, by
  rendering Tabs.vue's own <label> through a mounted ui/radix package and
  watching it silently render as Label.vue's markup instead, dropping
  Tabs.vue's own attributes. v-native below declares this <label> a genuine
  native HTML element up front, so this component's per-item visual renders
  correctly regardless of which other ui/radix components happen to be
  mounted alongside it — including Label.vue itself.

  On the `:checked="item.value === value"` binding and the unset-`value`
  case: this package has no notion of an optional prop with a real default
  (see Progress.vue's/Avatar.vue's header comments for the full
  "[missing: <name>]" placeholder trap) — `value` is therefore a required
  prop like every other prop here, and an unpassed `value` renders as the
  literal, truthy placeholder string "[missing: value]" rather than as
  `undefined`/empty. That placeholder string is exactly why this component
  never treats an absent selection as a special case needing extra guard
  logic: `item.value === value` is a plain string comparison, and
  "[missing: value]" cannot equal any real `item.value` a caller's `items`
  array would ever contain (it is not a legal value for this component's
  own `value` prop to intentionally hold), so every item's `:checked`
  correctly evaluates to `false` and the zero-JS baseline renders with no
  item selected — never an accidental match against the placeholder text.
  Callers that want a real "nothing selected yet" group should simply pass
  a `value` that does not match any item's `value` (e.g. an empty string),
  which this comparison already handles correctly for the same reason.

  Props (all REQUIRED — this package has no notion of an optional prop with
  a real default; see Progress.vue's/Avatar.vue's header comments for the
  full "[missing: <name>]" placeholder trap this convention avoids):
    items:    Array<{ id, value, label }> — matching Accordion.vue's/
              Tabs.vue's own `items` shape convention for consistency.
                - id:    stable, unique-within-this-instance identifier.
                         Combined with this instance's own `name` prop to
                         build this item's element id
                         (e.g. "radix-radiogroup-<name>-<id>"), which keeps
                         ids unique both within one instance's item list
                         *and* across multiple <RadioGroup> instances on the
                         same page (since `name` is itself required to be
                         distinct per instance — see above) — never sent to
                         the browser as the shared radio-group name itself.
                - value: the native `value` attribute for this item's
                         hidden radio input, and the value compared against
                         this component's own `value` prop to decide this
                         item's initial `checked` state.
                - label: this item's visible option text, rendered as plain
                         text inside its <label>.
    name:     string  — the native `name` attribute shared by every item's
                         hidden radio input in this instance; see the
                         per-instance-required rationale above.
    value:    string  — the currently-selected item's `value`; see the
                         unset-value rationale above for what happens when
                         it does not match any item.
    disabled: boolean — applied to every item's native `disabled` attribute
                         uniformly; this first pass has no per-item disabled
                         override in the `items` shape above.

  Usage:
    <RadioGroup
      name="notification-frequency"
      value="daily"
      :disabled="false"
      :items="[
        { id: 'daily', value: 'daily', label: 'Daily' },
        { id: 'weekly', value: 'weekly', label: 'Weekly' },
        { id: 'never', value: 'never', label: 'Never' },
      ]"
    />

  On there being no customelement enhancement script block in this file:
  per RFC 014 §3 Non-Goals, a script is only added when something is
  genuinely unreachable without it — the same discipline Switch.vue's own
  commit applied when it concluded role="switch" needed no script at all.
  Every piece of this
  component's real interactive behavior already comes from the browser for
  free, with nothing left for a script to add:
    - role="radiogroup" above is a plain static ARIA attribute, no runtime
      involvement required.
    - Each item's own implicit "radio" role, its checked/unchecked
      accessible state, and its Space-key/click activation all come from
      the native <input type="radio"> itself, the same free mapping
      Checkbox.vue's own header comment documents for its checkbox.
    - Arrow-key roving selection among the group's items, and a single
      Tab-key stop into/out of the whole group, are both intrinsic native
      behavior of a same-`name` group of radio inputs — see the keyboard
      note above, and Tabs.vue's own baseline, which already established
      this exact conclusion for an analogous CSS-only radio-input
      technique.
    - The dot indicator's checked/unchecked look is driven entirely by the
      real, native :checked pseudo-class via the adjacent-sibling selector
      in <style scoped> below — no class toggling or state sync required.
  Unlike Checkbox.vue's `indeterminate` state (genuinely impossible to
  reach from markup, because HTML has no `indeterminate` *attribute* at
  all — see that file's header comment), RadioGroup has no state in this
  prop surface that is unreachable from declarative markup, so there is
  nothing left for a script to enhance.
-->
<template>
  <div class="radix-radiogroup" role="radiogroup" :data-disabled="disabled ? '' : undefined">
    <span v-for="item in items" class="radix-radiogroup-item">
      <input
        type="radio"
        class="radix-radiogroup-input radix-visually-hidden-input"
        :id="'radix-radiogroup-' + name + '-' + item.id"
        :name="name"
        :value="item.value"
        :checked="item.value === value"
        :disabled="disabled"
      />
      <label
        v-native
        class="radix-radiogroup-item-label"
        :for="'radix-radiogroup-' + name + '-' + item.id"
      >{{ item.label }}</label>
    </span>
  </div>
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
 * Fixed vertical layout for this first pass — see the header comment's
 * "aria-orientation / layout direction" note for why a per-instance
 * `orientation` prop is a natural, purely additive follow-up rather than
 * something this pass needed to block on.
 */
.radix-radiogroup {
  display: inline-flex;
  flex-direction: column;
  gap: 0.75rem;
}

.radix-radiogroup-item {
  position: relative; /* containing block for the absolutely-positioned input */
  display: inline-flex;
  align-items: center;
}

.radix-radiogroup-item-label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  user-select: none;
}

/*
 * The dot indicator is a single ::before pseudo-element on the label — no
 * inner markup needed at all, and no inline SVG (same "a couple of CSS
 * shapes do the same job" call Checkbox.vue's own box already made). Drawn
 * as a plain circle by default; the :checked sibling-selector rule below
 * paints the "selected" inner dot via an inset box-shadow, so no second
 * pseudo-element is needed to layer a dot inside the circle.
 */
.radix-radiogroup-item-label::before {
  content: '';
  display: inline-block;
  box-sizing: border-box;
  flex-shrink: 0;
  width: 1.25rem;
  height: 1.25rem;
  border-radius: var(--radix-radius-thumb);
  border: 1px solid var(--radix-gray-8);
  background-color: #fff;
}

/* Checked: driven by the input's real, native :checked pseudo-class. */
.radix-radiogroup-input:checked + .radix-radiogroup-item-label::before {
  border-color: var(--radix-blue-9);
  box-shadow: inset 0 0 0 4px var(--radix-blue-9);
}

/* Never remove the focus outline — keep it visible for keyboard users.
   The input itself is visually hidden, so its focus ring is drawn on the
   paired label's dot instead, matching Checkbox.vue's/Switch.vue's
   identical treatment of their own hidden input + label pairs. */
.radix-radiogroup-input:focus-visible + .radix-radiogroup-item-label::before {
  outline: 2px solid var(--radix-blue-9);
  outline-offset: 2px;
}

.radix-radiogroup-input:disabled + .radix-radiogroup-item-label {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
