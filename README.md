<p align="center">
  <img src="logo.svg" width="256" height="256" alt="htmlc logo" />
</p>

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
9. [Structured Logging](#structured-logging)
10. [Component Error Handling](#component-error-handling)
11. [Custom Directives](#custom-directives)
12. [Compatibility with Vue.js](#compatibility-with-vuejs)
13. [Testing](#testing)
14. [html/template Integration](#htmltemplate-integration)

---

## Overview

`htmlc` lets you write reusable HTML components in `.vue` files — the same format used by Vue.js — and render them server-side in Go. There is no Node.js dependency and no JavaScript executed at runtime. The `<script>` section of a `.vue` file is parsed and preserved in the output but never executed by the engine.

Key characteristics:

- **Static output** — every render call produces a fixed HTML string.
- **Scoped styles** — `<style scoped>` is supported; the engine rewrites selectors and injects scope attributes automatically.
- **Component composition** — components can nest other components from the same registry.
- **No reactivity** — `v-model`, `@event`, and other client-side directives are stripped from the output; they have no meaning in a server-side renderer.

---

## Template Syntax

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

## Directives

### Supported directives

| Directive | Supported | Notes |
|---|---|---|
| `v-text` | Yes | Sets element text content (HTML-escaped). Replaces all children. |
| `v-html` | Yes | Sets element inner HTML (not escaped). Replaces all children. Use with trusted content only. |
| `v-show` | Yes | Adds `style="display:none"` when the expression is falsy. Merges with any existing `style` attribute. |
| `v-if` | Yes | Renders the element only when the expression is truthy. |
| `v-else-if` | Yes | Must immediately follow a `v-if` or `v-else-if` element (whitespace between is allowed). |
| `v-else` | Yes | Must immediately follow a `v-if` or `v-else-if` element. |
| `v-switch` | Yes | See [v-switch syntax](#v-switch-syntax) below. Must be on a `<template>` element. |
| `v-case` | Yes | Child of `<template v-switch>`. Renders when its expression equals the switch value. |
| `v-default` | Yes | Child of `<template v-switch>`. Renders when no preceding `v-case` matched. |
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

### v-switch syntax

`v-switch` provides a concise switch-statement pattern for conditional
rendering. It must be placed on a `<template>` element; its children carry
`v-case` or `v-default` directives.

```html
<!-- Switch on a string value -->
<template v-switch="user.role">
  <AdminPanel v-case="'admin'" />
  <ModPanel   v-case="'mod'" />
  <UserPanel  v-default />
</template>

<!-- Switch on a numeric value -->
<template v-switch="step">
  <StepOne   v-case="1" />
  <StepTwo   v-case="2" />
  <StepThree v-case="3" />
  <p v-default>Unknown step</p>
</template>
```

Rules:
- Only the **first matching** `v-case` branch is rendered.
- `v-default` renders when **no** `v-case` matched; only the first
  `v-default` is evaluated if multiple are present.
- Children of `<template v-switch>` that carry neither `v-case` nor
  `v-default` are silently ignored.
- Values are compared with Go `==` (strict equality; no type coercion).
- Using `v-switch` on a non-`<template>` element is an error.

**Difference from Vue.js:** Vue.js does not yet ship `v-switch`/`v-case`/
`v-default` as stable built-ins (as of 2026). htmlc implements the semantics
proposed in [RFC #482](https://github.com/vuejs/rfcs/discussions/482).

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

## Component System

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
engine.RenderPage(ctx, w, "HomePage", map[string]any{
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

The engine uses **proximity-based resolution**: when a tag is encountered in a template, the engine first searches the same directory as the calling component, then walks toward the root one level at a time until a match is found.

For each directory level, the following name-folding strategies are tried in order:

1. Exact match (e.g. `my-card` → `my-card`)
2. First letter capitalised (e.g. `card` → `Card`)
3. Kebab-case to PascalCase (e.g. `my-card` → `MyCard`)
4. Case-insensitive scan

If no match is found via the proximity walk, the engine falls back to the flat registry (backward-compatible with single-directory projects).

**Example:** given this component tree:

```
components/
  Card.vue          ← generic root card
  blog/
    Card.vue        ← blog-specific card
    PostPage.vue    ← references <Card>
  admin/
    Card.vue        ← admin-specific card
    Dashboard.vue   ← references <Card>
```

- `blog/PostPage.vue` referencing `<Card>` resolves to `blog/Card.vue`
- `admin/Dashboard.vue` referencing `<Card>` resolves to `admin/Card.vue`
- A root template referencing `<Card>` resolves to `Card.vue`

The template in `blog/PostPage.vue` uses an unqualified `<Card>` tag:

```vue
<!-- blog/PostPage.vue -->
<template>
  <Card :title="post.title">{{ post.summary }}</Card>
</template>
```

Because `blog/Card.vue` exists in the same directory, it wins over the
root-level `Card.vue` and the `admin/Card.vue`.  No import statement or
path qualifier is needed.

A component at the root level (`Shell.vue`) referencing `<Card>` resolves to
the root-level `Card.vue` because there is no `Card` in the root directory's
walk other than itself:

```vue
<!-- Shell.vue (at root of components/) -->
<template>
  <div><Card :title="title" /></div>
</template>
```

#### Explicit cross-directory references

Use `<component is="dir/Name">` to reference a component in a specific directory, bypassing proximity resolution:

```html
<!-- always resolves to blog/Card.vue, regardless of caller location -->
<component is="blog/Card" />

<!-- root-relative: always resolves to Card.vue at ComponentDir root -->
<component is="/Card" />

<!-- dynamic version -->
<component :is="'admin/Card'" />
```

Path-based `is` values are resolved exactly (no name-folding) and return an error if the component is not found.

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

Use `<component :is="expr">` to render a component whose name is determined at runtime. Use `<component is="...">` for a static name. The value must be a non-empty string that names a registered component, a path (`dir/Name`), or a native HTML element:

```html
<!-- resolve from a variable -->
<component :is="activeView" />

<!-- static name -->
<component is="Card" :title="pageTitle" />

<!-- explicit path — always blog/Card.vue, no proximity walk -->
<component is="blog/Card" />

<!-- root-relative path -->
<component is="/Card" />

<!-- switch between components in a loop -->
<div v-for="item in items">
  <component :is="item.type" :data="item" />
</div>
```

- All attributes other than `is`, `:is`, or `v-bind:is` are forwarded to the resolved component as props.
- Slot content (default and named) works exactly as with a statically-named component.
- If the resolved name is a known HTML element (e.g. `"div"`, `"input"`), the tag is rendered as-is rather than looked up in the component registry.
- `is` or `:is` is required; omitting it or supplying a non-string value is a render error.

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

## Special Attributes

| Attribute | Behavior |
|---|---|
| `:key` | Rendered as `data-key="value"` in the HTML output. Not used for diffing. |
| `class` + `:class` | Both are collected and merged into a single `class` attribute. |
| `style` + `:style` | Both are collected and merged into a single `style` attribute. |

---

## Go API Quick Reference

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
err = engine.RenderPage(ctx, w, "Page", map[string]any{
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

### Pass a struct as component data

Any Go struct (or pointer to a struct) can be used as a value within the data
map. The engine resolves field access using json struct tags when present and
falls back to the Go field name (case-insensitive on the first letter):

```go
type Post struct {
    Title   string   `json:"title"`
    Author  string   `json:"author"`
    Tags    []string `json:"tags"`
    Draft   bool     `json:"draft"`
}

post := Post{
    Title:  "Getting started with htmlc",
    Author: "Alice",
    Tags:   []string{"go", "templates"},
    Draft:  false,
}

err = engine.RenderPage(ctx, w, "PostPage", map[string]any{
    "post": post,
})
```

Inside `PostPage.vue` the struct fields are accessible by their json tag names:

```vue
<template>
  <article>
    <h1>{{ post.title }}</h1>
    <p class="byline">by {{ post.author }}</p>
    <ul>
      <li v-for="tag in post.tags">{{ tag }}</li>
    </ul>
    <span v-if="post.draft" class="badge">Draft</span>
  </article>
</template>
```

#### Spread a struct onto a child component with `v-bind`

When a child component expects the same set of fields that a struct exposes,
you can spread the struct directly instead of mapping each field individually:

```vue
<!-- PostPage.vue -->
<template>
  <Layout>
    <!-- Spread the post struct: title, author, tags, draft become props of PostCard -->
    <PostCard v-bind="post" />
  </Layout>
</template>
```

The engine accepts any struct or `map[string]any` as the right-hand side of
`v-bind`. Embedded struct fields are promoted and resolved just as if they had
been declared directly on the outer struct.

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

err = engine.RenderPage(ctx, w, "Page", data)
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

`ValidateAll` checks every registered component for unresolvable child component references and returns a slice of errors. It uses the same proximity-based resolution as the renderer, so a reference that would succeed at render time will not generate a false-positive validation error. Call it once at startup to surface missing-component problems before the first request:

```go
if errs := engine.ValidateAll(); len(errs) > 0 {
    for _, e := range errs {
        log.Printf("component error: %v", e)
    }
    os.Exit(1)
}
```

---

## Expression Language Reference

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

## Debug Mode

Enable debug mode via `Options.Debug` (or the `-debug` CLI flag) to annotate the rendered HTML with component metadata. When active, the root element of each rendered component receives three `data-htmlc-*` attributes:

| Attribute | Value |
|---|---|
| `data-htmlc-component` | Component name (registry key, original casing) |
| `data-htmlc-file` | Relative path to the `.vue` source file |
| `data-htmlc-props` | HTML-escaped JSON-encoded props passed to the component |

If the props cannot be JSON-serialised (for example, a prop value is an `io.Reader`), `data-htmlc-props-error` is emitted instead of `data-htmlc-props`.

Fragment templates (components with no single root element) are silently skipped — there is no element to annotate.

**Example output** for `<HeroBanner headline="Hello">` with debug mode enabled:

```html
<section
  id="hero"
  data-htmlc-component="HeroBanner"
  data-htmlc-file="components/HeroBanner.vue"
  data-htmlc-props="{&quot;headline&quot;:&quot;Hello&quot;}"
>
  ...
</section>
```

The `data-htmlc-*` attributes are standard HTML `data-*` attributes and are accessible via the browser's `dataset` API (`el.dataset.htmlcComponent`, `el.dataset.htmlcProps`, etc.).

> **Note:** Debug mode adds extra attributes and increases output size. Do not use it in production for performance reasons. It does not corrupt the document — all attribute values are HTML-escaped before emission.

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

## Structured Logging

The engine can emit one structured log record per component rendered using the standard library's `log/slog` package. Pass a `*slog.Logger` to `Options.Logger` to enable it.

Records are emitted at `slog.LevelDebug` for successful renders and `slog.LevelError` for failed renders. Each record includes:

| Attribute | Type | Description |
|---|---|---|
| `component` | string | Resolved component name (file base name without `.vue`) |
| `duration` | duration | Wall-clock time for the component subtree |
| `bytes` | int64 | Bytes written by the component subtree |
| `error` | error | Set only on `ERROR`-level records |

The two log messages are available as exported constants so test code can filter records without hardcoding strings:

```go
htmlc.MsgComponentRendered // "component rendered"
htmlc.MsgComponentFailed   // "component render failed"
```

A nil `Logger` (the default) disables all slog output with no overhead on the hot path.

### Example

```go
import (
    "log/slog"
    "os"

    "github.com/dhamidi/htmlc"
)

logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Logger:       logger,
})
```

Example log output (one line per component, formatted for readability):

```json
{"time":"...","level":"DEBUG","msg":"component rendered","component":"Card","duration":42000,"bytes":128}
{"time":"...","level":"DEBUG","msg":"component rendered","component":"Page","duration":95000,"bytes":512}
```

Records appear in post-order: leaf components are logged before their parents.

### Notes

- `Logger` and `Debug` mode are independent — both can be enabled simultaneously.
- If concurrent root renders sharing output are needed, use separate engine instances; a single engine's root-level instrumentation uses a shared `countingWriter`.

---

## Component Error Handling

By default, the first component render failure aborts the entire page render and the response writer receives nothing. Two complementary features make failures observable and recoverable.

### Structured component path

Every `*RenderError` now carries a `ComponentPath []string` field — the ordered list of component names from the page root to the failing component:

```go
var rerr *htmlc.RenderError
err := engine.RenderPage(ctx, w, "HomePage", data)
if errors.As(err, &rerr) {
    fmt.Println(strings.Join(rerr.ComponentPath, " > "))
    // Output: HomePage > Layout > Sidebar
}
```

`RenderError.Error()` automatically includes the path when depth > 1:

```text
HomePage > Layout > Sidebar: render Sidebar.vue: expr "items.length": type error
```

### In-place error handler

Register a `ComponentErrorHandler` on `Options` to intercept component failures, write a placeholder into the output buffer, and allow rendering to continue:

```go
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir:          "templates/",
    ComponentErrorHandler: htmlc.HTMLErrorHandler(), // built-in dev helper
})
err := engine.RenderPage(ctx, w, "HomePage", data)
// err == nil; w contains the page with <div class="htmlc-error"> placeholders
```

The `ComponentErrorHandler` type is:

```go
type ComponentErrorHandler func(w io.Writer, path []string, err error) error
```

- Return `nil` to write a placeholder and continue rendering sibling nodes.
- Return a non-nil error to abort the entire render immediately.

When the handler returns `nil` for every failure, `RenderPage` returns `nil` and the partial page (with placeholders) is written to `w` exactly like a successful render.

### Built-in development helper

`HTMLErrorHandler()` returns a handler that renders a visible `<div class="htmlc-error">` placeholder at each failure site:

```html
<div class="htmlc-error" data-path="HomePage &gt; Sidebar">
  render Sidebar.vue: expr "items.length": type error
</div>
```

Both `path` and the error message are HTML-escaped. The `htmlc-error` class lets you style the placeholder with CSS.

### Nil handler (default)

A nil `ComponentErrorHandler` preserves the existing behaviour: the first component error aborts the render and `w` receives nothing.

---

## Custom Directives

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

### Example: VHighlight

`VHighlight` is the canonical example of a custom directive in htmlc. It
mirrors the `v-highlight` directive from the
[Vue.js custom directives guide](https://vuejs.org/guide/reusability/custom-directives.html).

`VHighlight` is **not** auto-registered; you must opt in:

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Directives: htmlc.DirectiveRegistry{
        "highlight": &htmlc.VHighlight{},
    },
})
```

Then use `v-highlight` in templates:

```html
<p v-highlight="'yellow'">Highlight this text bright yellow</p>
<p v-highlight="theme.accentColour">Dynamic colour from scope</p>
```

The directive sets `background:<colour>` on the element's `style` attribute,
merging with any existing inline styles.

`VHighlight` implements only the `Created` hook because htmlc is server-side —
there is no DOM `mounted` event. This is the htmlc equivalent of Vue's
`mounted` hook on a custom directive.

### DirectiveWithContent

A directive that wants to replace the element's children with custom HTML can
implement the optional `DirectiveWithContent` interface:

```go
type DirectiveWithContent interface {
    Directive
    // InnerHTML returns the raw HTML to use as the element's inner content.
    // Return ("", false) to fall back to normal child rendering.
    InnerHTML() (html string, ok bool)
}
```

After `Created` is called, the renderer checks whether the directive implements
`DirectiveWithContent`.  If `InnerHTML()` returns `(s, true)` the string `s` is
written verbatim between the element's opening and closing tags — the template
children are skipped.  This is how external directives like `v-syntax-highlight`
inject processed markup without modifying the Go code.

> **Note:** The old `v-switch` component-dispatch directive (which replaced
> the host element's tag at runtime) has been removed. Use
> `<component :is="expr">` for dynamic component dispatch, or the new built-in
> `v-switch`/`v-case`/`v-default` for switch-style conditional rendering.

### External Directives

`htmlc build` can discover and invoke **external directives** — plain
executables living in the component file tree.  No recompilation of `htmlc` is
required.

#### Discovery

When `htmlc build` starts it walks the `-dir` component tree.  Any file whose
base name (without extension) matches `v-<directive-name>` **and** has at least
one executable bit set is registered as an external directive under the name
`<directive-name>` (the `v-` prefix is stripped).  Files inside hidden
directories (names starting with `.`) are skipped.  The directive name must be
lower-kebab-case (`[a-z][a-z0-9-]*`).

```
templates/
  directives/
    v-syntax-highlight    ← registered as "syntax-highlight"
    v-upper.sh            ← registered as "upper"
```

Convention: place directive executables alongside the components that use them
or in a dedicated `directives/` subdirectory.

#### Protocol

Each directive executable is spawned **once** at the start of a build.
`htmlc` communicates with it over **newline-delimited JSON** (NDJSON) on
stdin/stdout.  Stderr from the subprocess is forwarded verbatim to `htmlc`'s
stderr.  When the build finishes `htmlc` closes the subprocess's stdin; the
subprocess should drain its input and exit cleanly.

**Request envelope** (htmlc → directive):

```json
{
  "hook":    "created" | "mounted",
  "id":      "<opaque string echoed in response>",
  "tag":     "<element tag name>",
  "attrs":   { "<name>": "<value>", ... },
  "text":    "<concatenated text content of element's children>",
  "binding": {
    "value":     <evaluated expression>,
    "raw_expr":  "<unevaluated expression string>",
    "arg":       "<directive argument or ''>",
    "modifiers": { "<mod>": true, ... }
  }
}
```

**`created` response** (directive → htmlc):

```json
{
  "id":         "<same id as request>",
  "tag":        "<optional: new tag name>",
  "attrs":      { "<name>": "<value>", ... },
  "inner_html": "<optional: replacement inner HTML>",
  "error":      "<optional: non-empty aborts rendering of this element>"
}
```

- `inner_html` — if present and non-empty, replaces the element's children
  verbatim (not escaped).

**`mounted` response** (directive → htmlc):

```json
{
  "id":    "<same id as request>",
  "html":  "<optional: HTML injected after the element's closing tag>",
  "error": "<optional: non-empty aborts rendering>"
}
```

If the directive outputs a line that is not valid JSON or echoes the wrong
`id`, `htmlc` logs a warning to stderr and treats the hook as a no-op.

#### Example: syntax highlighting

```vue
<template>
  <pre v-syntax-highlight="'go'">
func main() {
    fmt.Println("hello")
}
  </pre>
</template>
```

A minimal `v-syntax-highlight` shell script skeleton:

```sh
#!/usr/bin/env node
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin, terminal: false });
rl.on('line', (line) => {
    const req = JSON.parse(line);
    if (req.hook === 'created') {
        const highlighted = highlight(req.binding.value, req.text); // your highlighter
        process.stdout.write(JSON.stringify({
            id: req.id,
            inner_html: highlighted,
        }) + '\n');
    } else {
        process.stdout.write(JSON.stringify({ id: req.id, html: '' }) + '\n');
    }
});
```

Test the directive interactively:

```sh
echo '{"hook":"created","id":"1","tag":"pre","attrs":{},"text":"fmt.Println(1)","binding":{"value":"go","raw_expr":"'\''go'\''","arg":"","modifiers":{}}}' \
  | ./v-syntax-highlight
```

---

## Custom Elements

A `.vue` component can opt in to **Web Component (Custom Element)** compilation by
including a `<script customelement>` block. The engine wraps the rendered template
in the component's derived tag name and writes the JavaScript to a `scripts/`
directory alongside the HTML.

### Declaring a custom element component

```html
<!-- components/ui/DatePicker.vue -->
<template>
  <div class="date-picker">
    <input type="date" />
  </div>
</template>

<script customelement>
class DatePickerElement extends HTMLElement {
  connectedCallback() {
    this.attachShadow({ mode: 'open' }).innerHTML = this.innerHTML;
  }
}
customElements.define('ui-date-picker', DatePickerElement);
</script>
```

The **tag name** is derived automatically from the component's file path using
kebab-case: `ui/DatePicker.vue` → `ui-date-picker`.

For **Declarative Shadow DOM** (streaming SSR), use the `shadowdom` attribute:

```html
<script customelement shadowdom>…</script>          <!-- mode: open  -->
<script customelement shadowdom="closed">…</script>  <!-- mode: closed -->
```

### Output structure

`htmlc build` writes collected scripts to `<out>/scripts/` alongside the HTML:

```
out/
  index.html
  about.html
  scripts/
    a1b2c3d4e5f6a7b8.js   ← ui-date-picker
    ff00112233445566.js   ← widgets-shape-canvas
```

Scripts are **deduplicated by content hash**: the same script used across
multiple pages produces exactly one file in `scripts/`. File names are the
first 16 hex characters of the SHA-256 hash of the script content.

The `scripts/` directory is only created when at least one custom element
component is rendered. Projects with no `<script customelement>` blocks
produce no `scripts/` directory.

### Dev server

`htmlc build -dev :8080` serves scripts from memory at `/scripts/`, rebuilding
automatically when source files change.

### Go API

```go
// Serve scripts over HTTP — the engine manages the collector internally
http.Handle("/scripts/", http.StripPrefix("/scripts/", engine.ScriptHandler()))

// Or write scripts to disk during a static build
engine.WriteScripts(filepath.Join(outDir, "scripts"))

// For advanced use, access the collector directly
collector := engine.Collector()
importMap := collector.ImportMapJSON("/scripts/")
```

### importMap template function

An `importMap` template function is automatically available in all page templates
and child components when rendering with `RenderPage` or `RenderFragment`:

```html
<head>
  <script type="importmap">{{ importMap("/scripts/") }}</script>
</head>
```

`importMap(urlPrefix)` returns the same JSON as `collector.ImportMapJSON(urlPrefix)`.
If no custom element scripts have been collected, it returns a valid empty import map.

---

## Compatibility with Vue.js

htmlc uses `.vue` Single File Component syntax and many of the same directive
names as Vue.js, but it is a **server-side-only renderer** with intentional
differences. This section documents where htmlc diverges from or extends
standard Vue.js behaviour.

### Directives

| Directive | Vue.js behaviour | htmlc behaviour |
|---|---|---|
| `v-switch` / `v-case` / `v-default` | Not in stable Vue.js (proposed in RFC #482) | Built-in; `v-switch` on `<template>`, children carry `v-case` / `v-default` |
| `v-on` / `@event` | Client-side event handler | Stripped from output |
| `v-model` | Two-way binding | Stripped from output |
| `v-cloak` | Hide until mounted | Not relevant (no mounting); ignored |
| `v-memo` | Memoised subtree | Not implemented |
| `v-once` | Render once, skip future updates | Accepted; no-op (server always renders once) |
| `<component :is="...">` | Dynamic component | Supported |

### Expression language

The htmlc expression evaluator supports a subset of JavaScript expressions. The
following are **not** supported:

- Template literals (backtick strings).
- Arrow functions and `function` keyword.
- `new`, `delete`, `typeof`, `instanceof`.
- Assignment operators (`=`, `+=`, etc.) and increment/decrement (`++`, `--`).
- Filters (`{{ value | filter }}`).

### Map iteration order

`v-for` over a Go `map` iterates in `reflect.MapKeys()` order, which is not
guaranteed to match insertion order. Vue.js preserves insertion order for
`Object.keys()`. Use a slice of objects if deterministic order is required.

### Scoped styles

htmlc supports `<style scoped>` with the same semantics as Vue.js SFCs: a
unique `data-v-XXXXXXXX` attribute is added to all elements rendered by the
component, and CSS selectors are rewritten to target that attribute.

### No reactivity

htmlc has no virtual DOM, no reactivity system, and no JavaScript runtime.
Every render call produces a fixed HTML string. Props are plain Go values
passed at render time.

---

## Testing

The `htmlctest` package provides a lightweight harness for writing unit and
integration tests for `.vue` components. Tests use an in-memory filesystem —
no temporary directories, no OS setup — and a fluent DOM-query API to assert
that rendered output contains the expected elements, text, and attributes.

### Import

```go
import "github.com/dhamidi/htmlc/htmlctest"
```

### Quick start: `Build` shorthand

`Build` wraps a template snippet in `<template>…</template>`, registers it as a
component named `Root`, and returns a `*Harness` ready to render:

```go
func TestGreeting(t *testing.T) {
    htmlctest.Build(t, `<p class="greeting">Hello {{ name }}!</p>`).
        Fragment("Root", map[string]any{"name": "World"}).
        Find(htmlctest.ByTag("p").WithClass("greeting")).
        AssertText("Hello World!")
}
```

### Multiple components: `NewHarness`

When the component under test references child components, use `NewHarness` to
register all required files in one call:

```go
func TestCard(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Badge.vue": `<template><span class="badge">{{ label }}</span></template>`,
        "Card.vue": `<template>
            <div class="card">
                <h2>{{ title }}</h2>
                <Badge :label="status" />
            </div>
        </template>`,
    })

    h.Fragment("Card", map[string]any{
        "title":  "Order #42",
        "status": "shipped",
    }).
        Find(htmlctest.ByTag("h2")).AssertText("Order #42").
        Find(htmlctest.ByClass("badge")).AssertText("shipped")
}
```

### Assertion methods

| Method | Checks |
|---|---|
| `AssertHTML(want)` | Exact HTML string after whitespace normalisation |
| `Find(query)` | Returns a `Selection` of matched nodes |
| `AssertExists()` | At least one node matched |
| `AssertNotExists()` | No nodes matched |
| `AssertCount(n)` | Exactly `n` nodes matched |
| `AssertText(text)` | Normalised text content of the first matched node |
| `AssertAttr(attr, value)` | Named attribute of the first matched node |

All `Assert*` methods call `t.Fatalf` on failure and return the receiver so
that assertions chain:

```go
result.Find(htmlctest.ByTag("li")).
    AssertCount(3).
    AssertText("First item") // checks text of the first <li>
```

### Query constructors

| Constructor | Matches |
|---|---|
| `ByTag("div")` | Elements by tag name (case-insensitive) |
| `ByClass("active")` | Elements that have the given CSS class |
| `ByAttr("data-id", "42")` | Elements where `data-id="42"` |

Queries are immutable values. Use combinators to narrow the match:

```go
// <li class="active" data-id="1"> inside a <ul>
htmlctest.ByTag("li").
    WithClass("active").
    WithAttr("data-id", "1").
    Descendant(htmlctest.ByTag("ul"))
```

### Testing v-for output

```go
func TestList(t *testing.T) {
    htmlctest.Build(t, `
        <ul>
            <li v-for="item in items">{{ item }}</li>
        </ul>
    `).
        Fragment("Root", map[string]any{
            "items": []string{"alpha", "beta", "gamma"},
        }).
        Find(htmlctest.ByTag("li")).
        AssertCount(3)
}
```

### Testing conditional rendering

```go
func TestBadge_Hidden(t *testing.T) {
    htmlctest.Build(t, `<span v-if="show" class="badge">NEW</span>`).
        Fragment("Root", map[string]any{"show": false}).
        Find(htmlctest.ByClass("badge")).
        AssertNotExists()
}
```

---

## html/template Integration

Already using `html/template`? htmlc works alongside your existing templates
with no changes required. Use `RegisterTemplate` to bring any existing partial
into an htmlc component tree, and `CompileToTemplate` to export new `.vue`
components to any library that requires `*html/template.Template`. Adopt
incrementally — one component at a time.

### CompileToTemplate — export a .vue component for stdlib code

`CompileToTemplate` compiles a named component (and all components it
transitively references) into a single `*html/template.Template`:

```go
tmpl, err := engine.CompileToTemplate("Card")
// tmpl is ready to Execute; sub-components appear as named define blocks.
// Template names are lowercased: "Card" → "card".

var buf strings.Builder
tmpl.Execute(&buf, map[string]any{"title": "Hello"})
```

Scoped `<style>` blocks are stripped from the output. Non-recoverable errors
are returned as `*ConversionError` (also testable with
`errors.Is(err, ErrConversion)`).

#### Supported constructs (vue → tmpl)

| Vue syntax | Go template output |
|---|---|
| `{{ ident }}` | `{{.ident}}` |
| `{{ a.b.c }}` | `{{.a.b.c}}` |
| `:attr="ident"` / `v-bind:attr="ident"` | `attr="{{.ident}}"` |
| `v-if="ident"` | `{{if .ident}}…{{end}}` |
| `v-else-if="ident"` | `{{else if .ident}}` |
| `v-else` | `{{else}}` |
| `v-for="item in list"` | `{{range .list}}…{{end}}` |
| `v-show="ident"` | injects `style="display:none"` conditionally |
| `v-html="ident"` | `{{.ident}}` + warning |
| `v-text="ident"` | `{{.ident}}` (children discarded) |
| `v-bind="ident"` (spread) | `{{.ident}}` + warning |
| `<template v-switch="ident">` | `{{if eq .ident "…"}}…` chain |
| `<slot>` | `{{block "default" .}}…{{end}}` |
| `<slot name="N">` | `{{block "N" .}}…{{end}}` |
| `<my-component>` (zero props) | `{{template "my-component" .}}` |

Complex expressions, bound props on child components, and custom directives
return `*ConversionError`.

### RegisterTemplate — use existing partials inside .vue files

`RegisterTemplate` imports an existing `*html/template.Template` into the
engine's registry as a virtual htmlc component under the given name:

```go
subTmpl, _ := html.template.New("foot-note").Parse("<footer>{{.year}}</footer>")
err := engine.RegisterTemplate("foot-note", subTmpl)
// "foot-note" can now be used as a child component in .vue files.
```

All `{{define}}` blocks within `tmpl` are also registered under their block
names. Conversion is validated at registration time; unsupported constructs
(e.g. `{{with}}`) return `*ConversionError` and nothing is registered.
Registering the same name twice keeps the latest value ("last write wins").

#### Supported constructs (tmpl → vue)

| Template syntax | Vue output |
|---|---|
| `{{.name}}` | `{{ name }}` |
| `{{.a.b}}` | `{{ a.b }}` |
| `{{if .cond}}…{{end}}` | `<div v-if="cond">…</div>` |
| `{{range .items}}…{{end}}` | `<ul><li v-for="item in items">…</li></ul>` |
| `{{block "N" .}}…{{end}}` | `<slot name="N">…</slot>` |
| `{{template "Name" .}}` | `<Name />` |

`{{with}}`, variable assignments, and multi-command pipelines return
`*ConversionError`.

For a complete walkthrough, see `docs/tutorial-template-integration.md`.
