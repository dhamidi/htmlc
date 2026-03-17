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
4. **Tree-structural failure reporting** so test output pinpoints the exact differing
   node path in the HTML tree rather than showing two full serialised strings.
5. **Extensibility via `SelectionChecker`**: callers can write custom assertion types
   that integrate with the same `t.Fatalf`/`t.Helper` reporting path as built-in
   assertions.

---

## 3. Non-Goals

1. **Browser / JavaScript rendering**: this RFC covers server-side HTML string
   output only. Interactive behaviour (click handlers, reactive state) is out of
   scope.
2. **XPath assertions**: the fluent `Query` builder covers the important cases;
   XPath is not planned.
3. **Parallel `t.Parallel()` management**: callers are responsible for calling
   `t.Parallel()` where desired. `Harness` does not call it automatically.
4. **Typed prop validation at compile time**: `Data` remains `map[string]any`.
   Static type-checked props are a separate concern.
5. **Integration with external test frameworks** (testify, gomega): the API uses
   the standard `testing.TB` interface only. Interoperability is not blocked, but
   not explicitly designed for.
6. **Snapshot / golden-file regression testing**: storing expected output in
   committed files makes the test suite stateful and is out of scope.
7. **Table-driven loop helper**: the fluent API makes each sub-test a one-liner;
   `RunCases` / `Case` are not included. Callers who want a loop write it themselves
   with `t.Run`.
8. **Pre-built assertion library**: `htmlctest` ships only the core assertions
   (`AssertExists`, `AssertText`, etc.). Domain-specific checkers are the caller's
   responsibility; `SelectionChecker` provides the integration point.

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

// Document returns the root *html.Node of the parsed rendered output.
// The document is parsed once and cached. Callers can walk the tree directly
// using golang.org/x/net/html without going through the Query/Selection API.
func (r *Result) Document() *html.Node

// --- Equality assertion ---

// AssertHTML asserts the rendered output equals want after normalising
// whitespace. On failure it reports a tree-structural diff.
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

// ByClass returns a Query matching elements that have the given CSS class.
func ByClass(class string) Query

// ByAttr returns a Query matching elements with attribute attr equal to value.
func ByAttr(attr, value string) Query

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

// Nodes returns the raw matched html.Node slice.
// Callers that want to write imperative assertions using t.Fatalf directly
// can obtain the node list and inspect it without going through the
// SelectionChecker interface.
func (s *Selection) Nodes() []*html.Node

// Check runs checker against the matched nodes.
// If checker.Check returns a non-nil error, the test fails with that error's
// message. Returns *Selection to allow chaining.
func (s *Selection) Check(checker SelectionChecker) *Selection
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

#### Query factory methods on `*Harness` and `*Result`

To avoid repeating the `htmlctest.` package qualifier at call sites, `*Harness` and
`*Result` each expose thin delegating wrappers over the package-level constructors:

```go
// pseudo-code — not implementation

// ByTag returns a Query matching elements with the given tag name.
// Identical to the package-level htmlctest.ByTag but avoids the package qualifier
// at call sites: h.Fragment(...).Find(h.ByTag("p")).
func (h *Harness) ByTag(name string) Query
func (h *Harness) ByClass(class string) Query
func (h *Harness) ByAttr(attr, value string) Query

// Same methods on *Result for callers who only have a result in scope.
func (r *Result) ByTag(name string) Query
func (r *Result) ByClass(class string) Query
func (r *Result) ByAttr(attr, value string) Query
```

These introduce no new logic; each delegates to the package-level constructor.
The package-level `ByTag`, `ByClass`, `ByAttr` remain exported for use in helper
functions that do not have a `*Harness` or `*Result` in scope.

### 4.4 Structured failure reporting via `AssertionFailure`

#### Current state

`AssertRendersHTML` and `AssertFragment` call `t.Fatalf` with:

```text
AssertFragment: got:
<full rendered html>

want:
<full expected html>
```

For large outputs this is unreadable. A string-level line diff is a poor fit
because the subject of every assertion is an HTML tree: a string diff obscures
structure and conflates whitespace differences with semantic differences.

#### Proposed extension

Introduce a private (unexported) interface:

```go
// pseudo-code — not implementation

// assertionFailure is implemented by each typed failure value.
// format returns the human-readable failure message for t.Fatalf.
type assertionFailure interface {
    format() string
}
```

Each assertion method constructs a typed failure value when the assertion does not
hold, then calls `r.t.Fatalf("%s", f.format())` (after `r.t.Helper()`). This
separates what-failed from how-to-report-it, and makes each assertion independently
testable by inspecting the failure value in unit tests for `htmlctest` itself.

Define one concrete failure type per assertion kind:

| Failure type | Fields | `format()` output |
|---|---|---|
| `textMismatch` | `want, got string`, `node *html.Node` | Shows element path, expected text, actual text — no raw diff |
| `attrMismatch` | `attr, want, got string`, `node *html.Node` | Shows element, attribute name, expected vs actual value |
| `existenceFailure` | `query Query`, `wantPresent bool` | Shows the query description and whether nothing / something was matched |
| `countMismatch` | `query Query`, `want, got int` | Shows query description, expected count, actual count |
| `htmlMismatch` | `want, got string`, `root *html.Node` | Shows a **tree-structural** comparison: tag paths that differ, not a line diff |

For `htmlMismatch`, the `format()` method walks both the want-parsed tree and the
got-parsed tree simultaneously, emitting lines like:

```text
AssertHTML: trees differ
  expected: <div class="card"><h2>Alice</h2></div>
  got:      <div class="card"><h2>Bob</h2></div>
  first difference at: div > h2 > #text
    want: "Alice"
    got:  "Bob"
```

This is implemented with zero external dependencies (only `golang.org/x/net/html`
and the standard library). The tree walk is a paired depth-first traversal.

**Verdict**: typed failure values provide tree-aware, human-readable output for
every assertion kind, with no external dependencies and independently testable
failure formatting logic.

### 4.5 User-defined assertions via `SelectionChecker`

#### Problem

There is no way for a caller to write a custom assertion (e.g.,
`AssertNoBrokenLinks`, `AssertAccessibleHeadings`) that receives the matched
`*html.Node` slice, reports failure through the same `t.Fatalf`/`t.Helper` path as
built-in assertions, and chains with the rest of the fluent API.

#### `SelectionChecker` interface

```go
// pseudo-code — not implementation

// SelectionChecker is implemented by user-defined assertion types.
// Check receives the matched nodes and returns a non-nil error if the assertion
// fails. The error message is used verbatim as the t.Fatalf argument, so it
// should be human-readable and include enough context.
//
// htmlctest calls t.Helper() before t.Fatalf so failure lines point to the
// caller, not into htmlctest itself.
type SelectionChecker interface {
    Check(nodes []*html.Node) error
}
```

`Selection.Check` (defined in §4.3) drives this interface. Callers can implement
domain-specific assertions and use them in the same fluent chain as built-in
assertions. See Example 4 in §6 for a complete worked example.

**Verdict**: a single-method interface is the minimal extension point that admits
arbitrary user logic, integrates with the existing failure path, and enables fluent
chaining — without requiring `htmlctest` to ship a large assertion library.

### 4.6 Migration stubs for removed free functions

The three existing free functions — `NewEngine`, `AssertRendersHTML`, and
`AssertFragment` — are not preserved as working code. Because there are no existing
users, backward compatibility has no value; forwarding wrappers would only clutter
the design and signal that the old API remains acceptable.

Instead, the three symbols are **kept as exported stubs** whose bodies call
`t.Helper(); t.Fatalf(...)` with a clear migration message:

```go
// pseudo-code — not implementation

// Deprecated: NewEngine is removed. Replace with:
//   h := htmlctest.NewHarness(t, files, opts...)
func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine {
    t.Helper()
    t.Fatalf("htmlctest.NewEngine is removed.\n" +
        "Replace with:\n" +
        "  h := htmlctest.NewHarness(t, files)\n" +
        "  h.Fragment(\"ComponentName\", data).AssertHTML(want)")
    return nil
}

// Deprecated: AssertRendersHTML is removed. Replace with:
//   h := htmlctest.NewHarness(t, files)
//   h.Page("ComponentName", data).AssertHTML(want)
func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
    t.Helper()
    t.Fatalf("htmlctest.AssertRendersHTML is removed.\n" +
        "Replace with:\n" +
        "  h := htmlctest.NewHarness(t, files)\n" +
        "  h.Page(\"%s\", data).AssertHTML(want)", name)
}

// Deprecated: AssertFragment is removed. Replace with:
//   h := htmlctest.NewHarness(t, files)
//   h.Fragment("ComponentName", data).AssertHTML(want)
func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
    t.Helper()
    t.Fatalf("htmlctest.AssertFragment is removed.\n" +
        "Replace with:\n" +
        "  h := htmlctest.NewHarness(t, files)\n" +
        "  h.Fragment(\"%s\", data).AssertHTML(want)", name)
}
```

The stubs compile (source-level compatibility) but fail immediately at test time
with a message that shows the exact replacement code, making migration mechanical.

**Verdict**: fail-fast stubs with migration instructions are preferable to silent
forwarding wrappers when there are no existing users. They make the break explicit
and self-documenting.

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
| `h.ByTag(name)`                                | `Query` matching by tag; avoids `htmlctest.` qualifier at call sites. |
| `h.ByClass(class)`                             | `Query` matching by CSS class; instance method on `*Harness`.        |
| `h.ByAttr(attr, value)`                        | `Query` matching by attribute; instance method on `*Harness`.        |
| `r.HTML()`                                     | Raw rendered string.                                                  |
| `r.Document()`                                 | Root `*html.Node` of the parsed rendered output (parsed once, cached).|
| `r.AssertHTML(want)`                           | Whole-output whitespace-normalized comparison with tree-structural diff.|
| `r.Find(q)`                                    | Return a `*Selection` of all nodes matching `Query` q.               |
| `r.ByTag(name)`                                | `Query` matching by tag; instance method on `*Result`.               |
| `r.ByClass(class)`                             | `Query` matching by CSS class; instance method on `*Result`.         |
| `r.ByAttr(attr, value)`                        | `Query` matching by attribute; instance method on `*Result`.         |
| `htmlctest.ByTag(name)`                        | `Query` matching elements by tag name (case-insensitive).            |
| `htmlctest.ByClass(class)`                     | `Query` matching elements by CSS class.                              |
| `htmlctest.ByAttr(attr, value)`                | `Query` matching elements by attribute value.                        |
| `q.WithClass(class)`                           | Extend `Query` to also require the given CSS class.                  |
| `q.WithAttr(attr, value)`                      | Extend `Query` to also require an attribute equals value.            |
| `q.Descendant(ancestor)`                       | Extend `Query` to only match descendants of `ancestor`.              |
| `s.AssertExists()`                             | Fail if no nodes were matched.                                        |
| `s.AssertNotExists()`                          | Fail if any node was matched.                                         |
| `s.AssertCount(n)`                             | Fail if the number of matched nodes is not `n`.                      |
| `s.AssertText(text)`                           | Fail if the first matched node's visible text is not `text`.         |
| `s.AssertAttr(attr, value)`                    | Fail if the first matched node's attribute is not `value`.           |
| `s.Nodes()`                                    | Return the raw `[]*html.Node` slice for imperative inspection.       |
| `s.Check(checker)`                             | Run a `SelectionChecker` against the matched nodes; chains.          |

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
    r.Find(r.ByTag("div").WithClass("card")).AssertExists()
    r.Find(r.ByTag("h2")).AssertText("Alice")
    r.Find(r.ByTag("img")).AssertAttr("src", "/img/alice.png")
    r.Find(r.ByTag("span").WithClass("badge")).AssertExists().AssertText("Admin")
}
```

Each assertion is independent: adding new elements to `UserCard.vue` does not
break the test, because only specific elements are checked. Instance methods
`r.ByTag`, `r.ByClass`, `r.ByAttr` avoid repeating the `htmlctest.` qualifier.

### Example 3 — Tree-structural failure output

When `AssertHTML` fails, the output identifies the first differing node path rather
than dumping two full strings:

```text
--- FAIL: TestUserCard (0.00s)
    htmlctest.go:47: AssertHTML: trees differ
          expected: <div class="card"><h2>Alice</h2></div>
          got:      <div class="card"><h2>Bob</h2></div>
          first difference at: div > h2 > #text
            want: "Alice"
            got:  "Bob"
```

### Example 4 — User-defined `SelectionChecker`

Callers can implement domain-specific assertions that integrate with the fluent
chain:

```go
// caller code — not part of htmlctest

// noEmptyAlt is a user-defined checker that fails if any matched <img> has an
// empty or missing alt attribute.
type noEmptyAlt struct{}

func (noEmptyAlt) Check(nodes []*html.Node) error {
    for _, n := range nodes {
        hasAlt := false
        for _, a := range n.Attr {
            if a.Key == "alt" {
                hasAlt = true
                if strings.TrimSpace(a.Val) == "" {
                    return fmt.Errorf("img element has empty alt attribute")
                }
            }
        }
        if !hasAlt {
            return fmt.Errorf("img element is missing alt attribute")
        }
    }
    return nil
}

func TestImagesHaveAlt(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Gallery.vue": `<template>
            <img :src="src" :alt="caption">
        </template>`,
    })

    r := h.Fragment("Gallery", map[string]any{"src": "/img/photo.jpg", "caption": "A photo"})
    r.Find(r.ByTag("img")).Check(noEmptyAlt{})
}
```

If the assertion fails, the test output is:

```text
--- FAIL: TestImagesHaveAlt (0.00s)
    htmlctest.go:82: img element has empty alt attribute
```

### Example 5 — Migration stub output

When an existing test calls one of the removed free functions, the test fails
immediately with a message showing the exact replacement:

```text
--- FAIL: TestLegacy (0.00s)
    htmlctest.go:12: htmlctest.AssertFragment is removed.
        Replace with:
          h := htmlctest.NewHarness(t, files)
          h.Fragment("Greeting", data).AssertHTML(want)
```

The old call site compiles without modification; the stub makes the required
migration mechanical and explicit.

---

## 7. Implementation Sketch

All new code lives in the `htmlctest` package. No changes to `htmlc` engine code
are required.

| File | Contents | Approx. lines |
|------|----------|--------------|
| `htmlctest/harness.go` | `Harness` struct, `NewHarness`, `Build`, `With`, `Engine`, `Page`, `Fragment`; plus `ByTag`, `ByClass`, `ByAttr` delegating methods (~15 additional lines) | ~75 |
| `htmlctest/result.go` | `Result` struct (holds `t testing.TB`, `html string`, lazy `*html.Node`); `HTML()`, `Document()`, `AssertHTML()`, `Find()`; plus `ByTag`, `ByClass`, `ByAttr` delegating methods (~15 additional lines) | ~55 |
| `htmlctest/query.go` | `Query` type and constructors (`ByTag`, `ByClass`, `ByAttr`, `WithClass`, `WithAttr`, `Descendant`); `Selection` type, assertion methods, `Nodes()`, `Check()`; `SelectionChecker` interface; recursive node-walk helpers | ~140 |
| `htmlctest/failure.go` | `assertionFailure` interface; concrete failure types (`textMismatch`, `attrMismatch`, `existenceFailure`, `countMismatch`, `htmlMismatch`) with `format()` methods; paired depth-first tree-walk for `htmlMismatch` | ~80 |
| `htmlctest/stubs.go` | Migration stubs for `NewEngine`, `AssertRendersHTML`, `AssertFragment` — each calls `t.Fatalf` with migration instructions | ~25 |

**`failure.go`**: `htmlMismatch.format()` performs a paired depth-first traversal
of the want-parsed and got-parsed `*html.Node` trees, emitting the path to the
first differing node. No external library; `golang.org/x/net/html` only.

**`query.go`**: the recursive node walk is a plain depth-first traversal of
`*html.Node`. `Query` matching uses `strings.EqualFold` for tag names and
`strings.Fields` for class tokenization. `Descendant` is checked by walking the
ancestor chain of candidate nodes. `SelectionChecker` is a single-method interface
defined here alongside `Selection`.

**New dependencies**: none. `golang.org/x/net/html` is already present as a
transitive dependency. It is listed as a direct dependency of `htmlctest`
(added to the `import` block) without adding a new module to `go.mod`.

---

## 8. Backward Compatibility

### `NewEngine`, `AssertRendersHTML`, `AssertFragment` (public)

These three functions are **not** preserved as working code. They remain as exported
symbols (source-level compatibility: existing code that references them still
compiles), but their bodies call `t.Fatalf` with a migration message. Any call site
that invokes them at test time will receive an immediate test failure with
instructions showing the exact replacement code.

This is intentional: there are no existing users, so backward compatibility has no
value. Fail-fast stubs make the migration path explicit and mechanical. See §4.6
for the stub implementations and Example 5 in §6 for sample output.

### New exports: `Harness`, `Result`, `Query`, `Selection`, `SelectionChecker`, `NewHarness`, `Build`, `ByTag`, `ByClass`, `ByAttr`

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
