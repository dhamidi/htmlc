# RFC 010: Component Error Handling — In-Place Rendering and Structured Paths

- **Status**: Accepted
- **Date**: 2026-03-17
- **Author**: TBD

---

## 1. Motivation

When a component deep in the rendering tree fails, `htmlc` discards all
partial output and returns a plain error string to the caller. The HTTP
handler has no rendered HTML to return and falls back to a plain-text 500
response. Debugging is hard: the error names only the immediate failing
component, not the full path from the page root to the failure site.

### The failure mode

Consider a three-level component tree:

```text
templates/
├── HomePage.vue        (root)
│   └── <Layout>
├── Layout.vue
│   └── <Sidebar>
└── Sidebar.vue         (← fails here: unknown variable "items")
```

`Sidebar.vue` contains `{{ items.length }}` but `items` was never passed as
a prop. The current error message is:

```text
component "Sidebar": render templates/Sidebar.vue: expr "items.length": type error
```

And the response body is **empty** — `RenderPageContext` buffers all output
into a `strings.Builder` and only writes it to `w` if `renderComponent`
returns `nil`. When the error reaches the top level, the buffer is discarded:

```go
// engine.go (current)
func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error {
    var buf strings.Builder
    sc, err := e.renderComponent(ctx, &buf, name, data)
    if err != nil {
        return err  // ← buf discarded; w receives nothing
    }
    // ... write buf to w
}
```

### Why this is silent and hard to debug

The developer sees an empty browser window. There is no visual indication of
*where* in the page the failure occurred. The error string names `Sidebar`
but does not say that `Sidebar` was reached via `HomePage → Layout → Sidebar`.
In a large component tree with multiple uses of `Sidebar`, the path is
essential to pinpoint which invocation failed.

### Why existing tooling is not the answer

The existing `Debug` mode writes HTML comments into the output — but only
*before* an error occurs. Once the error aborts rendering, no output reaches
the browser. The `slog.Logger` (RFC 005) emits a structured error record but
does not restore the partial HTML or provide an in-page visual signal.
There is no existing hook for callers to supply an error-placeholder renderer.

---

## 2. Goals

1. **Structured component path**: `RenderError` carries `ComponentPath
   []string`, the full ordered list of component names from the page root to
   the failing component (e.g. `["HomePage", "Layout", "Sidebar"]`).
2. **In-place error rendering**: callers can register a
   `ComponentErrorHandler` that is called at the point of failure, writes an
   HTML placeholder into the output stream, and allows rendering to continue
   for the rest of the page.
3. **Partial output delivered**: when a `ComponentErrorHandler` is registered
   and handles all errors, the (partial) rendered page is written to `w`
   exactly as it would be for a successful render.
4. **Fatal errors remain fatal**: errors that occur outside a component
   dispatch boundary (component not found, context cancellation, top-level
   prop validation) still abort the render and return an error without writing
   to `w`.
5. **Zero-cost default path**: callers that do not register a
   `ComponentErrorHandler` see no behaviour change and pay no extra cost.
6. **Testable handler contract**: the `ComponentErrorHandler` type is a plain
   `func`, requiring no interface implementation, so test doubles are trivial
   to write.

---

## 3. Non-Goals

1. **Recovery from panics** — `recover()` is out of scope. Panics in
   expression evaluation or directive hooks indicate programming errors, not
   template errors, and should propagate normally.
2. **Multiple accumulated errors returned from `RenderPage`** — the API
   surface (`RenderPage(w, name, data) error`) is unchanged. Error
   accumulation is the caller's responsibility via the handler closure.
3. **Template-syntax opt-in** — no new `.vue` syntax (e.g. `v-fallback`) is
   introduced. The feature is entirely a Go API.
4. **Retrying failed components** — the handler receives a read-only view of
   the error; it cannot retry the render with different data.
5. **Slot rendering errors** — slot content is rendered in the context of the
   component that authored it. Slot errors are attributed to the authoring
   component, not a separate "slot" path segment, and are handled by the
   same mechanism without special casing.

---

## 4. Proposed Design

### 4.1 Extending `RenderError` with `ComponentPath`

**Current state** (`errors.go`):

```go
type RenderError struct {
    Component string
    Expr      string
    Wrapped   error
    Location  *SourceLocation
}
```

**Proposed extension**:

```go
// pseudo-code — not implementation
type RenderError struct {
    Component     string
    ComponentPath []string // NEW: ordered path from root to failing component
    Expr          string
    Wrapped       error
    Location      *SourceLocation
}
```

`ComponentPath` is populated at each component boundary as the error travels
up the call stack. The last element of `ComponentPath` is the component that
directly produced the error (matching `Component`); the first element is the
page root. Example:

```text
ComponentPath = ["HomePage", "Layout", "Sidebar"]
Component     = "Sidebar"
```

`RenderError.Error()` is updated to include the path when `len(ComponentPath)
> 1`:

```go
// pseudo-code — not implementation
func (e *RenderError) Error() string {
    path := ""
    if len(e.ComponentPath) > 1 {
        path = strings.Join(e.ComponentPath, " > ") + ": "
    }
    // ... existing formatting using path prefix
}
```

### 4.2 Path tracking in `Renderer`

**Current state** (`renderer.go`): `Renderer` has no field tracking its
position in the component tree. Child renderers are created by
`rendererWithComponent` which copies the struct by value.

**Proposed extension**:

```go
// pseudo-code — not implementation
type Renderer struct {
    // ... existing fields ...
    componentPath []string // NEW: path from root to this component
}
```

When `renderComponentElement` creates a child renderer, it appends the
component name to the path:

```go
// pseudo-code — not implementation
childPath := make([]string, len(r.componentPath)+1)
copy(childPath, r.componentPath)
childPath[len(childPath)-1] = componentName  // e.g. "Sidebar"

childRenderer := &Renderer{
    // ... existing fields ...
    componentPath: childPath,
}
```

The root renderer's `componentPath` is initialised to
`[]string{rootComponentName}` in `renderComponent` (`engine.go`).

### 4.3 Populating `ComponentPath` on errors

**Current state**: `renderComponentElement` wraps errors as:

```go
return fmt.Errorf("component %q: %w", n.Data, err)
```

This wrapping chain is readable in error messages but not programmatically
inspectable.

**Proposed change**: Replace the `fmt.Errorf` wrapping with a function that
sets or extends `ComponentPath` on the outbound `RenderError`:

```go
// pseudo-code — not implementation
func wrapComponentError(path []string, componentName string, err error) error {
    var re *RenderError
    if errors.As(err, &re) {
        // Prepend the current component path to the existing RenderError.
        // The path slice is already correct on the child renderer;
        // just ensure ComponentPath reflects the full ancestry.
        if len(re.ComponentPath) == 0 {
            re.ComponentPath = path
        }
        return re
    }
    // Non-RenderError: wrap in a new RenderError.
    return &RenderError{
        Component:     componentName,
        ComponentPath: path,
        Wrapped:       err,
    }
}
```

Called as: `return wrapComponentError(childRenderer.componentPath, n.Data, err)`

### 4.4 `ComponentErrorHandler` type and `Options` field

**Proposed new type** (`errors.go` or a new `handler.go`):

```go
// pseudo-code — not implementation

// ComponentErrorHandler is called when a child component fails to render.
// w is the writer at the failure site; path is the full component path from
// the page root to the failing component. err is the render error (always a
// *RenderError when originating from a template expression; may be any error
// when originating from a directive or missing-prop handler).
//
// Return nil to write the placeholder and continue rendering sibling nodes.
// Return a non-nil error to abort the entire render immediately.
type ComponentErrorHandler func(w io.Writer, path []string, err error) error
```

**Options extension** (`engine.go`):

```go
// pseudo-code — not implementation
type Options struct {
    ComponentDir         string
    FS                   fs.FS
    Reload               bool
    Debug                bool
    Directives           DirectiveRegistry
    Logger               *slog.Logger

    // ComponentErrorHandler, if non-nil, is called in place of aborting the
    // render when a child component fails. The handler writes an HTML
    // placeholder to w and returns nil to continue, or returns a non-nil error
    // to abort. When the handler returns nil for all failures, the partial page
    // (with placeholders) is written to the io.Writer passed to RenderPage.
    ComponentErrorHandler ComponentErrorHandler // NEW
}
```

**Renderer field and builder method** (`renderer.go`):

```go
// pseudo-code — not implementation
type Renderer struct {
    // ... existing fields ...
    componentPath        []string
    componentErrorHandler ComponentErrorHandler // NEW
}

// WithComponentErrorHandler sets the handler called when a child component
// fails to render. It follows the existing With* builder convention.
func (r *Renderer) WithComponentErrorHandler(h ComponentErrorHandler) *Renderer {
    r.componentErrorHandler = h
    return r
}
```

The handler propagates to child renderers automatically because
`rendererWithComponent` copies the `Renderer` struct by value.

### 4.5 Modified `renderComponentElement` dispatch

**Current state** (`renderer.go`, simplified):

```go
// pseudo-code — not implementation
if err := childRenderer.Render(w, childScope); err != nil {
    return fmt.Errorf("component %q: %w", n.Data, err)
}
```

**Proposed change**:

```go
// pseudo-code — not implementation
renderErr := childRenderer.Render(w, childScope)
if renderErr != nil {
    wrapped := wrapComponentError(childRenderer.componentPath, n.Data, renderErr)
    if r.componentErrorHandler != nil {
        return r.componentErrorHandler(w, childRenderer.componentPath, wrapped)
    }
    return wrapped
}
```

When the handler returns `nil`, execution continues to the next sibling node.
When the handler returns a non-nil error, that error is propagated immediately
(aborting the render). The handler is passed `childRenderer.componentPath` so
it can access the structured path without unwrapping the error.

The existing slog-logging branch (`renderer.go:1692–1712`) is preserved and
updated symmetrically — the `fmt.Errorf` wrapping is replaced with
`wrapComponentError`, and the handler invocation happens after the log record
is emitted:

```go
// pseudo-code — not implementation
if renderErr != nil {
    r.logger.ErrorContext(r.ctx, MsgComponentFailed, ...)
    wrapped := wrapComponentError(childRenderer.componentPath, n.Data, renderErr)
    if r.componentErrorHandler != nil {
        return r.componentErrorHandler(w, childRenderer.componentPath, wrapped)
    }
    return wrapped
}
```

### 4.6 Delivering partial output when the handler handles all errors

**Current state**: `RenderPageContext` (`engine.go`) buffers output in
`strings.Builder` and writes it to `w` only if `renderComponent` returns
`nil`. When an error is returned, the buffer is discarded.

**Proposed change**: `renderComponent` is unchanged. When all component errors
are handled by `ComponentErrorHandler` (the handler returns `nil` for each),
`renderComponent` returns `nil`, and `RenderPageContext` writes the buffer
(which now contains error placeholders) to `w` as normal. No change to
`RenderPageContext` or `RenderFragmentContext` is required.

This is a key insight: the handler is invoked *inside* the render loop at the
point of failure, writing the placeholder directly to the in-progress buffer.
If the handler returns `nil`, rendering continues and the buffer grows
normally. Only if the handler returns a non-nil error does `renderComponent`
fail and the buffer get discarded.

### 4.7 Built-in development helper

A convenience constructor for a commonly needed development handler:

```go
// pseudo-code — not implementation

// HTMLErrorHandler returns a ComponentErrorHandler that renders a visible
// <div> placeholder for each failed component. It is intended for development
// use. The generated markup uses the class "htmlc-error" for easy targeting
// with CSS. path and err are HTML-escaped before inclusion.
func HTMLErrorHandler() ComponentErrorHandler {
    return func(w io.Writer, path []string, err error) error {
        msg := html.EscapeString(err.Error())
        p   := html.EscapeString(strings.Join(path, " > "))
        fmt.Fprintf(w,
            `<div class="htmlc-error" data-path=%q>%s</div>`,
            p, msg,
        )
        return nil
    }
}
```

This helper is exported but optional. Callers that need different error
presentation (e.g. JSON for HTMX responses) implement their own handler.

### 4.8 Design options for handler invocation site

Three options were considered for where to invoke the handler:

| Option | Invocation site | ✅ Pros | ❌ Cons |
|---|---|---|---|
| A (chosen) | Inside `renderComponentElement`, after `childRenderer.Render` | Handler writes directly to the live buffer; correct DOM position | Requires threading handler through all Renderer copies |
| B | In `renderComponent` (engine level), wrapping the whole tree | Simple; no Renderer changes | Handler cannot write to the correct DOM position; writes after the whole page buffer |
| C | In `renderChildren`, wrapping the `renderNode` call | Finer granularity (catches non-component errors too) | Mixes component and non-component error handling; hard to scope the path |

**Verdict**: Option A. The handler must write to `w` at the exact DOM position
of the failed component. Only `renderComponentElement` has both the writer
and the component context at the right moment. Threading the handler through
`Renderer` copies is low-cost since `rendererWithComponent` already copies
the struct.

---

## 5. Syntax Summary

No new template (`.vue`) syntax is introduced. All new surfaces are Go API.

| Go API | Description |
|---|---|
| `RenderError.ComponentPath []string` | Ordered path from root to failing component |
| `ComponentErrorHandler` type | `func(w io.Writer, path []string, err error) error` |
| `Options.ComponentErrorHandler` | Handler registered on the engine |
| `Renderer.WithComponentErrorHandler(h) *Renderer` | Handler registered on a standalone `Renderer` |
| `HTMLErrorHandler() ComponentErrorHandler` | Built-in dev handler: renders a `<div class="htmlc-error">` |

---

## 6. Examples

### Example 1: Default behaviour unchanged (no handler)

```text
templates/
├── HomePage.vue
└── Sidebar.vue   ({{ items.length }} — items is undefined)
```

```go
engine, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
err := engine.RenderPage(w, "HomePage", nil)
// err != nil; w has received no bytes
// err.Error() == "HomePage > Sidebar: render templates/Sidebar.vue: expr \"items.length\": type error"
```

`ComponentPath` is `["HomePage", "Sidebar"]`. The error string includes the
path. `w` is empty (unchanged from today).

### Example 2: Development — in-place HTML placeholder

```go
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir:         "templates/",
    ComponentErrorHandler: htmlc.HTMLErrorHandler(),
})
err := engine.RenderPage(w, "HomePage", nil)
// err == nil
// w contains the page HTML with:
//   <div class="htmlc-error" data-path="HomePage &gt; Sidebar">
//     render templates/Sidebar.vue: expr "items.length": type error
//   </div>
// in the position where <Sidebar> would have appeared.
```

### Example 3: Caller accumulates errors for logging

```go
var renderErrs []error
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    ComponentErrorHandler: func(w io.Writer, path []string, err error) error {
        renderErrs = append(renderErrs, err)
        fmt.Fprintf(w, `<!-- render error: %s -->`, html.EscapeString(err.Error()))
        return nil
    },
})

err := engine.RenderPage(w, "HomePage", data)
if err != nil {
    // fatal error (component not found, context cancelled, etc.)
    http.Error(rw, "internal error", 500)
    return
}
if len(renderErrs) != 0 {
    slog.Warn("partial render", "errors", renderErrs)
}
// Page was written to w with HTML comments at each failure site.
```

### Example 4: Abort on the first error from a specific subtree

```go
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    ComponentErrorHandler: func(w io.Writer, path []string, err error) error {
        if path[len(path)-1] == "PaymentForm" {
            // PaymentForm errors are unrecoverable — abort the render.
            return fmt.Errorf("payment form unavailable: %w", err)
        }
        // All other component errors: render a placeholder and continue.
        fmt.Fprintf(w, `<p class="widget-error">Widget unavailable</p>`)
        return nil
    },
})
```

### Example 5: Programmatic path inspection with `errors.As`

```go
var rerr *htmlc.RenderError
err := engine.RenderPage(w, "HomePage", nil)
if errors.As(err, &rerr) {
    fmt.Println("failed component:", rerr.Component)
    fmt.Println("full path:", strings.Join(rerr.ComponentPath, " > "))
    // Output:
    // failed component: Sidebar
    // full path: HomePage > Layout > Sidebar
}
```

---

## 7. Implementation Sketch

### `errors.go`

1. Add `ComponentPath []string` field to `RenderError` (one line).
2. Update `RenderError.Error()` to prefix the path when
   `len(ComponentPath) > 1` (~5 lines).
3. Add the `ComponentErrorHandler` type definition (~4 lines doc + 1 line type).
4. Add `HTMLErrorHandler() ComponentErrorHandler` (~10 lines, uses
   `html.EscapeString` from stdlib `"html"`).
5. Add private `wrapComponentError(path []string, name string, err error) error`
   helper (~12 lines).

### `engine.go`

1. Add `ComponentErrorHandler ComponentErrorHandler` field to `Options`
   (one line).
2. In `renderComponent`, initialise the root renderer's `componentPath` to
   `[]string{name}` using a new `WithComponentPath` builder call (~1 builder
   method + 1 call site).
3. Pass `opts.ComponentErrorHandler` to the root renderer via a new
   `WithComponentErrorHandler` call in the existing chain (one line).

### `renderer.go`

1. Add `componentPath []string` field to `Renderer` struct (one line).
2. Add `componentErrorHandler ComponentErrorHandler` field to `Renderer`
   struct (one line).
3. Add `WithComponentErrorHandler(h ComponentErrorHandler) *Renderer` builder
   method (~3 lines, mirrors `WithContext`).
4. Add `WithComponentPath(path []string) *Renderer` builder method (~3 lines).
5. In `renderComponentElement`, replace the `fmt.Errorf("component %q: %w",
   n.Data, err)` call (both the logger and non-logger paths) with
   `wrapComponentError(childRenderer.componentPath, n.Data, err)` and add the
   handler invocation block (~8 lines delta per path, ~16 lines total).
6. In the child renderer construction block, set `componentPath` to
   `childPath` (built with `append`, ~3 lines).

### Tests

New or extended in `engine_test.go` or a new `component_error_test.go`:

- `TestComponentPathInError`: assert `RenderError.ComponentPath` for a
  two-level and three-level tree.
- `TestComponentPathErrorString`: assert `Error()` output includes the path.
- `TestComponentErrorHandlerContinues`: assert the page is written to `w`
  when the handler returns nil, and that the placeholder appears at the
  correct position.
- `TestComponentErrorHandlerAborts`: assert the render aborts when the
  handler returns a non-nil error.
- `TestComponentErrorHandlerNil`: assert nil handler preserves the existing
  abort-on-error behaviour.
- `TestHTMLErrorHandler`: assert the built-in handler produces escaped HTML
  and continues rendering.
- `TestComponentPathPropagationDeep`: assert three-level path
  `["A", "B", "C"]` when `C` fails inside `B` inside `A`.

Platform note: component names in `componentPath` are derived from HTML tag
names (lowercased by the parser) or from the component registry key; they do
not include filesystem paths, so there are no `path` vs. `filepath`
portability concerns.

---

## 8. Backward Compatibility

### `RenderError` struct

`RenderError` gains one new exported field `ComponentPath []string`. Adding a
field to a struct is backward-compatible for callers that use `errors.As` to
extract the error (the established usage pattern). Callers that construct
`RenderError` values directly (test code) must add the field or use a named
literal; the existing convention in this codebase uses named fields, so no
breakage is expected.

`RenderError.Error()` output changes when `ComponentPath` has more than one
element — a path prefix is prepended. Callers that assert exact error strings
in tests must be updated. Callers that use `errors.Is`/`errors.As` are
unaffected.

### `Options` struct

`Options` gains one new field `ComponentErrorHandler ComponentErrorHandler`.
Adding a field is backward-compatible for callers using named-field
initialisation (the established pattern).

### `Renderer` struct

`Renderer` gains two new unexported fields and two new `With*` builder
methods. Adding unexported fields and exported methods is fully
backward-compatible.

### `RenderPage`, `RenderFragment`, `RenderPageContext`, `RenderFragmentContext`

Function signatures are unchanged. Default behaviour (nil handler) is
identical to today: the first component error aborts the render, `w` receives
nothing, and the error is returned.

### Module dependencies

`HTMLErrorHandler` uses `html.EscapeString` from the stdlib `"html"` package,
which `renderer.go` already imports (as `stdhtml`). No new module dependencies
are introduced.

---

## 9. Alternatives Considered

### A. Render the full partial buffer on error without a handler

Instead of a handler API, always write the partial buffer to `w` when an
error occurs, returning the error alongside.

**Rejected**: `io.Writer` does not allow "un-writing" partial output. If the
caller has already set response headers with `http.ResponseWriter`, a partial
body followed by an error leaves the response in an undefined state. A
handler-based API gives the caller explicit control over whether partial output
is acceptable.

### B. Return `([]error, error)` from `RenderPage`

Change `RenderPage` to return two values: a slice of handled component errors
and a fatal error.

**Rejected**: Breaking API change. All callers must be updated. The handler
closure achieves the same result without modifying the return signature — the
handler accumulates errors into a slice the caller controls.

### C. A new `RenderPagePartial` method alongside `RenderPage`

Add `RenderPagePartial(w, name, data, handler ComponentErrorHandler) error`
as a separate entry point, leaving `RenderPage` unchanged.

**Rejected**: The `ComponentErrorHandler` field in `Options` is cleaner for
engine-wide configuration (e.g. development mode) and avoids a parallel set of
entry-point methods. The `Renderer.WithComponentErrorHandler` builder supports
single-render overrides without requiring a separate method.

### D. An `ErrorComponent` name in `Options` (template-based fallback)

Specify the name of a registered component to render in place of a failed
child:

```go
Options{ErrorComponent: "ErrorBoundary"}
```

**Rejected**: The error component must receive the error and path as props,
requiring reflection-based prop injection or a fixed schema. A plain function
type is more flexible, testable, and does not require a `.vue` file for
error display.

### E. Recover partial `RenderError` from error chain instead of new field

Walk `errors.Unwrap` chain to reconstruct the component path from the nested
`component %q: ...` messages.

**Rejected**: Requires string parsing of error messages, which is fragile. A
structured `[]string` field is the correct representation.

---

## 10. Open Questions

1. **Should `ComponentPath` be set on non-`RenderError` errors?** (blocking)
   If a directive's `Created` hook returns a plain `fmt.Errorf(...)`, it is
   not a `*RenderError`. `wrapComponentError` wraps it in a new `RenderError`.
   Should the wrapper always be a `*RenderError`, or should a new
   `ComponentError` type own the path while wrapping the original? Tentative
   answer: always wrap in `*RenderError`; the path is most useful on render
   errors and the wrapper type is already established.

2. **Should the root component name appear in `ComponentPath`?** (non-blocking)
   Currently proposed: yes, path is `["Root", "Child", "Grandchild"]`.
   Alternative: omit the root so path is `["Child", "Grandchild"]` (the root
   is already known from the `RenderPage` call). Tentative answer: include the
   root for unambiguous self-contained paths.

3. **`HTMLErrorHandler` styling** (non-blocking)
   Should the built-in dev handler include inline CSS, or rely solely on the
   `htmlc-error` class? Inline CSS is more immediately visible without any
   stylesheet; the class enables easy customisation. Tentative answer: include
   minimal inline CSS (red border, monospace font) with the class so it is
   visible out-of-the-box.

4. **Interaction with `WithLogger`** (non-blocking)
   When both a logger and a handler are set, the log record is emitted at
   `LevelError` before the handler is called. After the handler returns nil,
   should a follow-up `LevelDebug` record indicate that the error was handled?
   Tentative answer: no second record; the `LevelError` record is sufficient
   and adding a second record would be noisy.

5. **`v-for` iteration errors** (non-blocking)
   If a child component inside a `v-for` loop fails on iteration 3 of 10,
   should the handler be called once per failing iteration? Tentative answer:
   yes — each `renderComponentElement` call is independent, and the handler is
   invoked at each failure site. The path alone identifies the component; the
   caller can inspect the error for iteration context if needed.
