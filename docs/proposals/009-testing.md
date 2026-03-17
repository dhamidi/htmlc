# RFC 009: htmlctest API Redesign — Fluent Harness and Query Builder

- **Status**: Draft
- **Date**: 2026-03-17
- **Author**: TBD

---

## 1. Motivation

The current `htmlctest` package exposes three free functions for testing `htmlc`
components. While sufficient for the simplest cases, the API forces every assertion
to thread `t` and `e` manually, supports only whole-string HTML comparison, and
provides no mechanism for querying specific elements within the rendered output.

### The failure in practice

A typical test today:

```go
func TestUserCard(t *testing.T) {
    e := htmlctest.NewEngine(t, map[string]string{
        "Avatar.vue":   `<template><img :src="src" alt="avatar"></template>`,
        "UserCard.vue": `<template>
            <div class="card">
                <Avatar :src="avatarURL" />
                <h2>{{ name }}</h2>
                <span v-if="admin" class="badge">Admin</span>
            </div>
        </template>`,
    })

    // Must pass t and e to every assertion.
    htmlctest.AssertFragment(t, e, "UserCard",
        map[string]any{"name": "Alice", "avatarURL": "/img/alice.png", "admin": true},
        `<div class="card"><img src="/img/alice.png" alt="avatar"><h2>Alice</h2><span class="badge">Admin</span></div>`,
    )
}
```

Pain points:

- `t` and `e` are threaded to every assertion — a form of manual dependency
  injection that adds noise in every call site.
- The only assertion is a whole-string normalized-whitespace comparison. To check
  that `<h2>` contains "Alice", the test must assert the entire rendered output.
  Adding a new element to the component breaks every existing test string.
- No way to assert "the `.badge` element is present and contains 'Admin'" without
  asserting the full surrounding markup.
- Table-driven tests need hand-rolled loops with repeated `htmlctest.AssertFragment`
  calls.
- Failure messages show raw strings side-by-side; there is no diff to pinpoint
  the changed region.

### Why the existing API cannot be extended in place

The three free-function signatures (`NewEngine`, `AssertRendersHTML`, `AssertFragment`)
require callers to pass `t` and `e` on every call. There is no object that
accumulates context between calls, so element-level queries and fluent chaining are
structurally impossible without introducing a new type.

---

## 2. Goals

1. **Introduce a `Harness` type** that owns the engine and captures `t`, eliminating
   manual threading of `t` and `e` to every assertion.
2. **Introduce a `Result` type** with fluent `Assert*` methods so assertions chain
   without intermediate variables.
3. **Fluent DOM queries**: a `Query` builder and `Selection` type allow asserting
   element presence, count, text content, and attributes without CSS selector
   strings or external dependencies.
4. **Table-driven helper**: `RunCases` runs a `[]Case` slice as sub-tests, reducing
   the boilerplate loop to a single call.
5. **Improved diff in failure messages** so test output pinpoints the changed region
   rather than showing two full strings.
6. **Full backward compatibility**: the three existing free functions are preserved
   with identical signatures.

---

## 3. Non-Goals

1. **Browser / JavaScript rendering**: this RFC covers server-side HTML string
   output only. Interactive behaviour (click handlers, reactive state) is out of
   scope.
2. **XPath assertions**: the fluent `Query` builder covers the important cases;
   XPath is not planned.
3. **Parallel `t.Parallel()` management**: callers are responsible for calling
   `t.Parallel()` where desired. `Harness` does not call it automatically.
4. **Typed prop validation at compile time**: `RunCases.Data` remains
   `map[string]any`. Static type-checked props are a separate concern.
5. **Integration with external test frameworks** (testify, gomega): the API uses
   the standard `testing.TB` interface only. Interoperability is not blocked, but
   not explicitly designed for.
6. **Snapshot / golden-file regression testing**: storing expected output in
   committed files makes the test suite stateful and is out of scope.

---

## 4. Proposed Design

### 4.1 `Harness` — central fixture

#### Current state

Callers create an `*htmlc.Engine` via `htmlctest.NewEngine` and pass it to every
assertion:

```go
// current — htmlctest/htmlctest.go
func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine
func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string)
func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string)
```

#### Proposed extension

A new **`Harness`** type holds `t` and `*htmlc.Engine` internally:

```go
// pseudo-code — not implementation
type Harness struct {
    t   testing.TB
    eng *htmlc.Engine
}

// NewHarness replaces NewEngine. Accepts the same file map + optional Options.
func NewHarness(t testing.TB, files map[string]string, opts ...htmlc.Options) *Harness

// With adds (or replaces) a single component after construction.
// Returns the same *Harness to allow chaining setup calls.
func (h *Harness) With(filename, src string) *Harness

// Engine exposes the underlying *htmlc.Engine for tests that need lower-level
// access (e.g. RegisterDirective, ServeComponent).
func (h *Harness) Engine() *htmlc.Engine

// Build is a shorthand constructor for single-component tests.
// The template string is automatically wrapped in <template>…</template>
// if it does not already contain one, so inline snippets work without ceremony.
func Build(t testing.TB, template string) *Harness
```

`Build` enables the most concise form:

```go
htmlctest.Build(t, `<p>Hello {{ name }}!</p>`).
    Fragment("Root", map[string]any{"name": "World"}).
    Find(htmlctest.ByTag("p")).AssertText("Hello World!")
```

`Build` infers the component name as `"Root"`. When the caller passes a string
without a `<template>` wrapper, `Build` wraps it automatically. This is a
convenience for one-off unit tests; multi-component tests should use `NewHarness`.

**Verdict**: `Harness` is the only structural change required to eliminate `t`/`e`
threading. All `Assert*` methods are on `Result` and `Selection`, not on `Harness`,
so `Harness` stays small.

### 4.2 `Result` — fluent assertion chain

#### Current state

Assertions are free functions. They call `t.Fatal` on failure and return nothing,
making chaining impossible.

#### Proposed extension

A new **`Result`** type holds the rendered HTML string and calls `t.Fatal` on
the embedded `t` when an assertion fails:

```go
// pseudo-code — not implementation
type Result struct {
    t    testing.TB
    html string
    root *html.Node // lazily parsed; cached after first Find call
}

// --- Rendering entry points on Harness ---

// Page renders name as a full HTML page (wraps RenderPageString).
func (h *Harness) Page(name string, data map[string]any) *Result

// Fragment renders name as an HTML fragment (wraps RenderFragmentString).
func (h *Harness) Fragment(name string, data map[string]any) *Result

// --- Accessor ---

// HTML returns the raw rendered string.
func (r *Result) HTML() string

// --- Equality assertion ---

// AssertHTML asserts the rendered output equals want after normalising
// whitespace. On failure it prints a line-level diff.
func (r *Result) AssertHTML(want string) *Result

// --- DOM query entry point ---

// Find returns a Selection of all nodes in the rendered HTML that match q.
// The HTML is parsed once and cached on first call.
func (r *Result) Find(q Query) *Selection
```

Every `Assert*` method returns `*Result` to allow chaining. Internally, each
assertion calls `r.t.Helper()` before `r.t.Fatal(…)` so that failure lines point
to the call site, not into the `htmlctest` package.

**HTML parsing**: `Result` parses the HTML string with `golang.org/x/net/html` on
first `Find` call; the root `*html.Node` is cached on `Result`. No external library
is required beyond this existing transitive dependency.

**Verdict**: a fluent `*Result` return is the cleanest way to enable chaining
without requiring callers to name intermediate variables.

### 4.3 `Query` and `Selection` — fluent DOM query builder

#### Current state

There is no way to query the rendered HTML tree for specific elements.

#### Proposed extension

A **`Query`** type is a composable, immutable element filter. It is interpreted
directly over the `*html.Node` tree; no CSS parser is needed.

```go
// pseudo-code — not implementation

// Query is a composable, immutable element filter.
// Constructors and combinators all return a new Query value.
type Query struct { ... }

// ByTag matches elements with the given tag name (case-insensitive).
func ByTag(name string) Query

// WithClass returns a new Query that additionally requires the element to have
// the given CSS class in its class attribute.
func (q Query) WithClass(class string) Query

// WithAttr returns a new Query that additionally requires the element to have
// attribute attr equal to value. Use value="" to assert attribute presence only.
func (q Query) WithAttr(attr, value string) Query

// Descendant returns a new Query that matches elements satisfying q that are
// descendants of an element satisfying ancestor.
func (q Query) Descendant(ancestor Query) Query
```

A **`Selection`** carries the matched nodes and exposes fluent assertion methods:

```go
// pseudo-code — not implementation

type Selection struct {
    t     testing.TB
    nodes []*html.Node // from golang.org/x/net/html (already a transitive dep)
}

// AssertExists fails the test if no nodes were matched.
func (s *Selection) AssertExists() *Selection

// AssertNotExists fails the test if any node was matched.
func (s *Selection) AssertNotExists() *Selection

// AssertCount fails the test if the number of matched nodes is not n.
func (s *Selection) AssertCount(n int) *Selection

// AssertText fails if the first matched node's visible text content
// (recursive TextNode concatenation, whitespace-normalised) is not equal to text.
func (s *Selection) AssertText(text string) *Selection

// AssertAttr fails if the first matched node does not have attribute attr
// equal to value.
func (s *Selection) AssertAttr(attr, value string) *Selection
```

`Selection` methods return `*Selection` to allow chaining:

```go
r.Find(ByTag("button").WithClass("primary")).
    AssertExists().
    AssertText("Save")
```

The tree walk is a plain recursive function over `*html.Node` — no external library
is needed beyond the existing `golang.org/x/net/html` transitive dependency.

**Verdict**: a typed query builder with a recursive node walk gives compile-time
checking of query structure, zero new dependencies, and a straightforward
implementation.

### 4.4 Table-driven helper — `RunCases`

#### Current state

Table-driven tests require a manual `for _, tc := range cases { t.Run(tc.Name, ...) }`
loop that duplicates boilerplate across every test file.

#### Proposed extension

```go
// pseudo-code — not implementation

// Case is one entry in a table-driven component test.
type Case struct {
    Name string         // sub-test name (passed to t.Run)
    Data map[string]any // props to pass to the component
    Want string         // expected HTML (normalised)
}

// RunCases runs each Case as a t.Run sub-test against component on h.
// Pass page=true for full-page rendering; omit (or false) for fragment rendering.
func RunCases(t *testing.T, h *Harness, component string, cases []Case, page ...bool)
```

`RunCases` calls `t.Run(tc.Name, ...)` for each case. Inside the sub-test it calls
`h.Fragment` (or `h.Page` when `page[0]` is true) and then `r.AssertHTML(tc.Want)`.

**Verdict**: a single helper reduces 6-line boilerplate loops to a single
`RunCases` call while retaining sub-test granularity for `go test -run`.

### 4.5 Line-level diff in failure messages

#### Current state

`AssertRendersHTML` and `AssertFragment` call `t.Fatalf` with:

```text
AssertFragment: got:
<full rendered html>

want:
<full expected html>
```

For large outputs this is unreadable.

#### Proposed extension

A private helper provides a `lineDiff(want, got string) string` function
implemented using only the standard library:

- Split `want` and `got` on newlines (`strings.Split`).
- Walk the lines to emit `+`/`-` prefix markers for differing regions, with up
  to ±5 lines of context around each changed region.
- No external library is required.

The diff output is embedded in the `AssertHTML` failure message.

**Verdict**: a stdlib-only line diff is sufficient to pinpoint the changed region
in HTML output. The implementation requires no new dependencies.

### 4.6 Backward-compatibility shims

The three existing free functions are preserved unchanged. Internally they become
thin wrappers over `Harness`/`Result`:

```go
// pseudo-code — not implementation

func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine {
    return NewHarness(t, files, opts...).Engine()
}

func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
    t.Helper()
    h := &Harness{t: t, eng: e}
    h.Page(name, data).AssertHTML(want)
}

func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
    t.Helper()
    h := &Harness{t: t, eng: e}
    h.Fragment(name, data).AssertHTML(want)
}
```

This guarantees that callers using the old API benefit from the improved diff
output without any code changes.

---

## 5. Syntax Summary

*This RFC introduces no new template syntax. The table below summarises the new
Go API surfaces.*

| Surface                                        | Meaning                                                               |
|------------------------------------------------|-----------------------------------------------------------------------|
| `htmlctest.NewHarness(t, files, opts...)`      | Create a `Harness`; replaces `NewEngine` for new tests.              |
| `htmlctest.Build(t, template)`                 | Shorthand harness for single-component tests; wraps template string.  |
| `h.With(filename, src)`                        | Add/replace a component file on an existing `Harness`.               |
| `h.Engine()`                                   | Access the underlying `*htmlc.Engine`.                               |
| `h.Page(name, data)`                           | Render a full HTML page; returns `*Result`.                          |
| `h.Fragment(name, data)`                       | Render an HTML fragment; returns `*Result`.                          |
| `r.HTML()`                                     | Raw rendered string.                                                  |
| `r.AssertHTML(want)`                           | Whole-output whitespace-normalized comparison with line-level diff.   |
| `r.Find(q)`                                    | Return a `*Selection` of all nodes matching `Query` q.               |
| `htmlctest.ByTag(name)`                        | `Query` matching elements by tag name (case-insensitive).            |
| `q.WithClass(class)`                           | Extend `Query` to also require the given CSS class.                  |
| `q.WithAttr(attr, value)`                      | Extend `Query` to also require an attribute equals value.            |
| `q.Descendant(ancestor)`                       | Extend `Query` to only match descendants of `ancestor`.              |
| `s.AssertExists()`                             | Fail if no nodes were matched.                                        |
| `s.AssertNotExists()`                          | Fail if any node was matched.                                         |
| `s.AssertCount(n)`                             | Fail if the number of matched nodes is not `n`.                      |
| `s.AssertText(text)`                           | Fail if the first matched node's visible text is not `text`.         |
| `s.AssertAttr(attr, value)`                    | Fail if the first matched node's attribute is not `value`.           |
| `htmlctest.RunCases(t, h, comp, cases, page?)` | Run `[]Case` as sub-tests.                                            |

---

## 6. Examples

### Example 1 — One-line component test

```go
func TestGreeting(t *testing.T) {
    htmlctest.Build(t, `<p>Hello {{ name }}!</p>`).
        Fragment("Root", map[string]any{"name": "World"}).
        Find(htmlctest.ByTag("p")).AssertText("Hello World!")
}
```

No file map, no engine variable, no full-string comparison.

### Example 2 — Multi-component harness with query assertions

```
components/
  Avatar.vue
  UserCard.vue
```

```go
func TestUserCard(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Avatar.vue":   `<template><img :src="src" alt="avatar"></template>`,
        "UserCard.vue": `<template>
            <div class="card">
                <Avatar :src="avatarURL" />
                <h2>{{ name }}</h2>
                <span v-if="admin" class="badge">Admin</span>
            </div>
        </template>`,
    })

    r := h.Fragment("UserCard", map[string]any{
        "name":      "Alice",
        "avatarURL": "/img/alice.png",
        "admin":     true,
    })
    r.Find(htmlctest.ByTag("div").WithClass("card")).AssertExists()
    r.Find(htmlctest.ByTag("h2")).AssertText("Alice")
    r.Find(htmlctest.ByTag("img")).AssertAttr("src", "/img/alice.png")
    r.Find(htmlctest.ByTag("span").WithClass("badge")).AssertExists().AssertText("Admin")
}
```

Each assertion is independent: adding new elements to `UserCard.vue` does not
break the test, because only specific elements are checked.

### Example 3 — Table-driven with `RunCases`

```
components/
  Button.vue
```

```go
func TestButton(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Button.vue": `<template>
            <button :disabled="disabled" :class="variant">{{ label }}</button>
        </template>`,
    })

    htmlctest.RunCases(t, h, "Button", []htmlctest.Case{
        {
            Name: "default",
            Data: map[string]any{"label": "Save", "variant": "primary"},
            Want: `<button class="primary">Save</button>`,
        },
        {
            Name: "disabled",
            Data: map[string]any{"label": "Save", "variant": "primary", "disabled": true},
            Want: `<button disabled class="primary">Save</button>`,
        },
    })
}
```

Sub-tests are named `TestButton/default` and `TestButton/disabled` and can be
targeted individually with `go test -run TestButton/disabled`.

### Example 4 — Backward-compatible existing test (unchanged)

```go
func TestLegacy(t *testing.T) {
    e := htmlctest.NewEngine(t, map[string]string{
        "Greeting.vue": `<template><p>Hello {{ name }}!</p></template>`,
    })
    htmlctest.AssertFragment(t, e, "Greeting",
        map[string]any{"name": "World"},
        "<p>Hello World!</p>",
    )
}
```

This test compiles and runs without modification. The failure message now includes
a line-level diff if the output does not match.

---

## 7. Implementation Sketch

All new code lives in the `htmlctest` package. No changes to `htmlc` engine code
are required.

| File | Contents | Approx. lines |
|------|----------|--------------|
| `htmlctest/harness.go` | `Harness` struct, `NewHarness`, `Build`, `With`, `Engine`, `Page`, `Fragment` | ~60 |
| `htmlctest/result.go` | `Result` struct (holds `t testing.TB`, `html string`, lazy `*html.Node`); `HTML()`, `AssertHTML()`, `Find()` | ~40 |
| `htmlctest/query.go` | `Query` type and constructors (`ByTag`, `WithClass`, `WithAttr`, `Descendant`); `Selection` type and assertion methods; recursive node-walk helpers | ~120 |
| `htmlctest/cases.go` | `Case` struct, `RunCases` | ~30 |
| `htmlctest/compat.go` | Backward-compat shims for `NewEngine`, `AssertRendersHTML`, `AssertFragment` | ~25 |

**`result.go`**: `AssertHTML` calls the private `lineDiff(want, got string) string`
helper (defined in the same file or a small private file) implemented with
`strings.Split` and a linear line walk. No external library.

**`query.go`**: the recursive node walk is a plain depth-first traversal of
`*html.Node`. `Query` matching uses `strings.EqualFold` for tag names and
`strings.Fields` for class tokenization. `Descendant` is checked by walking the
ancestor chain of candidate nodes.

**New dependencies**: none. `golang.org/x/net/html` is already present as a
transitive dependency. It is listed as a direct dependency of `htmlctest`
(added to the `import` block) without adding a new module to `go.mod`.

---

## 8. Backward Compatibility

### `NewEngine` (public)

Signature is unchanged. Internally re-implemented as `NewHarness(t, files, opts...).Engine()`.
All callers are source-compatible and binary-compatible.

### `AssertRendersHTML` (public)

Signature is unchanged. Now delegates to `h.Page(name, data).AssertHTML(want)`.
Failure messages now include a line-level diff; this is a strictly better output,
not a breaking change.

### `AssertFragment` (public)

Signature is unchanged. Now delegates to `h.Fragment(name, data).AssertHTML(want)`.
Same improved failure message as above.

### New exports: `Harness`, `Result`, `Query`, `Selection`, `Case`, `NewHarness`, `Build`, `ByTag`, `RunCases`

All new. No existing code references these identifiers.

### `go.mod` / `go.sum`

No new modules are added. `golang.org/x/net/html` moves from transitive to direct
dependency within `htmlctest`, but this does not modify `go.mod` if it is already
present in the module graph.

---

## 9. Alternatives Considered

### A. Extend the existing free functions with additional `...Option` parameters

Add functional options to `AssertFragment` for element assertions:

```go
htmlctest.AssertFragment(t, e, "UserCard", data, want,
    htmlctest.WithSelector(".card"),
    htmlctest.WithText("h2", "Alice"),
)
```

✅ No new types; backward-compatible signature extension via variadic options.
❌ Does not eliminate `t`/`e` threading.
❌ Options are not chainable; all assertions happen in a single call, making
   multiple assertions verbose.
❌ Cannot interleave assertions with conditional logic.

**Rejected**: the option pattern solves the chaining problem syntactically but not
structurally. `t` and `e` are still passed at every call site.

### B. Embed `t` and `e` in a context passed as the first argument

```go
ctx := htmlctest.NewContext(t, e)
htmlctest.AssertText(ctx, "UserCard", data, "h2", "Alice")
```

✅ No new method-receiver pattern; all functions remain free functions.
❌ Still requires a `ctx` variable at every call site.
❌ Cannot chain calls because free functions return nothing.
❌ `ctx` would need to carry render state (the last rendered `Result`) to avoid
   re-rendering for each assertion, which makes the API stateful in a surprising way.

**Rejected**: a context parameter reduces but does not eliminate the threading
problem and makes the result-sharing semantics implicit.

### C. CSS selector strings via `cascadia`

An earlier draft of this RFC proposed `AssertSelector(sel string)`,
`AssertText(sel, text string)`, and related methods powered by
`github.com/andybalholm/cascadia` and `golang.org/x/net/html`.

✅ CSS selector syntax is familiar to web developers.
⚠️ Selector strings are opaque — typos and unsupported pseudo-classes fail at
   runtime, not compile time.
❌ Adds `github.com/andybalholm/cascadia` as a new module dependency, violating
   the no-external-dependencies rule for this package.
❌ Selector strings cannot be composed or extended by the caller without string
   concatenation.

**Rejected**: the fluent `Query` builder provides the same expressiveness with
compile-time checking, no new dependencies, and a straightforward recursive
implementation over `*html.Node`.

---

## 10. Open Questions

1. **Component name inferred by `Build`**: `Build` currently infers the component
   name as `"Root"`. Should the name be customisable (e.g., `Build(t, template,
   htmlctest.WithName("MyComp"))`)? The inferred name only matters if the caller
   calls `h.Page("Root", …)` or `h.Fragment("Root", …)` explicitly; `Build` is
   designed for one-liner tests where this is not an issue.
   *Recommendation*: keep `"Root"` as a fixed convention for `Build`; callers
   who need a specific name should use `NewHarness`. Non-blocking.

2. **Whitespace normalisation in `AssertText`**: `AssertText` normalises whitespace
   in the text content of the matched element (collapses consecutive whitespace to
   a single space, trims leading/trailing). Should the raw text be available via a
   separate `AssertRawText` method?
   *Recommendation*: add `AssertRawText` only if a real need arises during
   implementation. Non-blocking.

3. **`RunCases` and `page` variadic**: the `page ...bool` variadic is slightly
   unusual. An alternative is `RunPageCases` / `RunFragmentCases` as two separate
   functions.
   *Recommendation*: use the variadic form for API simplicity; the two-function
   alternative is acceptable but adds surface area. Non-blocking.
