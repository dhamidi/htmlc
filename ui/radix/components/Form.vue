<!--
  Form — Radix-inspired accessible form field: label + control + validation
  message, wired together by id.

  ## Re-scoped, deliberately, from what "Form" means in real Radix

  Real Radix `Form` is a `<Form.Root>` that wraps an *entire, arbitrary,
  multi-field* form. A consumer freely composes `Form.Field` / `Form.Label`
  / `Form.Control` / `Form.Message` / `Form.ValidityState` as React children
  underneath it, and the root coordinates all of them through React context:
  per-field native `ValidityState`, custom sync/async matchers, and the set
  of message ids each field's control should be `aria-describedby`'d to —
  all discovered dynamically as children mount/unmount, and all recomputed
  live as the user types (validating on the control's native `change` event,
  clearing on user edit, focusing the first invalid control on submit).

  That whole shape — a parent component owning and coordinating arbitrary
  composed child field markup supplied via children — is exactly the same
  architectural mismatch Toolbar.vue's design note already worked around
  for heterogeneous toolbar items, and the same one Toast.vue's design note
  worked around for its provider/viewport pair: htmlc's `<slot>` inserts
  inert, already-rendered markup with no mechanism for a parent component to
  reach back into it and prop-inject context, walk a dynamic child list, or
  react to children mounting/unmounting the way React's context/children can.
  A `<Form>` that owns an arbitrary number of arbitrarily-typed fields,
  discovers them via composition, and cross-validates against `FormData`
  pulled from a live DOM form has no honest translation into this
  templating model.

  **The honest scope for this port**: `Form.vue` is not a form-owning root
  at all. It is **one self-contained, validated field unit** — a label, a
  slotted control, and a conditional accessible error message — used once
  per field inside the consumer's own plain, native `<form>` element (note
  the required `v-native` on that wrapping `<form>` — see "Critical: the
  consumer's own `<form>` tag needs `v-native` too" below for why — and note
  `<radix-form>`, not `<Form>`, for the field reference itself — see
  "Critical #2: nested `<form>`-tag-name references need the `radix-form`
  alias, not `<Form>`" further below for why):

    <form v-native method="post" action="/signup">
      <radix-form id="email" label="Email" error="{{ .Errors.Email }}">
        <input id="email" type="email" name="email"
               :aria-invalid="Errors.Email ? 'true' : undefined"
               :aria-describedby="Errors.Email ? 'email-message' : undefined" />
      </radix-form>
      <button type="submit">Sign up</button>
    </form>

  Real Radix's `Form.Field`/`Form.Label`/`Form.Control`/`Form.Message` are
  therefore collapsed into this one component (matching how ToggleGroup.vue
  collapsed Radix's `ToggleGroup.Root`/`ToggleGroup.Item` into one `items`-
  driven file); `Form.Root`'s cross-field client-side validation
  orchestration and `Form.ValidityState`'s render-prop access to a control's
  live `ValidityState` are out of scope entirely — see "error is a
  server-computed prop" below for why that is a reasonable trade for this
  model, not just a gap.

  ## Props (all REQUIRED — this package has no notion of an optional prop
  with a real default; see Progress.vue's/Avatar.vue's header comments for
  the full "[missing: <name>]" placeholder trap this convention avoids):

    id:    string — ties the label's `for`, the slotted control's own `id`
           (the consumer's responsibility to match — see below), and the
           generated message id (`id + '-message'`, matching Select.vue's
           own `id + '-native'` convention for a derived id) together.
    label: string — the field's visible label text.
    error: string — the field's current validation error message. Empty
           when the field is valid. This package has no optional-prop
           default (see above), so there is no same-file fallback for "no
           error" — the convention, matching Toast.vue's own `title`/
           `description` props exactly (also required strings, also
           `v-if`-gated, also documented as "pass '' to omit"), is that
           callers pass `error=""` explicitly for the valid/no-error state.
           An empty string is falsy under this engine's truthiness rules
           (see expr/doc.go's Truthiness section), so `v-if="error"` below
           genuinely omits the <p> element rather than rendering one with
           empty text content — this is the real, engine-native conditional
           mechanism, not a CSS display:none trick (an empty error message
           has no reason to exist in the DOM at all, let alone be a target
           any aria-describedby should be able to reach).

  ## The structural limitation this component cannot paper over

  This component renders a `<label>` and a message `<p>`, but it does
  **not** render the field's actual control — that is slotted in, supplied
  by the consumer, exactly like Radix's own `Form.Control` being an
  "asChild"-style wrapper around whatever the consumer passes it (an
  `<input>`, a `<select>`, a custom control). Unlike Radix's real
  `Form.Control`, though, this component has no way to inject props onto
  that slotted element: a `<slot>` here inserts already-rendered, inert
  markup (see the re-scoping note above), so `aria-describedby`/
  `aria-invalid` — the two attributes that actually connect the control to
  this component's label/message for assistive technology — **cannot** be
  set by this file on an element it does not own and cannot see inside of.

  This is a real, honest limitation, not a detail glossed over: **the
  consumer's own slotted control is responsible for setting both
  `aria-describedby` and `aria-invalid` on itself**, using this component's
  documented message-id convention (`id + '-message'`, exactly matching
  what this file's own `<p :id="id + '-message'">` renders) to reference
  the right element. The usage example above shows the full contract from
  the consumer's side: `:aria-describedby="Errors.Email ? 'email-message' :
  undefined"` (the same truthy-ternary-to-`undefined` attribute-omission
  idiom Label.vue's own `:for="for ? for : undefined"` already established
  in this package) and `:aria-invalid="Errors.Email ? 'true' : undefined"`.

  On why the example conditionally omits `aria-describedby` rather than
  binding it unconditionally to `id + '-message'`: a dangling ID reference
  (one whose target element does not exist in the document — which is
  exactly what `aria-describedby="email-message"` would be whenever `error`
  is empty and this file's own `v-if` has omitted the `<p>`) is not a
  spec violation and not a rendering hazard — per the WAI-ARIA ID Reference
  List processing model, an id token that resolves to no element is simply
  not part of the description; browsers and assistive technology drop the
  unresolved token rather than erroring on it. So an unconditional binding
  would have been *safe*, not just close-enough — this was verified against
  MDN's `aria-describedby` reference documentation and the WAI-ARIA 1.2 ID
  Reference List processing model, not assumed. The example still
  conditionally omits it anyway, matching Radix's own real `Form.Control`
  behavior (its `concatAriaDescribedby` helper only ever includes a
  message's id while that message is actually mounted, never leaves a
  reference to an absent one lying around) and because it is simply the
  more precise, defensively-written attribute for a consumer to copy —
  "correct and unambiguous" beats "correct but relies on a reader knowing a
  spec nuance" for a header-comment example other call sites will copy from.

  ## `error` as a server-computed prop — a better fit than it first looks

  Radix's real `Form` is client-validation-first: `Form.Control` listens
  for the control's native `change` event, runs the browser's own
  `ValidityState` (plus any custom sync/async matchers) against it, and
  re-renders `Form.Message` reactively, all without a page navigation.
  This port's `error` prop is the opposite shape: a plain string, computed
  once, server-side — the natural result of validating a submitted
  `<form>` on the backend and re-rendering the page with whichever fields
  failed carrying their message. For an SSR-first engine like htmlc, that
  is arguably a *better* default story than client-validation-first, not
  merely a fallback for one: it needs no client JS to work at all (a
  `<form method="post">` with no script, submitted, revalidated, and
  re-rendered by the server is a complete, fully-accessible round trip),
  it is exactly the shape a failed-POST-then-redisplay flow already
  produces, and it sidesteps the native-`ValidityState`-plus-custom-
  matchers machinery entirely by letting the server run whatever validation
  logic it already needs to run for the persistence layer anyway — there is
  no second, client-side copy of the same validation rules to keep in sync.

  ## `v-native` on the <label> below

  This component's own name ("Form") auto-registers a lowercase "form"
  alias in the component registry, but this file's template contains no
  literal `<form>` element for that alias to collide with (see Progress.vue's
  header comment for the identical "no collision, no v-native needed for
  the component's own name" reasoning) — the actual `<form>` element is the
  *consumer's*, wrapping this component from outside, never rendered here.
  The `<label>` below needs `v-native` for a different, cross-component
  reason instead: this package's own Label.vue registers a lowercase
  "label" alias engine-wide once ui/radix is mounted, and a literal
  `<label>` tag anywhere else in the same mount resolves against that alias
  before falling through to plain HTML (renderer.go resolves every
  implicit, non-native-declared tag against the component registry first).
  This is the exact trap Checkbox.vue's header comment documents in detail,
  confirmed there by actually mounting the full package and watching a
  plain `<label>` silently render as Label.vue's own markup instead,
  dropping this file's `for`/class attributes in the process — `v-native`
  is the same fix Checkbox.vue/Dialog.vue/Label.vue itself already apply to
  their own same-named tags.

  ## Critical: the consumer's own <form> tag needs `v-native` too — and so
  do Dialog.vue's/AlertDialog.vue's own internal ones (fixed in this commit)

  This is the same class of collision as the `<label>` one just above, but
  with a much larger blast radius, and it was not merely reasoned about —
  it was caught by actually rendering this component through a real
  htmlc.Engine with the full ui/radix package mounted, before being fixed.

  This component's own name ("Form") auto-registers a lowercase "form"
  alias in the component registry, exactly like every other component in
  this package (`entries[lower] = entry`, the same mechanism Label.vue's/
  Dialog.vue's header comments already document for "label"/"dialog"). Once
  that alias exists, **any literal, un-annotated `<form>` tag anywhere in
  the same mounted engine** — not just on a page that happens to use
  `<Form>` — resolves to this component instead of staying a plain native
  form: `renderer.go` resolves every implicit tag against the component
  registry before falling through to native HTML, regardless of whether
  the page author has ever heard of this file. Concretely, before this was
  caught: a page's own `<form method="post" action="/signup">` wrapping a
  `<Form>` field (exactly the pattern this component's own usage brief
  requires every consumer to write) silently became an *unwanted, props-less
  instance of this component itself* — rendering `for="[missing: id]"`/
  `[missing: label]` placeholders, a truthy `[missing: error]` message
  (the placeholder string is itself truthy, so `v-if="error"` did not save
  it), and swallowing the form's real children (the `<Form>` fields, the
  submit `<button>`) into its own `<slot>` instead of leaving them as
  siblings under a real `<form>` — the exact "children silently rewritten"
  failure mode Checkbox.vue's header comment warns about, just triggered by
  the single most common wrapping element this component's own usage
  pattern guarantees will be present.

  Worse, this is not limited to pages that use `<Form>` at all: Dialog.vue's
  and AlertDialog.vue's own zero-JS close/cancel mechanism is a literal
  `<form method="dialog">` *inside their own templates*, and adding this
  file to the same package broke both of them the same way — confirmed by
  rendering `<Dialog :open="true">` through the full mounted package and
  finding its Close button silently reparented into a props-less
  `<div class="radix-form-field">` instead of staying inside a real
  `<form method="dialog">`, which entirely disables the native
  close-the-nearest-ancestor-dialog behavior that mechanism depends on. That
  was fixed as part of this same commit: both files' own `<form
  method="dialog">` elements now carry `v-native` too (see their header
  comments for the specifics) — a small, direct, load-bearing consequence
  of this file's own name, not a hypothetical.

  The unavoidable, documented consequence for every consumer: **any project
  that mounts this package and writes its own literal `<form>` tag
  anywhere — whether or not that page uses `<Form>` at all — must mark that
  `<form>` `v-native`** (`Options.NativeElements` does not help here; per
  its own doc comment it only covers *hyphenated* tag names, and `form` has
  no hyphen). This is a real, load-bearing cost of this port's naming
  choice, kept because the alternative (naming the file something other
  than "Form", diverging from Radix's own name and every sibling
  component's `Component.vue`-name-matches-file convention) was judged
  worse — the same trade Label.vue/Dialog.vue already made for `<label>`/
  `<dialog>`, just with a wider blast radius this file owes it to the next
  reader to spell out plainly rather than downplay.

  ## Critical #2: nested `<form>`-tag-name references need the `radix-form`
  alias, not `<Form>` — a second, distinct trap that survives the fix above

  Marking the consumer's own wrapping `<form v-native>` (per "Critical"
  above) is necessary but **not sufficient**. Even correctly fixed that way,
  a literal `<Form>` field reference nested *inside* that same wrapping
  `<form>`, in the same template's raw HTML content, still silently
  disappears — found the same way as the trap above, by actually rendering
  this exact pattern through a real htmlc.Engine (in examples/radix-demo's
  own component gallery, the first real page to nest a `<Form>` reference
  inside a literal wrapping `<form>`), not reasoned about in the abstract.

  The cause is one layer beneath this engine's own component resolution,
  in the standard HTML tokenizer itself (`golang.org/x/net/html`, which
  `component.go`'s `parseTemplateHTML` feeds a template's *entire* raw HTML
  content to in one `html.Parse`/`html.ParseFragment` call — see that
  function's own header comment). `<Form>`, like every tag name, is
  lowercased to "form" during tokenization, same as the literal wrapping
  `<form>` around it — and the WHATWG HTML parsing spec has a dedicated,
  unconditional rule for this exact shape: a `<form>` start tag encountered
  while the parser's own form-element pointer is already set (i.e. while
  already inside another `<form>`) is a parse error, and *the token is
  simply ignored* — no element is created for it at all. This was confirmed
  directly, not inferred from reading the spec alone: a standalone
  `html.Parse` call on
  `<form method="post"><form id="inner"><input id="x"></form>...</form>`
  was run before writing this comment, and its rendered-back output
  contains exactly one `<form>` element, with `<input id="x">` reparented
  directly inside it — the inner `<form id="inner">` tag pair is entirely
  absent from the resulting tree, exactly as the spec's own "in body"
  insertion-mode algorithm describes for the form token. This happens
  *before* this engine's own component-resolution walk ever runs (parsing
  is a separate, earlier pass over the whole template — see engine.go's
  load sequence), so there is no hook in `renderer.go` that could
  compensate for it: by the time component resolution sees the tree, the
  inner `<Form>...</Form>` node pair, and every attribute it carried
  (`id`/`label`/`error`), has already ceased to exist. What survives is only
  its slotted children (e.g. the field's own `<input>`), silently reparented
  as direct children of the *outer* form — no error, no warning, no
  "[missing: <prop>]" placeholder (there is no node left to even attempt to
  render), just this component's entire label/message contribution quietly
  gone from the page.

  The fix is the same *kind* of fix "Critical" above already established
  for the outer `<form>`, applied to the inner reference instead: use a tag
  spelling that does not lowercase to the literal string "form". `v-native`
  itself does not apply here (`<Form>` is meant to resolve *as* the
  component, not to be excluded from resolution), but this package's own
  auto-registered kebab-case alias for this file, `radix-form` (assuming
  the standard `Mount{Prefix: "radix", ...}` this whole package hardcodes
  — see radix.go's header comment), works exactly like every other
  same-shape fix in this file: `<radix-form>` is not literally "form" to
  the tokenizer, so the nested-form suppression rule never matches it, and
  it survives standard HTML tree construction intact for this engine's own
  component resolution to correctly pick up afterward. Every usage example
  in this file's own header comment and the live examples/radix-demo
  gallery uses `<radix-form>` for exactly this reason — not a style
  preference, a load-bearing requirement for the reference to survive
  parsing at all whenever it is nested inside a literal `<form>` (which,
  per "Critical" above, is *every* real usage of this component).

  ## No customelement enhancement script: none, deliberately

  Per RFC 014 §3 Non-Goals, a script is only added when something is
  genuinely unreachable without it — the same discipline Switch.vue's and
  RadioGroup.vue's own commits already applied, both concluding that once
  every piece of a component's behavior is either a plain static attribute
  or something the browser already supplies for free, there is nothing
  left for a script to enhance. That reasoning applies here too, for a
  different reason than either of those two files: this component's
  `error`/`v-if`-gated message is pure server-computed conditional markup
  (an `error` string and a boolean check, both fully known at render time
  — the same "nothing depends on anything that happens after the page has
  already rendered" conclusion Progress.vue's header comment reaches for
  its own value/percentage rendering), so there is no client-side state
  intrinsic to *this* component for a script to own.

  A live-clearing enhancement was considered — hiding the error message the
  moment the user starts correcting the field, by listening for `input` on
  the slotted control — and rejected for this file specifically because of
  the same slot-content constraint documented above: a customelement
  enhancement script could in principle reach into its own light-DOM
  subtree (the slotted control is real, `querySelector`-able light DOM, not
  shadow DOM — Toolbar.vue's/Toast.vue's own scripts already read their own
  slotted/rendered subtrees this way), but it has no way to know *what
  kind* of element the consumer slotted in or what selector would reliably
  match it — unlike Toolbar's `items`-driven `.radix-toolbar-item` elements (this
  component's own markup, so a fixed class selector always matches), Form's
  slot content is arbitrary, consumer-authored markup with no fixed shape
  to select against. A caller who wants real-time clear-on-edit behavior
  already owns the slotted control directly in their own page and can wire
  an `input` listener to it themselves, targeting this component's
  documented `id + '-message'` convention to hide/clear the message — that
  is a small, genuinely optional enhancement squarely in the consumer's own
  script, not something this file can safely reach in and do for them.

  Usage (see also the full example at the top of this comment — note both
  the wrapping `<form v-native>`, required per "Critical" above, AND
  `<radix-form>` rather than `<Form>` for each field reference, required
  per "Critical #2" above):

    <form v-native method="post" action="/signup">
      <radix-form id="username" label="Username" error="">
        <input id="username" type="text" name="username" />
      </radix-form>

      <radix-form id="email" label="Email" error="Email is already taken">
        <input id="email" type="email" name="email"
               aria-invalid="true" aria-describedby="email-message" />
      </radix-form>

      <button type="submit">Sign up</button>
    </form>
-->
<template>
  <div class="radix-form-field" :data-invalid="error ? '' : undefined">
    <label v-native class="radix-label radix-form-label" :for="id">{{ label }}</label>
    <slot></slot>
    <p v-if="error" :id="id + '-message'" class="radix-form-message" role="alert">{{ error }}</p>
  </div>
</template>

<style scoped>
.radix-form-field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

/*
 * Same double-click-text-selection fix Label.vue's own <style scoped>
 * documents (`.radix-label`) — duplicated here, not composed via a nested
 * <Label> instance, since no other component in this package composes
 * <Label> into its own template either (every other file that needs a
 * label internally, e.g. Checkbox.vue's box, writes a plain
 * `<label v-native>` and matches Label's own class/CSS by hand instead).
 * This file follows that same established convention: the class name
 * ("radix-label") matches Label.vue's exactly, so a page that also mounts
 * Label.vue gets visually identical labels from either source, without
 * this file taking on a cross-component template dependency none of its
 * siblings have.
 */
.radix-form-label {
  display: inline-block;
  font-size: 0.9rem;
  font-weight: 600;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
}

.radix-form-message {
  margin: 0;
  font-size: 0.85rem;
  color: #dc2626;
}
</style>
