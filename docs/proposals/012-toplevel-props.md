# RFC 012: Struct Values as Top-Level Render Data

- **Status**: Draft
- **Date**: 2026-03-19
- **Author**: TBD

---

## 1. Motivation

RFC 007 introduced the `Props` interface (`props.go`) and `toProps()` so that
structs and custom types can be spread via `v-bind` inside component templates.
Callers who write `v-bind="user"` in a template can now pass a Go struct as a
prop value and the engine dispatches it correctly through `toProps()`.

However, every method in the `Render*` family still constrains the **top-level**
page data argument to `map[string]any`:

```go
// engine.go — current signatures
func (e *Engine) RenderPage(w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderFragment(w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderFragmentContext(ctx context.Context, w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderPageString(name string, data map[string]any) (string, error)
func (e *Engine) RenderFragmentString(name string, data map[string]any) (string, error)
```

The low-level renderer API has the same constraint:

```go
// renderer.go — current signatures
func (r *Renderer) Render(w io.Writer, scope map[string]any) error
func (r *Renderer) RenderString(scope map[string]any) (string, error)
func RenderString(c *Component, scope map[string]any) (string, error)
func Render(w io.Writer, c *Component, scope map[string]any) error
```

A Go HTTP handler that models its response data as a domain struct must
therefore convert it to `map[string]any` before calling any of these methods:

```go
// current — application code forced to convert
type PageData struct {
    User    User
    Product Product
}

func handler(w http.ResponseWriter, r *http.Request) {
    d := PageData{User: currentUser(r), Product: loadProduct(r)}
    eng.RenderPage(w, "ProductPage", map[string]any{
        "User":    d.User,
        "Product": d.Product,
    })
}
```

This conversion is mechanical, duplicates the type information already present
in the struct definition, and must be updated every time a field is added to
`PageData`. The failure mode is silent: a field omitted from the manual
conversion produces a `[missing: FieldName]` placeholder at render time, with no
compile-time warning.

RFC 007 solved this problem for `v-bind` spreads *inside* templates. The natural
completion of that work is to remove the same restriction at the call site where
top-level data enters the engine.

---

## 2. Goals

1. **Allow `map[string]any`, `Props`, struct, or pointer-to-struct to be passed
   as the `data` argument** to all six `Engine.Render*` methods.
2. **Allow the same types** as the `scope` argument to `Renderer.Render`,
   `Renderer.RenderString`, `RenderString`, and `Render`.
3. **Existing `map[string]any` callers compile and behave identically** — no
   source changes required.
4. **Reuse `toProps()` as the sole dispatch point** for the new argument
   normalisation, ensuring consistent behaviour between top-level data and
   `v-bind` spread.
5. **Provide a clear runtime error** when an unsupported type is passed (e.g.,
   a plain `int`), using the same error message already produced by `toProps()`.
6. **Update `validateProps`** so it operates on `Props` rather than
   `map[string]any`, preserving its existing case-insensitive fallback logic.

---

## 3. Non-Goals

1. **Compile-time type checking of the data argument**: this RFC does not use
   generics to preserve the existing type of `data` at call sites. Type safety
   at call sites is traded for a simpler, uniform signature. See §4.2 Option 3.
2. **HTTP middleware and `ServeComponent`**: `ServeComponent` and
   `ServePageComponent` accept `func(*http.Request) map[string]any` callbacks;
   widening those callbacks is a separate API evolution not addressed here.
3. **Static validation that a struct satisfies all template props**: the
   `Props()` analyser cannot know at parse time which struct type will be passed
   at the top level. This limitation is unchanged; see RFC 007 §4.4.
4. **Automatic marshalling to/from JSON or encoding/json tags at the top
   level**: field resolution follows the same rules as RFC 007 (`StructProps`):
   json tag name first, then Go field name, then first-rune-lowercase alias.
5. **Deep structural equality or diffing of props**: out of scope.

---

## 4. Proposed Design

### 4.1 Normalisation point

#### Current state

Each `Engine.Render*` method accepts `data map[string]any` and passes it
directly to `renderComponent`, which eventually reaches `Renderer.Render` as
a `scope map[string]any`. The chain never invokes `toProps()` on the top-level
value; `toProps()` is only called at `v-bind` spread sites inside the render
tree.

#### Proposed extension

Introduce a single normalisation step at each public entry point that converts
the caller-supplied data value into a `map[string]any` understood by the
existing render pipeline:

```go
// pseudo-code — not implementation
// propsToScope converts val to a map[string]any for the render pipeline.
// It delegates to toProps() for type dispatch, then materialises a flat map
// from the resulting Props value.
func propsToScope(val any) (map[string]any, error) {
    p, err := toProps(val)
    if err != nil {
        return nil, fmt.Errorf("render data: %w", err)
    }
    if p == nil {
        return map[string]any{}, nil
    }
    keys := p.Keys()
    m := make(map[string]any, len(keys))
    for _, k := range keys {
        v, _ := p.Get(k)
        m[k] = v
    }
    return m, nil
}
```

`propsToScope` is a thin adapter: it materialises a `map[string]any` from any
`Props` implementation, which is then handed to the unchanged internal render
pipeline. When `val` is already a `map[string]any`, `toProps()` returns a
`MapProps` wrapper; `propsToScope` iterates over its keys and produces an
identical `map[string]any` — the round-trip is allocation-neutral relative to
the current code path (the incoming map's capacity is reused).

The materialisation is done once per top-level render call, not per component
boundary. Individual `v-bind` spreads inside the template tree continue to use
`toProps()` directly without materialisation.

#### Why materialise a map rather than thread `Props` through the pipeline?

The internal render pipeline (`renderNode`, `renderElement`, `validateProps`,
the expression evaluator) operates pervasively on `map[string]any`. Threading
`Props` all the way through would require changing every internal function
signature and every scope-copy site. This RFC takes the pragmatic approach of
converting at the boundary so that the internal surface is untouched. A future
RFC could thread `Props` through the pipeline for zero-allocation rendering
of large struct-typed page data (see §10.3).

### 4.2 Primary API question: how should the `data` parameter type change?

Three options were evaluated:

#### Option 1 — Change `map[string]any` → `any` in all public signatures

All six `Engine.Render*` methods and the four package-level renderer functions
change their `data`/`scope` parameter from `map[string]any` to `any`. Internally,
each method calls `propsToScope(data)` immediately and returns the error if
`toProps()` rejects the type.

```go
// pseudo-code — not implementation
func (e *Engine) RenderPage(w io.Writer, name string, data any) error {
    scope, err := propsToScope(data)
    if err != nil {
        return err
    }
    return e.RenderPageContext(context.Background(), w, name, scope)
}
```

- ✅ Single, uniform signature for all render methods — no new API surface.
- ✅ Existing `map[string]any` callers compile unchanged; `toProps()` fast-path for
  `map[string]any` returns a `MapProps` without reflection.
- ✅ Struct callers pass data directly with no conversion boilerplate.
- ✅ Custom `Props` implementations work at the top level immediately.
- ⚠️ The parameter widens from `map[string]any` to `any`: the compiler no longer
  rejects `eng.RenderPage(w, "X", 42)` at compile time. The error surfaces at
  runtime instead (from `toProps()`).
- ⚠️ Callers that currently pass `nil` will now be accepted (normalised to an empty
  scope) rather than panicking on a nil map dereference.
- ❌ Loss of compile-time type checking for the outermost call site is the primary
  downside. Callers that previously had compiler-enforced type safety for the data
  argument must now rely on runtime validation.

**Verdict**: Option 1 is the recommended approach. The compile-time safety loss
is mitigated by (a) the clear, immediate runtime error from `toProps()` and (b)
the fact that the internal render pipeline already performs extensive runtime
validation (missing props, type errors in expressions). The uniform API is
consistent with how `v-bind` spreads already accept `any` at template boundaries,
and it requires no new exported names.

#### Option 2 — Add parallel `Props`-typed overloads

Leave the six `Engine` methods and four package-level functions unchanged.
Add six new `Engine` methods and four new package-level functions, each accepting
`Props` in place of `map[string]any`:

```go
// pseudo-code — not implementation
func (e *Engine) RenderPageProps(w io.Writer, name string, data Props) error
func (e *Engine) RenderFragmentProps(w io.Writer, name string, data Props) error
// …and so on for all six Engine methods and four package-level functions
```

- ✅ Existing signatures are completely unmodified — zero risk of breakage.
- ✅ The `Props`-typed overloads enforce that only `Props` implementations
  (including `MapProps`) are accepted; partial type safety at call sites.
- ⚠️ Struct callers must still wrap their value: `newStructProps(d)` or use a
  helper. They cannot pass a plain struct directly.
- ⚠️ Doubles the public API surface (10 new exported names). Discovery burden on
  new users who must learn which variant to call.
- ❌ `map[string]any` callers must either stay on the old overloads or explicitly
  construct `MapProps` — split userbase, two code paths to maintain forever.
- ❌ Does not solve the core ergonomic problem for plain-struct callers: they still
  need an explicit conversion step.

**Verdict**: Rejected. The doubled API surface and inability to accept plain
structs without explicit wrapping make this option inferior to Option 1. It
defers rather than solves the conversion burden.

#### Option 3 — Generic entry point `RenderPageWith[T any]`

Introduce a single generic function (or method, pending Go generics constraints)
that accepts any `T` and converts via `toProps()`:

```go
// pseudo-code — not implementation
func RenderPageWith[T any](e *Engine, w io.Writer, name string, data T) error {
    scope, err := propsToScope(any(data))
    if err != nil {
        return err
    }
    return e.RenderPageContext(context.Background(), w, name, scope)
}
```

- ✅ Preserves type information at call sites; `T` is statically known.
- ✅ Works for any type without changing the existing method signatures.
- ⚠️ Go generics do not support methods with type parameters on a receiver
  (`func (e *Engine) RenderPageWith[T any]` is not valid Go). The generic form
  must be a package-level function, breaking the `eng.RenderPage(…)` call pattern
  established throughout the codebase.
- ⚠️ The `T` constraint cannot be restricted to `Props | map[string]any | struct`
  in current Go generics syntax; the constraint is effectively `any`, so compile-time
  checking gains are illusory — `RenderPageWith[int](eng, …)` compiles and fails
  at runtime.
- ❌ Requires 10 new package-level generic functions (six `Engine`-equivalent
  wrappers plus four package-level renderer wrappers) for feature parity.
- ❌ Package-level function style (`RenderPageWith(eng, w, name, data)`) is
  inconsistent with the existing method-based API (`eng.RenderPage(w, name, data)`).

**Verdict**: Rejected. The apparent compile-time safety benefit is not
realised in practice (constraint is `any`), the method-receiver limitation forces
a package-level style inconsistent with the existing API, and the surface
expansion is large.

### 4.3 Scope: Engine API vs. low-level Renderer API

The four package-level renderer functions (`Render`, `RenderString`,
`Renderer.Render`, `Renderer.RenderString`) form a lower-level API, typically
used by callers that construct a `Renderer` directly. The same `map[string]any`
constraint applies to them.

**Proposed scope**: extend all four to accept `any`, applying `propsToScope` at
the entry point of each, consistent with the `Engine` API change. Rationale:
callers who use the low-level API also work with domain structs; leaving those
signatures unchanged would create an inconsistent API where the high-level path
accepts any, but the low-level path does not.

### 4.4 `validateProps` and the internal pipeline

#### Current state

`validateProps` (`renderer.go:298`) takes `scope map[string]any` and iterates
over it to perform case-insensitive prop key matching. The entire internal render
pipeline (`renderNode`, expression evaluation, scope merging) operates on
`map[string]any`.

#### Proposed extension

`validateProps` is called after `propsToScope` has already produced a
`map[string]any`. **No changes to `validateProps` are required.** The
normalisation at the entry point means that by the time `validateProps` is
called, the scope is always a `map[string]any`.

This is a deliberate design choice: the materialisation cost (iterating struct
fields once to build a `map[string]any`) is incurred once per render call, and
every downstream function continues to operate on the familiar `map[string]any`
type without any signature changes. The internal API surface is left completely
unchanged.

---

## 5. Syntax Summary

*No new template syntax is introduced.* This RFC is a Go API change only.
Template authors write `{{ FieldName }}` and `v-bind="prop"` exactly as before;
the field-name resolution rules are identical to those specified in RFC 007.

---

## 6. Examples

### Example 1 — Pass a plain struct directly (new behaviour)

```go
type PageData struct {
    Title   string
    User    User
    Product Product
}

func handler(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:   "Product Detail",
        User:    currentUser(r),
        Product: loadProduct(r),
    }
    // No conversion step — pass the struct directly.
    if err := eng.RenderPage(w, "ProductPage", data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

`ProductPage.vue` accesses `{{ Title }}`, `{{ User.Name }}`, `{{ Product.ID }}`
exactly as if a `map[string]any` had been passed.

### Example 2 — Struct with json tags

```go
type PageData struct {
    Title   string `json:"title"`
    Article Article `json:"article"`
}
```

The top-level keys seen by the template are `"title"` and `"article"` (the json
tag names). The template uses `{{ title }}` and `{{ article.Body }}`.

### Example 3 — Pointer to struct

```go
func handler(w http.ResponseWriter, r *http.Request) {
    data := &PageData{Title: "Hello"}
    eng.RenderPage(w, "Page", data) // pointer accepted; dereferenced by toProps()
}
```

### Example 4 — Custom `Props` implementation

```go
type LazyProps struct{ req *http.Request }

func (p LazyProps) Keys() []string { return []string{"user", "csrf"} }
func (p LazyProps) Get(key string) (any, bool) {
    switch key {
    case "user": return currentUser(p.req), true
    case "csrf": return csrfToken(p.req), true
    }
    return nil, false
}

func handler(w http.ResponseWriter, r *http.Request) {
    eng.RenderPage(w, "Page", LazyProps{r})
}
```

A custom `Props` implementation is passed directly. `toProps()` returns it via
the identity path (priority 2 in the dispatch order). `propsToScope` materialises
it once into a `map[string]any` before the render.

### Example 5 — Backward compatibility: existing `map[string]any` callers

```go
// This existing code compiles and behaves identically after the change.
eng.RenderPage(w, "Page", map[string]any{
    "title": "Hello",
    "user":  currentUser(r),
})
```

`toProps()` takes the `map[string]any` fast-path (priority 3), wrapping it in a
`MapProps`. `propsToScope` iterates its keys and produces the same map. No change
in behaviour.

### Example 6 — Low-level renderer with a struct

```go
r := htmlc.NewRenderer(comp)
data := MyPageData{Title: "Hello", Count: 3}
html, err := r.RenderString(data) // previously required map[string]any
```

### Example 7 — Runtime error for unsupported type

```go
// This compiles (parameter is `any`) but fails at runtime.
err := eng.RenderPage(w, "Page", 42)
// err.Error() == "render data: expected map or struct, got int"
```

---

## 7. Implementation Sketch

Changes are minimal and concentrated at the public boundary.

### `renderer.go`

1. Add `propsToScope(val any) (map[string]any, error)` — a private helper that
   calls `toProps(val)` and materialises the resulting `Props` into a
   `map[string]any`. ~12 lines.

2. **`Renderer.Render`** — change `scope map[string]any` to `scope any`; call
   `propsToScope(scope)` at the top, replacing the bare scope variable. One-liner
   change to signature + two added lines.

3. **`Renderer.RenderString`** — same signature change; delegates to `r.Render`
   so no additional logic needed. One-liner change.

4. **`RenderString` (package-level)** — same signature change; delegates to
   `NewRenderer(c).RenderString(scope)`. One-liner change.

5. **`Render` (package-level)** — same signature change; delegates to
   `NewRenderer(c).Render(w, scope)`. One-liner change.

6. **`validateProps`** — no changes required. It is called after `propsToScope`
   has already produced a `map[string]any`.

### `engine.go`

7. **All six `Engine.Render*` methods** — change `data map[string]any` to
   `data any`. The non-Context variants (`RenderPage`, `RenderFragment`,
   `RenderPageString`, `RenderFragmentString`) delegate to the Context variants;
   only the two Context variants need to call `propsToScope`. Approximately six
   one-line signature changes plus two `propsToScope` call sites.

8. **`renderComponent` (internal)** — if it currently takes `data map[string]any`,
   its signature changes to `data map[string]any` unchanged (the conversion happens
   in the public entry points above, before `renderComponent` is called).

### Tests

9. **`engine_test.go` / `renderer_test.go`** — add table-driven cases for:
   - plain struct top-level data
   - struct with json tags
   - pointer-to-struct
   - custom `Props` implementation
   - unsupported type returns the expected error
   - existing `map[string]any` callers behave identically (regression guard)

No changes to `props.go`, `component.go`, `expr/eval.go`, or any template-side code.

---

## 8. Backward Compatibility

### `Engine.RenderPage`, `Engine.RenderPageContext`, `Engine.RenderFragment`, `Engine.RenderFragmentContext`, `Engine.RenderPageString`, `Engine.RenderFragmentString`

Parameter type changes from `map[string]any` to `any`. This is a **source-level
compatible** change: every existing call site that passes a `map[string]any`
literal or variable continues to compile without modification because
`map[string]any` is assignable to `any`. The rendered output is identical.

### `Renderer.Render`, `Renderer.RenderString`

Same source-level compatible change from `map[string]any` to `any`. Existing
call sites compile unchanged.

### Package-level `Render`, `RenderString`

Same source-level compatible change. Existing call sites compile unchanged.

### `validateProps` (private)

No signature change. This function remains internal; its contract is unchanged.

### Interface implementations

Any type that currently implements `interface { Render(io.Writer, map[string]any) error }`
or similar (e.g., a mock in test code) will no longer satisfy that interface after
the signature change. Authors who define interface types based on the Renderer
method signatures must update those interfaces. This is the only source of
potential breakage; it is limited to code outside this package that wraps
`Renderer` methods in an interface.

**Mitigation**: the `Renderer` type is not itself defined by an interface in this
package. Direct usage of `*Renderer` (the common case) is unaffected.

### `nil` data

Previously, `eng.RenderPage(w, "Page", nil)` would compile (nil is assignable to
`map[string]any`) and produce a nil map scope, which could panic on a nil map
dereference in `validateProps`. After this change, `nil` is normalised to an
empty `map[string]any` by `propsToScope` (via `toProps()` returning `nil, nil`).
This is strictly a bug fix: callers that previously got a panic now get a clean,
empty render.

---

## 9. Alternatives Considered

### A. Add a top-level `PropsToMap` helper and document the pattern

Provide a public `PropsToMap(val any) (map[string]any, error)` so callers can
convert explicitly before calling the unchanged render methods.

**Rejected**: shifts the conversion burden back to callers, defeating the
ergonomic goal. Authors must remember to call the helper; there is no enforcement.
The internal `propsToScope` helper achieves the same effect automatically at the
entry point.

### B. Change only the `Engine` API, leave the low-level renderer unchanged

Widen only the six `Engine.Render*` methods; leave `Renderer.Render`,
`Renderer.RenderString`, and the package-level `Render`/`RenderString` with
`map[string]any`.

**Rejected**: inconsistent API. Callers who use the lower-level renderer API
(common for testing and for custom render pipelines) would not benefit from the
change and would see a surprising asymmetry.

### C. Thread `Props` through the entire internal pipeline

Change `validateProps`, `renderNode`, `renderElement`, and the expression
evaluator to accept and pass `Props` everywhere instead of `map[string]any`.

**Rejected** for this RFC (not forever): the internal change is large and
touches every scope-copy and scope-merge site. The materialisation at the entry
point is sufficient for the stated goals and preserves the internal API for a
future zero-allocation RFC. See §10.3.

### D. Use `encoding/json` round-trip for struct-to-map conversion

Marshal the struct to JSON and unmarshal into `map[string]any` at the entry
point.

**Rejected**: allocates a JSON string, runs two passes over the data, and loses
type information (all numbers become `float64`, typed structs become maps).
`toProps()` + `StructProps` already solves the problem without JSON.

### E. Require structs to implement a marker interface

Add a `TopLevelData` marker interface that callers must embed in their data
structs to signal intent.

**Rejected**: adds boilerplate to every domain type; inconsistent with RFC 007's
zero-configuration approach; no meaningful compile-time benefit since the engine
still calls `toProps()` at runtime.

---

## 10. Open Questions

1. **`ServeComponent` / `ServePageComponent` data callback signatures**: these
   methods accept `func(*http.Request) map[string]any`. Should they be widened to
   `func(*http.Request) any` in this RFC or a follow-on?
   *Tentative recommendation*: defer to a follow-on RFC. The callback ABI change
   is independent of the render method change and has its own backward-compat
   implications for closures defined by callers. **Non-blocking**.

2. **`propsToScope` materialisation cost for large structs**: materialising a
   `map[string]any` from a large struct at the top of every render call performs
   one allocation per render. For high-throughput servers this may be measurable.
   A benchmark (`BenchmarkRenderPage_StructData` with `b.ReportAllocs()`) should
   be added during implementation to quantify the cost before the RFC is accepted.
   *Recommendation*: require the benchmark result before merging.
   **Blocking**.

3. **Zero-allocation future: threading `Props` through the pipeline**: if the
   benchmark in Q2 shows that materialisation overhead is significant, a follow-on
   RFC should thread `Props` through `validateProps` and the full render pipeline,
   eliminating the materialisation step entirely. This RFC intentionally defers
   that work.
   **Non-blocking**.

4. **`nil` data behaviour change**: `eng.RenderPage(w, "Page", nil)` previously
   compiled with `nil` as a `map[string]any` and could panic; after this RFC it
   normalises to an empty scope. Callers relying on the nil-panic as a
   programming-error signal will lose that signal. Is silent normalisation
   preferable to returning an error on `nil`?
   *Recommendation*: normalise silently, consistent with the `v-bind="nil"` no-op
   behaviour established in RFC 007.
   **Non-blocking**.

5. **Widening `renderComponent` internal signature**: `renderComponent` currently
   takes `map[string]any`. After this RFC the conversion happens before
   `renderComponent` is called. If a future RFC threads `Props` through the
   pipeline, `renderComponent` will need its own signature change. Document this
   decision point in a comment at the call site.
   **Non-blocking**.
