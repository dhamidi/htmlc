# RFC 014: Extension Mechanism for Component Packages and Hypermedia Server Helpers

- **Status**: Draft
- **Date**: 2026-07-30
- **Author**: TBD

---

## 1. Motivation

Two increasingly common requests from `htmlc` users have no clean answer today:

1. **"I want to ship a reusable component library"** — something in the spirit of
   [Radix UI](https://www.radix-ui.com/)'s Accordion, Tabs, Dialog, and Popover:
   accessible, headless building blocks that a consuming project can vendor and
   compose, ideally with a zero-JS HTML/CSS baseline that a `<script customelement>`
   block progressively enhances.
2. **"I want server-side helpers for Turbo / htmx / Datastar"** — the three
   dominant "hypermedia-oriented" front-end libraries, each of which imposes a
   specific response-shaping contract (fragment vs. full page, response headers,
   content types, and — for Datastar — a long-lived SSE connection) that today's
   `RenderPage`/`RenderFragment` API supports only by accident, not by design.

This RFC is grounded in two working prototypes built against the real `htmlc`
API (not against a hypothetical one) — one exercising component-library
distribution, one exercising htmx/Turbo Streams/Datastar integration — plus a
full read of `directive.go`, `engine.go`, `renderer.go`, and `component.go`.
Every claim below cites the actual code or a reproduced behavior, not a guess.

### 1.1 Why the existing extension points don't solve this

`htmlc` already has three extension mechanisms in production, and each one was
evaluated and found insufficient for this problem:

- **`Directive`/`DirectiveWithContent`** (`directive.go:50-86`) lets Go code
  hook `Created`/`Mounted` on a single element. But `DirectiveContext`
  (`directive.go:29-39`) exposes only `Registry` and `RenderedChildHTML` — no
  render scope, no sibling access, no `context.Context`. A directive cannot
  see the value of an unrelated prop, cannot inject scope for its children,
  and cannot participate in "is this an htmx request" branching, because it
  never sees the `*http.Request` or any request-scoped data unless that data
  was already placed into *its own* expression's value.
- **External directives** (NDJSON subprocess protocol, `cmd/htmlc/external_directive.go`,
  spec in `docs/external-directive-protocol.md`) let a third-party executable
  participate in rendering without recompiling `htmlc`. But this is wired up
  **only** inside `cmd/htmlc/build_command.go:374-393` — `htmlc render`,
  `htmlc page`, and any Go program that calls `htmlc.New` directly get no
  external-directive discovery at all. It is a `build`-CLI-only feature, not a
  library feature, and therefore cannot host a request-time server extension.
- **`Options.FS fs.FS`** (`engine.go:51-57`) lets a project load components
  from an embedded filesystem instead of the OS. This is the closest thing to
  a "vendor a component library" mechanism that exists today — but `Options`
  holds exactly **one** `FS` and **one** `ComponentDir`
  (`engine.go:24-57`). There is no way to tell one engine "also load
  components from this second, independent `fs.FS`."

### 1.2 Concrete failure modes, reproduced

Two out-of-tree prototypes (component distribution, and htmx/Turbo/Datastar
integration — both built against this repository's real `htmlc` package, not
modified in tree) surfaced the following, all independently reproduced:

- **`Engine.Register` cannot cross a filesystem boundary.** `Register(name,
  path string) error` (`engine.go:456-461`) always resolves `path` through the
  *engine's own* `opts.FS` (or `os.ReadFile` if nil) — confirmed by reading
  `registerPathLocked` (`engine.go:388-393`). Calling `Register("Accordion",
  "components/Accordion.vue")` against an OS-backed engine, where that path
  actually lives in a *different* `embed.FS` belonging to a vendored library,
  fails immediately with `open components/Accordion.vue: no such file or
  directory` — reproduced verbatim in the prototype.
- **Hand-rolling a union `fs.FS` works, but is unsupported and undocumented.**
  The prototype wrote a ~40-line custom `fs.FS` implementing only `Open`,
  `fs.StatFS`, and `fs.ReadDirFS` (the exact minimal surface `discoverInto`,
  `engine.go:288-373`, actually calls) to mount a consumer's OS directory and a
  vendor's `embed.FS` under one root. It worked completely — proximity
  resolution, `<component is="dir/Name">`, and custom-element tag derivation
  all functioned correctly across the union boundary — but nothing in the
  public API says this is how you're supposed to distribute a component
  library, and the exact minimal interface `discoverInto` needs is an
  implementation detail that could change without notice.
- **Cross-mount name collisions resolve silently and can flip without
  warning.** When a consumer project and a vendored library both define
  `Accordion`, an unqualified `<Accordion>` correctly prefers the local one via
  proximity (`renderer.go:1472-1479`), matching documented RFC 001 semantics.
  But the flat-registry entry for `"Accordion"` is silently overwritten by
  whichever file the lexical walk visits last (`engine.go:38-39` documents
  this as existing, working-as-intended behavior for a single tree). The
  prototype reproduced a specific consequence: adding an unrelated,
  correctly-named local `Accordion.vue` to satisfy a naming-collision test
  caused an **existing, unmodified** template's `<Accordion>` reference —
  previously resolving to the vendor's component — to silently start
  resolving to the new local one instead. No error, no warning,
  `ValidateAll()` stayed clean. This is qualitatively different from RFC 001's
  proximity walk within one project tree, where the author controls every
  file; once components arrive from an independent, separately-versioned
  package, an author adding an unrelated file to their own project can
  silently repoint a reference into someone else's code.
- **The documented disambiguation syntax corrupts the page when self-closed.**
  The README's own "Explicit cross-directory references" section
  (README.md:502-506) shows `<component is="blog/Card" />` — self-closed.
  `normalizeSelfClosingComponents` (`component.go:475-488`) only rewrites
  self-closing tags whose name **starts with an uppercase letter**
  (`selfClosingComponentRe`, `component.go:475-477`); the lowercase built-in
  `<component>` tag is never rewritten. The HTML5 parser then treats the
  trailing `/` on this non-void element as meaningless and leaves the tag
  open, silently swallowing every subsequent sibling as a child of the
  unclosed `<component>` — which, since the resolved target has no `<slot>`,
  discards that entire subtree with **no error at all**. Reproduced: a
  three-section page using self-closing `<component is="..." />` for
  disambiguation rendered only its first two sections; switching to
  `<component is="...">...</component>` fixed it completely. This bug directly
  undermines the one mechanism §4.2 depends on for resolving cross-package
  name collisions.
- **An invalid custom-element tag name can pass `ValidateAll()` silently.**
  `component.go:133-140` warns when a derived custom-element tag has no
  hyphen — but it checks the tag computed **inside `ParseFile`**, before
  `discoverInto`/`registerPathLocked` re-derive `CustomElementTag` relative to
  `ComponentDir` (`engine.go:305-317`, `engine.go:395-407`). A root-level
  `Accordion.vue` typically has a hyphen in its pre-relative, path-based tag
  (e.g. from directory separators in the full OS path) and so passes the
  early check — but re-derives to the hyphen-less `accordion` afterward, which
  is invalid per the Custom Elements spec (`customElements.define('accordion',
  ...)` throws in a real browser). Reproduced: `eng.ValidateAll()` reported
  zero warnings while the emitted markup contained a literal `<accordion>`
  wrapper tag.
- **Any hyphenated tag not found in the registry is a hard render error, with
  no passthrough option.** `isComponentLike` (`renderer.go:1458-1460`) treats
  any tag containing a hyphen as a component reference; if `resolveComponent`
  fails to find it, rendering aborts with `unknown component: %q`
  (`renderer.go:1172-1174`). Reproduced directly: adding a literal
  `<turbo-frame id="...">` element to a `.vue` template (Turbo's own custom
  element, not an `htmlc` component) fails the render outright. Any hypermedia
  library or Web Component that ships its own custom element tag — Turbo's
  `<turbo-frame>`/`<turbo-stream>`, a hand-authored `<my-datepicker>`, or a
  third-party design system's tags — cannot appear literally in a template
  unless it is also registered as a dummy `.vue` component.
- **Conditionally omitting a non-native boolean attribute doesn't work.**
  `isBooleanAttr` (`renderer.go:1953-1958`) recognizes a fixed allowlist —
  `disabled, checked, selected, readonly, required, multiple, autofocus,
  open` — used at the two `:attr` binding sites (`renderer.go:1241`,
  `renderer.go:2078`) to decide whether a falsy bound value **omits** the
  attribute (real HTML boolean-attribute semantics) or **stringifies** it.
  Reproduced: `:hx-swap-oob="oob"` with `oob = false` renders literally as
  `hx-swap-oob="false"` (and `oob = nil` as `hx-swap-oob="null"`) — the
  attribute is never omitted, because `hx-swap-oob` isn't in the fixed list.
  This blocks the single most common hypermedia idiom — "include this
  attribute only when a condition holds" — for every non-native attribute
  vocabulary (`hx-*`, `data-turbo-*`, `data-*` in general).
- **Styles are re-emitted on every call, with no way to dedup across a
  connection.** `RenderFragment`/`RenderPage` always run a fresh internal
  `StyleCollector` per call (confirmed by reading `engine.go:988-1013`,
  `1033-1094` and by inspecting real output). Reproduced over a real SSE
  connection (see §1.3): three ticks of the same `Counter` component, sent as
  three separate Datastar `datastar-patch-elements` events on one open
  response, repeated the **identical** `<style scoped>` block — same
  `data-v-*` hash, byte-identical CSS — on every single tick. The only
  existing way to suppress the style header at all is
  `RenderWithCollector`/`RenderWithCollectorString` (`engine.go:907-928`),
  which happens to skip styles as a side effect of what its
  `*CustomElementCollector` parameter is actually for (custom-element script
  collection) — an undocumented, all-or-nothing accident, not a supported
  "render without style header" mode, and there is no way to keep a
  `StyleCollector` alive *across* calls so that only genuinely new styles are
  sent as a long-lived connection progresses.
- **A single fragment's own root element cannot be given an extra attribute
  at render time.** The idiomatic htmx out-of-band-swap pattern needs a
  target's own rendered root tag to carry `hx-swap-oob="true"` and a matching
  `id`. `htmlc` provides no way to inject an attribute into an
  already-rendered component's root element; the only two options are (a)
  wrap the rendered fragment in a synthetic container carrying the attribute
  — which, when the fragment's own root element already declares the same
  `id` (a very common case, since the ID is exactly what OOB swapping targets
  by), produces two nested elements with the same `id` in the final DOM after
  the swap — reproduced directly; or (b) bake the hypermedia attribute into
  the component's own template with a conditional `v-bind`, coupling an
  otherwise plain UI component to htmx vocabulary.

### 1.3 Datastar and the "long-lived connection" difference

Turbo Drive, Turbo Frames, Turbo Streams, and htmx are all satisfied by
`htmlc`'s existing per-request, single-write `Render*` model: pick full-page
vs. fragment based on a request header, optionally wrap the fragment or set a
response header, write once. Confirmed by building working prototypes of all
three response shapes with **no changes to `htmlc` itself** — see §6.

Datastar is architecturally different: its entire backend contract is a
single long-lived `text/event-stream` response
(`Content-Type: text/event-stream`) over which the server emits zero-to-many
`datastar-patch-elements`/`datastar-patch-signals` SSE events over time,
using the `ServerSentEventGenerator` pattern (confirmed against
`data-star.dev/reference/sse_events` and `data-star.dev/guide/backend_requests`;
a maintained Go SDK exists at `github.com/starfederation/datastar-go`, package
`datastar`, confirmed installable and used successfully in the prototype —
`v1.2.2` at time of writing). This is still compatible with `htmlc`'s
`io.Writer`-based rendering — the prototype confirmed genuine incremental
delivery (SSE events observed arriving ~200ms apart, not buffered) — but it is
the one case where "render once per response" stops being true, which is
exactly what surfaces the style-deduplication gap in §1.2.

---

## 2. Goals

1. **Multi-source component registration**: an engine can be configured with
   more than one component source (an OS directory, an `embed.FS` from a
   vendored Go package, or both) and resolve references across all of them
   using the existing proximity/explicit-path rules.
2. **Fail-fast collision detection across sources**: when two independently
   authored sources both define a component under the same flat name, the
   engine surfaces this at validation time rather than silently picking one
   based on lexical walk order or letting an unrelated later change flip
   which one an existing reference resolves to.
3. **Foreign/native custom element passthrough**: a template can contain a
   literal hyphenated tag that is not an `htmlc` component — a hypermedia
   library's own custom element, or any hand-authored Web Component — without
   registering a dummy `.vue` file for it, and without silently swallowing a
   genuine typo in an unregistered component reference.
4. **General falsy-attribute omission**: any `:attr`/`v-bind:attr` binding
   whose value is exactly `false`, `nil`, or JavaScript-style `undefined`
   omits the attribute entirely — not just the fixed eight native boolean
   attributes — matching real Vue.js semantics and unblocking the
   "include this attribute only when a condition holds" idiom for `hx-*`,
   `data-turbo-*`, and other third-party attribute vocabularies.
5. **A style/script collector that can outlive a single render call**, so
   that a caller driving multiple fragment renders into one open connection
   (SSE, or several fragments concatenated into one response) can dedup
   shared `<style scoped>` output across the whole session rather than
   resending it on every call.
6. **A thin, optional hypermedia helper layer** for htmx, Turbo (Drive,
   Frames, Streams), and Datastar that composes with the existing
   `Render*`/`ServeComponent` API rather than replacing it, so a project that
   doesn't use any of these three libraries pays zero cost and sees zero API
   surface change.
7. **Fix the two parser bugs this design depends on** (self-closing
   `<component>`, custom-element hyphen-check ordering) as a prerequisite,
   since both directly undermine mechanisms this RFC's design relies on.

---

## 3. Non-Goals

- **Shipping actual ports of Radix/Zag/Ark/etc. as bundled `htmlc` code.**
  This RFC makes it *possible* to distribute a Radix-style component library
  as a `.vue`-file package; authoring that library (or porting an existing
  one) is downstream work, not part of `htmlc` core.
- **Implementing accessibility behavior (focus trapping, roving tabindex,
  collision-aware positioning) inside the engine.** That remains the
  responsibility of the JavaScript in `<script customelement>`, exactly as
  today. This RFC only makes it easier to *ship and vendor* such components.
- **A package manager or versioning system for component packages.** Go
  modules already solve dependency resolution and versioning; a vendored
  component package is just a Go package that exposes an `fs.FS`. This RFC
  only needs `htmlc` to consume more than one `fs.FS`.
- **A bundler or transpiler for third-party client JS.** No npm resolution,
  no TypeScript, no tree-shaking — `<script customelement>` content remains
  hand-authored (or externally built) raw JavaScript text, exactly as RFC 006
  established.
- **WebSocket support for Turbo Streams' real-time broadcast path
  (`<turbo-stream-source>`).** Only the request/response shapes (Drive,
  Frames, single-response Streams) and Datastar's SSE model are addressed;
  broadcasting the same stream to multiple already-connected clients is a
  pub/sub concern outside `htmlc`'s scope.
- **Changing RFC 001's core proximity-resolution semantics for a single,
  single-source project.** Everything in §4 is additive; a project with one
  `ComponentDir` and no vendored packages sees no behavioral change.
- **A general-purpose plugin registry, marketplace, or discovery service.**
  This RFC solves distribution (how one engine consumes multiple sources) and
  a specific set of parser/renderer gaps — not a `go install`-able plugin
  ecosystem with its own tooling.

---

## 4. Proposed Design

### 4.1 Component packages: multi-source registration

#### Current state

`Options` (`engine.go:23-86`) holds exactly one `FS fs.FS` and one
`ComponentDir string`. `Engine.Register(name, path string) error`
(`engine.go:456-461`) reads `path` through `e.opts.FS` (or the OS filesystem)
— confirmed to fail when `path` belongs to a different `fs.FS` than the one
the engine was constructed with. There is no supported way to combine two
independent component sources into one engine.

#### Proposed extension

Add a new, additive field to `Options`:

```go
// pseudo-code — not implementation
type Options struct {
    // ... existing fields unchanged ...

    // Mounts registers additional component sources alongside the primary
    // FS/ComponentDir pair. Each Mount's components are addressed under
    // Mount.Prefix (e.g. Prefix: "radix" makes its Accordion.vue reachable
    // as <component is="radix/Accordion">). Mounts are additive: setting
    // Mounts does not require setting FS/ComponentDir, and vice versa.
    Mounts []Mount
}

type Mount struct {
    Prefix string // namespace prefix; must not contain "/" or collide with a
                  // top-level directory name already present in ComponentDir
    FS     fs.FS  // component source for this mount
    Dir    string // path within FS to scan (mirrors ComponentDir)
}
```

Internally, `New` builds a single union `fs.FS` from `Options.FS` (mounted at
`""`) plus each `Mount` (mounted at `Mount.Prefix + "/"`), and proceeds
exactly as today using that union as `e.opts.FS` and `""` as the effective
`ComponentDir`. This formalizes, as first-class supported API, the same union
approach the prototype hand-rolled and confirmed working (proximity
resolution, `<component is="dir/Name">`, and custom-element tag derivation
all functioned correctly across the union boundary). The union `fs.FS`
implementation itself — needing only `Open`, `fs.StatFS`, and
`fs.ReadDirFS`, per the exact surface `discoverInto` (`engine.go:288-373`)
calls — ships as an internal, tested implementation detail; callers never
write their own.

A component-package author needs no `htmlc` dependency at all — a Go package
that exposes `//go:embed components/*.vue` as an `embed.FS` (with a
documented convention, e.g. an exported `func FS() fs.FS`) is a complete
component package, exactly as the prototype's `vendor-radix-htmlc` module
demonstrated.

#### Evaluation

- **Option A — `Options.Mounts []Mount` (proposed above)**
  - ✅ Purely additive to `Options`; zero change for existing single-source
    projects.
  - ✅ Matches the exact working pattern already validated by prototype.
  - ✅ Namespace prefix is explicit and author-controlled, avoiding the
    silent root-level collisions in §1.2.
  - ⚠️ Requires a small amount of new internal union-`fs.FS` code, but this
    is a self-contained, easily-tested addition (~40 lines per the
    prototype).
  - **Verdict**: recommended.
- **Option B — leave the union `fs.FS` as a documented pattern only, no new
  `Options` field**
  - ✅ Zero new API surface.
  - ❌ Every project wanting to vendor a component package must hand-write
    (and maintain) the same ~40-line `fs.FS` shim, against an interface
    (`discoverInto`'s exact minimal `fs.FS` usage) that is an implementation
    detail and could change without notice — exactly the fragility the
    prototype flagged.
  - **Verdict**: rejected — pushes brittle, unsupported code onto every
    consumer of this feature.
- **Option C — a `Registry` interface, so a Go package can hand the engine
  components directly without any filesystem round-trip**
  - ✅ Would also solve non-FS-backed component sources (e.g. Go
    code-generated components, a database-backed registry).
  - ❌ `*Component` today can only be constructed via `ParseFile(path, src
    string)` on real `.vue` source text (`component.go:86-148`); there is no
    builder API or interface a "virtual component" could implement, and the
    renderer deals in the concrete `*Component` type throughout
    (`renderer.go:1168-1170`, `1648`). Making this work is a substantially
    larger change than the problem in front of us requires.
  - **Verdict**: out of scope for this RFC; noted as a candidate for a
    follow-up RFC in §10 (non-blocking).

### 4.2 Collision detection across mounts

#### Current state

Within a single `ComponentDir`, RFC 001's proximity walk resolves same-named
components deterministically and is well-understood by the author, who
controls every file in the tree. Across independent mounts (§4.1), the same
mechanism produces a qualitatively different risk: the flat-registry fallback
silently picks whichever source's file the lexical walk visits last
(`engine.go:38-39`), and — as reproduced in §1.2 — adding an unrelated,
correctly-named file anywhere in the *consumer's own* tree can silently
repoint an existing, unmodified template reference from a vendored
component to a newly-introduced local one, with no error and no warning from
`ValidateAll()`.

#### Proposed extension

Give each registry entry a **mount identity** (which `Mount.Prefix`, or the
empty string for the primary `Options.FS`/`ComponentDir`, produced it).
`ValidateAll()` (`engine.go:657-698`) gains a new check: for any flat name
that resolves to entries from **two or more distinct mount identities** with
no proximity-based tie-breaker available (i.e. the collision would actually
be reached by some caller through the flat-registry fallback, not shadowed by
a closer proximity match), report an error rather than silently registering
whichever one lexical order visited last.

```go
// pseudo-code — not implementation
type componentConflict struct {
    Name    string
    Mounts  []string // mount identities that all define this name
}

func (e *Engine) checkMountCollisions() []componentConflict {
    byName := map[string][]string{} // flat name -> mount identities
    for name, entry := range e.entries {
        byName[name] = append(byName[name], entry.mountID)
    }
    var conflicts []componentConflict
    for name, mounts := range byName {
        if len(distinct(mounts)) > 1 {
            conflicts = append(conflicts, componentConflict{name, distinct(mounts)})
        }
    }
    return conflicts
}
```

This runs as part of `ValidateAll()` whenever `Options.Mounts` is non-empty,
so single-source projects (the overwhelming majority today) see no new
behavior or performance cost. A collision does not have to be a hard startup
error — see Open Question 2 (§10) for whether it should default to a
returned validation error (matching `ValidateAll`'s existing contract, which
already returns `[]error` for the caller to act on) or a `Logger` warning.

### 4.3 Foreign/native custom element passthrough

#### Current state

`isComponentLike` (`renderer.go:1458-1460`) treats **any** hyphenated tag
name as an implied component reference; if `resolveComponent` fails to find
it, rendering aborts with `unknown component: %q` (`renderer.go:1172-1174`).
This is reproduced to fail on a literal `<turbo-frame id="...">` — a
perfectly valid native custom element belonging to a third-party library, not
an `htmlc` component.

#### Proposed extension

Add an explicit, opt-in allowlist:

```go
// pseudo-code — not implementation
type Options struct {
    // ... existing and Mounts fields ...

    // NativeElements lists hyphenated tag names that must never be resolved
    // against the component registry, even though they contain a hyphen.
    // Their attributes and children render exactly as a normal HTML element's
    // would (expressions evaluated, v-if/v-for honored, no component lookup).
    // Unlisted hyphenated tags retain today's behavior: resolve against the
    // registry, or fail with "unknown component" if not found.
    NativeElements []string
}
```

At the `isComponentLike` check site (`renderer.go:1172`), consult the
allowlist before treating the tag as component-like:

```go
// pseudo-code — not implementation
if r.registry != nil && isComponentLike(working.Data) && !r.isNativeElement(working.Data) {
    return fmt.Errorf("unknown component: %q", working.Data)
}
```

#### Evaluation

- **Option A — explicit `Options.NativeElements` allowlist (proposed above)**
  - ✅ No change to default behavior; existing projects are unaffected until
    they opt in.
  - ✅ A genuine typo in a component reference still fails loudly — this
    option requires the author to explicitly declare "this hyphenated tag is
    not one of my components," so an accidental unregistered reference is not
    silently swallowed as inert markup.
  - ⚠️ Requires the author to enumerate every foreign custom element tag used
    across the project; a large hypermedia library (many `<turbo-*>` tags,
    say) means a longer list, though this is a one-time, small cost.
  - **Verdict**: recommended.
- **Option B — auto-detect: if an unresolved hyphenated tag isn't in the
  registry, warn and pass it through instead of erroring**
  - ✅ No new `Options` field, zero authoring cost.
  - ❌ A genuinely misspelled or forgotten component reference (e.g.
    `<my-crad>` instead of `<my-card>` — assuming a hyphenated naming
    convention) would silently render as inert, unstyled markup instead of
    failing the build/render, trading a loud, catchable error for a subtle
    runtime bug.
  - **Verdict**: rejected — silently downgrading a real class of authoring
    mistakes is a worse failure mode than requiring an explicit allowlist.
- **Option C — per-tag opt-out via a template-level escape hatch, e.g.
  `v-native` attribute on the element**
  - ✅ Localized to the exact call site; no project-wide list to maintain.
  - ⚠️ Every occurrence of the foreign tag across every template needs the
    attribute, which is more repetitive than declaring it once in `Options`
    for a tag used dozens of times across a hypermedia-heavy project.
  - **Verdict**: viable as a *complement* to Option A (useful for a one-off
    foreign tag that doesn't warrant a project-wide allowlist entry), not a
    replacement. Left as a non-blocking open question in §10.

### 4.4 General falsy-attribute omission

#### Current state

`isBooleanAttr` (`renderer.go:1953-1958`) hardcodes exactly eight native HTML
boolean attributes. The two `:attr` binding sites (`renderer.go:1241`,
`renderer.go:2078`) consult this list to decide whether a falsy bound value
omits the attribute or stringifies it. Real Vue.js does not use a fixed
allowlist for this: **any** attribute bound to `false`/`null`/`undefined` is
omitted, regardless of attribute name. Reproduced: `:hx-swap-oob="false"`
renders as the literal attribute `hx-swap-oob="false"` today, blocking the
"include this attribute only when a condition holds" idiom for `hx-*`,
`data-turbo-*`, and any other third-party attribute vocabulary.

#### Proposed extension

Replace the allowlist check at both binding sites with a general rule: a
dynamically-bound attribute (`:attr`/`v-bind:attr`) whose evaluated value is
exactly Go `false`, `nil`, or the engine's `undefined` sentinel omits the
attribute; every other value (including the empty string) renders normally.
Static attributes (`attr="literal"`, no `:` prefix) are never affected — this
only changes behavior for `:`-bound expressions, which is where a caller
expresses a condition.

```go
// pseudo-code — not implementation
func attrShouldOmit(v any) bool {
    if v == nil {
        return true
    }
    if b, ok := v.(bool); ok && !b {
        return true
    }
    return v == expr.Undefined // however the engine represents undefined
}
```

#### Evaluation

- **Option A — general falsy-omission rule (proposed above)**
  - ✅ Matches real Vue.js semantics exactly — no surprising divergence for
    anyone coming from Vue.
  - ✅ Solves the problem for every attribute vocabulary at once (`hx-*`,
    `data-turbo-*`, arbitrary `data-*`), not just a curated list.
  - ⚠️ Backward compatibility: today, `:some-attr="false"` renders
    `some-attr="false"` for any attribute outside the fixed eight. This RFC's
    change makes that attribute disappear instead. See §8 for the specific
    compatibility analysis and recommendation.
  - **Verdict**: recommended, with the compatibility note in §8.
- **Option B — extend the fixed `isBooleanAttr` list with known hypermedia
  attribute names (`hx-swap-oob`, `hx-boost`, `data-turbo-*`, …)**
  - ✅ Smaller, narrowly-scoped change; no compatibility question at all for
    attributes outside the extended list.
  - ❌ Perpetually incomplete — every new hypermedia library or custom
    attribute vocabulary needs its own PR to extend the list, and does
    nothing for a project's own custom `data-*` attributes used the same way.
  - **Verdict**: rejected as the primary fix; the general rule in Option A
    subsumes this need without the maintenance burden.

### 4.5 A style collector that outlives a single render call

#### Current state

`RenderFragment`/`RenderPage` construct and consume a fresh `StyleCollector`
internally on every call (`engine.go:988-1013`, `1033-1094`). The only
existing way to render without a style header at all is
`RenderWithCollector`/`RenderWithCollectorString` (`engine.go:907-928`), and
that behavior is a side effect of what the `*CustomElementCollector`
parameter is actually documented for (script collection), not a supported
"skip styles" mode. Reproduced over a real Datastar SSE connection: three
successive `RenderFragmentContext` calls into the same open response repeated
an identical `<style scoped>` block on every tick.

#### Proposed extension

Expose `StyleCollector` as a public, reusable, cross-call object, mirroring
the existing `*CustomElementCollector` pattern that `RenderWithCollector`
already established:

```go
// pseudo-code — not implementation
func (e *Engine) NewStyleCollector() *StyleCollector

// RenderFragmentWithStyles renders name into w exactly like RenderFragment,
// but sources style output from sc instead of a fresh per-call collector.
// Passing the same *StyleCollector across multiple calls (e.g. across the
// life of one SSE connection) means only styles not already emitted by an
// earlier call on that same sc are written — sc tracks what has already
// been sent, the same way *CustomElementCollector already dedups scripts
// by content hash today.
func (e *Engine) RenderFragmentWithStyles(ctx context.Context, w io.Writer, name string, data map[string]any, sc *StyleCollector) error
```

A caller driving a Datastar SSE loop, or emitting several fragments
concatenated into one htmx/Turbo Streams response, creates one
`*StyleCollector` for the connection/response and passes it to every
`RenderFragmentWithStyles` call; the first call that touches a given
component's scoped styles emits them, every subsequent call touching the same
component does not repeat them. This directly targets the redundant-style-tag
problem reproduced in §1.2 for Datastar, and incidentally also benefits htmx
OOB responses and multi-action Turbo Streams responses (§6), which showed the
identical duplication for the same underlying reason: independent
`RenderFragment` calls concatenated into one response, each unaware of what a
sibling call already emitted.

### 4.6 Hypermedia server helpers

#### Current state

Nothing in `htmlc` today understands `HX-Request`, `Turbo-Frame`,
`Accept: text/vnd.turbo-stream.html`, or the Datastar SSE event format. The
README even points users at `RenderFragment` for "HTMX responses, turbo
frames, etc." (README.md, "Render an HTML fragment" section) without any
further support. Every one of the three prototypes in §6 needed the caller to
hand-write header checks, content-type strings, and wrapper markup from
scratch.

#### Proposed extension

A new, optional subpackage — `github.com/dhamidi/htmlc/hypermedia` — with
small, focused helpers per library. None of these change `Engine`'s core
behavior; they are thin wrappers a caller opts into. Sketch (see §7 for the
implementation-file breakdown):

```go
// pseudo-code — not implementation, package hypermedia

// htmx
func IsHTMXRequest(r *http.Request) bool          // checks HX-Request header
func IsBoosted(r *http.Request) bool              // checks HX-Boosted header
func SetTrigger(w http.ResponseWriter, event string) // sets HX-Trigger

// Turbo
const TurboStreamContentType = "text/vnd.turbo-stream.html"
func WantsTurboStream(r *http.Request) bool        // checks Accept header
func TurboFrameID(r *http.Request) (id string, ok bool) // checks Turbo-Frame header
func WriteTurboStream(w io.Writer, action, target, fragmentHTML string) error

// Datastar
func WriteDatastarPatchElements(w io.Writer, selector, mode, elementsHTML string) error
// (a thinner alternative: document the pattern of combining
// datastar-go's ServerSentEventGenerator directly with
// Engine.RenderFragmentWithStyles from §4.5, rather than wrapping the SDK)
```

#### Evaluation

- **Option A — ship a small `hypermedia` subpackage in the main module
  (proposed above)**
  - ✅ Gives users a tested, correct implementation of header/content-type
    details that are easy to get subtly wrong (e.g. the exact
    `text/vnd.turbo-stream.html` string, or which header htmx actually reads
    for boost detection).
  - ⚠️ Couples `htmlc`'s release cadence to three external libraries' evolving
    protocols — if htmx changes a header name in a future major version, this
    subpackage needs a matching update.
  - **Verdict**: recommended, but scoped deliberately small (header/content-
    type helpers and stream-wrapping only — no attempt to model Turbo's or
    Datastar's client-side behavior, no bundled JS).
- **Option B — publish this only as documentation/recipes, no shipped code**
  - ✅ Zero new maintenance surface, zero coupling to external protocol
    churn.
  - ❌ Every project re-implements the same handful of header checks and
    string constants, with the same opportunity to get the exact header name
    or content-type string wrong that Option A's subpackage exists to
    prevent.
  - **Verdict**: rejected as the sole answer, but the RFC's Examples (§6) and
    a docs-branch tutorial (per this repo's documentation convention) should
    still show the underlying pattern explicitly, not hide it behind the
    helper package, so a project using a fourth hypermedia library (or a
    future Turbo/htmx major version) can replicate the approach without
    waiting on an `htmlc` release.

### 4.7 Prerequisite bug fixes

Both of the following are correctness bugs independent of this RFC's new
features, but §4.1's collision-disambiguation story (`<component
is="mount/Name">`) and §4.3's custom-element passthrough both depend on the
surrounding mechanisms being reliable, so they are called out as
prerequisites rather than deferred cleanup:

1. **Self-closing `<component>` corrupts the page.**
   `selfClosingComponentRe` (`component.go:475-477`) must also match the
   lowercase built-in `component` tag name (`is`/`:is` self-closing form),
   not only uppercase PascalCase custom-component tags. This is the exact
   syntax the README already documents for explicit cross-directory
   references (README.md "Explicit cross-directory references" section); the
   fix makes existing documented usage actually safe rather than changing
   any public syntax.
2. **Custom-element hyphen validity check runs against the wrong tag.** The
   warning in `component.go:133-140` must be (re-)checked against the
   **final**, `ComponentDir`-relative tag computed in `discoverInto`
   (`engine.go:305-317`) and `registerPathLocked` (`engine.go:395-407`), not
   only the earlier, path-based tag computed inside `ParseFile`. This closes
   the gap where `ValidateAll()` reports no warning for a component whose
   actual emitted tag is invalid.

---

## 5. Syntax Summary

| Syntax | Meaning |
|---|---|
| `Options.Mounts = []Mount{{Prefix: "radix", FS: radixlib.FS(), Dir: "components"}}` | Registers a second, independently-sourced component tree, addressable under the `radix/` prefix alongside the primary `ComponentDir` |
| `<component is="radix/Accordion">...</component>` | Explicit reference into a specific mount, unaffected by proximity or naming collisions with the local tree (existing syntax; behavior fixed per §4.7 item 1) |
| `Options.NativeElements = []string{"turbo-frame", "turbo-stream"}` | Declares specific hyphenated tags as native HTML, never resolved against the component registry |
| `<turbo-frame id="todos">...</turbo-frame>` | Renders as a plain native element (attributes/children evaluated normally) when `turbo-frame` is declared in `NativeElements`; otherwise fails with `unknown component` as today |
| `:hx-swap-oob="isOOB"` | Omits the `hx-swap-oob` attribute entirely when `isOOB` is `false`/`nil`/`undefined`; renders `hx-swap-oob="true"` otherwise — works for any attribute name, not just the eight native HTML boolean attributes |
| `sc := engine.NewStyleCollector()` then repeated `engine.RenderFragmentWithStyles(ctx, w, name, data, sc)` | Dedups `<style scoped>` output across multiple render calls sharing `sc` (one SSE connection, or several fragments concatenated into one response) |
| `hypermedia.IsHTMXRequest(r)` / `hypermedia.WantsTurboStream(r)` / `hypermedia.TurboFrameID(r)` | Request-side detection helpers for branching page-vs-fragment rendering |
| `hypermedia.WriteTurboStream(w, "replace", "todo-list", fragmentHTML)` | Wraps rendered fragment HTML in a `<turbo-stream>` element with the correct `Content-Type` implications |

---

## 6. Examples

### Example 1 — Vendoring a dual-mode Accordion library (backward-compatible: no mounts)

A project with a single `ComponentDir` and no `Mounts` behaves identically to
today — this is the default, unaffected by anything in this RFC:

```
templates/
  HomePage.vue
  Card.vue
```

```go
engine, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
```

No `Mounts`, no `NativeElements`, no behavior change.

### Example 2 — Vendoring the library, with collision-free namespacing

```
// third-party module: github.com/example/radix-htmlc
radix-htmlc/
  components/
    Accordion.vue   ← HTML-only <details>/<summary> baseline + <script customelement>
    Tabs.vue
  radix.go          ← //go:embed components/*.vue ; func FS() fs.FS { return embeddedFS }
```

```go
import radixhtmlc "github.com/example/radix-htmlc"

engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Mounts: []htmlc.Mount{
        {Prefix: "radix", FS: radixhtmlc.FS(), Dir: "components"},
    },
})
```

```vue
<!-- templates/HomePage.vue -->
<template>
  <component is="radix/Accordion" :items="faqItems" />
</template>
```

If `templates/` also happens to define its own `Accordion.vue`, an
unqualified `<Accordion>` anywhere in `templates/` still resolves to the
*local* one via proximity (RFC 001 behavior, unchanged); only the explicit
`<component is="radix/Accordion">` reaches the vendored one. `ValidateAll()`
reports no conflict here because the two `Accordion` names are only ever
reached through disambiguated, mount-qualified paths in this example — see
Example 3 for the case where a genuine ambiguity exists.

### Example 3 — A genuine collision, caught at validation time

```
templates/
  Accordion.vue        ← local, unrelated component
  HomePage.vue          ← <Accordion /> (unqualified, no mount prefix used anywhere)
```

```go
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Mounts: []htmlc.Mount{
        {Prefix: "radix", FS: radixhtmlc.FS(), Dir: "components"},
    },
})
if errs := engine.ValidateAll(); len(errs) > 0 {
    // reports: component name "Accordion" is ambiguous across mounts ""
    // and "radix" — no conflict today, since templates/Accordion.vue
    // shadows the vendored one via proximity for every caller in
    // templates/. Only truly unreachable-except-by-flat-fallback
    // collisions are reported (see §4.2).
}
```

### Example 4 — htmx: fragment vs. full page, with an out-of-band swap and no duplicate styles

```go
func handleIncrement(w http.ResponseWriter, r *http.Request, engine *htmlc.Engine, sc *htmlc.StyleCollector) {
    count := incrementCounter()
    data := map[string]any{"count": count}

    if !hypermedia.IsHTMXRequest(r) {
        engine.RenderPage(r.Context(), w, "HomePage", data)
        return
    }

    hypermedia.SetTrigger(w, "counter-updated")
    engine.RenderFragmentWithStyles(r.Context(), w, "Counter", data, sc)
    fmt.Fprintf(w, `<div id="status-badge" hx-swap-oob="true">`)
    engine.RenderFragmentWithStyles(r.Context(), w, "StatusBadge", data, sc)
    fmt.Fprint(w, `</div>`)
}
```

Both fragments share `sc`, so `Counter` and `StatusBadge`'s scoped `<style>`
blocks are each emitted at most once across the response, even though both
components are rendered independently into the same writer.

### Example 5 — Turbo Streams: two actions in one response

```go
func handleTodoCreate(w http.ResponseWriter, r *http.Request, engine *htmlc.Engine) {
    todo := createTodo(r)
    w.Header().Set("Content-Type", hypermedia.TurboStreamContentType)

    itemHTML, _ := engine.RenderFragmentString(r.Context(), "TodoItem", map[string]any{"todo": todo})
    hypermedia.WriteTurboStream(w, "append", "todo-list", itemHTML)

    countHTML, _ := engine.RenderFragmentString(r.Context(), "TodoCount", map[string]any{"count": todoCount()})
    hypermedia.WriteTurboStream(w, "update", "todo-count", countHTML)
}
```

Confirmed (see §1.3 and the prototype behind it): concatenating multiple
`<turbo-stream>` elements requires no special framing — this is exactly the
pattern already validated end-to-end.

### Example 6 — Datastar: an SSE loop with deduplicated styles

```go
func handleCounterStream(w http.ResponseWriter, r *http.Request, engine *htmlc.Engine) {
    sse := datastar.NewSSE(w, r)
    sc := engine.NewStyleCollector()

    for i := 1; i <= 3; i++ {
        var buf bytes.Buffer
        engine.RenderFragmentWithStyles(r.Context(), &buf, "Counter", map[string]any{"count": i}, sc)
        sse.PatchElements(buf.String(), datastar.WithSelector("#ds-counter"))
        time.Sleep(200 * time.Millisecond)
    }
}
```

Only the first `PatchElements` call's `elements` payload carries the
`<style scoped>` block; the second and third carry bare HTML — reproduced as
a real, working pattern in the prototype using the existing
`RenderWithCollector` accident (§1.2), and formalized here as the supported
`RenderFragmentWithStyles` API.

---

## 7. Implementation Sketch

High-level Go-level changes only, grouped by file:

### `engine.go`

1. **`Options`**: add `Mounts []Mount` (§4.1) and `NativeElements []string`
   (§4.3).
2. **`Mount` type**: new exported struct (`Prefix`, `FS`, `Dir`).
3. **`New`**: when `Mounts` is non-empty, build an internal union `fs.FS`
   (new file, `unionfs.go`) wrapping `Options.FS`/`ComponentDir` at `""` and
   each mount at `Prefix + "/"`; use the union as the effective `opts.FS`,
   `""` as the effective `ComponentDir`. Track each registered entry's mount
   identity (extend `engineEntry` with a `mountID string` field) for §4.2.
4. **`ValidateAll`**: add the cross-mount collision check from §4.2 when
   `Mounts` is non-empty; unaffected otherwise.
5. **`NewStyleCollector`** (new method, §4.5): constructs and returns a
   `*StyleCollector` for reuse across calls.
6. **`RenderFragmentWithStyles`** (new method, §4.5): parallels
   `RenderFragment` but threads a caller-supplied `*StyleCollector` through
   instead of allocating a fresh one, and skips already-emitted style blocks
   the same way `CustomElementCollector.Add` (`customelement_collector.go:61-76`)
   already dedups scripts by content hash.
7. **`component.go:133-140` hyphen check**: move (or duplicate) this check
   to run against the final tag computed in `discoverInto`
   (`engine.go:305-317`) / `registerPathLocked` (`engine.go:395-407`) — §4.7
   item 2.

### `unionfs.go` (new file)

1. A small internal `fs.FS` implementation mounting N source filesystems at
   distinct path prefixes, implementing `Open`, `Stat` (`fs.StatFS`), and
   `ReadDir` (`fs.ReadDirFS`) — the exact minimal surface `discoverInto`
   requires, formalized as tested internal code instead of leaving every
   caller to reinvent it (§4.1, Option A vs. B).

### `renderer.go`

1. **`isComponentLike` call site** (`renderer.go:1172-1174`): consult
   `Renderer`'s native-elements set before erroring; new
   `Renderer.nativeElements map[string]bool` field populated from
   `Options.NativeElements` via a new `WithNativeElements` builder method,
   mirroring how `WithDirectives` (`renderer.go:226-232`) is threaded today.
2. **`isBooleanAttr` call sites** (`renderer.go:1241`, `renderer.go:2078`):
   replace the fixed-allowlist check with the general falsy-omission rule
   from §4.4 (`attrShouldOmit`). `isBooleanAttr` itself can remain as a
   narrower helper if still needed elsewhere, or be removed if these two call
   sites were its only callers — confirm during implementation.

### `component.go`

1. **`selfClosingComponentRe`** (`component.go:475-477`): extend the regex
   (or add a second regex + rewrite pass) to also match self-closing
   `<component ...\/>` (lowercase, exact tag name `component`), per §4.7 item
   1. Verify this does not also incorrectly rewrite genuinely void-like
   custom usages — `component` is never a void HTML element, so this is safe
   by construction.

### New package: `hypermedia/` (§4.6)

1. `hypermedia/htmx.go`: `IsHTMXRequest`, `IsBoosted`, `SetTrigger`, and
   similar small header-based helpers, each a direct wrapper over one
   documented htmx request/response header.
2. `hypermedia/turbo.go`: `TurboStreamContentType` constant,
   `WantsTurboStream`, `TurboFrameID`, `WriteTurboStream`.
3. `hypermedia/datastar.go`: either a thin re-export/helper layered on
   `github.com/starfederation/datastar-go`, or (if a hard dependency on that
   module is undesirable for `htmlc`'s own `go.mod`) documentation-only
   guidance showing the pattern from Example 6 — resolve as part of Open
   Question 9 (§10).
4. This package has its own `go.mod`-relative import path but should be
   released and versioned alongside the main module, matching the
   already-established `cmd/htmlc` sub-tooling pattern in this repo.

### Notes

- All new `Options` fields default to their zero value (`nil`), which is a
  strict no-op for every existing project — `Mounts: nil` skips the union-FS
  path entirely, `NativeElements: nil` preserves today's
  hard-error-on-unresolved-hyphenated-tag behavior exactly.
- `fs.FS` path handling continues to use `path` (forward slashes), not
  `filepath`, for anything touching FS-relative paths inside the new
  `unionfs.go`, per the same OS-portability note RFC 001 §10 already raised
  for `nsEntries`.

---

## 8. Backward Compatibility

### `Options` struct

Fully additive — `Mounts` and `NativeElements` are new fields defaulting to
`nil`/empty. No existing field changes meaning or type.

### `Engine.Register(name, path string) error`

Unchanged signature and semantics. Still resolves through the engine's own
(possibly union) `opts.FS` — since a union `fs.FS` is a valid `fs.FS`, no
change is needed here at all; `Register` automatically gains the ability to
reach mounted paths once `Mounts` is set, at no cost to callers who never set
it.

### `RenderFragment`/`RenderPage`

Unchanged. `RenderFragmentWithStyles` is a new, additive method; it does not
replace or deprecate the existing calls, which remain the simpler default
for the common case of a single, isolated render.

### `isBooleanAttr` / falsy-attribute rendering (§4.4)

This is the one change in this RFC with a real, non-trivial compatibility
question: the README's "v-bind notes" section (README.md:205-226) explicitly
documents "**Boolean attributes** (`disabled`, `checked`, `selected`,
`readonly`, `required`, `multiple`, `autofocus`, `open`) are omitted entirely
when the bound value is falsy" as a closed list — it does not merely omit
mentioning other attributes, it enumerates exactly these eight. Today,
`:some-attr="false"` for any attribute **outside** that list renders
`some-attr="false"` literally, and that is documented, not accidental,
behavior; after this change, the attribute would be omitted instead for
every attribute name. This is therefore a **documented-behavior change**,
not a silent-gray-area bug fix, and the README's own "Spread HTMX attributes
from a map" example (README.md:214-220, `v-bind="actions.delete.hxAttrs"`)
shows the exact scenario this RFC's motivation is built around already being
anticipated as a use case — strengthening the case for the change, but not
eliminating the compatibility question. This RFC recommends making the
change (matching real Vue.js semantics, and unblocking the hypermedia
attribute idiom this section exists to fix) but flags it explicitly as Open
Question 4 (§10), including whether it should ship behind a compatibility
flag, for reviewer sign-off before implementation.

### `ValidateAll`

Signature unchanged (`func (e *Engine) ValidateAll() []error`). The new
cross-mount collision check only activates when `Options.Mounts` is
non-empty, so no existing single-source project's `ValidateAll()` output
changes.

### Self-closing `<component>` fix (§4.7 item 1)

Strictly a correctness fix: today, self-closing `<component is="..." />`
silently corrupts the rest of the page. After the fix, it renders correctly.
There is no prior "intended" behavior this could break — any project
currently working around the bug (e.g. by always using the explicit
`<component>...</component>` form) is unaffected either way.

### Custom-element hyphen-check ordering fix (§4.7 item 2)

Also strictly a correctness fix: this only causes a *new* warning to be
surfaced by `ValidateAll()`/build-time validation for a component whose
final custom-element tag is invalid — a case that was previously silently
passing validation while producing broken output. No project relying on the
current silent-pass behavior is depending on anything the spec allows.

### New `hypermedia` subpackage

Entirely new, opt-in package; zero impact on the core `htmlc` module or any
project not importing it.

---

## 9. Alternatives Considered

### Build-time vendoring (`htmlc vendor` copies `.vue` files into the project tree)

Instead of a runtime `fs.FS` union, a new CLI subcommand could copy a
vendored library's `.vue` files into the consumer's own `ComponentDir` at
build time (analogous to `go mod vendor`), after which everything is one flat
tree with no runtime multi-source complexity at all.

**Rejected as the primary mechanism.** This trades away live updates when
using an `embed.FS`-backed library during development (`Reload: true` would
have no effect on vendored-and-copied files without a separate re-vendor
step), duplicates the library's source across every consuming project
(harder to tell "did I get a security fix from the upstream library" without
diffing vendored trees), and doesn't remove the underlying collision-naming
problem in §4.2 — it just changes where the collision is discovered (at
vendor-time file copy versus at `ValidateAll()` time). It may still be a
reasonable **complementary** option for projects that specifically want a
single frozen tree with no runtime `fs.FS` composition (e.g. producing a
fully static, dependency-free build artifact) — worth a narrower follow-up
RFC if there's demand, but not needed to solve the problem this RFC targets.

### `Registry` as a first-class interface

Considered in §4.1 (Option C) as a way to let a Go package hand the engine
already-constructed components with no filesystem round-trip at all —
rejected for this RFC because it requires a much larger change (a
constructible/interface-based `Component` abstraction where today only
`ParseFile`-on-real-source exists) than the actual problem (distributing
`.vue` file trees) requires. Left as a candidate for a future RFC.

### Auto-detecting native custom elements instead of an explicit allowlist

Considered in §4.3 (Option B) — rejected because it silently downgrades a
real class of authoring mistakes (a misspelled or forgotten component
reference) into inert, unstyled markup instead of a loud render error,
trading a debuggable failure for a much harder-to-notice one.

### Extending the fixed boolean-attribute list instead of a general falsy rule

Considered in §4.4 (Option B) — rejected because it is perpetually
incomplete: every new hypermedia library, every project's own custom
`data-*` attribute vocabulary, would need its own addition to a hardcoded
list in `htmlc` itself, rather than being solved once by matching real
Vue.js's general semantics.

### Building the hypermedia helpers as documentation only, no shipped code

Considered in §4.6 (Option B) — rejected as the *sole* answer because it
reproduces the exact class of easy-to-get-wrong details (precise header
names, the exact `text/vnd.turbo-stream.html` string) in every consuming
project independently, but retained as a **complement**: the Examples (§6)
and any docs-branch tutorial should still show the underlying pattern in
full, not just call the helper, so a project using a different or future
hypermedia library isn't blocked on an `htmlc` release.

---

## 10. Open Questions

1. **`Mount` field naming and shape** (blocking): is `Prefix`/`FS`/`Dir` the
   right shape, or should `Dir` be implied by `Prefix` (i.e. always scan the
   whole mounted `fs.FS` from its root, with no separate `Dir` sub-path)?
   Recommendation: keep `Dir` separate from `Prefix` — a vendored package's
   `embed.FS` commonly embeds a directory tree with a non-trivial root (e.g.
   `//go:embed components`), and conflating "where in the mounted FS to
   start" with "what prefix consumers use to address it" would force every
   package author to structure their embed exactly at FS-root.
2. **Collision severity** (blocking): should `ValidateAll()` return a hard
   `[]error` for a cross-mount collision (consistent with its existing
   "startup-time fail fast" contract, per README's "Validate components at
   startup" section), or should collisions instead be reported via
   `Options.Logger` as warnings, leaving the (fixed, proximity-first)
   resolution to proceed? Recommendation: hard error via `ValidateAll()` —
   this mirrors the existing contract exactly and a caller can always choose
   to log-and-continue rather than `os.Exit` on the returned errors, but a
   silent-by-default posture would reproduce exactly the failure mode this
   RFC exists to close.
3. **`v-native` per-element escape hatch** (non-blocking): should §4.3
   Option C (a template-level attribute marking one element as native,
   independent of any project-wide `NativeElements` list) be added as a
   complementary, finer-grained mechanism alongside the allowlist? Useful for
   a one-off foreign tag that doesn't warrant a permanent `Options` entry.
   Can be addressed in a follow-up change without disturbing this RFC's core
   design.
4. **Falsy-attribute-omission compatibility flag** (non-blocking, see §8):
   should the §4.4 behavior change ship unconditionally as a bug fix, or
   behind an `Options.LegacyAttrStringify bool` (default `false`, i.e. new
   behavior) for the rare project that has come to depend on the current
   stringify-everything-outside-eight-attributes behavior? Recommendation:
   ship unconditionally — the current behavior was never documented as
   intentional and matches no known real use case — but this is flagged
   explicitly for reviewer sign-off before implementation, since it is this
   RFC's only change with any observable-output difference for existing
   templates.
5. **Should `StyleCollector` reuse (§4.5) be unified with
   `CustomElementCollector` reuse into one "session" object**, since a
   long-lived SSE connection plausibly wants both style and script
   deduplication together, and `Engine.RenderWithCollector` already threads a
   `*CustomElementCollector` through per-call? Recommendation: yes in
   spirit — consider a combined `RenderFragmentWithSession(ctx, w, name,
   data, sess *RenderSession)` wrapping both collectors, with
   `RenderFragmentWithStyles` either becoming sugar over it or being
   subsumed entirely. Left open for implementation-time design rather than
   fixed here, since it doesn't change any of this RFC's public
   compatibility guarantees either way.
6. **Should a component-package author be able to express an intentional
   same-name override** (non-blocking) — e.g. "yes, I know `radix/Accordion`
   and my local `Accordion` share a name; prefer the local one by default
   without requiring every call site to use `<component is="/Accordion">`
   explicitly"? Not addressed by this RFC; explicit-path addressing (already
   existing syntax, fixed per §4.7) is the only disambiguation mechanism
   proposed. A convenience default-preference mechanism could be a narrow
   follow-up if this proves to be common friction in practice.
7. **Should external directives (the NDJSON subprocess protocol) gain a
   request-time, not just build-time, story** (blocking scope decision, but
   deferred to a follow-up RFC rather than blocking this one): §1.1 notes
   this mechanism is wired up only inside `cmd/htmlc/build_command.go` and
   has no path for a Go host application calling `htmlc.New` directly. This
   RFC's §4.6 hypermedia helpers work entirely within the existing
   `Render*`/`ServeComponent` API and do not require solving this, but a
   future RFC extending out-of-process directive execution to
   library-embedded, request-serving `htmlc` usage (e.g. for a third-party
   syntax highlighter or a hypermedia-protocol directive implemented in a
   language other than Go) is a natural next step this RFC's design does not
   preclude.
8. **Should `hypermedia.TurboFrameID`/`WantsTurboStream` also validate that
   the server's response actually contains a matching `<turbo-frame
   id="...">`** (non-blocking) — i.e. should the helper package catch the
   "response doesn't match Turbo Frame's expectations" class of bug at the
   `htmlc` layer, or is that left entirely to the author, matching Turbo's
   own "no special server validation" contract? Recommendation: leave to the
   author for this RFC — implementing response-shape validation would
   require buffering and re-parsing the rendered output, which conflicts
   with the streaming-friendly design elsewhere in this RFC (§4.5, §6 Example
   6). Revisit only if this proves to be a common source of bugs in
   practice.
9. **Should the `hypermedia` subpackage take a hard dependency on
   `github.com/starfederation/datastar-go`, or stay documentation-only for
   the Datastar portion specifically** (blocking, scoped to §4.6/§7's
   `hypermedia/datastar.go`): htmx and Turbo need no third-party dependency
   (they're just header/content-type conventions), but a genuinely useful
   Datastar helper likely means depending on the official SDK, whose release
   cadence `htmlc` would not control. Recommendation: ship
   `hypermedia/htmx.go` and `hypermedia/turbo.go` as proposed, but treat
   `hypermedia/datastar.go` as documentation/example code only (as shown in
   §6 Example 6) rather than a dependency-carrying package, until there's
   clear demand for a maintained wrapper.
