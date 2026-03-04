# Test Guidelines for htmlc

## Documentation quality (go doc -all)

Run `go doc -all github.com/dhamidi/htmlc` and verify each criterion below.
Treat any failure as a test failure — documentation quality is an acceptance criterion.

### Package overview (Diátaxis: explanation + reference)

- There is exactly **one** package-level doc comment. Multiple files contributing
  separate "Package htmlc …" sentences is a failure.
- The single package comment:
  - Explains in one sentence **what htmlc is** (a server-side Vue SFC renderer for Go)
  - Explains **why** it exists / the problem it solves (no JS runtime, scoped styles,
    standard net/http)
  - Describes the **main concepts** and how they relate: Engine → Component →
    Renderer → StyleCollector
  - Does **not** just repeat the package name ("Package htmlc provides htmlc.")

### Exported types (Diátaxis: reference)

For each exported type (`Engine`, `Options`, `Component`, `Renderer`, `Registry`,
`StyleCollector`, `StyleContribution`):

- The doc comment answers **what the type is**, not just what it stores.
- `Engine` must explain that it is the entry point for typical use.
- `Options` must explain each field's effect on runtime behaviour (not just restate
  the field name).
- `Renderer` and `Registry` should explain when a caller would use them directly
  (low-level API) vs. the Engine (high-level API).

### Exported functions and methods (Diátaxis: reference)

- Every exported function and method has a doc comment.
- Each comment states: what the function **does**, significant **parameters**,
  and what is returned or written on error.
- `RenderPage` and `RenderFragment` must explain the difference between them.
- `ServeComponent` must mention that it calls the data function on every request.

### Examples (Diátaxis: tutorial)

`go doc -all` must show at least one `Example` function demonstrating end-to-end
usage: create an Engine, register components, call RenderPage, write to
http.ResponseWriter. The example must be in an `_test.go` file so it appears in
`go doc` output and is verified by `go test`.
