# RFC 002: Template Integration

- **Status**: Draft
- **Date**: 2026-03-15
- **Author**: TBD

---

## 1. Motivation

`htmlc` and Go's standard `html/template` package have no interoperability bridge today. Adopting `htmlc` is currently an all-or-nothing decision: a project either rewrites every template at once or foregoes `htmlc`'s component model entirely.

### Scenario A — Vue-first project that needs stdlib compatibility

A project built with `htmlc` wants to hand off a rendered template to a Go library that accepts `*html/template.Template`. Common examples include email-sending libraries, PDF renderers, and framework middleware for response generation. There is currently no way to export an `htmlc` component as a stdlib template: `engine.go` exposes `RenderPage`, `RenderFragment`, `RenderPageString`, and `RenderFragmentString`, all of which write final rendered HTML to an `io.Writer` or return it as a string. None returns a reusable `*html/template.Template` that a library can call with its own data.

### Scenario B — stdlib-first project adopting htmlc

A mature Go web application owns hundreds of `html/template` partials — navigation bars, error pages, form elements — each integrated into framework middleware that injects request context, CSRF tokens, and session data. The team wants to start writing new UI features as `.vue` components while reusing existing templates as leaf components inside the `htmlc` tree. Today the only path is duplication: copy each template's output markup into a new `.vue` file, then keep both in sync.

### Why "just rewrite everything" is not viable

Template libraries in production applications are tightly coupled to framework middleware. Request-scoped data (CSRF tokens, user sessions, error pages) is injected by the framework before templates execute; changing the template engine means rewriting the middleware layer as well. This is a high-risk, high-cost migration that teams routinely defer indefinitely, which means `htmlc` is unavailable to the large population of existing Go web applications.

The goal of this RFC is to make the two systems interoperable incrementally: individual components can cross the boundary in either direction without requiring a full rewrite.

---

## 2. Goals

1. **`.vue` → `*html/template.Template`**: A root `.vue` component (and its statically-discovered sub-components) can be compiled to an `*html/template.Template` that stdlib `Execute` can call directly.
2. **`*html/template.Template` → htmlc component**: An existing `*html/template.Template` (including associated named `{{ define }}` blocks) can be registered in the `htmlc` engine as a virtual component, usable inside `.vue` component trees without materialising any new file on disk.
3. **CLI — `htmlc template vue-to-tmpl`**: Reads a `.vue` file and emits a Go `html/template`-compatible `.html` file to stdout, suitable for `go:embed` or direct file use.
4. **CLI — `htmlc template tmpl-to-vue`**: Reads a Go `html/template` file and emits a best-effort `.vue` component approximation to stdout.
5. **Documented interoperability path**: Developer-facing documentation describes how to use the two conversion directions together as a gradual migration strategy.

---

## 3. Non-Goals

1. **Source-text round-trip fidelity**: Rendered-output equivalence for constructs within the defined scope (all §4.1 rows that do not produce errors) IS a design target — `vue → tmpl → render(data)` must produce the same HTML as `vue → render(data)` for the same data, and `tmpl → vue → render(data)` must equal `tmpl → render(data)` for all constructs the `tmpl-to-vue` direction claims to support. Source-text identity after a round-trip is NOT a goal: whitespace, attribute ordering, and comment formatting may differ. Constructs outside the defined scope (complex expressions, arbitrary pipelines, `with` blocks, `$var` assignment) are not guaranteed to round-trip. Directives that support a partial mapping (`v-show`, `v-html`, `v-text`, `v-bind` spread, `v-switch`) still produce errors when used with complex expressions; only simple identifiers and dot-paths are supported.
2. **Client-side Vue.js features**: This RFC is strictly server-side. No JavaScript reactivity, `<script setup>`, or Composition API features are in scope.
3. **Automatic live sync**: Changes to a `.vue` file are not automatically propagated to a cached `*html/template.Template` at runtime; consumers re-compile explicitly.
4. **CSS/`<style>` block export**: Scoped styles are stripped in the vue-to-tmpl direction. Style handling is out of scope for the initial bridge.
5. **Changing the htmlc expression language**: This RFC does not alter the existing expression evaluator or add Go template action syntax to `.vue` files.
6. **Graceful degradation for unsupported `html/template` constructs in `tmpl-to-vue`**: unsupported constructs (arbitrary pipelines, `with` blocks, `$var` assignment, `{{ template "Name" expr }}` with non-`.` data) produce errors and halt conversion. The converter does not produce partial output. Templates containing unsupported constructs must be manually translated. This is a deliberate symmetry with `vue-to-tmpl`, which also errors rather than emitting partial output.

---

## 4. Proposed Design

### 4.1 Directive and Syntax Mapping Table

Before describing the API, this section establishes the mapping between the two template languages. The mapping is intentionally conservative: only constructs with unambiguous equivalents are translated; all others produce an error, aborting compilation.

Each entry below specifies the exact input that triggers it, the exact output produced, and what counts as unsupported (which produces an error).

| htmlc construct | `html/template` equivalent | Notes | Round-Trip Status |
|---|---|---|---|
| `{{ ident }}` (single identifier) | `{{ .ident }}` | Identifier is prefixed with `.` for dot-access | ✅ Lossless |
| `{{ a.b.c }}` (dot-path expression) | `{{ .a.b.c }}` | Dot-path is prefixed with `.` | ✅ Lossless |
| `{{ expr }}` (complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `:attr="name"` (simple identifier) | `attr="{{.name}}"` | Shorthand `:attr` with a simple identifier binding | ✅ Lossless |
| `:attr="a.b.c"` (dot-path) | `attr="{{.a.b.c}}"` | Shorthand `:attr` with a dot-path binding | ✅ Lossless |
| `v-bind:attr="name"` (simple identifier) | `attr="{{.name}}"` | Long-form equivalent of the shorthand above | ✅ Lossless |
| `v-bind:attr="a.b.c"` (dot-path) | `attr="{{.a.b.c}}"` | Long-form equivalent of the shorthand above | ✅ Lossless |
| `:attr="expr"` (complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `v-if="ident"` | `{{ if .ident }} … {{ end }}` | Condition must be a simple identifier or dot-path; complex expressions produce an error | ⚠️ Scope restriction — Go and htmlc/JS truthiness match for `bool`, non-empty `string`, non-zero numeric, and non-nil values; diverge for empty slices and maps (falsy in Go, truthy in JS/htmlc); restrict to types where truthiness is equivalent |
| `v-else` | `{{ else }}` | Direct equivalent | ✅ Lossless |
| `v-else-if="ident"` | `{{ else if .ident }}` | Same constraints as `v-if` | ⚠️ Scope restriction — same truthiness caveat as `v-if` |
| `v-for="item in list"` | `{{ range .list }} … {{ end }}` | Loop variable `item` translates to `.` inside the range block; loop body must not reference outer-scope variables; see §4.1.5 | ⚠️ Scope restriction — lossless only when the loop body does not reference outer-scope variables; see §4.1.5 |
| `v-show="ident"` (simple identifier or dot-path) | `style="{{ if not .ident }}display:none{{ end }}"` | Prepends `display:none` when falsy; merges with existing static `style` by emitting `{{ if not .ident }}display:none;{{ end }}<static-style>`; combined with dynamic `:style` produces an error; see §4.1.1 | ✅ Lossless — static literal `display:none` is not filtered by `html/template`'s CSS context |
| `v-show="expr"` (complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `v-html="ident"` (simple identifier or dot-path) | `<el>{{ .ident }}</el>` (all children replaced) | Data field must be `html/template.HTML`; plain `string` values are auto-escaped by `html/template`, diverging from htmlc semantics; see §4.1.2 | ⚠️ Data contract — caller must supply `html/template.HTML`; `html/template` passes `template.HTML` values through verbatim without additional escaping in an HTML body context |
| `v-html="expr"` (complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `v-text="ident"` (simple identifier or dot-path) | `<el>{{ .ident }}</el>` (all children replaced) | Direct equivalent; `html/template` auto-escapes output, matching `v-text` semantics; all existing children are discarded | ✅ Lossless |
| `v-text="expr"` (complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `v-bind="ident"` (argument-less spread, simple identifier or dot-path) | `<el {{ .ident }}>` | Data field must be `html/template.HTMLAttr` (pre-formatted attribute string); map-to-attribute conversion and class/style merging semantics are not preserved; see §4.1.3 | ⚠️ Data contract — caller must supply `html/template.HTMLAttr`; map-based spread semantics are not preserved |
| `v-bind="expr"` (argument-less spread, complex expression) | — | **Produces an error**; only simple identifiers and dot-paths are supported | — (error) |
| `<template v-switch="ident">` | `{{ if eq .ident … }}{{ else if eq .ident … }}{{ else }}{{ end }}` | Switch expression must be a simple identifier or dot-path; complex expressions produce an error; `<template>` element is not emitted; see §4.1.4 | ⚠️ Scope restriction — string case literals are lossless; numeric case literals require the caller to supply the switch value as Go `int` (not `float64`); see §4.1.4 |
| `<el v-case="literal">` | `{{ if eq .switchExpr literal }}` or `{{ else if eq .switchExpr literal }}` | String, number, and boolean literals are emitted as Go template literals; identifier and dot-path case expressions become `.ident`; equality uses Go's `eq`, not htmlc's JavaScript-style `==` | ⚠️ Data contract — string literals: ✅ lossless; numeric literals: caller must supply `int` to match Go template `eq`; see §4.1.4 |
| `<el v-default>` | `{{ else }}` | Direct equivalent; only the first `v-default` child is emitted; subsequent `v-default` children are silently dropped | ✅ Lossless |
| `v-switch` on non-`<template>` element | — | **Produces an error** | — (error) |
| Custom directives | — | **Produce an error**; no direct equivalent | — (error) |
| `<slot>` (default) | `{{ block "default" . }} … {{ end }}` | Overridable block | ⚠️ Scope restriction — fallback content renders identically when no caller override is provided; caller-override case (`<template #default>`) has no round-trippable equivalent in static tmpl output |
| `<slot name="N">` | `{{ block "N" . }} … {{ end }}` | Named block | ⚠️ Scope restriction — same as default slot |
| `<ComponentName>` | `{{ template "ComponentName" . }}` | Sub-component call; zero static props only — calls with static prop values produce an error; see §4.1.6 | ⚠️ Scope restriction — lossless only for components called with zero static props; see §4.1.6 |

#### 4.1.1 `v-show` — Style Injection

`v-show="ident"` evaluates the expression and, when falsy, prepends `display:none` to the element's `style` attribute. The mapping uses `html/template`'s `not` built-in inside the `style` attribute value.

**No existing `style` attribute:**

```html
<!-- htmlc input -->
<div v-show="visible">…</div>

<!-- html/template output -->
<div style="{{ if not .visible }}display:none{{ end }}">…</div>
```

**With existing static `style` attribute** — the static style is appended after the conditional `display:none`, so the existing declarations follow the injected one:

```html
<!-- htmlc input -->
<div v-show="visible" style="color:red">…</div>

<!-- html/template output -->
<div style="{{ if not .visible }}display:none;{{ end }}color:red">…</div>
```

**Combining `v-show` with `:style`** — the dynamic `:style` binding cannot be merged safely into a static template string; this combination **produces an error**.

**CSS context note**: `html/template` applies CSS-value filtering to dynamic `{{ .variable }}` outputs inside `style` attributes. The literal string `display:none` in the template source is static text, not a dynamic output; `html/template` writes it directly without filtering, making this pattern safe.

#### 4.1.2 `v-html` — Raw HTML Output

`v-html="ident"` discards all child nodes of the element and writes the expression value as raw, unescaped HTML. In `html/template`, a data field of type `template.HTML` is passed through verbatim in an HTML body context without escaping.

```html
<!-- htmlc input -->
<div v-html="content">fallback</div>

<!-- html/template output (fallback child discarded) -->
<div>{{ .content }}</div>
```

**Data contract change**: the caller must supply `.content` as `html/template.HTML`. If supplied as a plain `string`, `html/template` will HTML-escape it — producing visibly escaped markup — silently diverging from htmlc's unescaped behaviour. The generated template source should include a comment documenting this requirement:

```html
{{/* .content must be html/template.HTML; plain strings are auto-escaped */}}
<div>{{ .content }}</div>
```

#### 4.1.3 `v-bind` Spread — Pre-Formatted Attributes

Argument-less `v-bind="ident"` (no `:attr` suffix) evaluates the expression and spreads the result as attributes onto the element's opening tag. In `html/template`, a value of type `template.HTMLAttr` emitted between a tag name and `>` is written verbatim without escaping.

```html
<!-- htmlc input -->
<a v-bind="linkProps">text</a>

<!-- html/template output -->
<a {{ .linkProps }}>text</a>
```

**Data contract change**: in htmlc, `v-bind="obj"` accepts a `map[string]any` and performs per-key handling (class merging, style merging, boolean attribute toggling, per-key escaping). The `html/template` mapping requires the caller to pre-format the attributes as a single string of type `html/template.HTMLAttr`, for example:

```go
// In application code — not implementation
data["linkProps"] = template.HTMLAttr(`href="https://example.com" target="_blank"`)
```

None of htmlc's rich map semantics (class/style merging, boolean toggling) carry over; callers must handle all formatting themselves. If the caller supplies a plain `string`, `html/template` will escape `=` and `"` characters, producing broken HTML. The generated template source should include a comment:

```html
{{/* .linkProps must be html/template.HTMLAttr; plain strings produce broken HTML */}}
<a {{ .linkProps }}>text</a>
```

#### 4.1.4 `v-switch` / `v-case` / `v-default` — Conditional Switch

`v-switch` on a `<template>` element compiles to a chain of `{{ if eq … }}` / `{{ else if eq … }}` / `{{ else }}` / `{{ end }}` actions. The `<template>` wrapper element is not emitted.

```html
<!-- htmlc input -->
<template v-switch="tab">
  <div v-case="'home'">Home content</div>
  <div v-case="'settings'">Settings content</div>
  <div v-default>Default content</div>
</template>

<!-- html/template output -->
{{ if eq .tab "home" }}<div>Home content</div>{{ else if eq .tab "settings" }}<div>Settings content</div>{{ else }}<div>Default content</div>{{ end }}
```

**Case expression translation rules**:

| `v-case` expression | `html/template` equivalent |
|---|---|
| String literal `'home'` | `"home"` |
| Number literal `42` | `42` |
| Boolean literal `true` | `true` |
| Simple identifier `caseVar` | `.caseVar` |
| Dot-path `a.b` | `.a.b` |
| Complex expression | **Error** |

**Equality semantics**: htmlc's `==` uses JavaScript-style abstract equality (e.g. `null == undefined` is true; numeric string coercion applies). `html/template`'s `eq` uses Go's `==`, which requires comparable operands of the same type. Edge cases such as `float64(2) == int(2)` behave differently.

**Numeric type constraint**: the htmlc expression evaluator stores all numbers as `float64`. Numeric case literals in `.vue` source (e.g. `v-case="1"`) compile to Go integer literals in the template source (`{{ if eq .tab 1 }}`), where `1` is a Go `int`. If the caller supplies the switch value as `float64(1)` — as htmlc's own evaluator would — `html/template`'s `eq` will report `false` at runtime due to the type mismatch between `float64` and `int`. To avoid this, callers must supply the switch value as Go `int` when using numeric case literals. Alternatively, restrict numeric `v-case` usage to cases where the switch value is already a Go `int` in the data map. String case literals (`v-case="'home'"`) are unaffected by this constraint: Go `string` equality is type-safe. The `tmpl-to-vue` direction adds a comment to generated output documenting this restriction whenever numeric cases are present.

**`v-switch` on non-`<template>` elements**: already produces an error in the renderer; the compiler mirrors this.

**Multiple `v-default` elements**: only the first is emitted as the `{{ else }}` branch; subsequent `v-default` children are silently dropped, matching the renderer's behaviour.

**Nested `v-switch`**: each switch compiles to its own `{{ if }}` block; nesting works naturally.

#### 4.1.5 `v-for` — Loop Variable Handling

The `v-for` / `{{ range }}` mapping is the most semantically complex in this RFC because the two loop models differ in variable scoping and naming.

**Vue → tmpl direction**

`v-for="item in list"` generates `{{ range .list }}`. Inside the range body:

- References to the loop variable `{{ item }}` are translated to `{{ . }}` (the current range dot).
- References to `{{ item.field }}` are translated to `{{ .field }}`.
- References to any other outer-scope variable (e.g. `{{ title }}` from the parent data) are **flagged as errors**. Inside `{{ range }}`, `html/template` sets dot to the current element; the outer dot is inaccessible. Any loop body that references an outer-scope identifier that is not the loop variable cannot be translated faithfully and causes `vue-to-tmpl` to exit with a non-zero status.
- The index variable form `v-for="(item, index) in list"` is **out of scope** — it has no direct equivalent without `{{ range $i, $v := .list }}`; this form produces an error.

**tmpl → Vue direction**

`{{ range .list }} … {{ end }}` generates `<el v-for="item in list"> … </el>` (where `el` is the loop body's root element, or a `<template>` wrapper if the body has multiple sibling roots). Inside the range body:

- `{{ . }}` (bare dot) is translated to `{{ item }}`.
- `{{ .field }}` is translated to `{{ item.field }}`.
- The outer dot is inaccessible inside `{{ range }}` in `html/template`; this matches the constraint imposed in the Vue → tmpl direction — no outer-scope access is possible or promised.

**Nested loops**

Nested `v-for` loops require distinct loop variable names to avoid shadowing. The naming convention is:

- Outermost loop: `item`
- Second nesting level: `item2`
- Third nesting level: `item3`
- And so on (`item4`, `item5`, …)

In the tmpl → Vue direction, the converter tracks nesting depth and assigns variable names accordingly. In the Vue → tmpl direction, the Vue source already names its variables; the converter verifies that the innermost loop variable name is used consistently within its body and that no outer variable name (from an enclosing `v-for`) is referenced inside the inner loop body.

**Index variable**

`{{ range $i, $v := .list }}` is supported in `html/template` but has no direct htmlc equivalent. `tmpl-to-vue` produces an **error** for this form. `vue-to-tmpl` does not generate this form (the index is not accessible in a translated template).

**Concrete round-trip example**

tmpl input:

```html
<ul>{{ range .items }}<li>{{ .name }} ({{ .count }})</li>{{ end }}</ul>
```

tmpl → Vue output:

```html
<ul><li v-for="item in items">{{ item.name }} ({{ item.count }})</li></ul>
```

Vue → tmpl round-trip back:

```html
<ul>{{ range .items }}<li>{{ .name }} ({{ .count }})</li>{{ end }}</ul>
```

Rendered output for `data = {"items": [{"name": "Alice", "count": 3}]}` is identical at each stage:

```html
<ul><li>Alice (3)</li></ul>
```

The loop variable name `item` used in the intermediate Vue form does not collide with an outer-scope field named `item` because the `vue-to-tmpl` converter errors on any outer-scope reference from within the loop body — there is no path where `item` the field and `item` the loop variable are both accessed in the same body.

#### 4.1.6 `<ComponentName>` — Static Props

When a sub-component is called with static string prop values in htmlc — e.g. `<Card title="Welcome" body="Hello">` — those values cannot be forwarded to the generated `{{ template "Card" . }}` call, because `{{ template "Name" . }}` passes the entire parent dot unchanged; it has no mechanism for injecting additional key-value pairs into a new data scope without a helper function.

Three options were evaluated:

- ✅ **Option A — Restrict to zero static props**: calls with static prop values produce an **error** in `vue-to-tmpl`. The developer must move static values into the caller's data map before conversion. This is simple, consistent with the conservative mapping philosophy throughout §4.1, and avoids runtime dependencies.
- ⚠️ **Option B — Emit `{{ template "Name" (dict "k" "v" …) }}`**: uses a `dict` helper function registered on the template to construct a new map at render time. Requires the caller to register a `dict` function; adds a runtime dependency; the `dict` function is not part of `html/template`'s standard library, though it is common in template frameworks.
- ❌ **Option C — Inline the sub-component body**: substitutes the sub-component's template source inline instead of emitting a `{{ template }}` call. Eliminates the data-passing problem but loses the sub-component boundary and prevents deduplication when the same component is used multiple times.

**Verdict**: Option A. `vue-to-tmpl` produces an **error** for any `<ComponentName>` call that carries one or more static prop attributes. The error message identifies the component name and the offending prop, and suggests moving the static value into the caller's data map.

**Impact on round-trip status**: the `<ComponentName>` row in §4.1 carries ⚠️ Scope restriction — the rendered-output round-trip guarantee holds only for sub-component calls with zero static props.

**`tmpl-to-vue` direction**: `{{ template "Name" . }}` translates to `<Name />` (no props). The reverse mapping is unambiguous and lossless for the zero-static-prop case.

### 4.2 Go API — `.vue` → `*html/template.Template`

#### Current state

`engine.go` exposes `RenderPage`, `RenderPageContext`, `RenderFragment`, `RenderFragmentContext`, `RenderPageString`, and `RenderFragmentString`. All of these write final rendered HTML; none returns a reusable `*html/template.Template`. The `Engine` struct holds `entries map[string]*engineEntry` and `nsEntries map[string]map[string]*engineEntry`, both guarded by `mu sync.RWMutex`. The `engineEntry` type pairs a parsed `*Component` with the source path and modification time. There is no compilation path from a `*Component`'s AST to a `*html/template.Template`.

#### Proposed extension

Add two new exported methods to `*Engine`:

```go
// pseudo-code — not implementation

// ExportTemplate compiles the named component (and all sub-components
// reachable from it) into a *html/template.Template whose main template
// is named after the component. Associated sub-components are defined
// as named templates within the same set.
//
// The data contract of the returned template matches the props declared
// in the source .vue <script> block (if any); callers pass a map or
// struct with matching field names.
func (e *Engine) ExportTemplate(componentName string) (*html/template.Template, error)

// ExportTemplateSource is like ExportTemplate but returns the raw
// html/template source text instead of a parsed template. Useful for
// code generation, inspection, and testing.
func (e *Engine) ExportTemplateSource(componentName string) (string, error)
```

**Compilation steps** (high-level):

1. Acquire `e.mu` for reading; look up the component by name in `entries`.
2. Call the new internal function `compileToTemplateSource(entry *engineEntry, visited map[string]bool) string`, which walks the component's AST.
3. For each AST node, emit the `html/template` equivalent from the mapping table in §4.1.
4. Sub-component references (`<ComponentName>`) become `{{ template "ComponentName" . }}` calls; their source is recursively compiled by calling `compileToTemplateSource` on the referenced entry and appending the result as a `{{ define "ComponentName" }}…{{ end }}` block.
5. `<slot>` elements become `{{ block "slotName" . }}…{{ end }}`.
6. Unrecognised directives, complex expressions, or any construct with no `html/template` equivalent cause `compileToTemplateSource` to return an error immediately. No partial output is produced. The error includes the source file path and the offending construct.
7. The final assembled string is parsed with `html/template.New(name).Parse(src)`.

#### Evaluation: return type

Two options exist for the primary return value of `ExportTemplate`:

- ✅ **Parsed `*html/template.Template`**: caller can `Execute` immediately; satisfies library APIs that require a parsed template; errors surface at export time.
- ⚠️ **Parsed `*html/template.Template`**: compilation errors surface at export time rather than at engine boot — this is a deliberate behaviour change for callers who previously only called `RenderPage`.
- ✅ **Raw source text**: gives callers maximum flexibility; easier to inspect; no surprise parse errors.
- ❌ **Raw source text** as the primary return: requires a second public function or a struct return for the common case; makes the API awkward for library consumers.

**Verdict**: Return `*html/template.Template` directly from `ExportTemplate`. Add `ExportTemplateSource(componentName string) (string, error)` as a companion for callers that need the raw text (code generation, testing). `ExportTemplateSource` is the primitive; `ExportTemplate` calls it then parses the result.

### 4.3 Go API — `*html/template.Template` → htmlc component

#### Current state

Components are registered through two paths: (a) `discover` calls `registerPathLocked` for each `.vue` file found under `opts.ComponentDir`, and (b) `Register(name, path string)` provides a manual registration path. Both paths require a file on disk; there is no in-memory registration path. `registerPathLocked` reads the file, calls `ParseFile`, stores the result in `entries`, and (when `ComponentDir` is set) also populates `nsEntries`.

#### Proposed extension

Add a new exported method to `*Engine`:

```go
// pseudo-code — not implementation

// ImportTemplate wraps t as a virtual htmlc component registered under
// the template's name. The component accepts any data map and delegates
// rendering to t.Execute(w, data).
//
// Named templates defined within t (via {{ define "N" }}) are registered
// as separate components named "N", making them available as <N> tags
// within the htmlc component tree.
//
// ImportTemplate does not write any file to disk.
func (e *Engine) ImportTemplate(t *html/template.Template) error
```

**Implementation sketch**:

1. Acquire `e.mu` for writing (consistent with `registerPathLocked`'s locking discipline).
2. Create a **`syntheticComponent`** — a new unexported type wrapping a render function of the form `func(w io.Writer, data map[string]any) error`. Its `Render` method calls `t.Execute(w, data)`.
3. Wrap the `syntheticComponent` in an `engineEntry` with an empty path and zero `modTime` (hot-reload skips entries with no path).
4. Insert the entry into `e.entries` under `t.Name()`.
5. For each associated named template in `t.Templates()` (i.e. templates registered via `{{ define "N" }}`), create a separate `syntheticComponent` backed by that named template and register it under `"N"`.

**Naming collision**: if `e.entries` already contains an entry under `t.Name()`, `ImportTemplate` returns an error. A separate `ForceImportTemplate(t *html/template.Template) error` method is added for explicit overwrite (see §10 for discussion).

**Threading**: `ImportTemplate` acquires `e.mu` for writing, consistent with `Register`. The `mu sync.RWMutex` field already guards all writes to `entries` and `nsEntries`. No new locking is required.

**`nsEntries` impact**: synthetic components have no filesystem path and therefore no `relDir`. They are not inserted into `nsEntries`. They remain globally accessible via the flat `entries` fallback in `resolveComponent`, consistent with the behaviour of manually registered components via `Register`.

### 4.4 CLI — `template` subcommand group

#### Current state

`cmd/htmlc/main.go` dispatches on a command name string via the `cmds` map (of type `map[string]cmdFn`):

```go
cmds := map[string]cmdFn{
    "render": runRender,
    "page":   runPage,
    "props":  runProps,
    "ast":    runAst,
    "build":  runBuild,
}
```

Each subcommand is implemented in a separate `*_command.go` file under `cmd/htmlc/`. The `run` function dispatches to the appropriate `cmdFn` and writes errors to stderr.

#### Proposed extension

Add a new top-level subcommand `"template"` that itself dispatches to sub-subcommands:

```text
htmlc template <subcommand> [flags] [args]

Subcommands:
  vue-to-tmpl   Convert a .vue component to a Go html/template file
  tmpl-to-vue   Convert a Go html/template file to a .vue component
```

**`htmlc template vue-to-tmpl`**

```text
SYNOPSIS
  htmlc template vue-to-tmpl [-dir <components>] [-out <file>] <component-name>

FLAGS
  -dir string   Directory containing .vue components (default ".")
  -out string   Output file path. If omitted, writes to stdout.

DESCRIPTION
  Loads the named component from -dir, compiles it and all reachable
  sub-components to html/template syntax, and writes the result to -out
  (or stdout). The output is a valid Go html/template source file
  containing {{ define }} blocks for each sub-component.

  Constructs with no html/template equivalent (custom directives, complex
  expressions, etc.) cause the command to exit with a non-zero status and
  print an error to stderr identifying the offending construct, its source
  file, and its approximate location. No partial output is written.

  v-show, v-html, v-text, v-bind (spread), and v-switch are partially
  supported: they translate successfully when used with simple identifiers
  or dot-paths, and produce an error only when used with complex
  expressions.
```

**`htmlc template tmpl-to-vue`**

```text
SYNOPSIS
  htmlc template tmpl-to-vue [-out <file>] <template-file>

FLAGS
  -out string   Output .vue file path. If omitted, writes to stdout.

DESCRIPTION
  Reads <template-file> as a Go html/template source. Translates
  html/template actions to htmlc equivalents and emits a .vue Single
  File Component.

  Unsupported constructs (arbitrary pipelines, with blocks, $var
  assignment, {{ template "Name" expr }} with non-dot data, index
  variable range {{ range $i, $v := .list }}) cause the command to
  exit with a non-zero status and print an error to stderr. No partial
  output is written.

  The output file is prefixed with a generated-by comment.
```

**Inverse mapping table** — defines the `html/template` → htmlc translation for every supported action type:

| `html/template` action | htmlc / Vue equivalent | Notes |
|---|---|---|
| `{{ .field }}` | `{{ field }}` | Strip leading `.` |
| `{{ .a.b.c }}` | `{{ a.b.c }}` | Strip leading `.` on first segment |
| `{{ if .cond }} … {{ end }}` | `<el v-if="cond"> … </el>` or `<template v-if="cond"> … </template>` | Use `<template>` wrapper when the body has multiple sibling root elements |
| `{{ if .cond }} … {{ else }} … {{ end }}` | `v-if` + `v-else` pair | `{{ else }}` block wraps its root element(s) with `v-else` or `<template v-else>` |
| `{{ if .cond }} … {{ else if .cond2 }} … {{ end }}` | `v-if` + `v-else-if` chain | Translated recursively |
| `{{ if eq .x "a" }} … {{ else if eq .x "b" }} … {{ end }}` | `<template v-switch="x"><el v-case="'a'"> … </el><el v-case="'b'"> … </el></template>` | Pattern-matched as a `v-switch` chain when all branches are `eq` comparisons against the same left-hand variable |
| `{{ range .list }} … {{ end }}` | `<el v-for="item in list"> … </el>` | Loop body `{{ . }}` → `{{ item }}`; `{{ .field }}` → `{{ item.field }}`; nested ranges use `item2`, `item3`, …; see §4.1.5 |
| `{{ range $i, $v := .list }}` | — | **Produces an error**; no htmlc equivalent |
| `{{ template "Name" . }}` | `<Name />` | Sub-component call |
| `{{ block "N" . }} … {{ end }}` | `<slot name="N"> … </slot>` | Default content preserved as slot children |
| `{{ block "default" . }} … {{ end }}` | `<slot> … </slot>` | Unnamed (default) slot |
| `{{ define "N" }} … {{ end }}` | Separate component file | Cannot be inlined; **produces an error** with a message directing the author to extract into a separate `.vue` file |
| `{{ not .x }}` | Not directly usable as a directive expression | Only valid inside `v-if` / `v-else-if` values; translated as the condition |
| `{{ funcCall .arg }}` | — | **Produces an error**; arbitrary pipeline with function calls has no htmlc equivalent |
| `{{ $ }}` | — | **Produces an error**; root dot reference has no htmlc equivalent |
| `{{ $var := .field }}` | — | **Produces an error**; variable assignment has no htmlc equivalent |
| `{{ with .obj }} … {{ end }}` | — | **Produces an error**; `with` block has no htmlc equivalent (it differs from `v-if` by also rebinding dot) |
| `{{ template "Name" expr }}` (non-`.` data) | — | **Produces an error**; only `{{ template "Name" . }}` is translatable |

**`{{ if .cond }}` wrapping rule**: when a `{{ if .cond }}` block wraps multiple sibling elements with no single root element, `tmpl-to-vue` wraps them in a `<template v-if="cond">` element. This preserves the conditional without introducing a spurious HTML element. Example:

```html
<!-- html/template input -->
{{ if .showHeader }}<h1>Title</h1><p>Subtitle</p>{{ end }}

<!-- tmpl-to-vue output -->
<template v-if="showHeader"><h1>Title</h1><p>Subtitle</p></template>
```

**`{{ range }}` loop variable renaming rule**: inside a `{{ range .list }}` body, every occurrence of `{{ . }}` is renamed to `{{ item }}` and every occurrence of `{{ .field }}` is renamed to `{{ item.field }}`. Nested ranges increment the variable name suffix: the second nesting level uses `item2`, the third uses `item3`, and so on. See §4.1.5 for the full specification.

**`{{ if eq }}` chain pattern-matching rule**: a chain of `{{ if eq .x "a" }} … {{ else if eq .x "b" }} … {{ end }}` blocks is recognised as a `v-switch` pattern when all `if`/`else if` conditions use `eq` with the same left-hand variable and literal right-hand values. The chain is emitted as a `<template v-switch="x">` with `v-case` children. If any branch breaks the pattern (e.g. a non-`eq` condition, or a different left-hand variable), the entire chain is treated as a `v-if` / `v-else-if` sequence instead.

Both subcommands are implemented in a new file `cmd/htmlc/template_command.go`. The dispatch in `run()` adds `"template"` to the `cmds` map; the registered `cmdFn` extracts the first remaining argument as the sub-subcommand name and dispatches internally.

---

## 5. Syntax Summary

| Surface | Form | Description |
|---|---|---|
| Go API | `(*Engine).ExportTemplate(name string) (*html/template.Template, error)` | Compile component tree to stdlib template |
| Go API | `(*Engine).ExportTemplateSource(name string) (string, error)` | Emit raw template source text |
| Go API | `(*Engine).ImportTemplate(t *html/template.Template) error` | Register stdlib template as htmlc component |
| Go API | `(*Engine).ForceImportTemplate(t *html/template.Template) error` | Register stdlib template, overwriting any existing entry |
| CLI | `htmlc template vue-to-tmpl [-dir d] [-out f] <name>` | CLI wrapper for `ExportTemplateSource` |
| CLI | `htmlc template tmpl-to-vue [-out f] <file>` | CLI wrapper for tmpl→vue conversion |

---

## 6. Examples

### Example 1 — Export a simple card component

Project layout:

```text
components/
  Card.vue        ← accepts props: title, body
```

`Card.vue`:

```html
<template>
  <div class="card">
    <h2>{{ title }}</h2>
    <p>{{ body }}</p>
  </div>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Card
```

Expected stdout:

```html
{{ define "Card" }}
<div class="card">
  <h2>{{ .title }}</h2>
  <p>{{ .body }}</p>
</div>
{{ end }}
```

The interpolations `{{ title }}` and `{{ body }}` are translated to `{{ .title }}` and `{{ .body }}` by prefixing identifiers with `.` to match `html/template`'s dot-access convention.

### Example 2 — Export a component with a sub-component

Project layout:

```text
components/
  Card.vue        ← as in Example 1
  Page.vue        ← uses <Card>
```

`Page.vue`:

```html
<template>
  <main>
    <Card title="Welcome" body="Hello from Page" />
  </main>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Page
```

Expected stdout (both defines in one self-contained file):

```html
{{ define "Page" }}
<main>
  {{ template "Card" . }}
</main>
{{ end }}

{{ define "Card" }}
<div class="card">
  <h2>{{ .title }}</h2>
  <p>{{ .body }}</p>
</div>
{{ end }}
```

Running `vue-to-tmpl` on a root component walks all reachable sub-components recursively and appends a `{{ define "Name" }}…{{ end }}` block for each one, producing a self-contained file that `html/template.ParseFiles` can load directly.

### Example 3 — Import an existing stdlib template

```go
// In application code — not implementation
import (
    "html/template"
    htmlc "github.com/dhamidi/htmlc"
)

t := template.Must(template.ParseFiles("legacy/nav.html"))
engine, _ := htmlc.New(htmlc.Options{ComponentDir: "./components"})
engine.ImportTemplate(t)
// Now <Nav> is usable in any .vue component
html, _ := engine.RenderPageString("Home", map[string]any{"user": "Alice"})
```

`components/Home.vue`:

```html
<template>
  <!DOCTYPE html>
  <html>
    <body>
      <Nav user="{{ user }}" />
      <main>
        <h1>Welcome, {{ user }}</h1>
      </main>
    </body>
  </html>
</template>
```

When the renderer encounters `<Nav>`, it looks up `"Nav"` in `entries`, finds the synthetic component wrapping `t`, and calls `t.Execute(w, data)` with the current data scope. The legacy nav template renders as if it had been a native `.vue` component from the start. No changes to `legacy/nav.html` are required.

If `legacy/nav.html` contains named `{{ define }}` blocks (e.g. `{{ define "NavItem" }}`), `ImportTemplate` registers each as a separate synthetic component — `"NavItem"` becomes usable as `<NavItem>` in any `.vue` file.

### Example 4 — Round-trip check (supported constructs)

Starting with `components/Button.vue`:

```html
<template>
  <button v-if="enabled" :class="variant">{{ label }}</button>
</template>
```

Running `vue-to-tmpl`:

```text
htmlc template vue-to-tmpl -dir ./components Button
```

Produces:

```html
{{ define "Button" }}
{{ if .enabled }}<button class="{{.variant}}">{{ .label }}</button>{{ end }}
{{ end }}
```

The `v-if` translates cleanly to `{{ if .enabled }}`. The `:class="variant"` binding uses a simple identifier, so it translates to `class="{{.variant}}"`.

If the template used a complex expression such as `:class="isActive ? 'active' : ''"`, `vue-to-tmpl` would return an error:

```text
htmlc template vue-to-tmpl: Button.vue: :class="isActive ? 'active' : ''": complex expression not supported; only simple identifiers and dot-paths are allowed
```

Running `tmpl-to-vue` on the generated output:

```text
htmlc template tmpl-to-vue Button.html
```

Produces a `.vue` file with `v-if="enabled"` restored and the bound attribute approximated, with a header warning:

```html
<!-- generated by htmlc template tmpl-to-vue; review required -->
<template>
  <button v-if="enabled" :class="variant">{{ label }}</button>
</template>
```

This example illustrates the round-trip guarantee for supported constructs: `v-if` and simple identifier bindings survive the round-trip with identical rendered output. Unsupported constructs (complex expressions) produce errors at export time, forcing the developer to resolve them before proceeding — consistent with the rendered-output equivalence guarantee described in §3.

### Example 5 — Backward compatibility (no change for existing projects)

An existing `htmlc` project that calls only `RenderPage`, `RenderFragment`, `RenderPageString`, or `RenderFragmentString` and does not invoke `ExportTemplate`, `ExportTemplateSource`, `ImportTemplate`, `ForceImportTemplate`, or the `htmlc template` CLI subcommand sees no change in behaviour. No existing method signatures are modified. No existing CLI subcommands (`render`, `page`, `props`, `ast`, `build`) change. The `entries`, `nsEntries`, and `mu` fields of `Engine` are unchanged.

### Example 6 — `v-show` visibility toggle

`Tooltip.vue`:

```html
<template>
  <div class="tooltip" v-show="visible" style="position:absolute">
    {{ message }}
  </div>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Tooltip
```

Expected stdout:

```html
{{ define "Tooltip" }}
<div class="tooltip" style="{{ if not .visible }}display:none;{{ end }}position:absolute">
  {{ .message }}
</div>
{{ end }}
```

When `.visible` is falsy, the rendered output includes `display:none;position:absolute` in the `style` attribute. When truthy, only `position:absolute` appears. The static style value follows the conditional injection, matching the htmlc renderer's prepend-then-append behaviour.

### Example 7 — `v-html` raw HTML insertion

`RichText.vue`:

```html
<template>
  <article v-html="body">Loading…</article>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components RichText
```

Expected stdout:

```html
{{ define "RichText" }}
{{/* .body must be html/template.HTML; plain strings are auto-escaped */}}
<article>{{ .body }}</article>
{{ end }}
```

The child node `Loading…` is discarded. The caller must supply `.body` as `html/template.HTML`:

```go
// In application code — not implementation
import "html/template"

data := map[string]any{
    "body": template.HTML("<p>Hello <strong>world</strong></p>"),
}
engine.ExportTemplate("RichText") // then Execute with data
```

If `.body` is a plain `string`, `html/template` will HTML-escape it, producing `&lt;p&gt;Hello…&lt;/p&gt;` — visibly broken markup. The generated comment reminds callers of this contract.

### Example 8 — `v-text` text content replacement

`Badge.vue`:

```html
<template>
  <span class="badge" v-text="count">0</span>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Badge
```

Expected stdout:

```html
{{ define "Badge" }}
<span class="badge">{{ .count }}</span>
{{ end }}
```

The fallback child `0` is discarded; `.count` is HTML-escaped by `html/template`, matching `v-text`'s behaviour exactly. No data contract change: `.count` can be any type whose string representation is safe to display.

### Example 9 — `v-bind` spread attributes

`Link.vue`:

```html
<template>
  <a v-bind="attrs">click here</a>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Link
```

Expected stdout:

```html
{{ define "Link" }}
{{/* .attrs must be html/template.HTMLAttr; plain strings produce broken HTML */}}
<a {{ .attrs }}>click here</a>
{{ end }}
```

The caller must supply `.attrs` as `html/template.HTMLAttr`:

```go
// In application code — not implementation
import "html/template"

data := map[string]any{
    "attrs": template.HTMLAttr(`href="https://example.com" target="_blank"`),
}
```

Map-based attribute spreading (class merging, style merging, boolean toggling) is not available through this mapping. Callers that need rich spread semantics must pre-format the attribute string themselves.

### Example 10 — `v-switch` tab switcher

`Tabs.vue`:

```html
<template>
  <section>
    <template v-switch="activeTab">
      <div v-case="'home'"><h2>Home</h2><p>Welcome.</p></div>
      <div v-case="'profile'"><h2>Profile</h2><p>Your settings.</p></div>
      <div v-default><h2>Not found</h2></div>
    </template>
  </section>
</template>
```

Command:

```text
htmlc template vue-to-tmpl -dir ./components Tabs
```

Expected stdout:

```html
{{ define "Tabs" }}
<section>
{{ if eq .activeTab "home" }}<div><h2>Home</h2><p>Welcome.</p></div>{{ else if eq .activeTab "profile" }}<div><h2>Profile</h2><p>Your settings.</p></div>{{ else }}<div><h2>Not found</h2></div>{{ end }}
</section>
{{ end }}
```

The `<template v-switch>` wrapper is not emitted. Each `v-case` string literal is translated to a Go double-quoted string. The `v-default` branch becomes the trailing `{{ else }}` block.

**Type note**: when the switch expression is numeric (e.g. `v-switch="page"` with `v-case="1"`), callers must supply `.page` as Go `int` (not `float64`) to avoid a type mismatch with `html/template`'s `eq`. See §4.1.4 for the full numeric type constraint.

### Example 11 — Full round-trip fidelity demonstration

This example demonstrates rendered-output equivalence in both directions for constructs within the defined scope.

**Step 1 — Source `.vue` component** (`components/ArticleList.vue`):

```html
<template>
  <section>
    <h1>{{ title }}</h1>
    <ul v-if="hasItems">
      <li v-for="item in items">{{ item.name }}</li>
    </ul>
    <p v-else>No items.</p>
  </section>
</template>
```

The component uses `{{ ident }}` interpolation, `v-if` / `v-else`, and `v-for` — all within the defined round-trip scope. The `v-for` loop body references only the loop variable `item`; no outer-scope variables are accessed inside the loop.

**Step 2 — `vue-to-tmpl` output**:

```text
htmlc template vue-to-tmpl -dir ./components ArticleList
```

```html
{{ define "ArticleList" }}
<section>
  <h1>{{ .title }}</h1>
  {{ if .hasItems }}<ul>
    {{ range .items }}<li>{{ .name }}</li>{{ end }}
  </ul>{{ else }}<p>No items.</p>{{ end }}
</section>
{{ end }}
```

Translation notes:
- `{{ title }}` → `{{ .title }}` (identifier prefixed with `.`)
- `v-if="hasItems"` → `{{ if .hasItems }}` / `{{ else }}` / `{{ end }}`
- `v-for="item in items"` → `{{ range .items }}`
- `{{ item.name }}` → `{{ .name }}` (loop variable `item` replaced with dot)

**Step 3 — Rendered output equivalence**

Data `D`:

```json
{"title": "News", "hasItems": true, "items": [{"name": "First"}, {"name": "Second"}]}
```

Rendering the original `.vue` with data `D` via `htmlc`:

```html
<section>
  <h1>News</h1>
  <ul>
    <li>First</li><li>Second</li>
  </ul>
</section>
```

Rendering the generated `html/template` with data `D` via `html/template.Execute`:

```html
<section>
  <h1>News</h1>
  <ul>
    <li>First</li><li>Second</li>
  </ul>
</section>
```

The rendered HTML is identical. (Whitespace within the template source may differ, but the HTML content is byte-for-byte equivalent.)

**Step 4 — `tmpl-to-vue` round-trip back**:

```text
htmlc template tmpl-to-vue ArticleList.html
```

```html
<!-- generated by htmlc template tmpl-to-vue; review required -->
<template>
  <section>
    <h1>{{ title }}</h1>
    <template v-if="hasItems"><ul>
      <li v-for="item in items">{{ item.name }}</li>
    </ul></template><p v-else>No items.</p>
  </section>
</template>
```

The round-tripped `.vue` differs from the original in source formatting (the `v-if` / `v-else` pair uses a `<template v-if>` wrapper because the original `{{ if }}` block had multiple sibling elements in the else branch), but the rendered output for data `D` is again identical:

```html
<section>
  <h1>News</h1>
  <ul>
    <li>First</li><li>Second</li>
  </ul>
</section>
```

**Constructs deliberately excluded from this example**: static props on sub-components (`<Card title="Welcome">`), outer-scope variable access inside `v-for`, and numeric `v-switch` cases — these all fall outside the round-trip scope and would produce errors if used.

---

## 7. Implementation Sketch

This section describes Go-level changes at a high level. Full implementations are out of scope.

### `component.go`

1. Add a new unexported **`syntheticComponent`** type:

   ```go
   // pseudo-code — not implementation
   type syntheticComponent struct {
       name   string
       render func(w io.Writer, data map[string]any) error
   }
   ```

   `syntheticComponent` implements the same internal render interface used by `renderComponent` in `engine.go`. Its `render` field delegates directly to `t.Execute(w, data)` for the wrapped `*html/template.Template`.

2. No changes to the `Component` struct or its parser (`ParseFile`, `extractSections`, `parseTemplateHTML`).

### `engine.go`

1. **`ExportTemplateSource(name string) (string, error)`** — new exported method:
   1. Acquire `e.mu.RLock`; look up `name` in `entries`; release lock.
   2. Call `compileToTemplateSource(entry, make(map[string]bool))`.
   3. Return the assembled source string.

2. **`ExportTemplate(name string) (*html/template.Template, error)`** — new exported method:
   1. Call `ExportTemplateSource(name)`.
   2. Parse the result with `html/template.New(name).Parse(src)`.
   3. Return the parsed template or the first error.

3. **`ImportTemplate(t *html/template.Template) error`** — new exported method:
   1. Acquire `e.mu.Lock`.
   2. If `entries[t.Name()]` is non-nil, return a `"component already registered"` error.
   3. For each template in `t.Templates()`, create a `syntheticComponent` and wrap it in an `engineEntry` with an empty path.
   4. Insert each entry into `entries` under the template's name.
   5. Release lock.

4. **`ForceImportTemplate(t *html/template.Template) error`** — new exported method: same as `ImportTemplate` but skips the collision check at step 2.

5. **`compileToTemplateSource(entry *engineEntry, visited map[string]bool) (string, error)`** — new private function (~100–200 lines):
   - Walks `entry.comp.Template` (the `*html.Node` tree stored by `ParseFile`).
   - For each node type, emits the `html/template` equivalent from the mapping in §4.1.
   - Tracks `visited` to prevent infinite recursion in circular component graphs.
   - Sub-component references recurse into `entries` to compile the referenced component's source as a `{{ define }}` block.
   - Returns an error for any unrecognised directive, unsupported construct, or complex expression that cannot be translated.

### `renderer.go`

No changes. Synthetic components bypass the renderer entirely and write directly to the `io.Writer` via their `render` function. The engine's existing `renderComponent` method checks whether the resolved entry holds a `syntheticComponent` and, if so, calls its `render` function in place of constructing a `Renderer`.

### `cmd/htmlc/template_command.go` (new file)

1. **`runTemplateCommand(args []string, stdout, stderr io.Writer, strict bool) error`** — top-level dispatcher:
   - If `len(args) == 0`, print usage and return an error.
   - Dispatch on `args[0]`: `"vue-to-tmpl"` → `runVueToTmpl`; `"tmpl-to-vue"` → `runTmplToVue`; otherwise print unknown-subcommand error.

2. **`runVueToTmpl(args []string, stdout, stderr io.Writer, strict bool) error`**:
   - Parses `-dir` (default `"."`) and `-out` flags.
   - Creates an engine with `htmlc.New(htmlc.Options{ComponentDir: dir})`.
   - Calls `engine.ExportTemplateSource(name)`.
   - On success, writes the result to `-out` (or `stdout` if `-out` is omitted).
   - On error (unsupported construct encountered), writes the error to `stderr` and exits with a non-zero status. No partial output is written.

3. **`runTmplToVue(args []string, stdout, stderr io.Writer, strict bool) error`**:
   - Parses `-out` flag.
   - Calls `html/template.ParseFiles(args[0])` to load the template.
   - Calls `convertTmplToVue(t *html/template.Template) string` — a new private function that performs the inverse mapping.
   - Writes the result (prefixed with the review-required comment) to `-out` or `stdout`.

### `cmd/htmlc/main.go`

One change: add `"template": runTemplateCommand` to the `cmds` map. No other modifications.

### Platform notes

All file path operations in `template_command.go` use `filepath` (not `path`) for OS portability, consistent with the rest of `cmd/htmlc/`. The `compileToTemplateSource` function in `engine.go` uses `path` (forward slashes) when computing component lookup keys, consistent with how `nsRelDir` and `relDirForPath` handle FS-relative paths.

---

## 8. Backward Compatibility

### `(*Engine).RenderPage`, `RenderPageContext`, `RenderFragment`, `RenderFragmentContext`, `RenderPageString`, `RenderFragmentString`

No change in signature or semantics.

### `(*Engine).Register(name, path string) error`

No change. Manual registration via file path continues to work exactly as today.

### `(*Engine).ValidateAll() []ValidationError`

No change in signature. `ValidateAll` uses `entries` for validation; synthetic components inserted by `ImportTemplate` will appear in `entries` and be considered valid component targets. No false positives are introduced.

### `Engine` struct fields (`entries`, `nsEntries`, `mu`, `opts`, etc.)

All unexported. No public impact.

### `Options` struct

No new fields. `New(opts Options)` is unchanged.

### CLI — existing subcommands (`render`, `page`, `props`, `ast`, `build`)

Unchanged. Adding `"template"` to the `cmds` map cannot conflict with existing subcommand names. The `run` function's dispatch logic is unaffected for all existing subcommand names.

### CLI — `htmlc template` called without a sub-subcommand

Prints usage and exits with a non-zero status, consistent with how other subcommands handle missing required arguments (e.g. `htmlc render` without a component name).

### `(*Engine).ExportTemplate`, `ExportTemplateSource`, `ImportTemplate`, `ForceImportTemplate`

All four are new methods. No existing method signatures change. Adding methods to `*Engine` is backward-compatible in Go.

---

## 9. Alternatives Considered

### A — Emit `text/template` instead of `html/template`

`text/template` and `html/template` share identical syntax. Emitting `text/template` would avoid an import-path choice and make the output usable with either package.

Rejected: `html/template` is the correct choice for HTML output because it provides automatic contextual escaping. Using `text/template` for HTML content silently removes XSS protection that `html/template` provides. Any caller rendering the output in a browser would be exposed to injection attacks. This is an unacceptable security regression.

### B — Return raw source text from `ExportTemplate` instead of a parsed template

Returning a string gives callers maximum flexibility and avoids runtime parse errors surfacing at export time. This was considered seriously because it keeps `ExportTemplate` pure (no stdlib dependency at the call site) and makes the output inspectable.

Rejected as the sole primary return: the common use case — handing off to a library that accepts `*html/template.Template` — requires a parsed value. Two separate entry points (`ExportTemplate` and `ExportTemplateSource`) cover both use cases without forcing callers to parse manually.

### C — Implement `.vue` → `html/template` at the renderer level (reuse `renderer.go`)

`renderer.go` already walks the `*Component` AST. Reusing the renderer's AST traversal for template source generation would avoid duplicating traversal logic.

Rejected: the renderer is tightly coupled to writing final HTML output — it evaluates expressions, resolves props, and renders slots with concrete data. Generating template source requires emitting *unevaluated* placeholder actions (`{{ .field }}`) instead of concrete values. Reusing the renderer would require threading a "generation mode" flag through its internal state machine, introducing significant complexity and coupling. A dedicated `compileToTemplateSource` function that traverses the same `*html.Node` tree with a different emission strategy is cleaner.

### D — File-based `ImportTemplate` (accept a `.html` path instead of `*html/template.Template`)

Accepting a file path mirrors the signature of `Register(name, path string)` and is simpler to document.

Rejected: coupling the API to the filesystem contradicts the "no materialise to disk" goal and makes the API harder to test and use in memory-only environments — for example, when templates are loaded from `embed.FS` or generated programmatically. Accepting a `*html/template.Template` directly is idiomatic Go and more composable.

### E — New `.vue` directive `v-import-template` for in-template stdlib imports

A new directive could allow `.vue` files to declare a dependency on a stdlib template inline:

```html
<!-- hypothetical — not proposed -->
<template v-import-template="legacy/nav.html">
```

This would avoid a Go API change and keep the import declaration visible in the template source.

Rejected: this would add a directive whose semantics require Go-level side effects (reading a file, parsing a stdlib template) at parse time, violating the engine's separation between parsing and rendering. The `Component` parser (`ParseFile`, `extractSections`) has no access to the engine's registry. Implementing this would couple the parser to the engine, which is a deeper architectural change than the Go API approach. The Go API (`ImportTemplate`) is explicit, testable, and consistent with how `Register` works.

---

## 10. Open Questions

1. **Sub-component recursion depth** (`non-blocking`): `ExportTemplate` walks sub-components recursively via `compileToTemplateSource`. Should there be a depth limit to prevent accidental infinite recursion in pathological component graphs (e.g. a component that references itself)? Tentative recommendation: track `visited map[string]bool` as proposed, which already prevents cycles; add a `MaxDepth` option (default 50) consistent with other tree-walking limits in the engine.

2. **Naming collision on `ImportTemplate`** (`blocking`): If `ImportTemplate` is called with a template whose name matches an already-registered component (whether auto-discovered or manually registered via `Register`), should it error, silently overwrite, or require an explicit opt-in? Recommendation: error by default to prevent silent shadowing; provide `ForceImportTemplate` for deliberate overwrite. This aligns with Go's general principle of explicit over implicit and must be decided before implementation because it defines the observable API contract.

3. **`tmpl-to-vue` output quality** (`non-blocking`): `html/template` action syntax is more terse than `htmlc` directives, and the conversion from tmpl→vue is inherently lossy. Should `runTmplToVue` produce a warning header in the output file reminding the developer that manual review is required? Tentative recommendation: yes, emit `<!-- generated by htmlc template tmpl-to-vue; review required -->` at the top of the output. This is low-cost, visible, and prevents the false impression that the conversion is lossless.

4. **`<ComponentName>` with static props translation** (`blocking`): How should `vue-to-tmpl` handle a sub-component call that carries one or more static prop attributes (e.g. `<Card title="Welcome">`)? Three options are described in §4.1.6. **Verdict**: Option A — produce an error, require the developer to move static values into the caller's data map. This is the simplest approach, consistent with the conservative mapping philosophy, and avoids runtime dependencies. Resolved by §4.1.6; recorded here for traceability.

5. **`tmpl-to-vue` error-on-unsupported vs. partial output with markers** (`blocking`): Should `tmpl-to-vue` error on unsupported constructs (no partial output, non-zero exit) or emit HTML comments for them (partial output with `<!-- tmpl: … -->` markers) to support gradual migration? The rendered-output round-trip fidelity goal requires erroring — a partial output with markers does not round-trip. The "gradual migration" use case benefits from partial output. **Verdict**: error on unsupported constructs, no partial output. This is consistent with `vue-to-tmpl` behaviour and with Non-Goal 6 (§3). Resolved here; recorded for traceability. A `--permissive` flag may be added in a future RFC to opt out of the round-trip guarantee and enable partial output with markers (see question 6).

6. **`--permissive` flag for `tmpl-to-vue`** (`non-blocking`): Should a `--permissive` flag be added to `tmpl-to-vue` that falls back to `<!-- tmpl: original action -->` HTML comments instead of erroring, explicitly opting out of the round-trip guarantee? This would serve the "I just want a starting point" gradual-migration use case without compromising the default strict mode. Tentative recommendation: defer to a follow-up RFC; the default strict mode must be established first before introducing an opt-out.

---

## 11. Resolved Design Decisions

The following questions were raised during review and are now resolved.

### Prop introspection for imported templates

**Resolution**: feasible, non-blocking. The Go stdlib exports the parse tree of any `*html/template.Template` via the `text/template/parse` package. Walking the tree and collecting all `{{.field}}` accesses yields the set of props the template expects — the same strategy used by `Component.Props()` for native `.vue` components. `ImportTemplate` can populate a prop list that `ValidateAll` and other introspection tools consume without any new file-system dependency.

### Expression translation and unsupported features

**Resolution**: unsupported features produce **errors**, not warnings or HTML comments. `compileToTemplateSource` returns an error immediately on the first untranslatable construct. `runVueToTmpl` exits with a non-zero status and writes one error line to stderr per untranslatable construct, including the source file path and the offending construct. No partial output is written. The CLI error format follows the pattern established in §4.4.
