# htmlc — Server-Side Vue.js Component Engine for Go

## Overview

`htmlc` is a Go library that renders Vue.js Single File Components (SFCs) on the server. It implements the subset of Vue.js that can be evaluated without a browser: the template language, directives, expression evaluation, scoped styles, and component composition. The output is plain HTML that can be served directly or used as a fragment in HTMX-style partial updates.

The library exposes a minimal, idiomatic Go API that plugs directly into `net/http`.

---

## Package layout

```
htmlc/
  engine.go          — Engine, component registry, top-level render API
  component.go       — SFC parser (.vue files → template/script/style sections)
  renderer.go        — Template walker: directives, interpolation, composition
  expr/
    lexer.go         — Tokeniser for the JS-compatible expression language
    parser.go        — Recursive-descent parser → AST
    eval.go          — AST evaluator against a Go data scope
    ast.go           — AST node types
  style.go           — Style extractor, scope-ID stamper, merger
  go.mod
```

---

## 1. Expression Language (`htmlc/expr`)

A self-contained lexer → parser → evaluator that understands the JavaScript expression syntax needed by Vue templates. It has no external dependencies.

### 1.1 Literals

| Literal | Example |
|---------|---------|
| Integer | `42`, `-7` |
| Float | `3.14`, `.5` |
| String (single-quoted) | `'hello'` |
| String (double-quoted) | `"world"` |
| Boolean | `true`, `false` |
| Null | `null` |
| Undefined | `undefined` |
| Array | `[1, 'two', item]` |
| Object | `{ key: val, 'x': 1 }` |

### 1.2 Unary operators

All JavaScript unary prefix operators:

| Operator | Meaning |
|----------|---------|
| `!` | logical NOT |
| `-` | arithmetic negation |
| `+` | unary plus (convert to number) |
| `~` | bitwise NOT |
| `typeof` | type string (`"number"`, `"string"`, `"boolean"`, `"object"`, `"undefined"`, `"function"`) |
| `void` | evaluates operand, returns `undefined` |

### 1.3 Binary operators (highest to lowest precedence)

| Operators | Category |
|-----------|----------|
| `**` | exponentiation (right-associative) |
| `*` `/` `%` | multiplicative |
| `+` `-` | additive |
| `<<` `>>` `>>>` | bitwise shift |
| `<` `<=` `>` `>=` | relational |
| `in` | property membership (key in object/map) |
| `instanceof` | type check (always false in Go context, included for compatibility) |
| `==` `!=` | loose equality (with JS-style type coercion: `0 == false`, `"" == false`, `null == undefined`) |
| `===` `!==` | strict equality (no coercion) |
| `&` | bitwise AND |
| `^` | bitwise XOR |
| `\|` | bitwise OR |
| `&&` | logical AND (short-circuit, returns operand not bool) |
| `\|\|` | logical OR (short-circuit, returns operand not bool) |
| `??` | nullish coalescing (returns right if left is null/undefined) |

### 1.4 Ternary

`condition ? consequent : alternate`

### 1.5 Member access

- Dot notation: `user.name`, `item.price`
- Bracket notation: `items[0]`, `obj['key']`, `obj[dynKey]`
- Chained: `user.address.city`

### 1.6 Function calls

`fn(a, b, c)` — calls a function found in the evaluation scope. Spread (`...args`) is not supported.

### 1.7 Scope and Go interop

The evaluator takes a `map[string]any` scope. Go values are accessed transparently:

- `map[string]any` and `map[string]V` → accessed by string key
- Structs → accessed by exported field name or `json` tag
- Slices → accessed by integer index
- Functions (`func(...any) (any, error)`) → callable
- `nil` maps to `null`; zero values of numeric types map to `0`

### 1.8 API

```go
package expr

// Compile parses an expression string into a reusable AST.
func Compile(src string) (Expr, error)

// Expr is a compiled expression ready for repeated evaluation.
type Expr interface {
    Eval(scope map[string]any) (any, error)
}

// Eval is a convenience wrapper: compile + evaluate in one step.
func Eval(src string, scope map[string]any) (any, error)
```

---

## 2. Single File Component format

Components are `.vue` files with up to three top-level sections:

```html
<template>
  <div class="card">
    <h2>{{ title }}</h2>
    <slot />
  </div>
</template>

<script>
// Optional. Returns the component's default props/data as a Go map literal
// (parsed at load time, not executed).
// Alternatively, components receive their data from the caller.
</script>

<style scoped>
.card { border: 1px solid #ccc; padding: 1rem; }
h2    { color: navy; }
</style>
```

The `<script>` section is reserved for future use and is parsed but not evaluated in this version.

---

## 3. Template Directives

All directives are processed server-side. Client-side-only directives (`v-model`, `v-on`/`@`) are **preserved as-is** in the output (they are not stripped, so the browser can handle them if Vue.js is also loaded client-side).

### 3.1 Conditional rendering

```html
<div v-if="count > 0">has items</div>
<div v-else-if="count === 0">empty</div>
<div v-else>negative?</div>
```

- The first truthy branch is rendered; the rest are omitted.
- `v-else-if` and `v-else` must immediately follow a sibling with `v-if`/`v-else-if` (no text nodes between).

### 3.2 List rendering

```html
<li v-for="item in items" :key="item.id">{{ item.name }}</li>
<li v-for="(item, index) in items">{{ index }}: {{ item }}</li>
<li v-for="(value, key) in obj">{{ key }} = {{ value }}</li>
<li v-for="n in 5">{{ n }}</li>
```

- Iterates arrays (element, index), objects/maps (value, key), and integers (1..n).
- `:key` is evaluated and rendered as a `data-key` attribute.

### 3.3 Attribute binding

```html
<a :href="url" :class="{ active: isActive }" :style="{ color: textColor }">
<img :src="imgSrc" :alt="description">
<button :disabled="!canSubmit">Submit</button>
```

- `:attr` is shorthand for `v-bind:attr`.
- Object syntax for `:class`: each key is added to the class list if its value is truthy.
- Array syntax for `:class`: `['a', condition ? 'b' : '']` — falsy strings are omitted.
- Object syntax for `:style`: keys are CSS property names (camelCase or kebab-case), values are the CSS values.
- Boolean attributes (`disabled`, `checked`, `selected`, `readonly`, `required`, `multiple`, `autofocus`, `open`): rendered without value when truthy, omitted when falsy.

### 3.4 Text content

```html
<p v-text="message"></p>
```

Sets the element's text content to the evaluated expression (HTML-escaped).

### 3.5 Raw HTML

```html
<div v-html="rawHtml"></div>
```

Sets the element's inner HTML to the evaluated expression (not escaped). The surrounding element is still rendered.

### 3.6 Visibility

```html
<div v-show="isVisible">content</div>
```

Rendered as `style="display:none"` when falsy, rendered normally when truthy. Element is always present in the DOM.

### 3.7 Skip compilation

```html
<div v-pre>{{ this is not interpolated }}</div>
```

The element and all its descendants are emitted verbatim without any template processing.

### 3.8 Render once

```html
<div v-once>{{ expensive }}</div>
```

Processed once at render time like a normal expression (equivalent to normal rendering in a server-side context).

### 3.9 Mustache interpolation

```html
<p>Hello, {{ user.name }}!</p>
<p>Total: {{ price * qty }}</p>
```

Expressions inside `{{ }}` are evaluated and HTML-escaped. Whitespace inside the braces is trimmed.

### 3.10 `<template>` wrapper

```html
<template v-if="show">
  <p>first</p>
  <p>second</p>
</template>
<template v-for="item in list">
  <dt>{{ item.key }}</dt>
  <dd>{{ item.val }}</dd>
</template>
```

`<template>` is a virtual wrapper — it is not rendered as an HTML element; only its children are emitted.

### 3.11 Component usage

```html
<Card :title="pageTitle" class="main-card">
  <p>Slot content here</p>
</Card>
```

- PascalCase or kebab-case tag names that match a registered component are rendered by recursively invoking that component's template.
- Props (`:prop`) are evaluated and passed as the child component's scope.
- Static attributes are passed as string props.
- Inner content becomes the default slot (`<slot />`).

---

## 4. Style Scoping

### 4.1 Scope ID generation

Each component gets a stable scope ID derived from its file path: `data-v-` followed by the first 8 hex characters of the FNV-1a hash of the component path.

### 4.2 Scoping mechanism

When a component has a `<style scoped>` block:

1. Every HTML element rendered by that component receives the scope attribute (e.g., `data-v-a1b2c3d4`).
2. Every CSS rule in the style block is rewritten by appending `[data-v-a1b2c3d4]` to the last simple selector of each rule.
3. Global styles (`<style>` without `scoped`) are included as-is.

### 4.3 Style merging

During a render, all `<style>` contributions (scoped and global) from all rendered components are collected into a single `<style>` block. Pages inject this block into `<head>`. Fragments return the style block prepended to the HTML output.

---

## 5. Engine API

### 5.1 Creating an engine

```go
engine := htmlc.New(htmlc.Options{
    ComponentDir: "./components",   // root directory for .vue files
    Reload:       true,             // re-parse components on every request (dev mode)
})
```

`Reload: true` means the engine checks the file's mtime before every render and re-parses if it has changed. `Reload: false` parses each component once and caches it indefinitely (production mode).

### 5.2 Registering components

Components are discovered automatically by recursively scanning `ComponentDir` for `*.vue` files. The component name is the filename without extension (e.g., `Card.vue` → `Card`, `ui/Button.vue` → `Button`). If two files produce the same name, the last one wins (alphabetical traversal order).

Additional components can be registered manually:

```go
engine.Register("MyCard", "/abs/path/to/Card.vue")
```

### 5.3 Rendering a page

```go
html, err := engine.RenderPage(componentName string, data map[string]any) (string, error)
```

- Renders the named component with the provided data scope.
- Collects all style contributions and inserts a `<style>` tag immediately before `</head>`. If no `<head>` is present, prepends the style block to the output.
- Returns the complete HTML string.

### 5.4 Rendering a fragment

```go
html, err := engine.RenderFragment(componentName string, data map[string]any) (string, error)
```

- Renders the named component with the provided data scope.
- Prepends all collected styles as a `<style>` tag to the output.
- Intended for HTMX-style partial page updates.

### 5.5 http.Handler integration

```go
// ServeComponent returns an http.HandlerFunc that renders the named component.
// Data is sourced from a user-supplied DataFunc.
func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc

// Example wiring:
mux := http.NewServeMux()
mux.Handle("/", engine.ServeComponent("HomePage", func(r *http.Request) map[string]any {
    return map[string]any{"title": "Welcome"}
}))
```

The handler sets `Content-Type: text/html; charset=utf-8` and writes the rendered page.

---

## 6. Error handling

- Parse errors (malformed template, bad expression) are returned as Go errors from the relevant function.
- Runtime errors in expressions (type mismatch, nil dereference, index out of bounds) are returned as errors from `RenderPage`/`RenderFragment`.
- Unknown component names referenced in a template return an error.

---

## 7. Constraints and non-goals

- No JavaScript runtime. The expression language is a Go implementation; it does not call into V8 or any JS engine.
- No client-side reactivity. There is no virtual DOM, no diff algorithm, no WebSocket connection.
- `v-model` and `v-on`/`@` directives are passed through to the HTML unchanged so that client-side Vue.js can pick them up if present.
- No TypeScript support in `<script>`.
- No `<script setup>` composition API.
- Async components and `<Suspense>` are not supported.
- SSR hydration markers are not emitted.
