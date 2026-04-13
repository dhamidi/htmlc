---
name: htmlc
description: Explains how htmlc works — a server-side Go template engine that uses Vue.js .vue file syntax but renders entirely in Go with no JavaScript runtime. Use when working with htmlc components, writing .vue templates for Go servers, or understanding how htmlc differs from Vue.js. Triggers on "htmlc", "vue template in Go", "server-side vue", "htmlc component", "htmlc directive", "htmlc slot", "htmlc scoped style".
---

# htmlc

htmlc is a **server-side Go template engine** that borrows Vue.js Single File Component (`.vue`) syntax for authoring but renders entirely in Go. There is no JavaScript runtime, no reactivity, no virtual DOM, and no client-side hydration. Every render call produces a plain HTML string.

## Core mental model

- `.vue` files are **Go templates in disguise** — the format is Vue's, the execution is Go's.
- `<script>` and `<script setup>` blocks are **rejected at parse time** — they have no meaning server-side.
- Directives like `v-model` and `@event` are **stripped silently** — they are client-side only.
- Data flows **one way**: props passed in → HTML string out.

## Single File Component structure

```vue
<template>
  <!-- required; the HTML template -->
</template>

<style scoped>
  /* optional; collected and injected as a <style> block */
</style>

<script customelement>
  // optional; JS for Web Component registration only
</script>
```

`<script>` and `<script setup>` cause a **parse error**. Use `<script customelement>` only for Web Component registration.

## Expression language

htmlc uses a custom JS-like expression evaluator — not Go's `html/template` syntax.

Supported: arithmetic (`+`, `-`, `*`, `/`, `%`, `**`), comparison (`===`, `!==`, `>`, `<`, `>=`, `<=`), logical (`&&`, `||`, `!`), nullish coalescing (`??`), optional chaining (`obj?.key`, `arr?.[i]`), ternary (`cond ? a : b`), member access (`obj.key`, `arr[i]`, `arr.length`), function calls, array literals `[a, b]`, object literals `{ key: val }`.

Not supported: arrow functions, template literals, `new`, `delete`, assignment operators, filters (`| filterName`).

Numbers are always `float64`. Truthiness follows JavaScript semantics (non-empty arrays are truthy).

## Directives

See `references/directives.md` for the full reference. Key directives:

| Directive | Behavior |
|---|---|
| `{{ expr }}` | Mustache interpolation; HTML-escaped |
| `v-if` / `v-else-if` / `v-else` | Conditional rendering |
| `v-for="item in items"` | Iterate slices, maps, or integer ranges |
| `v-show="expr"` | Adds `style="display:none"` when falsy; element always present in DOM |
| `v-bind:attr` / `:attr` | Dynamic attribute binding |
| `v-bind="obj"` | Spread `map[string]any` or struct as attributes |
| `:class` | Object `{ active: bool }` or array `[classA, classB]` |
| `:style` | Object with camelCase keys → kebab-case CSS |
| `v-text="expr"` | Sets text content; HTML-escaped; replaces children |
| `v-html="expr"` | Sets inner HTML; **not** escaped |
| `v-switch="expr"` | Switch pattern on `<template>` (implements Vue RFC #482) |
| `v-pre` | Skips all processing for the subtree |
| `v-slot` / `#name` | Targets named or scoped slots |

Stripped silently (client-side only): `v-on` / `@event`, `v-model`, `v-cloak`, `v-memo`, `v-once`.

## Component system

Components are discovered automatically by walking the `ComponentDir`. No imports or registration needed.

**Resolution order** for a tag like `<Card />`:
1. Same directory as the calling component
2. Walk toward root, one level at a time
3. At each level: exact match → capitalize first letter → kebab-to-PascalCase → case-insensitive scan
4. Fall back to flat registry

Cross-directory: `<component is="blog/Card" />` or `<component :is="'admin/Card'" />`.

**Props** are passed as HTML attributes:
- Static: `title="Hello"`
- Dynamic: `:title="expr"`
- Spread: `v-bind="propsMap"`

Each component renders in an **isolated scope** — no automatic parent scope inheritance.

**Slots:**
- Default: `<slot />` in child; inner content in caller
- Named: `<slot name="header" />` in child; `<template #header>` in caller
- Scoped: `<slot :item="item" />` in child; `<template #default="{ item }">` in caller
- Slot content is always evaluated in the **caller's** scope

## Go API

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    FS:           templateFS,  // optional embed.FS
    Reload:       true,        // hot-reload for dev
})

engine.RegisterFunc("url", buildURL)  // available in all templates

// Full page (injects <style> before </head>)
engine.RenderPage(w, "HomePage", map[string]any{"title": "Hello"})

// Fragment (prepends <style> block)
engine.RenderFragment(ctx, w, "Card", map[string]any{"title": "Card"})

// http.Handler
http.Handle("/card", engine.ServeComponent("Card"))
```

## How htmlc differs from Vue.js

See `references/vue-comparison.md` for the full comparison table.

The fundamental difference: **Vue.js is a client-side reactive framework that supports SSR. htmlc is a server-side Go template engine that uses Vue's `.vue` file format as its authoring syntax.** The format is borrowed for ergonomics — not for runtime compatibility.

Key differences to keep in mind:
- `<script>` blocks → **parse error** in htmlc (not silently ignored)
- `v-model`, `@event` → **stripped** from output
- No reactivity, no computed properties, no watchers
- Expression language is a JS-like **subset** — no arrow functions, no template literals
- Component registration is **automatic** from directory walk, not via `import`
- Props have **no validation and no defaults**
- Map iteration order is **not guaranteed** (Go `reflect.MapKeys()`)
- `v-switch` is **implemented** in htmlc but not yet stable in Vue.js
