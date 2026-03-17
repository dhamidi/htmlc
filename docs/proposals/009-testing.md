# RFC 009: htmlctest API Redesign — Fluent Harness, CSS Selectors, and Snapshots

- **Status**: Draft
- **Date**: 2026-03-17
- **Author**: TBD

---

## 1. Motivation

The current `htmlctest` package exposes three free functions for testing `htmlc`
components. While sufficient for the simplest cases, the API forces every assertion
to thread `t` and `e` manually, supports only whole-string HTML comparison, and
provides no mechanism for querying specific elements by CSS selector or for
snapshot-based regression testing.

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
- No CSS selector support (like Capybara's `have_selector`, `have_text`, `find`).
  There is no way to assert "the `.badge` element is present and contains 'Admin'"
  without asserting the full surrounding markup.
- No snapshot/golden-file workflow. Complex layouts with many elements require
  either hand-written expected strings (fragile) or no assertion at all.
- Table-driven tests need hand-rolled loops with repeated `htmlctest.AssertFragment`
  calls.
- Failure messages show raw strings side-by-side; there is no unified diff to
  pinpoint the changed region.

### Why the existing API cannot be extended in place

The three free-function signatures (`NewEngine`, `AssertRendersHTML`, `AssertFragment`)
require callers to pass `t` and `e` on every call. There is no object that
accumulates context between calls, so selector assertions and fluent chaining are
structurally impossible without introducing a new type.

---

## 2. Goals

1. **Introduce a `Harness` type** that owns the engine and captures `t`, eliminating
   manual threading of `t` and `e` to every assertion.
2. **Introduce a `Result` type** with fluent `Assert*` methods so assertions chain
   without intermediate variables.
3. **CSS-selector assertions**: `AssertSelector`, `AssertNoSelector`, `AssertCount`,
   `AssertText`, `AssertAttr`, `AssertAttrContains` — powered by `cascadia` +
   `golang.org/x/net/html`.
4. **Snapshot/golden-file workflow**: `AssertMatchesSnapshot` stores expected output
   in `testdata/snapshots/` and regenerates it with `-update`.
5. **Table-driven helper**: `RunCases` runs a `[]Case` slice as sub-tests, reducing
   the boilerplate loop to a single call.
6. **Unified diff in failure messages** so test output pinpoints the changed region
   rather than showing two full strings.
7. **Full backward compatibility**: the three existing free functions are preserved
   with identical signatures.

---

## 3. Non-Goals

1. **Browser / JavaScript rendering**: this RFC covers server-side HTML string
   output only. Interactive behaviour (click handlers, reactive state) is out of
   scope.
2. **XPath assertions**: CSS selectors via `cascadia` are sufficient; XPath is not
   planned.
3. **Parallel `t.Parallel()` management**: callers are responsible for calling
   `t.Parallel()` where desired. `Harness` does not call it automatically.
4. **Typed prop validation at compile time**: `RunCases.Data` remains
   `map[string]any`. Static type-checked props are a separate concern.
5. **Integration with external test frameworks** (testify, gomega): the API uses
   the standard `testing.TB` interface only. Interoperability is not blocked, but
   not explicitly designed for.

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
    AssertText("p", "Hello World!")
```

`Build` infers the component name as `"Root"`. When the caller passes a string
without a `<template>` wrapper, `Build` wraps it automatically. This is a
convenience for one-off unit tests; multi-component tests should use `NewHarness`.

**Verdict**: `Harness` is the only structural change required to eliminate `t`/`e`
threading. All `Assert*` methods are on `Result`, not on `Harness`, so `Harness`
stays small.

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
}

// --- Rendering entry points on Harness ---

// Page renders name as a full HTML page (wraps RenderPageString).
func (h *Harness) Page(name string, data map[string]any) *Result

// Fragment renders name as an HTML fragment (wraps RenderFragmentString).
func (h *Harness) Fragment(name string, data map[string]any) *Result

// --- Accessor ---

// HTML returns the raw rendered string.
func (r *Result) HTML() string

// --- Equality assertions ---

// AssertHTML asserts the rendered output equals want after normalising
// whitespace. On failure it prints a unified diff.
func (r *Result) AssertHTML(want string) *Result

// AssertContains asserts the rendered output contains the literal substring.
func (r *Result) AssertContains(fragment string) *Result

// --- CSS-selector assertions ---

// AssertSelector asserts at least one element matching sel is present.
func (r *Result) AssertSelector(sel string) *Result

// AssertNoSelector asserts no element matching sel is present.
func (r *Result) AssertNoSelector(sel string) *Result

// AssertCount asserts exactly n elements match sel.
func (r *Result) AssertCount(sel string, n int) *Result

// AssertText asserts the first element matching sel has the given
// visible text content (whitespace-normalised, child element text included).
func (r *Result) AssertText(sel, text string) *Result

// AssertAttr asserts the first element matching sel has attribute attr
// equal to value.
func (r *Result) AssertAttr(sel, attr, value string) *Result

// AssertAttrContains asserts attribute attr of the first element matching sel
// contains the substring value (useful for class lists).
func (r *Result) AssertAttrContains(sel, attr, value string) *Result

// --- Snapshot assertion ---

// AssertMatchesSnapshot compares the output against a golden file at
// testdata/snapshots/<name>.html relative to the calling test file.
// Pass -update to go test to write/overwrite the snapshot.
func (r *Result) AssertMatchesSnapshot(name string) *Result
```

Every `Assert*` method returns `*Result` to allow chaining. Internally, each
assertion calls `r.t.Helper()` before `r.t.Fatal(…)` so that failure lines point
to the call site, not into the `htmlctest` package.

**CSS selector evaluation**: `Result` parses the HTML string with
`golang.org/x/net/html` on first selector use (result is cached on `Result`).
`cascadia.Parse` compiles the selector; `cascadia.QueryAll` returns the matching
nodes. Text extraction walks the `html.Node` tree collecting `TextNode` data.

**Verdict**: a fluent `*Result` return is the cleanest way to enable chaining
without requiring callers to name intermediate variables.

### 4.3 Table-driven helper — `RunCases`

#### Current state

Table-driven tests require a manual `for _, tc := range cases { t.Run(tc.Name, ...) }`
loop that duplicates boilerplate across every test file.

#### Proposed extension

```go
// pseudo-code — not implementation

// Case is one entry in a table-driven component test.
type Case struct {
    Name     string         // sub-test name (passed to t.Run)
    Data     map[string]any // props to pass to the component
    Want     string         // expected HTML (normalised); mutually exclusive with Snapshot
    Snapshot string         // golden-file name; mutually exclusive with Want
}

// RunCases runs each Case as a t.Run sub-test against component on h.
// Pass page=true for full-page rendering; omit (or false) for fragment rendering.
func RunCases(t *testing.T, h *Harness, component string, cases []Case, page ...bool)
```

`RunCases` calls `t.Run(tc.Name, ...)` for each case. Inside the sub-test it calls
`h.Fragment` (or `h.Page` when `page[0]` is true) and then:
- If `tc.Snapshot != ""`: calls `r.AssertMatchesSnapshot(tc.Snapshot)`.
- Else: calls `r.AssertHTML(tc.Want)`.

`Want` and `Snapshot` are mutually exclusive. If both are non-empty `RunCases`
calls `t.Fatal` immediately.

**Verdict**: a single helper reduces 6-line boilerplate loops to a single
`RunCases` call while retaining sub-test granularity for `go test -run`.

### 4.4 Snapshot testing — `AssertMatchesSnapshot`

#### Current state

No golden-file mechanism exists. Large expected strings are embedded inline or
not tested at all.

#### Proposed extension

`AssertMatchesSnapshot(name string)` resolves the snapshot path as:

```text
<dir of calling test file>/testdata/snapshots/<name>.html
```

The calling test file's directory is determined via `runtime.Caller(1)`. This is
the same pattern used by `go test`'s `testdata` convention.

**Update flag**: `AssertMatchesSnapshot` reads the global `flag.Bool` named
`"update"`. Callers register it in `TestMain`:

```go
// pseudo-code — not implementation
var update = flag.Bool("update", false, "overwrite snapshot files")

func TestMain(m *testing.M) {
    flag.Parse()
    os.Exit(m.Run())
}
```

When `-update` is set:
- If the snapshot file does not exist, it is created with the current output.
- If it already exists, it is overwritten.
- The test is marked as passed (not failed).

When `-update` is not set:
- If the snapshot file does not exist, the test fails with an actionable message:
  `snapshot not found: run go test -update to create it`.
- If it exists, the output is compared after normalising whitespace. Differences
  are reported as a unified diff.

Snapshot files are stored under `testdata/snapshots/` which is the standard Go
convention for test fixture data; these files are committed to the repository.

**Verdict**: `runtime.Caller`-based path resolution is the established Go
pattern for test helpers that locate files relative to the calling test. Storing
snapshots in `testdata/snapshots/` follows the existing `go test` convention.

### 4.5 Unified diff in failure messages

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

A private `diff.go` file provides a `unifiedDiff(want, got string) string` helper
that computes a line-level diff. The diff library is selected from:

| Option                        | Notes                                                                    |
|-------------------------------|--------------------------------------------------------------------------|
| `github.com/google/go-cmp`    | ✅ widely used in Go test tooling, ✅ already common in Go ecosystems, ⚠️ not a diff-format library — provides structural diffs, not unified text diffs |
| `github.com/sergi/go-diff`    | ✅ produces unified text diffs, ✅ lightweight, ⚠️ less commonly audited  |

**Verdict**: `github.com/sergi/go-diff` produces the output format (`--- want`,
`+++ got`, `@@ … @@`) that test authors are accustomed to from `git diff`. Use
`go-diff`'s `diffmatchpatch.DiffMain` + `DiffToPretty` path for the failure
message in `AssertHTML`.

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
| `r.AssertHTML(want)`                           | Whole-output whitespace-normalized comparison with unified diff.      |
| `r.AssertContains(fragment)`                   | Substring presence assertion.                                         |
| `r.AssertSelector(sel)`                        | At least one element matches CSS selector.                            |
| `r.AssertNoSelector(sel)`                      | No element matches CSS selector.                                      |
| `r.AssertCount(sel, n)`                        | Exactly `n` elements match CSS selector.                              |
| `r.AssertText(sel, text)`                      | First matching element's visible text equals `text`.                  |
| `r.AssertAttr(sel, attr, value)`               | First matching element's attribute equals `value`.                    |
| `r.AssertAttrContains(sel, attr, value)`       | First matching element's attribute contains substring `value`.        |
| `r.AssertMatchesSnapshot(name)`                | Compare against golden file; regenerate with `-update`.               |
| `htmlctest.RunCases(t, h, comp, cases, page?)` | Run `[]Case` as sub-tests.                                            |

---

## 6. Examples

### Example 1 — One-line component test

```go
func TestGreeting(t *testing.T) {
    htmlctest.Build(t, `<p>Hello {{ name }}!</p>`).
        Fragment("Root", map[string]any{"name": "World"}).
        AssertText("p", "Hello World!")
}
```

No file map, no engine variable, no full-string comparison.

### Example 2 — Multi-component harness with selector assertions

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

    h.Fragment("UserCard", map[string]any{
        "name":      "Alice",
        "avatarURL": "/img/alice.png",
        "admin":     true,
    }).
        AssertSelector(".card").
        AssertText("h2", "Alice").
        AssertAttr("img", "src", "/img/alice.png").
        AssertSelector(".badge").
        AssertText(".badge", "Admin")
}
```

Each assertion is independent: adding new elements to `UserCard.vue` does not
break the test, because only specific selectors are checked.

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

### Example 4 — Snapshot testing

```
components/
  Layout.vue
  Nav.vue
  Footer.vue
testdata/
  snapshots/
    layout-logged-in.html   ← generated on first run
```

```go
func TestComplexLayout(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Layout.vue": `...`,
        "Nav.vue":    `...`,
        "Footer.vue": `...`,
    })

    h.Page("Layout", map[string]any{"title": "Home", "user": "alice"}).
        AssertMatchesSnapshot("layout-logged-in")
    // First run (go test -update): writes testdata/snapshots/layout-logged-in.html
    // Subsequent runs: compares; diff printed on mismatch
}
```

### Example 5 — Backward-compatible existing test (unchanged)

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
a unified diff if the output does not match.

---

## 7. Implementation Sketch

All new code lives in the `htmlctest` package. No changes to `htmlc` engine code
are required.

1. **`htmlctest/harness.go`** — `Harness` struct, `NewHarness`, `Build`, `With`,
   `Engine`, `Page`, `Fragment`. `Build` wraps the template string in
   `<template>…</template>` when no `<template>` tag is present (string contains
   check), registers it as `"Root.vue"`, and creates a `NewHarness` with a
   single-entry map. ~60 lines.

2. **`htmlctest/result.go`** — `Result` struct (holds `t testing.TB`, `html string`,
   lazy-parsed `*html.Node`). All `Assert*` methods. CSS selector evaluation uses
   `cascadia.Parse` + `cascadia.QueryAll`. Text extraction walks `html.Node`
   collecting `html.TextNode` data recursively. Each `Assert*` calls
   `r.t.Helper()` then `r.t.Fatal(…)` on failure and returns `r` on success.
   ~150 lines.

3. **`htmlctest/snapshot.go`** — `AssertMatchesSnapshot`. Resolves path with
   `runtime.Caller(2)` (two frames up: one for the method, one for the call site
   in the test). Creates `testdata/snapshots/` with `os.MkdirAll` if absent.
   Reads the `-update` flag registered in `init()` via `flag.Bool`. On `-update`,
   writes the current output; otherwise reads and diffs. ~70 lines.

4. **`htmlctest/cases.go`** — `Case` struct, `RunCases`. Validates `Want`/`Snapshot`
   mutual exclusion. Calls `t.Run` with `h.Fragment` or `h.Page` depending on
   the `page` variadic. ~40 lines.

5. **`htmlctest/compat.go`** — Backward-compat shims for `NewEngine`,
   `AssertRendersHTML`, `AssertFragment` as thin wrappers. ~25 lines.

6. **`htmlctest/diff.go`** — Private `unifiedDiff(want, got string) string` helper
   using `github.com/sergi/go-diff/diffmatchpatch`. ~20 lines.

**New dependencies** (add to `go.mod`):
- `github.com/andybalholm/cascadia` — CSS selector parsing and evaluation over
  `golang.org/x/net/html` nodes. `golang.org/x/net/html` is already a transitive
  dependency.
- `github.com/sergi/go-diff` — unified diff output for `AssertHTML` failure
  messages.

**Platform notes**: snapshot file paths use `filepath.Join` (OS-native separators)
since they touch the real filesystem, not `path.Join`. The `testdata/snapshots/`
convention is portable across UNIX and Windows.

---

## 8. Backward Compatibility

### `NewEngine` (public)

Signature is unchanged. Internally re-implemented as `NewHarness(t, files, opts...).Engine()`.
All callers are source-compatible and binary-compatible.

### `AssertRendersHTML` (public)

Signature is unchanged. Now delegates to `h.Page(name, data).AssertHTML(want)`.
Failure messages now include a unified diff; this is a strictly better output,
not a breaking change.

### `AssertFragment` (public)

Signature is unchanged. Now delegates to `h.Fragment(name, data).AssertHTML(want)`.
Same improved failure message as above.

### New exports: `Harness`, `Result`, `Case`, `NewHarness`, `Build`, `RunCases`

All new. No existing code references these identifiers.

### `go.mod` / `go.sum`

Two new indirect-to-test dependencies are added: `cascadia` and `go-diff`. These
are test-only (`htmlctest` is a test-helper package); they do not affect the
`htmlc` engine's dependency graph for production builds.

---

## 9. Alternatives Considered

### A. Extend the existing free functions with additional `...Option` parameters

Add functional options to `AssertFragment` for selector assertions:

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
htmlctest.AssertSelector(ctx, "UserCard", data, ".card")
htmlctest.AssertText(ctx, "UserCard", data, "h2", "Alice")
```

✅ No new method-receiver pattern; all functions remain free functions.
❌ Still requires a `ctx` variable at every call site.
❌ Cannot chain calls because free functions return nothing.
❌ `ctx` would need to carry render state (the last rendered `Result`) to avoid
   re-rendering for each assertion, which makes the API stateful in a surprising way.

**Rejected**: a context parameter reduces but does not eliminate the threading
problem and makes the result-sharing semantics implicit.

### C. Use `github.com/google/go-cmp` for diff output instead of `go-diff`

`go-cmp` produces structured diffs for Go values. For string comparison, it
would show the two strings as Go string literals with `−`/`+` markers.

✅ Already common in Go test codebases.
⚠️ Its diff output for long multi-line HTML strings is less readable than a
   unified text diff with line-level context; it does not produce `@@ … @@`
   context headers.
❌ Adds a heavier dependency than needed for a simple text diff.

**Rejected**: `go-diff` produces the unified format that developers expect when
diffing HTML output; `go-cmp` is better suited to structured Go value comparison.

### D. Place snapshot files relative to `os.Getwd()` rather than the calling file

Use the working directory (the module root when `go test ./...` is run) to anchor
snapshot paths.

✅ No `runtime.Caller` magic.
❌ The working directory is not stable across different `go test` invocations
   (e.g., `go test ./pkg/...` vs `cd pkg && go test`).
❌ Violates the `testdata` convention, which locates fixtures next to the test
   file that uses them.

**Rejected**: `runtime.Caller`-based path resolution is the established Go
pattern for helpers that locate test fixtures relative to the calling test.

### E. Require callers to register the `-update` flag themselves

Do not register `flag.Bool("update", …)` inside the `htmlctest` package. Require
callers to add a `TestMain` that calls `flag.Parse()` and passes the flag value
to `AssertMatchesSnapshot`.

✅ More explicit; no global flag registration.
❌ Every package using snapshots must add a `TestMain` boilerplate and a flag
   definition. The burden on the caller outweighs the benefit.

**Rejected**: registering the flag in `htmlctest`'s `init()` (as a package-level
side effect) is acceptable because it follows the same pattern used by `net/http/httptest`
and other standard library test helpers. The flag name `"update"` is conventional
and unlikely to conflict.

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

3. **CSS pseudo-classes in `cascadia`**: `cascadia` supports a subset of CSS Level
   3 selectors. Pseudo-elements (e.g., `::before`) and dynamic pseudo-classes
   (e.g., `:hover`) are not meaningful for server-rendered HTML and are not
   supported. Should the API document the supported selector subset?
   *Recommendation*: document the `cascadia` limitation in the `AssertSelector`
   godoc. Non-blocking.

4. **Snapshot diff format**: should `AssertMatchesSnapshot` normalise whitespace
   before comparing, or compare the raw rendered string?
   *Recommendation*: normalise whitespace (same as `AssertHTML`) so that
   inconsequential formatting changes do not trigger snapshot failures. However,
   the snapshot file should store the normalised form so that `-update` produces
   a readable, stable file. Blocking — must be decided before implementation to
   avoid inconsistent snapshot files.

5. **`RunCases` and `page` variadic**: the `page ...bool` variadic is slightly
   unusual. An alternative is `RunPageCases` / `RunFragmentCases` as two separate
   functions.
   *Recommendation*: use the variadic form for API simplicity; the two-function
   alternative is acceptable but adds surface area. Non-blocking.
