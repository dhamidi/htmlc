# RFC 004: OpenTelemetry Per-Component Tracing

- **Status**: Rejected
- **Rejection reason**: Too much complexity, can be served by slog / otel integrations
- **Date**: 2026-03-15
- **Author**: TBD

---

## 1. Motivation

`htmlc` renders a tree of Vue-style components server-side in Go. Each call to
`RenderPageContext` or `RenderFragmentContext` descends recursively through the
component tree: `HomePage` renders `Shell`, which renders `NavBar`, `ProductList`,
and so on. Applications that instrument their HTTP layer with `otelhttp` already
receive a single span for the entire HTTP handler, but there is **no visibility
inside the component tree** — which components were rendered, how long each took,
or which ones failed.

### The observability gap

Consider a slow page render. The `otelhttp` span shows `"GET /"` taking 450ms.
Without component-level tracing, there is no way to know whether the bottleneck
is in `ProductList` (fetching dozens of rows), `Shell` (expensive style
computation), or a deeply-nested `Analytics` component that calls an external
service. The operator must instrument the application code, not the template layer.

This failure mode is **silent and hard to attribute**: the HTML output is
correct, the HTTP span is present, but there is no structured signal indicating
which part of the template tree is responsible for the latency.

### Why middleware is not the answer

HTTP middleware (`otelhttp`) wraps the handler boundary. It cannot observe what
happens inside `htmlc`'s recursive render loop. Adding manual timing code to
individual components in Go is impossible — components are `.vue` files, not Go
code. The only place to instrument component dispatch is inside the renderer
itself.

---

## 2. Goals

1. Emit one OTel span per component rendered, automatically nested to mirror the
   component tree depth.
2. Introduce **no import of any OTel package** in the `htmlc` core module — the
   core must remain dependency-free.
3. Users who do not use OTel pay **zero cost**: no allocations and no extra
   function calls when the hook is not set.
4. Span nesting is automatic via `context.Context` propagation; callers do not
   wire parent-child relationships manually.
5. Errors during component rendering are recorded on the corresponding span.
6. The integration surface mirrors `otelhttp` so it is immediately familiar to
   Go OTel users.

---

## 3. Non-Goals

1. **Metrics (histograms, counters)** — render duration histograms can be
   implemented by users via `ComponentHookFunc`; they are not needed in core.
2. **`v-for` iteration spans** — each loop iteration is not a component
   boundary; span-per-iteration would be too noisy.
3. **Slot render spans** — slots are rendered as part of the calling component,
   not as a separate component dispatch.
4. **Expression evaluation spans** — too granular; the existing debug mode
   addresses this use case.
5. **`WithDataMiddleware` spans** — this hook fires outside the component tree
   and belongs in HTTP middleware, not the renderer.
6. **Automatic OTel SDK setup** — callers initialise their own provider and pass
   it to `htmlcotel`; this RFC does not touch provider bootstrapping.

---

## 4. Proposed Design

### 4.1 New type: `ComponentHookFunc` (core, `hook.go`)

A single new exported function type carries the hook contract:

```go
// ComponentHookFunc is called by the renderer immediately before each component
// in the tree begins rendering.
//
// Parameters:
//   - ctx is the context in effect at that point in the component tree.
//     It already contains any span created by a parent component's hook call,
//     so span nesting is automatic.
//   - name is the resolved component name (e.g. "Shell", "NavBar", "ProductCard").
//
// Return values:
//   - The returned context MUST be passed to all child components rendered
//     within this component. This is what creates parent-child span relationships.
//     Returning the same ctx unchanged is valid but spans will appear as siblings.
//   - The returned function is called after the component finishes rendering.
//     The error argument is non-nil if the render failed; nil on success.
//     This function must always be called — use defer inside the renderer.
//     It must never be nil; return a no-op func if nothing needs to happen.
//
// The zero value (nil) is valid and means no hook is registered.
type ComponentHookFunc func(ctx context.Context, name string) (context.Context, func(error))
```

This type is placed in a new file `hook.go` in the core package. It has no
dependencies beyond `context` from the standard library.

### 4.2 New field in `Options`

```go
type Options struct {
    ComponentDir string
    FS           fs.FS
    Reload       bool
    Debug        bool
    Directives   DirectiveRegistry

    // OnComponent, if non-nil, is called before each component in the render
    // tree begins rendering. See ComponentHookFunc for the full contract.
    // The default (nil) disables component-level hooks entirely.
    OnComponent ComponentHookFunc
}
```

The `Engine` stores the hook and propagates it to every `Renderer` it creates,
using the existing `With*` builder pattern already present on `Renderer`.

### 4.3 Threading the hook through the renderer

**Current state**: `Renderer` (defined in `renderer.go:149`) holds a `ctx
context.Context` field. The `rendererWithComponent` method (`renderer.go:1681`)
copies the current renderer by value and sets a new `component`. This copy
propagates all fields — `ctx`, `registry`, `funcs`, etc. — to child renderers.

**Proposed extension**: Add a `hook ComponentHookFunc` field to `Renderer` and a
corresponding `WithHook` builder. Because `rendererWithComponent` copies by
value, the hook propagates to child renderers automatically with no additional
wiring.

```go
// pseudo-code — not implementation
type Renderer struct {
    // ... existing fields ...
    hook ComponentHookFunc // new field
}

func (r *Renderer) WithHook(h ComponentHookFunc) *Renderer {
    r.hook = h
    return r
}
```

`Engine` calls `WithHook(opts.OnComponent)` when constructing the root renderer,
alongside the existing `WithContext`, `WithComponents`, etc. calls.

### 4.4 Calling the hook at component dispatch

**Current state**: `renderComponentElement` (`renderer.go:1535`) is the single
code path where the renderer dispatches to a child component. It currently calls
`r.rendererWithComponent(comp)` to build a child renderer and then invokes the
child's `Render` method.

**Proposed change**: Fire the hook immediately before the child render begins,
replace `r.ctx` with the context returned by the hook, and call the done
function via `defer`. The child renderer (obtained from `rendererWithComponent`)
must receive the post-hook context, not the pre-hook context.

```go
// pseudo-code — not implementation
func (r *Renderer) renderComponentElement(w io.Writer, n *html.Node, scope map[string]any, comp *Component) error {
    var renderErr error

    if r.hook != nil {
        ctx := r.ctx
        if ctx == nil {
            ctx = context.Background()
        }
        var done func(error)
        ctx, done = r.hook(ctx, comp.Name)
        defer func() { done(renderErr) }()
        r = r.rendererWithComponent(nil) // shallow copy
        r.ctx = ctx
    }

    // ... existing prop-collection and child render logic, now using r.ctx ...
    renderErr = r.rendererWithComponent(comp).Render(w, childScope)
    return renderErr
}
```

The nil-check (`if r.hook != nil`) is a single branch prediction and adds no
measurable overhead on the unhook path.

**Verdict**: This is the minimal change. No new abstraction layers, no interface,
no allocation when the hook is nil.

### 4.5 Calling the hook for the root component

The root component — the one named in `RenderPageContext` or
`RenderFragmentContext` — must also fire the hook so that the entire render tree
is bracketed by a span, linkable to the incoming `otelhttp` span.

**Current state**: `RenderPageContext` (`engine.go:650`) and
`RenderFragmentContext` (`engine.go:695`) construct a `Renderer` and call
`Render` directly; they do not go through `renderComponentElement`.

**Proposed change**: Extract a small helper that wraps the root render call with
the hook, symmetrically to the child dispatch path:

```go
// pseudo-code — not implementation
func (e *Engine) renderRoot(ctx context.Context, name string, render func(context.Context) error) error {
    if e.opts.OnComponent == nil {
        return render(ctx)
    }
    var renderErr error
    ctx, done := e.opts.OnComponent(ctx, name)
    defer func() { done(renderErr) }()
    renderErr = render(ctx)
    return renderErr
}
```

`RenderPageContext` and `RenderFragmentContext` call `renderRoot`, passing a
closure that builds the renderer and calls `Render`. This avoids duplicating the
nil-check at every entry point.

### 4.6 Design options considered for the hook signature

Three signatures were evaluated:

| Option | Signature | ✅ Pros | ❌ Cons |
|---|---|---|---|
| A (chosen) | `func(ctx, name) (ctx, func(error))` | Mirrors `otelhttp` middleware pattern; done func allows both success and error reporting | Caller must remember to `defer done` |
| B | `func(ctx, name) (ctx, func())` + separate error hook | Simpler done func | Two hooks to configure; error path is a second surface |
| C | Interface `ComponentObserver` | Extensible without signature change | Forces interface allocation on every component; overkill for one hook |

**Verdict**: Option A. The `func(error)` done-func is the established Go pattern
(see `trace.Span.End`, `pprof.Do`, `httptrace.WithClientTrace`). Option B
fragments the contract unnecessarily. Option C over-engineers for a single use
case; if additional hook data is needed later, a `ComponentHookContext` struct
can be introduced alongside `ComponentHookFunc` without removing the original.

---

## 5. Syntax Summary

No new template syntax is introduced. The hook is configured entirely in Go at
engine construction time.

| Go API | Description |
|---|---|
| `ComponentHookFunc` | Type alias for the hook function signature |
| `Options.OnComponent` | Field to attach a hook to an `Engine` |
| `htmlcotel.WithTracerProvider(tp)` | Returns a `ComponentHookFunc` backed by the given `TracerProvider` |
| `htmlcotel.WithGlobalTracer()` | Returns a `ComponentHookFunc` backed by the global OTel provider |
| `htmlcotel.ScopeName` | Instrumentation scope string (`"github.com/dhamidi/htmlc"`) |

---

## 6. Examples

### Example 1: OTel tracing with a custom provider

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    OnComponent:  htmlcotel.WithTracerProvider(otel.GetTracerProvider()),
})
```

```
GET /                           150ms   ← otelhttp
  └─ HomePage                   148ms   ← htmlcotel
       ├─ Shell                 140ms
       │    ├─ NavBar            12ms
       │    │    ├─ NavLink       2ms
       │    │    └─ NavLink       2ms
       │    └─ ProductList       95ms
       │         ├─ ProductCard   7ms
       │         └─ ...
       └─ Footer                  3ms
```

Span nesting is automatic: each hook call receives the context returned by its
parent's hook call, so OTel sees the correct parent-child chain without any
manual wiring.

### Example 2: Global tracer (convenience)

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    OnComponent:  htmlcotel.WithGlobalTracer(),
})
```

Equivalent to Example 1 when `otel.GetTracerProvider()` is the configured
provider. Useful for applications that initialise a single global provider at
startup.

### Example 3: Zero-cost path — no hook

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    // OnComponent not set — nil
})
```

`renderComponentElement` tests `r.hook != nil`; it is false, so the branch is
not taken and no hook-related code runs. No allocations occur.

### Example 4: Custom logging hook (no OTel dependency)

```go
loggingHook := func(ctx context.Context, name string) (context.Context, func(error)) {
    start := time.Now()
    return ctx, func(err error) {
        if err != nil {
            log.Printf("component %s failed in %s: %v", name, time.Since(start), err)
        } else {
            log.Printf("component %s rendered in %s", name, time.Since(start))
        }
    }
}

engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    OnComponent:  loggingHook,
})
```

`ComponentHookFunc` is a plain function type — no OTel import required for users
who only want logging or metrics.

### Example 5: Testing with a recording hook

```go
type callRecord struct {
    name string
    err  error
}

func recordingHook(records *[]callRecord) htmlc.ComponentHookFunc {
    return func(ctx context.Context, name string) (context.Context, func(error)) {
        return ctx, func(err error) {
            *records = append(*records, callRecord{name: name, err: err})
        }
    }
}

func TestOnComponentHook(t *testing.T) {
    var records []callRecord

    engine, _ := htmlc.New(htmlc.Options{
        ComponentDir: "testdata/templates/",
        OnComponent:  recordingHook(&records),
    })

    engine.RenderFragmentString("Page", nil)

    // Page renders Shell, which renders NavBar.
    assert.Equal(t, []callRecord{
        {name: "Page"},
        {name: "Shell"},
        {name: "NavBar"},
    }, records)
}
```

No OTel dependency is needed in tests of the core hook mechanism.

---

## 7. Implementation Sketch

### `htmlc` core

**`hook.go`** (new file):
- Declare `ComponentHookFunc` type with full godoc.

**`engine.go`**:
- Add `OnComponent ComponentHookFunc` field to `Options` struct (after `Directives`).
- In `RenderPageContext` and `RenderFragmentContext`, call `e.renderRoot(ctx,
  name, ...)` instead of directly constructing the renderer.
- Add private `renderRoot` helper (~10 lines) that wraps the root render with
  the hook when non-nil.
- Pass `opts.OnComponent` to the root renderer via a new `WithHook` call in the
  renderer setup chain (alongside existing `WithContext`, `WithComponents`, etc.).

**`renderer.go`**:
- Add `hook ComponentHookFunc` field to `Renderer` struct (`renderer.go:149`).
- Add `WithHook(h ComponentHookFunc) *Renderer` builder (mirrors `WithContext`,
  `WithFuncs`, etc.).
- In `renderComponentElement` (`renderer.go:1535`): add the nil-check + hook
  call + `defer done` block immediately after the existing debug-mode block.
  Pass the post-hook context to the child renderer via `r.ctx`.
- `rendererWithComponent` copies by value, so `hook` propagates automatically.

**Tests** (new or extended in `engine_test.go` or a new `hook_test.go`):
- `TestOnComponentHookFires`: assert hook is called for root and each child.
- `TestOnComponentHookOrder`: assert call order matches tree traversal.
- `TestOnComponentHookErrorPropagation`: assert done func receives non-nil error
  when a component fails.
- `TestOnComponentHookNil`: assert nil hook causes no panic and no behaviour
  change.

### `htmlcotel` submodule (new directory)

**`htmlcotel/go.mod`**:
```
module github.com/dhamidi/htmlc/htmlcotel

require (
    github.com/dhamidi/htmlc v0.x.x
    go.opentelemetry.io/otel v1.x.x
    go.opentelemetry.io/otel/trace v1.x.x
)
```

**`htmlcotel/otel.go`**:
- `const ScopeName = "github.com/dhamidi/htmlc"`
- `func WithTracerProvider(tp trace.TracerProvider) htmlc.ComponentHookFunc`
- `func WithGlobalTracer() htmlc.ComponentHookFunc`
- Private `makeHook(tracer trace.Tracer) htmlc.ComponentHookFunc`

**`htmlcotel/otel_test.go`**:
- Integration test using `go.opentelemetry.io/otel/trace/tracetest` in-memory exporter.
- Assert parent-child span relationships mirror the component tree.
- Assert error is recorded (`span.RecordError`) and status set to `codes.Error`
  when a component render returns an error.

Platform note: all file path operations inside the hook use the component name
string (not a filesystem path), so there are no `path` vs `filepath` concerns.

---

## 8. Backward Compatibility

### `Options` struct

`Options` gains one new field `OnComponent ComponentHookFunc`. In Go, adding a
field to a struct is backward-compatible for all callers that use named-field
initialisation (`Options{ComponentDir: "..."}`) — this is the existing usage
pattern throughout the test suite and documentation. Callers using positional
struct literals (not used in this codebase) would break, but that is a pre-
existing issue unrelated to this change.

### `Renderer` struct

`Renderer` is exported but its fields are unexported. Adding a new unexported
field is fully backward-compatible. The new `WithHook` builder follows the
established `With*` convention.

### `RenderPage`, `RenderFragment`, `RenderPageContext`, `RenderFragmentContext`

Function signatures are unchanged. Behaviour is unchanged when `OnComponent` is
nil (the default).

### Module graph

`github.com/dhamidi/htmlc` gains no new dependencies. Callers who do not import
`htmlcotel` are entirely unaffected. The `htmlcotel` submodule lives under a
separate `go.mod` and is never pulled in transitively by the core module.

---

## 9. Alternatives Considered

### A. OTel import directly in core

Add `go.opentelemetry.io/otel` as a direct dependency of `github.com/dhamidi/htmlc`
and emit spans from the renderer.

**Rejected**: Adds a non-trivial dependency tree (OTel SDK, exporters, etc.) to
every application that uses `htmlc`, even those that never use tracing. This
violates the Go convention for library authors and conflicts with Goal 2.

### B. Interface-based observer

Define a `ComponentObserver` interface instead of a function type:

```go
type ComponentObserver interface {
    BeforeComponent(ctx context.Context, name string) context.Context
    AfterComponent(name string, err error)
}
```

**Rejected**: Forces an interface allocation on every component dispatch even
when the observer is a struct with no state. The function type achieves the same
result with a single pointer comparison for the nil check. An interface also
makes it harder for users to compose multiple observers (they need a multiplexer
struct); function composition is simpler. If a richer surface is needed in the
future, a struct parameter can be introduced alongside the existing type.

### C. Separate `BeforeComponent` / `AfterComponent` hooks

Two separate `ComponentHookFunc` fields — one called before, one after — instead
of the single "start returns done" signature.

**Rejected**: The "start returns done" pattern is the established Go idiom for
paired lifecycle operations (see `trace.Tracer.Start`, `pprof.Do`,
`context.WithCancel`). It guarantees the before and after operations share the
same closure state without requiring a matching mechanism. Two separate hooks
would require callers to correlate them manually and could not share a span
handle without external state.

### D. Middleware wrapping `RenderPageContext`

Provide a function that wraps the engine's render methods rather than adding a
hook to `Options`:

```go
func TracedEngine(e *htmlc.Engine, tp trace.TracerProvider) *htmlc.Engine
```

**Rejected**: `Engine` is not an interface, so wrapping it would require either
making it an interface (a breaking change) or embedding it (leaking the inner
engine's untraced methods). The hook approach is less invasive and handles
child-component spans naturally without any wrapping at all.

---

## 10. Open Questions

1. **Should `htmlcotel` live in the same repository?** (non-blocking)
   Current plan: `htmlcotel/` is a subdirectory of the `htmlc` repo with its
   own `go.mod`, mirroring the `bridge` submodule pattern already used in this
   repo. Tentative answer: yes, co-locate for easier versioned releases.

2. **What `htmlc.component` attribute value should be used for anonymous or
   dynamically-resolved components?** (non-blocking)
   If the component name is a dynamic expression (e.g. `<component :is="...">`)
   the resolved name may differ from the tag name. The hook receives the
   resolved `comp.Name`, which is always the canonical name. No change needed.

3. **Should the hook also fire for `RegisterFunc`-registered render functions?**
   (non-blocking)
   `RegisterFunc` callbacks are called during expression evaluation, not at
   component dispatch. Firing the hook for them would require a different
   mechanism. Tentative answer: out of scope for this RFC; label as non-goal.

4. **Version pinning for `go.opentelemetry.io/otel` in `htmlcotel/go.mod`**
   (blocking before implementation)
   The minimum compatible OTel version must be chosen before `htmlcotel` can be
   published. Tentative recommendation: target the latest stable `v1.x` release
   at time of implementation; document the minimum in the submodule README.

5. **Should `WithHook` be exported on `Renderer`?** (non-blocking)
   The low-level `Renderer` API is already exported (`NewRenderer`, `WithContext`,
   `WithFuncs`, etc.) for callers who bypass `Engine`. Exporting `WithHook`
   is consistent with this pattern. Tentative answer: yes, export it.
