# RFC 006 (Updated): Custom Element Compilation

- **Status**: Draft (updated 2026-03-25)
- **Original date**: 2026-03-16
- **Author**: TBD

---

## 1. Motivation

`htmlc` today is a **server-side-only renderer**: it processes `.vue` files into static HTML strings and has no mechanism for delivering client-side interactivity. When a page needs an interactive island — a date-picker, a live canvas, a tabbed panel — the author must either reach for a full JavaScript framework bundled separately, or hand-write Custom Element boilerplate that duplicates the component boundary already declared in the `.vue` file.

The core problem this RFC solves is best illustrated by what today's workaround looks like in practice:

```go
// Without RFC 006 — today's only option
fmt.Fprintf(w, `<shape-canvas>
  <canvas width="%d" height="%d" data-src="%s"></canvas>
</shape-canvas>
<script>...paste 30 lines of JS here...</script>`, width, height, url)
```

One escaped wrong quote and the whole thing silently breaks. The component boundary in `.vue` provides zero protection because the JS is authored separately.

### Why Web Components, and why now

Custom Elements (part of the Web Components standard) are the right primitive: the browser instantiates a class when the element appears in the document; no runtime, no bundler, no framework. But Web Components have a historically poor developer experience. The table below maps the most commonly cited complaints to what this RFC delivers.

#### Common complaints vs. RFC 006

| Complaint | RFC 006 verdict |
|---|---|
| **SSR is broken** — no standard story; components flash blank until JS runs | **Solved** — htmlc renders the template server-side; with `shadowdom`, output is Declarative Shadow DOM that browsers attach before any JS runs |
| **Async upgrade breaks frameworks** — CE upgrade is asynchronous, breaks synchronous reactivity | **Sidesteps** — htmlc has no client-side reactivity loop to break; interactivity is opt-in |
| **Shadow DOM styling friction** — global styles don't cascade in; scoping must be duplicated per component | **Improved** — `<style scoped>` is automatically placed inside `<template shadowrootmode>` for shadow DOM components; authors write styles once |
| **Form participation requires FACE boilerplate** — CE inputs are excluded from form submissions by default | **Solved** — server renders `<input type="hidden" :name :value>` directly; form submission works without JS |
| **Attribute/property reflection is confusing** — non-string values, casing mismatches, unclear reflection semantics | **Solved** — props are typed and validated in Go; the CE reads server-rendered `data-*` attributes; no client-side reflection required for initial state |
| **TypeScript/tooling gaps** — editors don't know CE types; some tools break on Shadow DOM | **Partial** — Go side is fully typed; `<script customelement>` block is vanilla JS with no TS story yet |
| **React interoperability nightmare** — every JS framework handles CEs differently | **N/A** — htmlc is a Go server-side renderer; no JS framework is involved |
| **No built-in reactivity** — must reinvent or pull in Lit/FAST/etc. | **By design** — islands model: reach for a CE only where client-side behaviour is genuinely needed |

**Key insight from this analysis**: RFC 006's biggest value is not just "nicer web component authoring syntax." It is that server-side rendering resolves most of the hardest web component problems *before the browser ever sees the HTML*. The complaints that remain (TypeScript, CSS design tokens leaking across shadow boundaries) are inherent to the shadow DOM boundary itself — any solution faces them.

#### Sources

- Nolan Lawson, [Web components are okay](https://nolanlawson.com/2024/09/28/web-components-are-okay/) (2024)
- Adam Silver, [The problem with web components](https://adamsilver.io/blog/the-problem-with-web-components/)
- Ryan Carniato, [Web Components Are Not the Future](https://dev.to/ryansolid/web-components-are-not-the-future-48bh) — DEV Community
- [Web Components and SSR - 2024 Edition](https://dev.to/stuffbreaker/web-components-and-ssr-2024-edition-1nel) — DEV Community
- Manuel Matuzovic, [Pros and cons of using Shadow DOM and style encapsulation](https://www.matuzo.at/blog/2023/pros-and-cons-of-shadow-dom/) (2023)
- ICT Institute, [Evaluating the Role of Web Components in 2024](https://ictinstitute.nl/webcomponents-in-2024/)

---

## 2. Goals

1. **100% Vue SFC syntax compatibility**: `<script customelement>` must not collide with any attribute already used by Vue on `<script>` blocks (`lang`, `src`, `generic`, `setup`).
2. **Author interactivity inside the `.vue` file**: a single block (`<script customelement>`) where the author writes client-side JavaScript; `htmlc` emits it verbatim into the page.
3. **Automatic SSR wrapping**: when a component carries `<script customelement>`, `htmlc` automatically wraps the rendered template output in the component's derived tag name (e.g. `<date-picker>`, `<shape-canvas>`).
4. **Error on `<script>` and `<script setup>`**: descriptive compile-time error preventing silent failures.
5. **Deduplication across the render pass**: the same custom element script is referenced at most once per page via the importmap and `ScriptsFS`.
6. **In-memory script FS**: compiled scripts collected into an `fs.FS` accessible on the engine for serving, cache-busting, or embedding.
7. **Zero impact on components without `<script customelement>`**: existing rendering behaviour is identical for all current components.
8. **Declarative Shadow DOM opt-in**: `<script customelement shadowdom>` wraps the rendered template in `<template shadowrootmode="open">` (or `"closed"`) automatically.

---

## 3. Non-Goals

- Vue 3 reactivity, Options API, or Composition API on the client.
- `<script>` or `<script setup>` blocks (intentional compile-time errors).
- SSR hydration or resumability.
- Bundling or tree-shaking.
- Dynamic imports or lazy loading.
- Customised built-in elements (`is="..."` syntax) — Safari does not support them.
- Generating class boilerplate or autoregistering elements — the script is emitted verbatim.

---

## 4. Proposed Design

### 4.1 Block Parsing

Extend `Component` with new fields:

```go
// pseudo-code — not implementation
type Component struct {
    Template            *html.Node
    Script              string   // non-empty → error
    Style               string
    Scoped              bool
    Path                string
    Source              string
    Warnings            []string
    CustomElementScript string   // verbatim body of <script customelement>
    CustomElementTag    string   // derived tag name, set at engine load time
    ShadowDOMMode       string   // "": light DOM; "open" or "closed": DSD
}
```

In `extractSections`, detect `customelement` attribute on `<script>` tags and parse `shadowdom` attribute:

```go
// pseudo-code — not implementation
switch {
case attrs["setup"] != "":
    sections["script:setup"] = rawBody(tokenizer)
case attrPresent(attrs, "customelement"):
    sections["script:customelement"] = rawBody(tokenizer)
    if v, ok := attrs["shadowdom"]; ok {
        sections["script:customelement:shadowdom"] = map[bool]string{true: "open"}[v != "closed"]
        if v == "closed" { sections["script:customelement:shadowdom"] = "closed" }
    }
default:
    sections["script"] = rawBody(tokenizer)
}
```

Compile-time validations in `ParseFile`:
- `<script>` present → error: "use `<script customelement>`"
- `<script setup>` present → error: "use `<script customelement>`"
- `<script customelement src="...">` → error: inline body required
- `CustomElementScript` does not contain `customElements.define` → error

### 4.2 Tag-Name Derivation

Derived deterministically from the component path:

1. Split into directory segments + filename (no extension).
2. Each segment: PascalCase/camelCase → kebab-case.
3. Join all segments with `-`.

| File path | Derived tag name |
|---|---|
| `DatePicker.vue` | ❌ `date-picker` — no hyphen, single segment; compile error |
| `ui/DatePicker.vue` | `ui-date-picker` |
| `widgets/ShapeCanvas.vue` | `widgets-shape-canvas` |
| `admin/Card.vue` | `admin-card` |

**Compile-time error**: derived tag name with no hyphen (e.g. top-level `Counter.vue` → `counter`) is rejected with an actionable message directing the author to move to a subdirectory.

### 4.3 SSR Wrapping

When `CustomElementScript` is non-empty, the renderer wraps the rendered template output:

**Light DOM** (`<script customelement>`):
```html
<widgets-shape-canvas>
  [rendered template HTML]
</widgets-shape-canvas>
```

**Declarative Shadow DOM** (`<script customelement shadowdom>`):
```html
<ui-date-picker>
  <template shadowrootmode="open">
    <style>[scoped styles for this component]</style>
    [rendered template HTML]
  </template>
</ui-date-picker>
```

The author's `<template>` block contains only the inner content. The wrapping is fully automatic.

### 4.4 Script Collection and ScriptsFS

A `CustomElementCollector` (mirroring `StyleCollector`) accumulates custom element entries during a render pass. The engine exposes:

```go
// pseudo-code — not implementation
func (e *Engine) ScriptsFS() fs.FS
func (e *Engine) NewScriptFSServer() http.Handler  // Cache-Control: immutable for hashed files
```

Script files are content-hashed:

```
widgets-shape-canvas.a1b2c3d4.js
ui-date-picker.e5f6a7b8.js
index.js                            // not hashed; imports all custom elements
```

### 4.5 Importmap Injection

For `RenderPage`, htmlc injects before `</head>`:

```html
<script type="importmap">
{"imports":{"widgets-shape-canvas":"/components/widgets-shape-canvas.a1b2c3d4.js","ui-date-picker":"/components/ui-date-picker.e5f6a7b8.js"}}
</script>
<script type="module" src="/components/index.js"></script>
```

For `RenderFragment`, authors use `{{ importMap() }}` explicitly in their fragment template.

An optional `NonceFunc func(context.Context) string` engine option injects CSP nonces on both tags.

---

## 5. Syntax Summary

| Syntax | Meaning |
|---|---|
| `<script customelement>` | Light DOM custom element; template rendered as direct children of the CE tag |
| `<script customelement shadowdom>` | Shadow DOM custom element; template wrapped in `<template shadowrootmode="open">` |
| `<script customelement shadowdom="closed">` | Shadow DOM with closed shadow root |
| JS: `this.querySelector(...)` | For light DOM components |
| JS: `this.shadowRoot.querySelector(...)` | For shadow DOM components (`this.shadowRoot` is pre-attached by DSD) |

---

## 6. Examples

### Example 1: Shape Canvas (light DOM, EventSource)

```
components/
  widgets/
    ShapeCanvas.vue
pages/
  Dashboard.vue
```

`components/widgets/ShapeCanvas.vue`:

```vue
<template>
  <canvas :width="width" :height="height" :data-src="src"></canvas>
</template>

<style scoped>
canvas { border: 1px solid #ccc }
</style>

<script customelement>
class WidgetsShapeCanvas extends HTMLElement {
  #source = null
  #ctx = null

  connectedCallback() {
    const canvas = this.querySelector('canvas')
    this.#ctx = canvas.getContext('2d')
    this.#source = new EventSource(canvas.dataset.src)
    this.#source.onmessage = ({ data }) => this.#draw(JSON.parse(data))
  }

  disconnectedCallback() { this.#source?.close() }

  #draw({ type, color = '#000', x, y, w, h, r }) {
    const ctx = this.#ctx
    ctx.fillStyle = color
    if (type === 'rect')   { ctx.fillRect(x, y, w, h) }
    if (type === 'circle') { ctx.beginPath(); ctx.arc(x, y, r, 0, 2*Math.PI); ctx.fill() }
    if (type === 'clear')  { ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height) }
  }
}
customElements.define('widgets-shape-canvas', WidgetsShapeCanvas)
</script>
```

Usage:

```html
<widgets-shape-canvas src="/api/shapes/stream" :width="800" :height="600"></widgets-shape-canvas>
```

Server emits:

```html
<widgets-shape-canvas>
  <canvas width="800" height="600"
          data-src="/api/shapes/stream"
          data-v-a1b2c3d4></canvas>
</widgets-shape-canvas>
<style>canvas[data-v-a1b2c3d4] { border: 1px solid #ccc }</style>
<script type="importmap">{"imports":{"widgets-shape-canvas":"/components/widgets-shape-canvas.a1b2c3d4.js"}}</script>
<script type="module" src="/components/index.js"></script>
```

**What the server controls:** canvas dimensions, stream URL — computed in Go, vary per user. Adding a second canvas on the same page just works; the importmap deduplicates the script reference.

---

### Example 2: Date Picker (shadow DOM, form integration)

```
components/
  ui/
    DatePicker.vue
```

`components/ui/DatePicker.vue`:

```vue
<template>
  <!-- Visible display: server-renders the value immediately, no flash -->
  <span class="display">{{ value || placeholder }}</span>
  <!-- Hidden field: form submission works without JS -->
  <input type="hidden" :name="name" :value="value">
</template>

<style scoped>
:host {
  display: inline-block;
  position: relative;
}
.display {
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  min-width: 140px;
  display: inline-block;
}
.display:hover { border-color: #9ca3af }
.picker {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0,0,0,.15);
  padding: 8px;
}
input[type=date] { border: none; outline: none; font-size: 14px }
</style>

<script customelement shadowdom>
class UiDatePicker extends HTMLElement {
  #hidden  = null
  #display = null

  connectedCallback() {
    const root = this.shadowRoot
    this.#hidden  = root.querySelector('input[type=hidden]')
    this.#display = root.querySelector('.display')

    this.#display.addEventListener('click', () => this.#openPicker())
    document.addEventListener('click', (e) => {
      if (!this.contains(e.target)) this.#closePicker()
    }, { capture: true })
  }

  #openPicker() {
    const root = this.shadowRoot
    if (root.querySelector('.picker')) return

    const picker = document.createElement('div')
    picker.className = 'picker'

    const input = document.createElement('input')
    input.type = 'date'
    input.value = this.#hidden.value

    input.addEventListener('change', () => {
      this.#hidden.value = input.value
      this.#display.textContent = input.value
      this.#closePicker()
      this.dispatchEvent(new CustomEvent('change', {
        detail: input.value, bubbles: true, composed: true
      }))
    })

    picker.append(input)
    root.append(picker)
    input.showPicker?.()
  }

  #closePicker() {
    this.shadowRoot.querySelector('.picker')?.remove()
  }
}
customElements.define('ui-date-picker', UiDatePicker)
</script>
```

Usage:

```html
<ui-date-picker name="start_date" :value="startDate" placeholder="Pick a date"></ui-date-picker>
```

Server emits:

```html
<ui-date-picker>
  <template shadowrootmode="open">
    <style>
      :host { display: inline-block; position: relative }
      .display[data-v-b2c3d4e5] { padding: 6px 12px; border: 1px solid #d1d5db; ... }
      .picker[data-v-b2c3d4e5] { position: absolute; top: calc(100% + 4px); ... }
    </style>
    <span class="display" data-v-b2c3d4e5>2026-03-15</span>
    <input type="hidden" name="start_date" value="2026-03-15" data-v-b2c3d4e5>
  </template>
</ui-date-picker>
```

**Key properties of the shadow DOM variant:**
- `.picker` popup `z-index: 100` is isolated inside the shadow root — no page-level stacking context conflict
- `:host` styles the custom element itself from inside — impossible with light DOM scoped styles
- `composed: true` on the `change` event crosses the shadow boundary so parent forms can listen
- `this.shadowRoot` is already attached by the browser (from DSD) before `connectedCallback` fires — no `attachShadow()` needed
- Form submission works without JS: the hidden input is plain server-rendered HTML

---

## 7. Implementation Sketch

Changes required in core htmlc (all in the same module):

**`component.go`**
- Extend `extractSections` to read `customelement` and `shadowdom` attributes on `<script>` tags
- Extend `Component` struct with `CustomElementScript`, `CustomElementTag`, `ShadowDOMMode`
- Add compile-time validations in `ParseFile`

**`engine.go`**
- Derive `CustomElementTag` from component path at load time; error if no hyphen
- Build `ScriptsFS` (in-memory `fs.FS`) from all loaded custom element scripts, content-hashed
- Add `ScriptsFS() fs.FS` and `NewScriptFSServer() http.Handler` methods
- Add `WithNonceFunc(func(context.Context) string) *Engine` option
- Inject importmap + loader script before `</head>` in `RenderPage`

**`renderer.go`**
- After rendering a component with non-empty `CustomElementScript`: wrap output in CE tag
- For shadow DOM mode: wrap rendered HTML + scoped styles in `<template shadowrootmode>`
- Register `importMap() template.HTML` as a template function for use in fragment templates
- Add `CustomElementCollector` (mirroring `StyleCollector`) for deduplication

**`cmd/htmlc/`**
- `build` subcommand: write `ScriptsFS` contents to output directory alongside rendered HTML

---

## 8. Backward Compatibility

- All components without `<script customelement>` are unaffected.
- `Engine` API: new methods (`ScriptsFS`, `NewScriptFSServer`, `WithNonceFunc`) are additive.
- `Component` struct: new fields (`CustomElementScript`, `CustomElementTag`, `ShadowDOMMode`) are zero-valued for all existing components.
- No changes to `RenderFragment` behavior for components without custom element scripts.
- The importmap injection in `RenderPage` is new behavior, but only activates when at least one custom element is present in the rendered output.

---

## 9. Alternatives Considered

**Build RFC 006 as a separate package** — analyzed at length. The engine currently has 4 tight coupling points (component parsing, render wrapping, script collection, page finalization) that would need new extension interfaces before a separate package could implement RFC 006. The changes to open those seams (~100–200 lines) are modest but require deliberate design. Given the goal of an out-of-the-box coherent experience, keeping the implementation in core is preferred. A future refactor to separate it remains possible.

**Require a full JS bundler** — adds significant operational complexity for what is intentionally a minimal, no-Node-required tool. ES module `import` statements in the inline script body can reference external utilities without a bundler; the browser's native module system handles it.

**Support `<script setup>` for composable behavior** — Vue Composition API semantics would require shipping a reactivity runtime. Out of scope by design.

---

## 10. Open Questions

1. **`shadowdom="closed"` support** — Should closed shadow roots be supported in v1? Closed shadow roots prevent external JS (`element.shadowRoot === null`) and complicate devtools inspection. Tentative recommendation: support it (it's a one-liner change to the DSD `shadowrootmode` attribute), but document that `open` is the recommended default. *(Non-blocking)*

2. **Acronym casing in tag derivation** — `XMLParser.vue` → `x-m-l-parser` is ugly. Recommend documenting that files should use title-case acronyms (`XmlParser.vue` → `xml-parser`). A future heuristic could handle common acronyms. *(Non-blocking)*

3. **`RenderFragmentWithElements` convenience method** — Deferred from v1. Authors use `{{ importMap() }}` explicitly in fragment templates. *(Non-blocking)*

4. **TypeScript declarations for custom elements** — The `<script customelement>` block is vanilla JS. A future `htmlc lsp` or codegen step could emit `.d.ts` declarations from prop definitions. *(Non-blocking)*
