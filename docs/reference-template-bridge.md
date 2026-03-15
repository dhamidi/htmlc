# Reference: Template Bridge

Authoritative specification for the htmlc ↔ `html/template` conversion layer. No narrative prose beyond what is needed for precision.

---

## 3.1 Directive mapping table

Full mapping as defined in RFC 002 §4.1. Columns: htmlc construct | `html/template` equivalent | Round-trip status | Notes.

| htmlc construct | `html/template` equivalent | Round-trip status | Notes |
|---|---|---|---|
| `{{ ident }}` | `{{ .ident }}` | ✅ Lossless | Single identifier; prefixed with `.` |
| `{{ a.b.c }}` | `{{ .a.b.c }}` | ✅ Lossless | Dot-path; prefixed with `.` |
| `{{ expr }}` (complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `:attr="name"` | `attr="{{.name}}"` | ✅ Lossless | Shorthand bound attribute |
| `:attr="a.b.c"` | `attr="{{.a.b.c}}"` | ✅ Lossless | Dot-path bound attribute |
| `v-bind:attr="name"` | `attr="{{.name}}"` | ✅ Lossless | Long-form equivalent of `:attr` shorthand |
| `v-bind:attr="a.b.c"` | `attr="{{.a.b.c}}"` | ✅ Lossless | Long-form dot-path |
| `:attr="expr"` (complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `v-if="ident"` | `{{ if .ident }} … {{ end }}` | ⚠️ Scope restriction | Condition must be simple identifier or dot-path; truthiness diverges for empty slices/maps (see §3.7) |
| `v-else-if="ident"` | `{{ else if .ident }}` | ⚠️ Scope restriction | Same truthiness caveat as `v-if` |
| `v-else` | `{{ else }}` | ✅ Lossless | Direct equivalent |
| `v-for="item in list"` | `{{ range .list }} … {{ end }}` | ⚠️ Scope restriction | Loop variable `item` maps to `.` inside range; outer-scope variable references produce an error |
| `v-show="ident"` | `style="{{ if not .ident }}display:none{{ end }}"` | ✅ Lossless | Prepends `display:none` when falsy; merges with existing static `style` |
| `v-show="expr"` (complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `v-html="ident"` | `<el>{{ .ident }}</el>` | ⚠️ Data contract | Children discarded; field must be `html/template.HTML` (see §3.6) |
| `v-html="expr"` (complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `v-text="ident"` | `<el>{{ .ident }}</el>` | ✅ Lossless | Children discarded; `html/template` auto-escapes, matching `v-text` semantics |
| `v-text="expr"` (complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `v-bind="ident"` (spread) | `<el {{ .ident }}>` | ⚠️ Data contract | Field must be `html/template.HTMLAttr` (see §3.6) |
| `v-bind="expr"` (spread, complex) | — | ❌ Error | Only simple identifiers and dot-paths are supported |
| `<template v-switch="ident">` | `{{ if eq .ident … }} … {{ end }}` | ⚠️ Scope restriction | Expression must be simple identifier or dot-path; `<template>` element not emitted |
| `<el v-case="literal">` | `{{ if eq .switchExpr literal }}` / `{{ else if … }}` | ⚠️ Data contract | String literals: lossless; numeric literals: caller must supply Go `int` to match `eq` |
| `<el v-default>` | `{{ else }}` | ✅ Lossless | Only the first `v-default` child is emitted |
| `v-switch` on non-`<template>` | — | ❌ Error | `v-switch` is only valid on `<template>` elements |
| Custom directives (`v-xxx`) | — | ❌ Error | No equivalent; all unrecognised `v-` attributes produce an error |
| `<slot>` (default) | `{{ block "default" . }} … {{ end }}` | ⚠️ Scope restriction | Fallback content is lossless; caller-override has no round-trippable equivalent |
| `<slot name="N">` | `{{ block "N" . }} … {{ end }}` | ⚠️ Scope restriction | Named block; same caveat as default slot |
| `<ComponentName>` (zero props) | `{{ template "componentname" . }}` | ⚠️ Scope restriction | Sub-component calls with zero static props only; calls with any static prop values produce an error |

**tmpl→vue direction** (`TemplateToVue`):

| `html/template` construct | `.vue` equivalent | Notes |
|---|---|---|
| `{{ .ident }}` | `{{ ident }}` | Dot stripped |
| `{{ .a.b.c }}` | `{{ a.b.c }}` | Dot stripped |
| `{{ if .cond }} … {{ end }}` | `<div v-if="cond"> … </div>` | Synthetic `<div>` wrapper |
| `{{ if .cond }} … {{ else }} … {{ end }}` | `<div v-if="cond"> … </div><div v-else> … </div>` | Two synthetic `<div>` elements |
| `{{ range .items }} … {{ end }}` | `<ul><li v-for="item in items"> … </li></ul>` | Synthetic `<ul><li>` wrapper |
| `{{ block "name" . }} … {{ end }}` | `<slot name="name"> … </slot>` | `"default"` → `<slot>` |
| `{{ template "Name" . }}` | `<Name />` | Component call |
| `{{ .field \| func }}` | — | ❌ Error |
| `{{ with .x }}` | — | ❌ Error |
| `$x := .field` | — | ❌ Error |
| `{{ template "Name" expr }}` (non-`.` data) | — | ❌ Error |

---

## 3.2 `Engine.CompileToTemplate` reference

```
func (e *Engine) CompileToTemplate(componentName string) (*html/template.Template, error)
```

**Parameters**

| Parameter | Type | Description |
|---|---|---|
| `componentName` | `string` | Name of the root component to compile. Case-insensitive lookup; the returned template is named with the lowercased value. |

**Return values**

| Value | Description |
|---|---|
| `*html/template.Template` | Ready-to-execute template set. The root template is named with the lowercased `componentName`. Sub-components are included as additional named templates in the same set. |
| `error` | Non-nil when the component is not found or conversion fails. |

**Error types**

| Error | Condition |
|---|---|
| `ErrComponentNotFound` (sentinel) | `componentName` is not registered in the engine. Test with `errors.Is(err, htmlc.ErrComponentNotFound)`. |
| `*bridge.ConversionError` (wrapped with `ErrConversion`) | A directive or expression in the component tree cannot be converted. Extract with `errors.As(err, &cerr)`. The `Location` field points to the source position. |

**Template name convention**

The root template name is `strings.ToLower(componentName)`. For example, `"MyCard"` → `"mycard"`. All sub-component templates in the set follow the same lowercasing rule.

**Scoped style handling**

`<style scoped>` blocks in `.vue` components are stripped. The compiled template contains no `<style>` elements.

**Thread safety**

`CompileToTemplate` acquires a read lock on the engine registry. Concurrent calls to `CompileToTemplate`, `RenderFragment`, and `Has` are safe. Concurrent calls to `RegisterTemplate` while `CompileToTemplate` is running are also safe.

---

## 3.3 `Engine.RegisterTemplate` reference

```
func (e *Engine) RegisterTemplate(name string, tmpl *html/template.Template) error
```

**Parameters**

| Parameter | Type | Description |
|---|---|---|
| `name` | `string` | Name under which the root template is registered as a component. |
| `tmpl` | `*html/template.Template` | Template set to convert and register. All named `{{ define }}` blocks in the set are also registered under their own names. |

**Return values**

| Value | Description |
|---|---|
| `error` | Non-nil when conversion of any template in the set fails. |

**Error types**

| Error | Condition |
|---|---|
| `*bridge.ConversionError` (wrapped with `ErrConversion`) | A template construct in `tmpl` is not supported by the tmpl→vue converter. |

**Validation timing**

Conversion and validation happen at registration time. If any template in the set fails conversion, `RegisterTemplate` returns the error and registers nothing (atomic: either all succeed or none are registered).

**Thread safety**

`RegisterTemplate` acquires a write lock on the engine registry. Safe to call concurrently with read operations; concurrent writes serialise.

**Behaviour when name is already registered**

Last write wins. A second call with the same `name` overwrites the first registration. This is consistent with the engine's flat-registry policy.

---

## 3.4 CLI reference: `htmlc template vue-to-tmpl`

```
htmlc template vue-to-tmpl [flags] <ComponentName>
```

Reads the named `.vue` component (and all statically-referenced sub-components) from the component directory, converts them to `html/template {{ define }}` blocks, and writes the result to stdout.

**Flags**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-dir` | `string` | `"."` | Directory to scan for `.vue` files. |
| `-quiet` | `bool` | `false` | Suppress non-fatal conversion warnings on stderr. |

**Output format**

One `{{ define "name" }} … {{ end }}` block per component, written to stdout. Sub-components appear before the root component (dependency order: leaves first). Scoped `<style>` blocks are stripped.

**Exit codes**

| Code | Meaning |
|---|---|
| 0 | Conversion succeeded. Output written to stdout. |
| 1 | Error (component not found, conversion error, I/O error). Error message written to stderr. |

**stdout / stderr split**

- stdout: converted template text only
- stderr: non-fatal warnings (unless `-quiet`) and error messages

---

## 3.5 CLI reference: `htmlc template tmpl-to-vue`

```
htmlc template tmpl-to-vue [flags]
```

Reads `html/template` source from stdin, converts it to a `.vue` component, and writes the result to stdout.

**Flags**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-name` | `string` | `"Component"` | Component name used for error messages and the `<template>` root. |
| `-quiet` | `bool` | `false` | Suppress non-fatal conversion warnings on stderr. |

**Input**

Raw `html/template` source text on stdin. The input may contain a single root-level template or multiple `{{ define }}` blocks.

**Output format**

A `.vue` component source on stdout, wrapped in `<template> … </template>`.

**Exit codes**

| Code | Meaning |
|---|---|
| 0 | Conversion succeeded. Output written to stdout. |
| 1 | Error (parse failure, unsupported construct). Error message written to stderr. |

**stdout / stderr split**

- stdout: converted `.vue` source only
- stderr: non-fatal warnings (unless `-quiet`) and error messages

---

## 3.6 Type contracts

Directives that impose runtime type constraints on data fields when a `.vue` component is compiled to `html/template`:

| Directive | Required Go type | Why |
|---|---|---|
| `v-html="field"` | `html/template.HTML` | `html/template` auto-escapes plain `string` in HTML body context; `template.HTML` is passed through verbatim, matching `v-html` unescaped semantics. |
| `v-bind="attrs"` (spread) | `html/template.HTMLAttr` | The attribute-position `{{ .attrs }}` in `html/template` requires a pre-formatted attribute string of type `HTMLAttr`; map-based attribute spreading is not available in `html/template`. |

**Passing the wrong type is not a compilation error.** The template executes, but the output is wrong: a plain `string` passed to a `v-html` field renders with escaped angle brackets.

---

## 3.7 Truthiness divergence table

`v-if` and `v-else-if` evaluate their expression according to htmlc's JavaScript-derived truthiness rules. Go's `html/template` `{{ if }}` uses Go's truthiness rules. The two diverge for empty collections:

| Value | htmlc (JS rules) | Go template | Safe to cross the bridge? |
|---|---|---|---|
| `false` | Falsy | Falsy | ✅ |
| `0` | Falsy | Falsy | ✅ |
| `""` | Falsy | Falsy | ✅ |
| `nil` | Falsy | Falsy | ✅ |
| `[]T{}` (empty slice) | **Truthy** | Falsy | ❌ Diverge |
| `map[K]V{}` (empty map) | **Truthy** | Falsy | ❌ Diverge |

A non-nil, non-zero-length slice is truthy in both systems. The divergence only occurs when the slice or map is non-nil but empty. Restrict `v-if` conditions to `bool`, non-empty `string`, non-zero numeric, or non-nil pointer types to avoid silent behavioural differences.

The converter emits a warning for `v-if` expressions whose type cannot be statically verified as safe. This is a warning, not an error; conversion proceeds.

---

## 3.8 Unsupported constructs (error catalogue)

The following constructs produce a `*bridge.ConversionError` with `errors.Is(err, htmlc.ErrConversion) == true`.

**vue→tmpl direction** (`CompileToTemplate`, `TemplateText`, `htmlc template vue-to-tmpl`):

| Construct | Error message (partial) | Remedy |
|---|---|---|
| `{{ items[0] }}` | `unsupported expression kind` | Use only simple identifiers (`name`) or dot-paths (`a.b.c`) |
| `{{ a + b }}` | `unsupported expression kind` | Same as above |
| `:href="getUrl(id)"` | `unsupported expression kind` | Compute derived values in the handler and pass the result as a named field |
| `<MyComponent title="static">` | `static props on sub-component calls are not supported` | Remove static props; pass all data through the root data map |
| `v-my-directive="x"` | `unsupported directive` | Remove the custom directive or manually translate to html/template syntax |
| `v-for="item in list"` body references outer variable | `outer-scope variable reference inside v-for` | Embed the outer-scope data into each list item struct (see §2.6 in the how-to guide) |
| `v-show` combined with `:style` | `v-show cannot be combined with :style` | Separate the concerns: use `v-show` alone or manage visibility via a class |

**tmpl→vue direction** (`RegisterTemplate`, `htmlc template tmpl-to-vue`):

| Construct | Error message (partial) | Remedy |
|---|---|---|
| `{{ .items \| len }}` | `multi-command pipeline` | Replace pipeline with a pre-computed field in the data |
| `{{ with .x }}` | `unsupported action: with` | Restructure data so the template receives the nested value at the root |
| `$x := .field` | `variable assignment` | Move `$x` computation to the Go handler |
| `{{ template "Name" expr }}` (expr ≠ `.`) | `template call with non-dot data` | Ensure sub-template calls pass `.` as data |
