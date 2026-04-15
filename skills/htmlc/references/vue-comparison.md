# htmlc vs Vue.js: Comparison

The fundamental difference: **Vue.js is a client-side reactive framework that supports SSR. htmlc is a server-side Go template engine that uses Vue's `.vue` file format as its authoring syntax.** The `.vue` format is borrowed for ergonomics — not for runtime compatibility with Vue.js.

## Feature comparison

| Dimension | Vue.js | htmlc |
|---|---|---|
| **Execution environment** | Browser or Node.js (SSR) | Go server only; no JS runtime |
| **Reactivity** | Full reactive system (`ref`, `reactive`, `computed`, watchers) | None — renders once per request |
| **Virtual DOM** | Yes — diffing and patching | No — direct HTML string output |
| **`<script>` section** | Defines component logic, data, methods | **Rejected with parse error** |
| **`<script setup>`** | Composition API setup | **Rejected with parse error** |
| **`v-model`** | Two-way data binding | Stripped from output |
| **`@event` / `v-on`** | Event handlers | Stripped from output |
| **`v-switch`** | Not yet stable (RFC #482) | Implemented |
| **Map iteration order** | Insertion order (JS Map) | Not guaranteed (Go `reflect.MapKeys()`) |
| **Expression language** | Full JavaScript | Custom JS-like subset |
| **Filters** (`\| filterName`) | Vue 2 only | Not implemented |
| **Component registration** | `import` + `components: {}` | Automatic from directory walk |
| **Props validation** | `defineProps` with types and defaults | No validation; no defaults |
| **Scoped styles** | `data-v-*` attributes | `data-v-*` attributes (same mechanism, different hash) |
| **Slot scope** | Caller scope | Caller scope (same) |
| **Custom directive lifecycle** | 7 hooks (created, beforeMount, mounted, beforeUpdate, updated, beforeUnmount, unmounted) | 2 hooks: `Created` and `Mounted` |
| **Data flow** | Props down, events up; Vuex/Pinia for global state | Props down only; `RegisterFunc` for global helpers |
| **Hot reload** | Vite HMR | `Options{Reload: true}` — mtime-based file re-parsing |
| **Build output** | JS bundles | Plain HTML strings (or static files via `htmlc build`) |
| **Web Components** | `defineCustomElement` | `<script customelement>` section |
| **Testing** | Vue Test Utils | `htmlctest` package with fluent harness |
| **Interop** | — | Bidirectional conversion with Go `html/template` |

## Expression language differences

htmlc's expression evaluator is a JS-like subset. These Vue.js expression patterns do **not** work in htmlc:

```js
// Arrow functions — NOT supported
items.filter(i => i.active)

// Template literals — NOT supported
`Hello, ${name}!`

// new keyword — NOT supported
new Date()

// Assignment — NOT supported
count++

// Filters (Vue 2) — NOT supported
{{ date | formatDate }}
```

These patterns **do** work (same in both):

```js
// Ternary
isActive ? 'active' : 'inactive'

// Optional chaining
user?.profile?.avatar

// Nullish coalescing
title ?? 'Untitled'

// Member access and length
items.length
items[0].name
```

## What `<script>` means in each context

In Vue.js, `<script>` defines the component's logic:
```vue
<script setup>
const props = defineProps({ title: String })
const count = ref(0)
</script>
```

In htmlc, `<script>` causes a **parse error**. There is no component logic — data comes entirely from props passed by the Go caller. Use `<script customelement>` only for Web Component registration.

## Component registration

Vue.js requires explicit registration:
```js
import Card from './Card.vue'
export default { components: { Card } }
```

htmlc discovers components automatically by walking the `ComponentDir`. Any `.vue` file in the directory tree is available by its base name. No imports, no registration.

## Props

Vue.js:
```js
const props = defineProps({
  title: { type: String, required: true, default: 'Untitled' },
  count: { type: Number, default: 0 }
})
```

htmlc — no validation, no defaults, just pass attributes:
```html
<Card title="Hello" :count="42" />
```

Missing props evaluate to `undefined` in expressions. There is no way to declare defaults in the component itself.

## Reactivity

Vue.js components re-render when reactive state changes. htmlc components render exactly once per call. There is no concept of state, watchers, computed properties, or re-rendering. If data changes, the Go handler calls `RenderPage` or `RenderFragment` again with new data.

## Scoped styles

Both Vue.js and htmlc use `data-v-*` attribute stamping for scoped styles. The mechanism is the same; the hash algorithm differs (htmlc uses FNV-1a 32-bit hash of the file path). CSS written for Vue's scoped styles will work in htmlc without modification.
