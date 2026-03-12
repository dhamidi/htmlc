# htmlc

A server-side Go template engine that uses Vue.js Single File Component (`.vue`) syntax for authoring but renders entirely in Go with no JavaScript runtime.

**This is a static rendering engine.** There is no reactivity, virtual DOM, or client-side hydration. Templates are evaluated once per request and produce plain HTML.

---

## Table of Contents

1. [Overview](#overview)
2. [Template Syntax](#template-syntax)
3. [Directives](#directives)
4. [Component System](#component-system)
5. [Special Attributes](#special-attributes)
6. [Go API Quick Reference](#go-api-quick-reference)
7. [Expression Language Reference](#expression-language-reference)
8. [Debug Mode](#debug-mode)
9. [Custom Directives](#custom-directives)

---

## 1. Overview

`htmlc` lets you write reusable HTML components in `.vue` files — the same format used by Vue.js — and render them server-side in Go. There is no Node.js dependency and no JavaScript executed at runtime. The `<script>` section of a `.vue` file is parsed and preserved in the output but never executed by the engine.

Key characteristics:

- **Static output** — every render call produces a fixed HTML string.
- **Scoped styles** — `<style scoped>` is supported; the engine rewrites selectors and injects scope attributes automatically.
- **Component composition** — components can nest other components from the same registry.
- **No reactivity** — `v-model`, `@event`, and other client-side directives are stripped from the output; they have no meaning in a server-side renderer.

---

## 2. Template Syntax

### Supported

#### Text interpolation

`{{ expr }}` evaluates the expression against the current render scope and HTML-escapes the result.

```html
<p>Hello, {{ name }}!</p>
<p>{{ a }} + {{ b }} = {{ a + b }}</p>
```

Multiple interpolations in a single text node are supported.

#### Expression language

| Category | Operators / Syntax |
|---|---|
| Arithmetic | `+`, `-`, `*`, `/`, `%`, `**` |
| Comparison | `===`, `!==`, `>`, `<`, `>=`, `<=`, `==`, `!=` |
| Logical | `&&`, `\|\|`, `!` |
| Nullish coalescing | `??` |
| Optional chaining | `obj?.key`, `arr?.[i]` |
| Ternary | `condition ? then : else` |
| Member access | `obj.key`, `arr[i]`, `arr.length` |
| Function calls | `fn(args)` (via `expr.RegisterBuiltin` or `engine.RegisterFunc`) |
| Array literals | `[a, b, c]` |
| Object literals | `{ key: value }` |

#### Built-in functions

The engine ships with no pre-registered built-in functions. Use `expr.RegisterBuiltin` to add custom functions that are callable from templates by name. For measuring collection sizes, use the `.length` member property instead — it works on strings, slices, arrays, and maps with no registration required:

```html
<!-- number of elements in a slice -->
<span>{{ items.length }}</span>

<!-- number of bytes in a string -->
<span>{{ name.length }}</span>
```

### Not supported

- Filters (`{{ value | filterName }}`) — Vue 2 syntax, not implemented.
- JavaScript function definitions, arrow functions (`=>`), `new`, `delete`.
- Template literals (backtick strings).
- Assignment operators (`=`, `+=`, etc.) and increment/decrement (`++`, `--`).

---

## 3. Directives

### Supported directives

| Directive | Supported | Notes |
|---|---|---|
| `v-text` | Yes | Sets element text content (HTML-escaped). Replaces all children. |
| `v-html` | Yes | Sets element inner HTML (not escaped). Replaces all children. Use with trusted content only. |
| `v-show` | Yes | Adds `style="display:none"` when the expression is falsy. Merges with any existing `style` attribute. |
| `v-if` | Yes | Renders the element only when the expression is truthy. |
| `v-else-if` | Yes | Must immediately follow a `v-if` or `v-else-if` element (whitespace between is allowed). |
| `v-else` | Yes | Must immediately follow a `v-if` or `v-else-if` element. |
| `v-for` | Yes | See [v-for syntax](#v-for-syntax) below. |
| `v-bind` / `:attr` | Yes | Dynamic attribute binding. See [v-bind notes](#v-bind-notes) below. |
| `v-pre` | Yes | Skips all interpolation and directive processing for the element and all its descendants. The `v-pre` attribute itself is stripped from the output. |
| `v-slot` / `#name` | Yes | Used on `<template>` elements (or directly on a component tag) to target named or scoped slots. Shorthand: `#name`. See [Slots](#slots) under §4. |
| `v-once` | No-op | Accepted and stripped; server-side rendering always renders once, so this directive has no effect. |

### Not supported

| Directive | Status |
|---|---|
| `v-on` / `@event` | Stripped. Client-side event handlers have no meaning in server-side rendering. |
| `v-model` | Stripped. Two-way data binding has no meaning in server-side rendering. |
| `v-cloak` | Not relevant for server-side rendering. |
| `v-memo` | Not implemented. |

### v-for syntax

```html
<!-- Array: item only -->
<li v-for="item in items">{{ item }}</li>

<!-- Array: item + index -->
<li v-for="(item, index) in items">{{ index }}: {{ item }}</li>

<!-- Map: value + key -->
<li v-for="(value, key) in obj">{{ key }}: {{ value }}</li>

<!-- Map: value + key + index (index is position in iteration order) -->
<li v-for="(value, key, index) in obj">{{ index }}. {{ key }}: {{ value }}</li>

<!-- Integer range: iterates 1..n inclusive -->
<li v-for="n in 5">{{ n }}</li>

<!-- Multi-element group using <template> -->
<template v-for="item in items">
  <dt>{{ item.term }}</dt>
  <dd>{{ item.def }}</dd>
</template>
```

**Difference from Vue.js:** Map iteration order follows Go's `reflect.MapKeys()` order, which is not guaranteed to be insertion order.

### v-bind notes

- `:class` supports **object syntax** (`{ active: isActive }`) and **array syntax** (`[classA, classB]`).
- `:style` supports **object syntax** with camelCase keys (`{ fontSize: '14px' }`); keys are converted to kebab-case in the output.
- **Boolean attributes** (`disabled`, `checked`, `selected`, `readonly`, `required`, `multiple`, `autofocus`, `open`) are omitted entirely when the bound value is falsy.
- `:key` is rendered as `data-key="value"` in the output (not as a `key` attribute).
- `class` and `:class` are merged into a single `class` attribute.
- `style` and `:style` are merged into a single `style` attribute.
- `v-bind:attr` (long form) is equivalent to `:attr` (shorthand).
- **`v-bind="obj"` (attribute spreading)**: When `v-bind` is used without
  an attribute name and its value is a `map[string]any`, each entry is spread
  as an HTML attribute. Keys `class` and `style` follow the same merge rules
  as `:class`/`:style`. Boolean attribute semantics apply per key.

  ```html
  <!-- Spread HTMX attributes from a map: -->
  <button v-bind="actions.delete.hxAttrs">Delete</button>

  <!-- Spread props into a child component: -->
  <Card v-bind="cardProps" :title="override" />
  ```

  On child components, the spread map values are lower priority than
  explicitly named `:prop` bindings.

---

## 4. Component System

### Supported

#### Single File Components

A `.vue` file may have three top-level sections:

```vue
<template>
  <!-- required; HTML template -->
</template>

<script>
  // optional; preserved verbatim in output but NOT executed
</script>

<style>
  /* optional; collected and injected as a <style> block */
</style>
```

#### Props

Pass data to child components via attributes:

```html
<!-- Dynamic prop (expression evaluated in caller scope) -->
<Card :title="pageTitle" :count="items.length" />

<!-- Static prop (always a string) -->
<Card title="Hello" />
```

No prop type validation or default values — the engine passes whatever you provide.

#### Default slot

Use `<slot />` inside a component to render the caller's inner content:

```html
<!-- Card.vue -->
<template>
  <div class="card">
    <slot />
  </div>
</template>
```

```html
<!-- caller -->
<Card>
  <p>This goes into the slot.</p>
</Card>
```

Slot content is evaluated in the **caller's** scope, not the child component's scope.

#### Slots

##### Default slot

As shown above, `<slot />` renders the caller's inner content. Children of `<slot>` act as **fallback content** — rendered only when the caller provides nothing:

```html
<!-- Button.vue -->
<template>
  <button>
    <slot>Click me</slot>
  </button>
</template>
```

```html
<!-- renders "Click me" because no content provided -->
<Button />

<!-- renders "Submit" -->
<Button>Submit</Button>
```

##### Named slots

A component can expose multiple insertion points by giving each `<slot>` a `name` attribute. The caller targets a named slot with `<template v-slot:name>` or the `#` shorthand `<template #name>`:

```html
<!-- Layout.vue -->
<template>
  <div class="layout">
    <header><slot name="header" /></header>
    <main><slot /></main>
    <footer><slot name="footer" /></footer>
  </div>
</template>
```

```html
<!-- caller -->
<Layout>
  <template #header><h1>{{ pageTitle }}</h1></template>
  <p>Main body content.</p>
  <template #footer><small>© 2024</small></template>
</Layout>
```

Content without a `v-slot` / `#` target goes to the default slot.

##### Scoped slots

A component can pass data back to the caller's slot content by binding props on the `<slot>` element. The caller receives them via `v-slot="{ … }"` or `#name="{ … }"`:

```html
<!-- List.vue -->
<template>
  <ul>
    <li v-for="item in items">
      <slot :item="item" :index="index" />
    </li>
  </ul>
</template>
```

```html
<!-- caller — destructured binding -->
<List :items="products">
  <template #default="{ item, index }">
    <strong>{{ index }}.</strong> {{ item.name }}
  </template>
</List>
```

Binding patterns:

| Syntax | Effect |
|---|---|
| `v-slot` | Slot targeted, no props exposed |
| `v-slot="slotProps"` | All slot props available as `slotProps.x` |
| `v-slot="{ item }"` | Destructured; `item` available directly |
| `v-slot="{ item, index }"` | Multiple destructured props |

##### Scope rules

- Slot content is always evaluated in the **caller's** scope.
- Slot props (from `:prop="expr"` on `<slot>`) are merged into the scope when rendering that slot's content — they do not leak into the rest of the caller's template.
- Named-slot props are scoped to the `<template #name="…">` block.

#### Scope rules for props and engine functions

Each component renders in an **isolated scope** containing only its own props.
It does not automatically inherit variables from the parent component's scope.
This is intentional: it prevents accidental coupling between components and
makes data-flow explicit.

Engine-level functions registered via `engine.RegisterFunc` are an exception:
they are injected into every component's scope automatically, at every level
of the component tree. Treat them as a lightweight, read-only global namespace
for helper functions (URL builders, route matchers, formatters).

**WithDataMiddleware** values are **not** propagated automatically — they are
available only in the top-level page scope. If a deeply-nested component needs
a value supplied by middleware (such as the current user), pass it down as an
explicit prop or register it as an engine function instead.

| Mechanism | Available in page | Available in child components |
|-----------|:-----------------:|:-----------------------------:|
| `RenderPage` / `RenderFragment` data map | Yes | No (pass as props) |
| `WithDataMiddleware` values | Yes | No (pass as props) |
| `RegisterFunc` functions | Yes | Yes (automatic) |
| Explicit `:prop="expr"` | — | Yes |

#### The page-to-shell pattern

A common layout structure has a page component that passes request-specific
data into a shared shell (layout) component:

```vue
<!-- HomePage.vue -->
<template>
  <Shell :title="title">
    <h1>Welcome</h1>
    <p>{{ intro }}</p>
  </Shell>
</template>
```

```vue
<!-- Shell.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body>
      <nav>
        <a :href="url('home')">Home</a>  <!-- url() from RegisterFunc -->
      </nav>
      <main><slot /></main>
    </body>
  </html>
</template>
```

Render data for the page:

```go
engine.RenderPage(w, "HomePage", map[string]any{
    "title": "Welcome",
    "intro": "Hello from the server.",
})
```

Key points:
- The `Shell` component receives `title` as an explicit prop.
- Helper functions like `url` are available in `Shell` automatically via
  `RegisterFunc` — they do not need to be passed as props.
- Slot content (`<h1>Welcome</h1>`) is evaluated in the **page's** scope,
  not Shell's scope, so it can reference `title` and `intro` directly.

#### Component resolution

Given a tag name, the engine tries these strategies in order:

1. Exact match in the registry (e.g. `my-card` → `my-card`)
2. First letter capitalised (e.g. `card` → `Card`)
3. Kebab-case to PascalCase (e.g. `my-card` → `MyCard`)
4. Case-insensitive scan

#### Scoped styles

```vue
<style scoped>
.button { color: red; }
</style>
```

The engine rewrites CSS selectors with a `data-v-*` scope attribute (e.g. `.button[data-v-abc123]`) and adds that attribute to every HTML element rendered by the component.

Slot content is stamped with the **authoring component's** scope attribute, not the child component's. This mirrors Vue SFC behaviour: CSS rules in the parent apply to elements the parent authors, even when those elements are rendered inside a child's `<slot>`.

| Content authored in | Scope attribute stamped |
|---------------------|------------------------|
| Child component template | `data-v-child` |
| Parent slot content | `data-v-parent` |
| Fallback children of `<slot>` | `data-v-child` (fallback belongs to the child) |

CSS content is extracted verbatim from `<style>` blocks — quoted string values, `@font-face` declarations, data URIs, and special characters (`&`, `<`, `>`) are preserved exactly as written and are never HTML-escaped. Only non-`@`-rule selectors are rewritten for scoping.

#### Nested composition

Components can freely use other components registered in the same engine.

#### Dynamic components

Use `<component :is="expr">` to render a component whose name is determined at runtime. The expression must evaluate to a non-empty string that names a registered component or a native HTML element:

```html
<!-- resolve from a variable -->
<component :is="activeView" />

<!-- inline string literal -->
<component :is="'Card'" :title="pageTitle">
  <p>slot content</p>
</component>

<!-- switch between components in a loop -->
<div v-for="item in items">
  <component :is="item.type" :data="item" />
</div>
```

- All attributes other than `:is` (or `v-bind:is`) are forwarded to the resolved component as props.
- Slot content (default and named) works exactly as with a statically-named component.
- If the resolved name is a known HTML element (e.g. `"div"`, `"input"`), the tag is rendered as-is rather than looked up in the component registry.
- `:is` is required; omitting it or supplying a non-string value is a render error.

### Not supported

| Feature | Status |
|---|---|
| `<script setup>` / Composition API | Not supported. `<script>` content is never executed. |
| Computed properties, watchers, lifecycle hooks | Not applicable (no runtime). |
| `$emit` / custom events | Not implemented. |
| `provide` / `inject` | Not implemented. |
| Async components | Not applicable. |
| `defineProps` / `defineEmits` / `withDefaults` | Not applicable. |
| Teleport, Suspense, KeepAlive | Not applicable. |

---

## 5. Special Attributes

| Attribute | Behavior |
|---|---|
| `:key` | Rendered as `data-key="value"` in the HTML output. Not used for diffing. |
| `class` + `:class` | Both are collected and merged into a single `class` attribute. |
| `style` + `:style` | Both are collected and merged into a single `style` attribute. |

---

## 6. Go API Quick Reference

### Create an engine

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",  // recursively scanned for *.vue files
    Reload:       false,         // set true for hot-reload during development
})
```

### Render a full HTML page

Scoped styles are injected before the first `</head>` tag.

```go
err = engine.RenderPage(w, "Page", map[string]any{
    "title": "Home",
    "items": []string{"a", "b"},
})
```

### Render an HTML fragment

Scoped styles are prepended before the HTML. Use this for HTMX responses, turbo frames, etc.

```go
err = engine.RenderFragment(w, "Card", map[string]any{
    "title": "My Card",
})
```

### Serve a component as an HTTP handler

```go
http.Handle("/widget", engine.ServeComponent("Widget", func(r *http.Request) map[string]any {
    return map[string]any{"id": r.URL.Query().Get("id")}
}))
```

Pass `nil` as the second argument if the component needs no data.

### Parse a component manually

```go
comp, err := htmlc.ParseFile("path/to/Button.vue", srcString)
```

### Discover expected props

```go
for _, p := range comp.Props() {
    fmt.Println(p.Name, p.Expressions)
}
```

### Inspect parse and render errors

Parse and render failures carry structured location information when the
source position can be determined. Use `errors.As` to inspect them:

```go
import "errors"

_, err := htmlc.ParseFile("Card.vue", src)
var pe *htmlc.ParseError
if errors.As(err, &pe) {
    fmt.Println(pe.Path)             // "Card.vue"
    if pe.Location != nil {
        fmt.Println(pe.Location.Line)    // 1-based line number
        fmt.Println(pe.Location.Snippet) // 3-line source context
    }
}

err = engine.RenderFragment(w, "Card", data)
var re *htmlc.RenderError
if errors.As(err, &re) {
    fmt.Println(re.Component)        // component path
    fmt.Println(re.Expr)             // expression that failed, e.g. "post.Title"
    if re.Location != nil {
        fmt.Println(re.Location.Line)    // approximate line number
        fmt.Println(re.Location.Snippet) // 3-line source context
    }
}
```

When location information is available, `err.Error()` prints a compiler-style
message with file, line, and a source snippet:

```
Card.vue:14:5: render Card.vue: expr "post.Title": cannot access property "Title" of null
  13 |   <div class="card">
> 14 |     {{ post.Title }}
  15 |   </div>
```

When position cannot be determined, the traditional `htmlc: ...` format is used
as a fallback so existing error-checking code continues to work.

### Configure missing prop behavior

By default, when a prop is missing from the render scope, the engine renders a
visible `[missing: propName]` placeholder in its place so the page still loads
and you can immediately see which prop is absent.

To restore strict error behaviour (rendering aborts with an error), use the
built-in `ErrorOnMissingProp` handler:

```go
engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)
```

To silence missing props entirely or substitute a custom value:

```go
// silently substitute an empty string
engine.WithMissingPropHandler(func(name string) (any, error) {
    return "", nil
})
```

### Development hot-reload

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Reload:       true,  // re-parses changed files before each render
})
```

### Load components from an embedded filesystem

Set `Options.FS` to any `fs.FS` implementation — including `embed.FS` — and
the engine reads and walks component files through that FS instead of the OS
filesystem. `ComponentDir` is then interpreted as a path within the FS.

```go
import "embed"

//go:embed templates
var templateFS embed.FS

engine, err := htmlc.New(htmlc.Options{
    FS:           templateFS,
    ComponentDir: "templates",
})
```

This embeds the entire `templates/` directory into the binary at compile time.
Any `fs.FS` implementation works — `embed.FS`, `testing/fstest.MapFS`,
`fs.Sub`, or a custom virtual filesystem.

Hot-reload (`Reload: true`) is supported when the FS implements `fs.StatFS`
(which `embed.FS` does not — embedded files have no mtime). When the FS does
not implement `fs.StatFS`, reload checks are silently skipped.

### Context-aware rendering

Pass a `context.Context` to propagate cancellation and deadlines through the render pipeline:

```go
ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel()

err = engine.RenderPageContext(ctx, w, "Page", data)
err = engine.RenderFragmentContext(ctx, w, "Card", data)
```

`ServeComponent` and `ServePageComponent` automatically forward `r.Context()`.

### Register per-engine template functions

`RegisterFunc` makes a Go function callable from any template expression rendered by this engine. Engine-level functions act as lower-priority builtins — the render data scope overrides them:

```go
engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    t, _ := args[0].(time.Time)
    return t.Format("2 Jan 2006"), nil
})
```

```html
<span>{{ formatDate(post.CreatedAt) }}</span>
```

### Share helper functions across the component tree

Functions registered with `RegisterFunc` are available in **every** component
at any nesting depth — you do not need to pass them as props:

```go
engine.RegisterFunc("url", func(args ...any) (any, error) {
    name, _ := args[0].(string)
    return router.URLFor(name), nil
})
engine.RegisterFunc("routeActive", func(args ...any) (any, error) {
    name, _ := args[0].(string)
    return r.URL.Path == router.URLFor(name), nil
})
```

```vue
<!-- Shell.vue — url() and routeActive() are available without any prop wiring -->
<template>
  <nav>
    <a :href="url('home')" :class="{ active: routeActive('home') }">Home</a>
    <a :href="url('about')" :class="{ active: routeActive('about') }">About</a>
  </nav>
  <slot />
</template>
```

This is the recommended approach for router helpers, auth utilities, and any
other function that many components across the tree need to call.

For **data values** (structs, strings, booleans) that vary per request, pass
them as explicit props or use `WithDataMiddleware` and thread them down through
the component tree where needed.

### Serve a full-page component as an HTTP handler

`ServePageComponent` is like `ServeComponent` but renders a full HTML page and lets the data function return an HTTP status code:

```go
http.Handle("/post", engine.ServePageComponent("PostPage", func(r *http.Request) (map[string]any, int) {
    post, err := db.GetPost(r.URL.Query().Get("slug"))
    if err != nil {
        return nil, http.StatusNotFound
    }
    return map[string]any{"post": post}, http.StatusOK
}))
```

### Mount multiple routes at once

`Mount` registers a set of component routes on an `http.ServeMux` in one call. Each component is served as a full HTML page. Use `WithDataMiddleware` to inject common data (auth, CSRF, etc.) shared across all routes:

```go
engine.Mount(mux, map[string]string{
    "GET /{$}":    "HomePage",
    "GET /about":  "AboutPage",
    "GET /posts":  "PostsPage",
})
```

### Inject data for all HTTP routes

`WithDataMiddleware` adds a function that enriches the data map on every HTTP-triggered render. Multiple middleware functions are applied in registration order:

```go
engine.WithDataMiddleware(func(r *http.Request, data map[string]any) map[string]any {
    data["currentUser"] = sessionUser(r)
    data["csrfToken"]   = csrf.Token(r)
    return data
})
```

### Validate components at startup

`ValidateAll` checks every registered component for unresolvable child component references and returns a slice of errors. Call it once at startup to surface missing-component problems before the first request:

```go
if errs := engine.ValidateAll(); len(errs) > 0 {
    for _, e := range errs {
        log.Printf("component error: %v", e)
    }
    os.Exit(1)
}
```

---

## 7. Expression Language Reference

Expressions are JavaScript-compatible in syntax and truthiness rules but are evaluated entirely in Go.

### Operators (highest to lowest precedence)

| Precedence | Operators | Example |
|---|---|---|
| 7 (highest) | Unary `!`, unary `-` | `!active`, `-x` |
| 6 | `**` (exponentiation) | `2 ** 10` |
| 5 | `*`, `/`, `%` | `price * qty` |
| 4 | `+`, `-` | `a + b` |
| 3 | `>`, `<`, `>=`, `<=`, `==`, `!=`, `===`, `!==` | `count > 0` |
| 2 | `&&` | `a && b` |
| 2 | `\|\|`, `??` | `a \|\| 'default'`, `val ?? 'n/a'` |
| 1 (lowest) | `? :` (ternary) | `ok ? 'yes' : 'no'` |

Member access (`obj.key`, `arr[i]`), optional chaining (`obj?.key`, `arr?.[i]`), and function calls (`fn(args)`) have the highest binding and are parsed as primary expressions.

**Optional chaining** short-circuits to `undefined` when the left-hand side is `null` or `undefined`, preventing runtime errors from missing nested data:

```html
{{ user?.address?.city ?? "Unknown" }}
{{ items?.[0]?.name }}
```

### Truthiness (JavaScript-compatible)

Falsy values: `false`, `0`, `""` (empty string), `null`, `undefined`.
Everything else is truthy, including empty arrays and empty objects.

### Type notes

- All numbers are `float64` internally (JavaScript number semantics).
- Accessing a missing map key or an out-of-range index returns `undefined` (not an error).
- `null` and `undefined` are distinct values. `null == undefined` is `true`; `null === undefined` is `false`.
- The `??` operator returns the right-hand side only when the left-hand side is `null` or `undefined` (not when it is `0` or `""`).

### Examples

```
{{ count > 0 ? count : "none" }}
{{ user.name ?? "Guest" }}
{{ items[0].title }}
{{ tags.length }}
{{ price * 1.2 }}
{{ active ? "active" : "" }}
```

---

## 8. Debug Mode

Debug mode annotates rendered HTML with structured HTML comments that describe component boundaries, expression values, conditional branch outcomes, and slot contents. The annotated output is valid HTML that renders identically in a browser but carries diagnostic information visible in DevTools or via `curl | grep -i debug`.

**Debug output is intended for development only. Never enable it in production.**

### Enabling via Go API

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Debug:        true,
})
```

### Enabling via CLI

```
htmlc render --debug -dir ./templates Card -props '{"title":"Hello"}'
htmlc page --debug -dir ./templates PostPage -props '{"slug":"intro"}'
```

### Example annotated output

```html
<!-- [htmlc:debug] component=PostPage file=templates/PostPage.vue -->
<article>
  <!-- [htmlc:debug] expr="post.Title" value=Hello World -->
  <h1>Hello World</h1>
  <!-- [htmlc:debug] v-if="post.Draft" → false: node skipped -->
  <!-- [htmlc:debug] slot=default nodes=2 -->
  <p>Body content here</p>
  <!-- [htmlc:debug] /slot=default -->
</article>
<!-- [htmlc:debug] /component=PostPage -->
```

### What the comments describe

| Comment pattern | Meaning |
|---|---|
| `component=Name file=path` | Start of a child component render |
| `/component=Name` | End of a child component render |
| `expr="..." value=...` | Expression evaluated during text interpolation |
| `v-if="..." → false: node skipped` | Conditional node that was not rendered |
| `slot=name nodes=N` | Start of slot content being rendered |
| `/slot=name` | End of slot content |

### AST inspection

The `htmlc ast` subcommand parses a `.vue` file and prints its template AST as indented pseudo-XML, without executing the render pipeline:

```
htmlc ast -dir ./templates PostPage
```

Example output:

```
Document
  Element[article] attrs=[]
    Element[h1] attrs=[]
      Text: "{{ post.Title }}"
    Element[p] v-if="post.Draft" attrs=[]
      Text: "Draft"
```

---

## 9. Custom Directives

The engine supports user-defined directives that extend the template language. A custom directive is any Go type that implements the `Directive` interface:

```go
type Directive interface {
    // Created is called before the element is rendered.
    // Mutate node.Attr or node.Data to affect what is emitted.
    Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error

    // Mounted is called after the element's closing tag has been written.
    // Bytes written to w appear immediately after the element.
    Mounted(w io.Writer, node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
}
```

Register a directive on the engine with `RegisterDirective` (no `v-` prefix):

```go
engine.RegisterDirective("my-dir", &MyDirective{})
```

Then use it in templates as `v-my-dir`:

```html
<div v-my-dir="someExpr" class="wrapper">content</div>
```

The `DirectiveBinding` passed to both hooks contains:

| Field | Type | Description |
|---|---|---|
| `Value` | `any` | Evaluated directive expression |
| `RawExpr` | `string` | Un-evaluated expression string |
| `Arg` | `string` | Argument after `:` (e.g. `"href"` in `v-my-dir:href`) |
| `Modifiers` | `map[string]bool` | Dot-separated modifiers (e.g. `{"prevent": true}`) |

### Built-in: VSwitch

`VSwitch` is a built-in custom directive that replaces the host element with a registered component determined at runtime by the directive expression — similar to `<component :is="...">` but using a directive syntax.

`v-switch` is **pre-registered and enabled by default** — no setup is required:

```go
// No explicit registration needed; v-switch works out of the box.
engine, err := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
```

```html
<!-- item.type evaluates to e.g. "CardWidget" — that component is rendered -->
<div v-switch="item.type" :title="item.title" />
```

All attributes other than `v-switch` (and any `v-switch:*` argument forms) are forwarded to the resolved component as props. Component names are matched case-insensitively. An error is returned if the component name does not exist in the registry.

To override the built-in with a custom implementation, supply it via `Options.Directives` or `Engine.RegisterDirective`:

```go
engine.RegisterDirective("switch", &myCustomSwitch{})
```
