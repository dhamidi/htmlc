---
name: htmlc
description: Explains how htmlc works — a server-side Go template engine that uses Vue.js .vue file syntax but renders entirely in Go with no JavaScript runtime. Use when working with htmlc components, writing .vue templates for Go servers, or understanding how htmlc differs from Vue.js. Triggers on "htmlc", "vue template in Go", "server-side vue", "htmlc component", "htmlc directive", "htmlc slot", "htmlc scoped style".
---

# htmlc

htmlc is a server-side Go template engine that uses `.vue` file syntax but renders entirely in Go — no JavaScript runtime, no reactivity, no hydration.

## Key differences from Vue.js

- `<script>` / `<script setup>` → **parse error** (not silently ignored)
- `<script customelement>` is the only valid `<script>` variant (Web Component registration only)
- `v-model`, `@event` → stripped from output (no-op, no error)
- No reactivity, computed properties, or watchers
- Expression language is a JS-like subset: no arrow functions, no template literals, no filters (`| filterName`)
- Component registration is automatic (directory walk) — no `import` or `components: {}` needed
- Props have no validation and no defaults
- Map iteration order is not guaranteed (Go `reflect.MapKeys`)
- `v-switch` / `v-case` / `v-default` are implemented (Vue RFC #482) — not standard Vue
- Render output is a plain HTML string; `<style scoped>` blocks are collected and injected server-side
- Numbers are always `float64`; truthiness follows JavaScript semantics

See `references/directives.md` for the full directive reference and `references/vue-comparison.md` for a complete comparison table.

## Go API (quick reference)

```go
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    FS:           templateFS, // optional embed.FS
    Reload:       true,       // hot-reload for dev
})

engine.RegisterFunc("url", buildURL) // available in all templates

// Full page (injects <style> before </head>)
engine.RenderPage(w, "HomePage", map[string]any{"title": "Hello"})

// Fragment (prepends <style> block)
engine.RenderFragment(ctx, w, "Card", map[string]any{"title": "Card"})
```
