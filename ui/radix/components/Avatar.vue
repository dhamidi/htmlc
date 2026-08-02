<!--
  Avatar — Radix-inspired profile image with a graceful fallback.

  Unlike every prior component in this directory except Dialog, whether an
  <img> actually loaded cannot be known declaratively or with pure CSS: CSS
  has no selector for "this <img>'s src 404'd" (:has() cannot reach into an
  <img>'s network-fetch outcome, and there is no error-state pseudo-class
  for images), so detecting a load failure genuinely requires JavaScript —
  specifically the <img> element's own native `error` event. That is why
  this component, like Dialog, needs a real <script customelement> rather
  than a CSS-only enhancement.

  Zero-JS baseline: both the <img> and the fallback are rendered. The <img>
  is shown by default and the fallback stays hidden (`display: none`, see
  <style scoped> below). Reasoning: with no JS present, there is no
  client-side way to know in advance whether the image will fail, so the
  only genuinely zero-JS-available signal is the browser's own native
  handling of a failed <img> load — a broken-image icon together with the
  element's `alt` text. That native treatment already *is* a form of
  graceful degradation (it is, after all, exactly what "alt text" is for),
  so defaulting to "show the <img>" costs nothing over defaulting to "show
  the fallback": a working image looks identical either way, and a broken
  image still communicates *something* useful (the alt text) without JS,
  whereas defaulting to "show the fallback" would leave a *working* image
  permanently hidden behind its fallback on any browser/request that never
  runs the enhancement script (JS disabled, script blocked, script still
  loading, etc.) — a strictly worse failure mode for the common case.

  Slots:
    fallback — caller-composed fallback content (initials, a generic-user
    icon, whatever), rendered inside a <span> that starts hidden. Chosen as
    a *named* slot (README.md, "Named slots") rather than a single
    `fallbackText` string prop, because a real fallback is very often not
    plain text (an inline SVG icon, a styled initials badge, etc.) — a named
    slot lets callers compose arbitrary markup the same way Dialog.vue's
    default slot does for its body content, without this component needing
    to special-case every possible fallback shape.

  Props (all REQUIRED — see the default-value note below):
    src:      string  — passed straight through to the <img>'s `src`.
    alt:      string  — passed straight through to the <img>'s `alt`.
    delayMs:  number  — optional-*feeling* enhancement knob (see below);
                         pass 0 for "show the fallback immediately on
                         error, no delay" (this port's documented
                         convention for Radix's own `delayMs={undefined}`
                         default), or a positive integer to wait that many
                         milliseconds after the error is detected before
                         swapping to the fallback — useful for avoiding an
                         unnecessary flash-to-fallback on a connection that
                         might still recover (e.g. a retry/cache warm-up)
                         before the delay elapses. Ignored entirely in the
                         zero-JS baseline; it only affects the
                         <script customelement> enhancement below.

  On all three props being required, including the optional-feeling
  `delayMs`: this package has no notion of an optional prop with a real
  default — any identifier referenced anywhere in the template becomes a
  required prop, and an unpassed required prop renders as a visible, truthy
  "[missing: <name>]" placeholder (see component.go's validateProps).
  `src`/`alt` follow the exact same required-prop precedent as every prior
  component here (Separator.vue's orientation/decorative, AspectRatio.vue's
  ratio, Label.vue's for): document the expectation, have every call site
  pass it explicitly. `delayMs` gets the identical treatment rather than a
  `delayMs ?? 0`-style fallback, for the same reason documented at length in
  Label.vue's header comment: the placeholder string is itself truthy, so
  `??` would never fall through to a literal default — it would leak
  "[missing: delayMs]" straight into the rendered `data-delay-ms` attribute
  instead. (Incidentally, `Number("[missing: delayMs]")` evaluates to `NaN`,
  and this component's script below reads the attribute as
  `Number(...) || 0`, so a caller who forgets `delayMs` still degrades to
  "no delay" rather than a broken timer — but that is a defensive fallback
  in the *script*, not a substitute for documenting the prop as required.)

  Usage (note: :delayMs, camelCase — this engine matches a v-bind
  attribute to a prop by its exact identifier, recovering the original
  casing the HTML parser would otherwise lowercase away, rather than
  converting kebab-case attribute names to camelCase; see engine.go /
  integration_test.go's TestIntegration_CamelCasePropViaSlot):
    <Avatar :src="user.avatarUrl" :alt="user.name" :delayMs="0">
      <template #fallback>{{ initials(user.name) }}</template>
    </Avatar>

  On self-reference-cycle risk (the trap documented in Label.vue's/
  Dialog.vue's header comments, where this component's own name
  auto-registers a lowercase alias in the component registry that can
  collide with a literal native tag of the same name inside its own
  template): "avatar" is *not* a native HTML element — there is no such
  thing as a literal `<avatar>` tag for the auto-registered lowercase
  "avatar" alias to collide with, unlike Label's `<label>` or Dialog's
  `<dialog>`. The elements actually used in this template (`<span>`,
  `<img>`, `<slot>`) are unrelated native tags with their own names, none of
  which match "avatar", so no `v-native` escape hatch is needed anywhere in
  this file. (Confirmed empirically below: this file contains no literal
  `<avatar` tag anywhere, so there is nothing for the lowercase alias to
  resolve back onto.)
-->
<template>
  <span class="radix-avatar">
    <img
      class="radix-avatar-image"
      :src="src"
      :alt="alt"
      :data-delay-ms="delayMs"
    />
    <span class="radix-avatar-fallback" data-state="hidden">
      <slot name="fallback"></slot>
    </span>
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

/*
 * Baseline sizing/shape: a fixed-size circular frame is this port's own
 * sensible default appearance (matching the "give it a reasonable default
 * look" precedent set by Dialog.vue's border/box-shadow and Tabs.vue's
 * background colors elsewhere in this directory) — not a value transcribed
 * from Radix's own (unstyled-by-default) primitive. Callers needing a
 * different size/shape can override via their own CSS targeting
 * .radix-avatar.
 */
.radix-avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  overflow: hidden;
  border-radius: var(--radix-radius-thumb);
  background-color: var(--radix-gray-4);
  vertical-align: middle;
  user-select: none;
}

.radix-avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.radix-avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1;
  color: var(--radix-gray-11);
}

/*
 * The zero-JS default: fallback stays hidden until the enhancement script
 * below (a) detects a load failure and (b) flips this attribute to
 * "visible". `display: none` here is deliberate and load-bearing — see
 * this file's header comment for why "show the <img> by default" is the
 * defensible zero-JS baseline this attribute encodes.
 */
.radix-avatar-fallback[data-state='hidden'] {
  display: none;
}
</style>

<script customelement>
// Progressive enhancement on top of the "always show <img>" zero-JS
// baseline above. Per RFC 014 §3 Non-Goals, this does not reimplement
// anything the browser already gives for free — it listens for the
// <img>'s own native `error` event (fired when the resource referenced by
// `src` fails to load — a real, spec'd DOM event, not something hand-
// rolled here) and swaps visibility between the <img> and the fallback
// <span> by toggling the fallback's `data-state` attribute the baseline
// CSS above already keys off of. `load` is also listened for so that a
// later successful load (e.g. a caller mutating `src` after this element
// is already connected) explicitly restores the "show the image" state
// instead of leaving a previously-shown fallback stuck visible.
class RadixAvatar extends HTMLElement {
  #image = null
  #fallback = null
  #timerId = null

  connectedCallback() {
    this.#image = this.querySelector('.radix-avatar-image')
    this.#fallback = this.querySelector('.radix-avatar-fallback')
    if (!this.#image || !this.#fallback) return

    this.#image.addEventListener('error', this.#onError)
    this.#image.addEventListener('load', this.#onLoad)

    // An <img> whose load already failed *before* this custom element's
    // connectedCallback ran (e.g. a cached 404 that resolves
    // synchronously) never fires a listener attached after the fact:
    // `complete` is already true, and no further `error`/`load` event is
    // coming. `complete && naturalWidth === 0` is the standard way to spot
    // that already-settled failure case, so it is checked once here too.
    if (this.#image.complete && this.#image.naturalWidth === 0) {
      this.#onError()
    }
  }

  disconnectedCallback() {
    if (this.#image) {
      this.#image.removeEventListener('error', this.#onError)
      this.#image.removeEventListener('load', this.#onLoad)
    }
    if (this.#timerId !== null) {
      window.clearTimeout(this.#timerId)
      this.#timerId = null
    }
  }

  // Explicit success path: cancel any pending delayed fallback swap, make
  // sure the <img> is visible, and make sure the fallback is hidden. Never
  // runs concurrently with #onError's own visibility changes below, since
  // a given <img> load only ever settles as success XOR failure.
  #onLoad = () => {
    if (this.#timerId !== null) {
      window.clearTimeout(this.#timerId)
      this.#timerId = null
    }
    this.#image.style.removeProperty('display')
    this.#fallback.setAttribute('data-state', 'hidden')
  }

  // Failure path: hide the <img> immediately (a broken-image icon is
  // never preferable to the caller's own fallback once JS is available to
  // do better), then reveal the fallback — either immediately, or after
  // `delayMs` milliseconds if the caller opted into a delay via the
  // data-delay-ms attribute (see this file's header comment on the
  // `delayMs` prop). The <img> is always hidden right away, in both
  // cases: `delayMs` only ever delays *when the fallback becomes
  // visible*, producing a brief window with neither element shown rather
  // than ever showing the broken-image icon and the fallback together.
  // Both branches below only ever move data-state to "visible" — never
  // back to "hidden" — so this handler can never leave the fallback in a
  // state that contradicts the <img> being hidden.
  #onError = () => {
    this.#image.style.display = 'none'
    const delayMs = Number(this.#image.dataset.delayMs) || 0
    const show = () => this.#fallback.setAttribute('data-state', 'visible')
    if (delayMs > 0) {
      this.#timerId = window.setTimeout(show, delayMs)
    } else {
      show()
    }
  }
}

customElements.define('radix-avatar', RadixAvatar)
</script>
