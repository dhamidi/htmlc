# RFC 005: Structured Logging with `log/slog`

- **Status**: Accepted
- **Date**: 2026-03-16
- **Author**: TBD

---

## 1. Motivation

`htmlc` renders a tree of Vue-style components server-side in Go. In
production, when a render is unexpectedly slow or produces unexpectedly large
output, there is no structured log signal that records which component was
responsible, how long it took, or how many bytes it produced. The existing
`Debug` mode writes HTML comments into the rendered output — useful during
development but unreadable by log aggregators and unsuitable for production.

### The production observability gap

Consider a page that renders slowly in production but not in development. The
HTTP request log shows `"GET /"` taking 800ms. Without structured logging,
there is no way to determine which component in the tree is responsible for
the slowdown without adding ad-hoc `log.Printf` calls around the render call
site, which cannot see inside `htmlc`'s recursive render loop. The output is
correct, the error is nil, and nothing in the existing tooling points at
the bottleneck.

This failure mode is **silent and hard to attribute**: the HTML output is
correct, no error is returned, but there is no structured signal indicating
which component rendered slowly or produced unexpectedly large output.

### Why debug mode is not the answer

The `Debug` option (`engine.go`) writes HTML comments such as
`<!-- component=ProductList file=templates/ProductList.vue -->` directly into
the rendered output. This is incompatible with production use: it inflates
response size, may break parsers that do not expect comments in specific
locations, and cannot be routed to a log aggregator. It provides no timing or
byte-count information.

### Why HTTP middleware is not the answer

HTTP middleware can time the entire request but cannot observe what happens
inside `htmlc`'s recursive render loop. Components are `.vue` files, not Go
code, so manual instrumentation of individual components is impossible. The
only place to instrument component dispatch is inside the renderer itself.

---

## 2. Goals

1. Accept a `*slog.Logger` in `Options` and use it to emit one structured log
   record per component rendered, automatically covering the full component
   tree.
2. Each log record includes: component name, render duration (subtree), bytes
   written (subtree), and error (if any).
3. Introduce **no new external dependencies** in the `htmlc` core module —
   `log/slog` is part of the standard library since Go 1.21.
4. Users who do not set a logger pay **zero cost**: no allocations and no
   extra work when `Logger` is nil.
5. All log calls receive the context in effect at that point in the component
   tree, enabling slog handlers that extract trace IDs, request IDs, or other
   context values.
6. A `nil` logger disables all slog output without any behaviour change; a
   `slog.Default()` logger routes records to the process-wide default handler.

---

## 3. Non-Goals

1. **Distributed tracing** — span creation and cross-service propagation are
   out of scope for this RFC; use an OTel integration directly.
2. **Aggregate performance counters** — total render counts, cumulative
   latency, and similar aggregate metrics belong in RFC 003 (expvar
   integration).
3. **Expression evaluation logging** — too granular; the existing debug mode
   covers this use case.
4. **`v-for` iteration logging** — each loop iteration is not a component
   boundary; per-iteration records would be excessively noisy.
5. **Slot render logging** — slots are rendered as part of the calling
   component and do not constitute a separate dispatch.
6. **Log level configuration on the engine** — callers configure their slog
   handler level; the engine always uses `slog.LevelDebug` for normal records
   and `slog.LevelError` for failures.
7. **Replacing the existing `Debug` mode** — HTML-comment debug mode serves a
   different use case (in-browser, development-time inspection) and is not
   removed.

---

## 4. Proposed Design

### 4.1 New field in `Options`

```go
// pseudo-code — not implementation
type Options struct {
    ComponentDir string
    FS           fs.FS
    Reload       bool
    Debug        bool
    Directives   DirectiveRegistry

    // Logger, if non-nil, receives one structured log record per component
    // rendered. Records are emitted at slog.LevelDebug for successful renders
    // and slog.LevelError for failed renders. Each record includes the
    // component name, render duration (subtree), bytes written (subtree), and
    // any error. The nil value (default) disables all slog output.
    Logger *slog.Logger
}
```

`log/slog` is part of the Go standard library since Go 1.21. `Logger` is
independent of any hook field and adds no external dependencies to the module.

### 4.2 New field in `Renderer` and builder method

```go
// pseudo-code — not implementation
type Renderer struct {
    // ... existing fields ...
    logger *slog.Logger   // new field
    cw     countingWriter // new field — reused across component dispatches
}

func (r *Renderer) WithLogger(l *slog.Logger) *Renderer {
    r.logger = l
    return r
}
```

The `Engine` calls `WithLogger(opts.Logger)` when constructing the root
renderer, alongside the existing `WithContext`, `WithComponents`, and similar
calls. Because `rendererWithComponent` (`renderer.go:1681`) copies the
`Renderer` by value, the logger propagates to all child renderers
automatically with no additional wiring.

### 4.3 Measuring output size with a counting writer

To record bytes written per component, the renderer wraps the `io.Writer`
passed to each component's render call with a `countingWriter` when a logger
is set:

```go
// pseudo-code — not implementation
type countingWriter struct {
    w io.Writer
    n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
    n, err := cw.w.Write(p)
    cw.n += int64(n)
    return n, err
}

// Reset reinitialises the counter and sets the underlying writer.
// Calling Reset before each component dispatch avoids allocating a new
// countingWriter on every logged render.
func (cw *countingWriter) Reset(w io.Writer) {
    cw.w = w
    cw.n = 0
}
```

This type is private. It is embedded as a value field `cw countingWriter`
in `Renderer`, so it is allocated once as part of the `Renderer` struct
(which `rendererWithComponent` already creates for each child dispatch).
`Reset` reinitialises the counter and redirects writes to the new underlying
writer, so no additional heap allocation is needed per component when the
logger is non-nil.

### 4.4 Emitting the log record at component dispatch

**Current state**: `renderComponentElement` (`renderer.go:1535`) dispatches
to a child component by calling `r.rendererWithComponent(comp).Render(w, scope)`.

**Proposed change**: When `r.logger` is non-nil, wrap `w` in a
`countingWriter`, record `start := time.Now()`, run the existing render path,
then emit the log record using the renderer's context:

```go
// pseudo-code — not implementation
func (r *Renderer) renderComponentElement(w io.Writer, n *html.Node,
    scope map[string]any, comp *Component) error {

    if r.logger == nil {
        return r.rendererWithComponent(comp).Render(w, scope)
    }

    child := r.rendererWithComponent(comp)
    child.cw.Reset(w)
    start := time.Now()
    renderErr := child.Render(&child.cw, scope)
    elapsed := time.Since(start)

    if renderErr != nil {
        r.logger.ErrorContext(r.ctx, "component render failed",
            slog.String("component", comp.Name),
            slog.Duration("duration", elapsed),
            slog.Int64("bytes", child.cw.n),
            slog.Any("error", renderErr),
        )
    } else {
        r.logger.DebugContext(r.ctx, "component rendered",
            slog.String("component", comp.Name),
            slog.Duration("duration", elapsed),
            slog.Int64("bytes", child.cw.n),
        )
    }

    return renderErr
}
```

The `if r.logger == nil` guard is a single pointer comparison and adds no
measurable overhead on the no-logger path.

### 4.5 Logging the root component

The root component — the one named in `RenderPageContext` or
`RenderFragmentContext` — must also be logged so the entire render tree is
visible, not only the child dispatches.

**Current state**: `RenderPageContext` (`engine.go:650`) and
`RenderFragmentContext` (`engine.go:695`) construct a `Renderer` and call
`Render` directly; they do not go through `renderComponentElement`.

**Proposed change**: Extract a private `loggedRender` helper that wraps a
root render call with slog instrumentation, symmetrically to the child path:

```go
// pseudo-code — not implementation
func (e *Engine) loggedRender(
    ctx context.Context,
    name string,
    w io.Writer,
    render func(io.Writer) error,
) error {
    if e.opts.Logger == nil {
        return render(w)
    }

    e.cw.Reset(w)
    start := time.Now()
    renderErr := render(&e.cw)
    elapsed := time.Since(start)

    if renderErr != nil {
        e.opts.Logger.ErrorContext(ctx, "component render failed",
            slog.String("component", name),
            slog.Duration("duration", elapsed),
            slog.Int64("bytes", e.cw.n),
            slog.Any("error", renderErr),
        )
    } else {
        e.opts.Logger.DebugContext(ctx, "component rendered",
            slog.String("component", name),
            slog.Duration("duration", elapsed),
            slog.Int64("bytes", e.cw.n),
        )
    }

    return renderErr
}
```

`Engine` embeds a `cw countingWriter` field for root-level logging. Because
`RenderPageContext` and `RenderFragmentContext` are not called concurrently
on the same `Engine` with shared output (each call supplies its own `w`),
the single embedded field is safe for sequential use. If concurrent root
renders are needed, callers must use separate `Engine` instances.

`RenderPageContext` and `RenderFragmentContext` call `loggedRender`, passing
a closure that builds the renderer and calls `Render`. This avoids
duplicating the nil-check at every entry point.

### 4.6 Log attributes and record ordering

Records appear in **post-order** (leaf components first, root last) because
the log call is emitted after the component subtree finishes rendering. This
ordering is a natural consequence of the recursive descent and requires no
special handling.

The `bytes` and `duration` attributes measure the **subtree** rooted at the
logged component — not the component's direct output alone. A `ProductList`
that renders ten `ProductCard` children will report the combined bytes and
duration of the entire subtree. This is the most actionable metric for
identifying large or slow output subtrees.

All log records emitted by `htmlc` use the following fixed attribute keys:

| Key | `slog` constructor | Type | Description |
|---|---|---|---|
| `component` | `slog.String` | `string` | Resolved component name |
| `duration` | `slog.Duration` | `time.Duration` | Subtree render duration |
| `bytes` | `slog.Int64` | `int64` | Subtree bytes written |
| `error` | `slog.Any` | `error` | Non-nil only on `LevelError` records |

Using `time.Duration` (nanoseconds as `int64`) allows slog handlers to
format durations according to their own preferences: nanoseconds in JSON,
human-readable in text.

### 4.7 Design options for output size measurement

Three approaches were evaluated:

| Option | Approach | ✅ Pros | ❌ Cons |
|---|---|---|---|
| A (chosen) | `countingWriter` wrapping `w` at dispatch | Exact subtree byte count; no API changes | No extra allocation: `cw` is embedded in `Renderer`; `Reset()` reinitialises it at zero additional cost. |
| B | Return `(int64, error)` from `Render` | No allocation; byte counts propagate up | Breaking change to the `Render` signature |
| C | Measure at the HTTP response writer | Zero changes to htmlc | Cannot attribute bytes to individual components |

**Verdict**: Option A. The heap allocation per component is acceptable on an
observability path that is off by default. Option B is a breaking API change.
Option C cannot attribute output to individual components, which is the
primary use case.

---

## 5. Syntax Summary

No new template syntax is introduced. The logger is configured entirely in Go
at engine construction time.

| Go API | Description |
|---|---|
| `Options.Logger` | `*slog.Logger` to receive per-component render records |
| `Renderer.WithLogger(l *slog.Logger) *Renderer` | Set logger on a standalone `Renderer` (low-level API) |
| Log message `"component rendered"` | Emitted at `slog.LevelDebug` on success |
| Log message `"component render failed"` | Emitted at `slog.LevelError` on failure |
| Attribute `component` | `string` — component name |
| Attribute `duration` | `time.Duration` — subtree render duration |
| Attribute `bytes` | `int64` — subtree bytes written |
| Attribute `error` | `error` — render error (error records only) |

---

## 6. Examples

### Example 1: Default slog handler

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Logger:       slog.Default(),
})
```

Log output (text handler, debug level enabled):

```text
time=2026-03-16T12:00:00.001Z level=DEBUG msg="component rendered" component=NavLink duration=1.2ms bytes=142
time=2026-03-16T12:00:00.001Z level=DEBUG msg="component rendered" component=NavLink duration=1.1ms bytes=142
time=2026-03-16T12:00:00.002Z level=DEBUG msg="component rendered" component=NavBar duration=4.5ms bytes=612
time=2026-03-16T12:00:00.018Z level=DEBUG msg="component rendered" component=ProductCard duration=6.8ms bytes=1024
...
time=2026-03-16T12:00:00.098Z level=DEBUG msg="component rendered" component=ProductList duration=95.3ms bytes=18432
time=2026-03-16T12:00:00.148Z level=DEBUG msg="component rendered" component=Shell duration=140.1ms bytes=22016
time=2026-03-16T12:00:00.149Z level=DEBUG msg="component rendered" component=HomePage duration=148.6ms bytes=24576
```

Records appear in post-order: leaf components are logged before the
components that contain them.

### Example 2: JSON handler for log aggregation

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Logger:       logger,
})
```

JSON output:

```json
{"time":"2026-03-16T12:00:00.001Z","level":"DEBUG","msg":"component rendered","component":"NavLink","duration":1200000,"bytes":142}
{"time":"2026-03-16T12:00:00.002Z","level":"DEBUG","msg":"component rendered","component":"NavBar","duration":4500000,"bytes":612}
{"time":"2026-03-16T12:00:00.149Z","level":"DEBUG","msg":"component rendered","component":"HomePage","duration":148600000,"bytes":24576}
```

(`duration` is nanoseconds as an integer in JSON — the standard
`time.Duration` representation.)

### Example 3: Request-scoped logger with request ID

```go
func handler(w http.ResponseWriter, r *http.Request) {
    reqLogger := slog.Default().With(
        "request_id", r.Header.Get("X-Request-ID"),
    )

    engine, _ := htmlc.New(htmlc.Options{
        ComponentDir: "templates/",
        Logger:       reqLogger,
    })

    engine.RenderPageContext(r.Context(), w, "HomePage", pageProps)
}
```

Every log record automatically carries `request_id` because the logger was
constructed with `.With(...)`. The context passed to `RenderPageContext` is
threaded through to all `DebugContext`/`ErrorContext` calls, enabling handlers
that extract additional values such as OpenTelemetry trace IDs.

### Example 4: Zero-cost path — no logger

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    // Logger not set — nil
})
```

`renderComponentElement` tests `r.logger != nil`; the test is false, so no
`countingWriter` is allocated, no timing is recorded, and no log record is
emitted. Behaviour is identical to the current version.

### Example 5: Verifying log output in unit tests

```go
func TestRendererEmitsLogRecords(t *testing.T) {
    var buf bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    engine, _ := htmlc.New(htmlc.Options{
        ComponentDir: "testdata/templates/",
        Logger:       logger,
    })

    engine.RenderFragmentString("Page", nil)

    if !strings.Contains(buf.String(), `"component":"Shell"`) {
        t.Errorf("expected log record for Shell, got: %s", buf.String())
    }
}
```

No external test dependencies are needed; `log/slog` and `bytes` are stdlib.

---

## 7. Implementation Sketch

### `engine.go`

- Add `Logger *slog.Logger` field to the `Options` struct (after `Directives`). One line.
- Add `cw countingWriter` value field to the `Engine` struct. One line.
- Add private `loggedRender(ctx, name, w, renderFn)` helper (~20 lines) that
  wraps a root render with slog instrumentation when `opts.Logger != nil`.
- Update `loggedRender` to call `e.cw.Reset(w)` and pass `&e.cw` instead of
  allocating a new `countingWriter`.
- In `RenderPageContext` and `RenderFragmentContext`: call
  `e.loggedRender(ctx, name, w, ...)` instead of directly constructing and
  invoking the renderer. Each change is a one-liner replacement.
- Pass `opts.Logger` to the root renderer via a new `WithLogger` call in the
  existing renderer setup chain (alongside `WithContext`, `WithComponents`,
  etc.). One line.

### `renderer.go`

- Add `logger *slog.Logger` field to the `Renderer` struct
  (`renderer.go:149`). One line.
- Add `cw countingWriter` value field to the `Renderer` struct (no pointer;
  embedded inline). One line.
- Add `WithLogger(l *slog.Logger) *Renderer` builder method (mirrors
  `WithContext`, `WithFuncs`, etc.). Three lines.
- Add private `countingWriter` struct with its `Write` method and a
  `Reset(w io.Writer)` method that zeroes `n` and sets `w` (~13 lines).
  Place in a small private helper section or a new `slog.go` file.
- In `renderComponentElement` (`renderer.go:1535`): add the `nil` check and
  the wrapped render+log block (~15 lines). Replace `cw := &countingWriter{w: w}`
  and the subsequent `r.rendererWithComponent(comp).Render(cw, scope)` with:
  ```go
  child := r.rendererWithComponent(comp)
  child.cw.Reset(w)
  renderErr = child.Render(&child.cw, scope)
  ```
  Access `child.cw.n` for the byte count. No net increase in allocations vs.
  the non-logging path.
- `rendererWithComponent` copies the `Renderer` by value, so `logger`
  propagates to child renderers automatically — no additional wiring.

### Tests

New or extended in `engine_test.go` or a new `slog_test.go`:

- `TestLoggerEmitsRecordPerComponent`: assert one debug record per component
  in a known tree.
- `TestLoggerRecordsBytes`: assert the `bytes` attribute is positive and
  larger for a component with children than for a leaf.
- `TestLoggerRecordsDuration`: assert the `duration` attribute is positive.
- `TestLoggerErrorRecord`: assert an error record is emitted at `LevelError`
  when a component fails to render.
- `TestLoggerNil`: assert a nil logger causes no panic and no behaviour change.
- `TestLoggerContextPropagation`: assert `DebugContext` receives the context
  passed to `RenderPageContext`.
- `TestLoggerPostOrder`: assert leaf components are logged before their parent.

No external test dependencies are needed. `log/slog`, `bytes`, and `strings`
are all standard library packages.

Platform note: all instrumented paths use component name strings (not
filesystem paths), so there are no `path` vs. `filepath` portability concerns.

---

## 8. Backward Compatibility

### `Options` struct

`Options` gains one new field `Logger *slog.Logger`. Adding a field to a
struct is backward-compatible for all callers that use named-field
initialisation (`Options{ComponentDir: "..."}`), which is the established
pattern in the test suite and all documentation examples. Callers using
positional struct literals (not present in this codebase) would break, but
that is a pre-existing concern unrelated to this RFC.

### `Renderer` struct

`Renderer` is exported but its fields are unexported. Adding a new unexported
field and a new `WithLogger` builder method is fully backward-compatible. The
`WithLogger` method follows the established `With*` convention already used
by `WithContext`, `WithFuncs`, and `WithComponents`.

### `RenderPage`, `RenderFragment`, `RenderPageContext`, `RenderFragmentContext`

Function signatures are unchanged. Behaviour is unchanged when `Logger` is
nil (the default). The `countingWriter` wrapper is only allocated when a
non-nil logger is set, so the hot path for applications that do not use
logging is entirely unaffected.

### Module dependencies

`log/slog` is part of the Go standard library since Go 1.21. No new entries
in `go.mod`. If `htmlc` currently declares a minimum Go version lower than
1.21, the `go` directive in `go.mod` must be bumped to `go 1.21`. This is the
only potential compatibility concern and must be verified before
implementation (see §10, question 1).

---

## 9. Alternatives Considered

### A. Wrap `ComponentHookFunc` as the slog integration point

Provide an `htmlcslog` subpackage that returns a `ComponentHookFunc` backed
by a `*slog.Logger`, using a hypothetical hook-based approach:

```go
// hypothetical
engine, _ := htmlc.New(htmlc.Options{
    OnComponent: htmlcslog.WithLogger(slog.Default()),
})
```

**Rejected as the primary surface**: The `ComponentHookFunc` signature
(`func(ctx, name) (ctx, func(error))`) does not receive the `io.Writer`, so
byte counting is impossible without changes to the hook type. Since `log/slog`
is a stdlib package (unlike OTel), there is no dependency-hygiene reason to
isolate it behind a hook. No `htmlcslog` package will be provided.

### B. Add an `io.Writer` parameter to `ComponentHookFunc`

Extend the hook signature to include the writer, enabling byte counting from
within a hook:

```go
// hypothetical
type ComponentHookFunc func(ctx context.Context, name string, w io.Writer) (context.Context, io.Writer, func(error))
```

**Rejected**: This would change the hook type signature and force all hook
implementations — including ones with no interest in counting bytes — to handle
a writer parameter they do not need.

### C. Log at `slog.LevelInfo` instead of `slog.LevelDebug`

Emit component render records at `slog.LevelInfo` so they appear by default
without configuring the handler level.

**Rejected**: A page with 20 components emits 20 records per request. In
production under load this would produce thousands of records per second,
making the application logs unusable. `slog.LevelDebug` requires explicit
opt-in from the operator, which is appropriate for per-component granularity.

### D. One aggregate record per `RenderPageContext` call

Emit a single log record per top-level render call summarising total duration
and bytes, without per-component breakdown.

**Rejected**: The primary use case is identifying which component in the tree
is slow or unexpectedly large. A single aggregate record cannot attribute the
bottleneck to a specific component, which is exactly the observability gap
described in §1.

### E. A `SetLogger` method on `Engine` instead of an `Options` field

Provide `engine.SetLogger(l *slog.Logger)` as a mutable post-construction
setter.

**Rejected**: `Engine` is configured through `Options` at construction time —
this is the established pattern throughout the codebase. A mutable setter
would require synchronisation for concurrent use (renders can run concurrently
on the same engine) and diverges from the existing convention.

---

## 10. Open Questions

1. **Minimum Go version** (blocking before implementation)
   `log/slog` was stabilised in Go 1.21. Verify that `go.mod` declares
   `go 1.21` or higher before implementing; bump if necessary.

2. **Export log message strings as constants?** (non-blocking)
   Export `const MsgComponentRendered = "component rendered"` and
   `const MsgComponentFailed = "component render failed"` for testability.
   Callers can filter by message without hardcoding strings. Tentative answer:
   export both.

3. **Subtree bytes vs. direct bytes only** — *decided: count subtree bytes.*
   The `bytes` attribute measures the entire subtree rooted at the logged
   component. This is the most actionable metric for identifying large output
   regions and is a natural consequence of wrapping `w` in a `countingWriter`
   before passing it to `Render`.

4. **`reloaded=true` log attribute** — *decided: not added.*
   Reload state is out of scope for this RFC. Reload counts are tracked by
   RFC-003 (expvar). Adding this attribute would require threading reload state
   through the renderer for a marginal benefit.
