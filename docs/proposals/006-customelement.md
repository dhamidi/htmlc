# RFC 006: Custom Element Compilation

- **Status**: Draft
- **Date**: 2026-03-16
- **Author**: TBD

---

## 1. Motivation

`htmlc` today is a **server-side-only renderer**: it processes `.vue` files into static HTML strings and has no mechanism for delivering client-side interactivity. When a page needs an interactive island — a date-picker, an accordion, a live counter, a tabbed panel — the author must either reach for a full JavaScript framework (Vue, React, Svelte) bundled and delivered separately, or hand-write Custom Element boilerplate that duplicates the component boundary already declared in the `.vue` file.

### The problem in practice

Consider a simple counter component:

```
components/
  Counter.vue      ← has a <template> for SSR, but needs click interactivity
pages/
  Home.vue         ← embeds <Counter>
```

Today, `Counter.vue` can render static HTML perfectly. But there is no `htmlc`-native path to attach a click handler that increments a displayed value. The author must write a separate `.js` file, figure out how to register a custom element, replicate the element's markup in JS string concatenation, and manually wire the two together. If they refactor `Counter.vue` they must remember to update the JS file separately. The component boundary provides no protection.

### Why a full client-side framework is not the answer

Adding full Vue 3 client-side compilation (reactivity, the Composition API, VDOM) would require shipping and initialising a large runtime on every page. `htmlc` is designed for server-rendered Go applications where pages are complete HTML documents — the framework's job ends at the server. What is missing is only a **thin, standards-based bridge** between the `.vue` component boundary and the browser's own component model: the Custom Elements API.

Custom Elements (part of the Web Components standard) let any DOM element register a class that the browser instantiates when the element appears in the document. The author writes a plain class body. No runtime is needed. No bundler is needed. This is exactly the right primitive for `htmlc`'s model.

---

## 2. Goals

1. **100% Vue SFC syntax compatibility**: `<script customelement>` must not collide with any attribute already used by Vue on `<script>` blocks (`lang`, `src`, `generic`, `setup`).
2. **Author interactivity inside the `.vue` file**: provide a single block (`<script customelement>`) where the author writes the Custom Element class body (`connectedCallback`, `attributeChangedCallback`, `observedAttributes`, etc.); `htmlc` wraps it in the required boilerplate.
3. **Error on `<script>` and `<script setup>`**: emit a descriptive compile-time error when these blocks appear, preventing silent failures for authors who accidentally write standard Vue script blocks.
4. **Import Map integration**: when a page includes at least one custom element, `htmlc` can optionally emit a `<script type="importmap">` mapping tag names to their script source.
5. **Deduplication across the render pass**: the same custom element definition is emitted at most once per page, even if the component is used multiple times.
6. **Zero impact on components without `<script customelement>`**: server-side rendering behaviour is identical to today for all existing components.

---

## 3. Non-Goals

- **Implementing Vue 3 reactivity, the Options API, or the Composition API on the client.** The emitted JS is a plain class body; no reactivity primitives are provided or planned.
- **Supporting `<script>` or `<script setup>` blocks.** These are intentional compile-time errors. Implementing them would require re-implementing Vue's compiler and runtime on the client side.
- **SSR hydration or resumability.** The server renders static HTML; the Custom Element enhances it after the fact. There is no serialised component state passed from server to client.
- **Bundling or tree-shaking.** Each custom element script is independent. There is no module bundler integration.
- **Dynamic imports or lazy loading.** All emitted scripts are eager.
- **Customised built-in elements (`is="..."` syntax).** Safari does not support them; autonomous custom elements are the only viable cross-browser target.

---

## 4. Proposed Design

### 4.1 Block Parsing

#### Current state

`extractSections` in `component.go` tokenises the top level of a `.vue` file with `golang.org/x/net/html`. It recognises three tag names (`"template"`, `"script"`, `"style"`). For `<style>`, it reads the `scoped` attribute and records it in a side map. For `<script>`, it reads **no attributes** — it merely stores the raw text content in `sections["script"]`. A second `<script>` block returns `"duplicate <script> section"` immediately.

The `Component` struct currently carries:

```go
// current — component.go
type Component struct {
    Template *html.Node
    Script   string   // raw <script> body (currently always rejected at higher level)
    Style    string
    Scoped   bool
    Path     string
    Source   string
    Warnings []string
}
```

#### Proposed extension

Extend `Component` with a new field **`CustomElementScript`** that stores the raw body of a `<script customelement>` block:

```go
// pseudo-code — not implementation
type Component struct {
    Template            *html.Node
    Script              string   // non-empty → error: not supported
    Style               string
    Scoped              bool
    Path                string
    Source              string
    Warnings            []string
    CustomElementScript string   // new: body of <script customelement>, empty if absent
    CustomElementTag    string   // new: explicit tag name override, empty = derive from filename
}
```

In `extractSections`, when the tokeniser encounters a `<script>` start tag, read all its attributes before consuming the body:

```go
// pseudo-code — not implementation
attrs := attrsMap(token)   // map[string]string of all attributes on the tag

switch {
case attrs["setup"] != "":
    // existing: record as "script:setup"
    sections["script:setup"] = rawBody(tokenizer)
case attrs["customelement"] != "":
    // new: record as custom element body
    tagOverride := attrs["tag"]   // optional explicit tag name
    sections["script:customelement"] = rawBody(tokenizer)
    sections["script:customelement:tag"] = tagOverride
default:
    // existing: plain <script> — stored; will be rejected later
    sections["script"] = rawBody(tokenizer)
}
```

`ParseFile` then populates the struct:

```go
// pseudo-code — not implementation
comp.CustomElementScript = sections["script:customelement"]
comp.CustomElementTag    = sections["script:customelement:tag"]

// Keep existing error for plain <script>:
if sections["script"] != "" {
    return nil, fmt.Errorf(
        "%s: <script> blocks are not supported by htmlc; " +
        "use <script customelement> to define a Custom Element", path)
}
// New error for <script setup>:
if sections["script:setup"] != "" {
    return nil, fmt.Errorf(
        "%s: <script setup> blocks are not supported by htmlc; " +
        "use <script customelement> to define a Custom Element", path)
}
```

**Evaluation**

- ✅ Reads all script-block variants from a single tokeniser pass — no second parse.
- ✅ The `customelement` attribute is confirmed absent from Vue's SFC spec (`lang`, `src`, `generic`, `setup` are the only recognised attributes on `<script>`). No collision.
- ✅ `CustomElementTag` supports the `tag="my-name"` override without a separate block.
- ⚠️ `sections` map grows two new keys; ensure the "duplicate section" guard covers all key combinations.

**Verdict**: extend attribute reading in `extractSections` to detect `customelement`; store body and optional `tag` override as new `Component` fields.

---

### 4.2 Tag-Name and Class-Name Derivation

When `CustomElementTag` is empty, `htmlc` derives both the HTML custom-element tag name and the JS class name from the component's file name.

**Algorithm** (applied to the base name without extension, e.g. `DatePicker` from `DatePicker.vue`):

1. **Class name**: prefix with `Htmlc` → `HtmlcDatePicker`.
2. **Tag name**: insert hyphens at PascalCase word boundaries, lowercase all, prefix with `htmlc-` → `htmlc-date-picker`.

Examples:

| File name    | Class name         | Tag name              |
|--------------|--------------------|-----------------------|
| `Counter.vue`     | `HtmlcCounter`     | `htmlc-counter`       |
| `DatePicker.vue`  | `HtmlcDatePicker`  | `htmlc-date-picker`   |
| `LiveChart.vue`   | `HtmlcLiveChart`   | `htmlc-live-chart`    |
| `Button.vue`      | `HtmlcButton`      | `htmlc-button`        |
| `MyXYZWidget.vue` | `HtmlcMyXYZWidget` | `htmlc-my-x-y-z-widget` |

When `CustomElementTag` is non-empty, validate that it contains at least one hyphen and starts with a lowercase ASCII letter (per the Custom Elements specification). The class name is derived from the provided tag name: split on `-`, title-case each segment, prefix with `Htmlc`.

**Collision considerations**: the `htmlc-` prefix ensures generated tag names do not collide with standard HTML elements (which contain no hyphen or whose names are reserved). Two distinct components whose file names produce the same derived tag name — e.g. `DatePicker.vue` and `Date-Picker.vue` — will produce a duplicate registration error at emit time, which is a deterministic, loud failure.

**Evaluation**

- ✅ Deterministic: same file always produces the same names.
- ✅ `htmlc-` prefix avoids clashes with plain HTML or third-party elements.
- ⚠️ Acronym sequences (e.g. `XMLParser`) produce `htmlc-x-m-l-parser`; authors can override with `tag="htmlc-xml-parser"`.
- ⚠️ Deeply nested files with the same base name (e.g. `blog/Counter.vue` and `admin/Counter.vue`) produce the same tag name; the engine should detect this and error.

**Verdict**: derive from file name with `htmlc-` prefix; provide `tag="..."` override attribute for edge cases.

---

### 4.3 JS Emission — Wrapping the Author's Code

When `Component.CustomElementScript` is non-empty, `htmlc` wraps the author's code in a minimal `class extends HTMLElement { … }` shell and appends a `customElements.define(…)` call.

#### Emission pseudocode

```go
// pseudo-code — not implementation
func emitCustomElement(tagName, className, scriptBody string) template.HTML {
    return template.HTML(fmt.Sprintf(`<script>
class %s extends HTMLElement {
%s
}
customElements.define(%q, %s);
</script>`, className, scriptBody, tagName, className))
}
```

The author's `scriptBody` is the verbatim content of the `<script customelement>` block — it forms the body of the class. The author writes lifecycle callbacks and any helper methods directly:

```html
<!-- Counter.vue -->
<script customelement>
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
</script>
```

This produces:

```html
<script>
class HtmlcCounter extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
}
customElements.define("htmlc-counter", HtmlcCounter);
</script>
```

**Evaluation**

- ✅ No runtime dependency: the emitted script is self-contained.
- ✅ Verbatim body insertion means the author can use any JS syntax supported by their target browsers.
- ⚠️ No automatic `observedAttributes` injection — the author must declare the static getter manually if needed. (This keeps the emitter trivially simple; a future RFC can add automatic derivation from `Component.Props()`.)
- ⚠️ The body is inserted verbatim, so `htmlc` cannot validate JS syntax. Syntax errors surface in the browser console, not at build time. A future lint step could run `esbuild --bundle=false` on the body.

**Verdict**: verbatim body insertion wrapped in a minimal class shell. No automatic attribute wiring in v1; defer to a future RFC.

---

### 4.4 Deduplication

The same component may be rendered many times on a single page (e.g. a `<Counter>` inside a list of 50 items). The `customElements.define` call must appear **exactly once** per tag name per page; calling it twice with the same tag name throws a `DOMException`.

#### Current state

`StyleCollector` (in `style.go`) already implements deduplication for CSS: it accumulates `StyleContribution` values keyed by `ScopeID+"\x00"+CSS`, and each unique contribution is flushed once via `styleBlock(sc)`.

#### Proposed extension

Introduce a parallel **`CustomElementCollector`** type:

```go
// pseudo-code — not implementation
type CustomElementEntry struct {
    TagName   string
    ClassName string
    Script    string // verbatim body
}

type CustomElementCollector struct {
    seen    map[string]struct{}   // keyed by tag name
    entries []CustomElementEntry
}

func (c *CustomElementCollector) Add(e CustomElementEntry) {
    if _, ok := c.seen[e.TagName]; ok {
        return   // already registered for this render pass
    }
    c.seen[e.TagName] = struct{}{}
    c.entries = append(c.entries, e)
}

func (c *CustomElementCollector) FlushCustomElements() template.HTML {
    var b strings.Builder
    for _, e := range c.entries {
        b.WriteString(string(emitCustomElement(e.TagName, e.ClassName, e.Script)))
    }
    return template.HTML(b.String())
}
```

The `Renderer` receives a `*CustomElementCollector` (analogous to `*StyleCollector`). Whenever `renderElement` processes a component tag whose `Component.CustomElementScript` is non-empty, it calls `collector.Add(...)`. The collector is allocated per render pass (per `RenderPage` / `RenderFragment` call) and is not shared across concurrent requests.

`FlushCustomElements()` is called by the page author (or by `RenderPage` automatically before `</body>`) to emit the deduplicated script tags.

**Evaluation**

- ✅ Exact mirror of the existing style deduplication pattern — low conceptual overhead.
- ✅ Per-render-pass allocation means no cross-request state leakage.
- ⚠️ `FlushCustomElements` placement (end of `<body>` vs. `<head>`) affects when the Custom Element class is available during parsing. Deferring to end-of-`<body>` is the safest default because the element's HTML is already in the DOM when the script executes.
- ⚠️ `RenderFragment` users who do not call `FlushCustomElements` get orphaned elements in the DOM that are never upgraded. This should be documented prominently.

**Verdict**: introduce `CustomElementCollector` mirroring `StyleCollector`; `FlushCustomElements()` returns a `template.HTML` string of deduplicated `<script>` tags.

---

### 4.5 Import Map Integration

An Import Map (`<script type="importmap">`) tells the browser how to resolve bare module specifiers. While the custom element scripts emitted in §4.3 do not themselves use bare imports, the Import Map can be used to map the tag name's "module identity" to an external URL — enabling CDN delivery, versioning, and cache busting.

Two sub-options:

#### Option A — Inline (recommended for v1)

The custom element JS is embedded directly in a `<script>` tag in the page. No URL is needed. No Import Map entry is required for the script itself, though one may still be written for documentation or future tooling.

- ✅ No file I/O at render time.
- ✅ No CDN configuration required.
- ✅ Works with strict CSP if a nonce is propagated (see §10).
- ⚠️ Script is not independently cacheable by the browser.
- ⚠️ Repeated across pages (though a cache on the HTML document itself mitigates this).

#### Option B — URL-based

`htmlc` writes the emitted JS to a static file under a configurable `AssetsDir`, and the Import Map maps `"htmlc-counter"` → `"/assets/htmlc-counter.HASH.js"`.

- ✅ Script is cacheable independently.
- ✅ CDN-deliverable.
- ❌ Requires file I/O during startup or first render — complicates the render pipeline.
- ❌ Requires a mechanism to serve the generated assets (static file server, CDN upload).
- ⚠️ Hash must be derived from script content to enable cache busting.

**Verdict**: implement Option A (inline) in v1. Design `FlushCustomElements()` to accept an optional nonce string for CSP compatibility. Defer Option B to a follow-up RFC.

#### Import Map emission

When using Option A, `htmlc` can optionally emit a `<script type="importmap">` that maps each tag name to `"inline"` (a sentinel) or simply omit the import map. The import map becomes meaningful only when Option B is implemented. For v1, no import map is emitted automatically.

---

### 4.6 Interaction with `<style scoped>` and `<style>`

A component that declares `<script customelement>` may also declare a `<style>` block. Two sub-cases:

#### Case 1: Component does not use shadow DOM

The component's `<style>` (scoped or global) is handled exactly as today by `StyleCollector`. The custom element tag in the DOM inherits the scoped attribute (e.g., `data-v-a1b2c3d4`) stamped on it by the renderer. No change required.

#### Case 2: Component uses shadow DOM (author calls `this.attachShadow(...)` in `connectedCallback`)

Styles in the document `<head>` do not pierce the shadow boundary. The author must include the CSS inside the shadow root. Two approaches:

1. **Author-managed**: the author manually injects a `<style>` element inside `connectedCallback`. The `<style>` block in the `.vue` file continues to contribute to the page's `<head>` (for SSR appearance). This is the v1 behaviour — no automatic shadow-DOM style injection.
2. **Automatic injection** (future): if `<script customelement shadowdom>` is detected, `htmlc` serialises the scoped CSS and injects it into the emitted class body as a template literal, e.g.:

```js
// pseudo-code — not implementation
connectedCallback() {
    const shadow = this.attachShadow({ mode: "open" });
    const style = document.createElement("style");
    style.textContent = `/* scoped CSS here */`;
    shadow.appendChild(style);
    // ... author body follows
}
```

**Verdict**: v1 leaves shadow-DOM style injection to the author. The `shadowdom` attribute on `<script customelement>` is reserved for a future RFC.

---

### 4.7 Error Behaviour for `<script>` and `<script setup>`

`ParseFile` currently accepts a `<script>` block and stores it in `Component.Script`, but no rendering path uses it — it is silently ignored. This is a latent confusion vector: an author familiar with Vue may write `<script setup>` expecting reactivity, see no error, and waste debugging time.

Under this RFC:

- **`<script>` (plain)**: `ParseFile` returns an error:
  ```
  path/to/Component.vue: <script> blocks are not supported by htmlc;
  use <script customelement> to define a Custom Element
  ```
- **`<script setup>`**: `ParseFile` returns an error:
  ```
  path/to/Component.vue: <script setup> blocks are not supported by htmlc;
  use <script customelement> to define a Custom Element
  ```

Both errors are emitted at component-load time (engine startup or first `ParseFile` call), not at render time.

**Verdict**: promote the current silent ignore to a loud compile-time error with an actionable message.

---

## 5. Syntax Summary

| Block | Attribute | Meaning in `htmlc` |
|---|---|---|
| `<script customelement>` | *(none)* | Custom Element body; compiled to `class HtmlcName extends HTMLElement { … }` with tag derived from filename |
| `<script customelement tag="my-tag">` | `tag` | Same as above, but uses the provided tag name instead of the filename-derived one |
| `<script customelement shadowdom>` | `shadowdom` | **Reserved** — shadow DOM opt-in; deferred to a future RFC |
| `<script>` | *(any)* | **Error**: `<script> blocks are not supported by htmlc` |
| `<script setup>` | `setup` | **Error**: `<script setup> blocks are not supported by htmlc` |
| `<style>` | *(none)* | Global stylesheet contribution; unchanged from today |
| `<style scoped>` | `scoped` | Scoped stylesheet contribution; unchanged from today |
| `<template>` | *(none)* | Server-side render template; unchanged from today |

---

## 6. Examples

### Example 1 — Minimal Interactive Counter

A standalone counter with no server-side template (pure custom element).

**Directory tree**

```
components/
  Counter.vue
pages/
  Home.vue
```

**`Counter.vue`**

```html
<template>
  <htmlc-counter>
    <button>Click me</button>
    <span>0</span>
  </htmlc-counter>
</template>

<script customelement>
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
</script>
```

**`Home.vue`**

```html
<template>
  <html>
    <head><title>Home</title></head>
    <body>
      <Counter />
      {{FlushCustomElements}}
    </body>
  </html>
</template>
```

**Rendered output (simplified)**

```html
<html>
  <head><title>Home</title></head>
  <body>
    <htmlc-counter>
      <button>Click me</button>
      <span>0</span>
    </htmlc-counter>
    <script>
class HtmlcCounter extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
}
customElements.define("htmlc-counter", HtmlcCounter);
</script>
  </body>
</html>
```

---

### Example 2 — Progressive Enhancement (Template + Custom Element)

A `<Tabs>` component that renders all tab panels statically for SEO and no-JS users, and then uses a `<script customelement>` to add client-side tab switching.

**`Tabs.vue`**

```html
<template>
  <htmlc-tabs>
    <nav class="tab-bar">
      <button data-tab="0">Overview</button>
      <button data-tab="1">Details</button>
      <button data-tab="2">Reviews</button>
    </nav>
    <div class="tab-panel" data-panel="0"><slot name="overview" /></div>
    <div class="tab-panel" data-panel="1"><slot name="details" /></div>
    <div class="tab-panel" data-panel="2"><slot name="reviews" /></div>
  </htmlc-tabs>
</template>

<style scoped>
.tab-panel { display: block; }
</style>

<script customelement>
  connectedCallback() {
    this.panels = Array.from(this.querySelectorAll('.tab-panel'));
    this.buttons = Array.from(this.querySelectorAll('[data-tab]'));
    this._activate(0);
    this.querySelectorAll('[data-tab]').forEach(btn => {
      btn.addEventListener('click', () => this._activate(Number(btn.dataset.tab)));
    });
  }

  _activate(index) {
    this.panels.forEach((p, i) => { p.hidden = i !== index; });
    this.buttons.forEach((b, i) => { b.setAttribute('aria-selected', i === index); });
  }
</script>
```

**Behaviour**

- Without JS: all three panels are visible; users see all content (SEO-friendly, accessible).
- With JS: the Custom Element's `connectedCallback` hides panels 1 and 2 and activates tab-bar button events.
- The `<style scoped>` contribution is flushed into `<head>` by the existing `StyleCollector` path.
- The custom element script is flushed by `FlushCustomElements()` before `</body>`.

---

### Example 3 — Multiple Custom Elements on One Page (Deduplication)

A dashboard page that uses `<Counter>` three times and `<Toggle>` once.

**`Toggle.vue`** (abbreviated)

```html
<template>
  <htmlc-toggle>
    <input type="checkbox" /><label><slot /></label>
  </htmlc-toggle>
</template>

<script customelement>
  connectedCallback() {
    this.input = this.querySelector('input');
    this.input.addEventListener('change', () => {
      this.dispatchEvent(new CustomEvent('toggle', { detail: this.input.checked }));
    });
  }
</script>
```

**`Dashboard.vue`** (abbreviated)

```html
<template>
  <html>
    <head><title>Dashboard</title></head>
    <body>
      <h1>Dashboard</h1>
      <Counter /> <!-- rendered three times via v-for equivalent -->
      <Counter />
      <Counter />
      <Toggle>Dark mode</Toggle>
      {{FlushCustomElements}}
    </body>
  </html>
</template>
```

**Rendered output (script section only)**

```html
<!-- Only ONE definition per element, regardless of how many times used -->
<script>
class HtmlcCounter extends HTMLElement {
  connectedCallback() { /* ... */ }
}
customElements.define("htmlc-counter", HtmlcCounter);
</script>
<script>
class HtmlcToggle extends HTMLElement {
  connectedCallback() { /* ... */ }
}
customElements.define("htmlc-toggle", HtmlcToggle);
</script>
```

The `CustomElementCollector` tracks `{ "htmlc-counter", "htmlc-toggle" }` as its seen set. The three `<Counter>` renders each call `collector.Add(...)`, but only the first produces an entry. `FlushCustomElements()` emits exactly two `<script>` blocks.

---

### Example 4 — Backward Compatibility (No `<script customelement>`)

An existing project with no interactive components.

**`Card.vue`**

```html
<template>
  <div class="card">
    <h2>{{ title }}</h2>
    <p>{{ body }}</p>
  </div>
</template>

<style scoped>
.card { border: 1px solid #ccc; padding: 1rem; }
</style>
```

- `Component.CustomElementScript` is `""`.
- `Component.CustomElementTag` is `""`.
- `CustomElementCollector.Add` is never called.
- `FlushCustomElements()` returns `template.HTML("")`.
- Output is identical to today.

---

## 7. Implementation Sketch

### `component.go`

1. Add two new fields to `Component`: `CustomElementScript string` and `CustomElementTag string`.
2. In `extractSections`, after reading a `<script>` start tag, collect all its attributes into a `map[string]string`.
3. If `attrs["customelement"] != ""` or the attribute name `"customelement"` is present (boolean attribute), store the body in `sections["script:customelement"]` and `attrs["tag"]` in `sections["script:customelement:tag"]`.
4. In `ParseFile`, populate the two new fields from `sections`.
5. Convert the current silent-ignore of `sections["script"]` and `sections["script:setup"]` into explicit error returns with the messages defined in §4.7.

### `style.go` (or new `customelement.go`)

1. Define `CustomElementEntry` struct with `TagName`, `ClassName`, `Script string` fields.
2. Define `CustomElementCollector` struct with `seen map[string]struct{}` and `entries []CustomElementEntry`.
3. Implement `(c *CustomElementCollector) Add(e CustomElementEntry)` — no-op if `TagName` already in `seen`.
4. Implement `(c *CustomElementCollector) FlushCustomElements() template.HTML` — iterates `entries`, calls `emitCustomElement` for each, concatenates results.
5. Add standalone `emitCustomElement(tagName, className, scriptBody string) template.HTML` helper.
6. Add `DeriveTagName(filename string) (tagName, className string)` helper implementing the algorithm in §4.2.

### `engine.go`

1. In `renderComponent` (and its callers), allocate a `*CustomElementCollector` per render pass alongside the existing `*StyleCollector`.
2. Pass the collector into the `Renderer` via a new `WithCustomElements(cc *CustomElementCollector)` builder method (mirroring `WithStyles`).
3. In `RenderPage`, after injecting the style block, call `collector.FlushCustomElements()` and inject the result before `</body>`. Alternatively, expose `FlushCustomElements` as a template function so page authors control placement.
4. Add `FlushCustomElements() template.HTML` as a public method on `Engine` (delegating to the per-pass collector) for use in `CompileToTemplate` / `TemplateText` workflows.

### `renderer.go`

1. Add `customElementCollector *CustomElementCollector` field to `Renderer`.
2. In `renderElement`, when resolving a component tag: if the resolved `Component.CustomElementScript != ""`, call `customElementCollector.Add(CustomElementEntry{...})`.
3. Propagate the collector into child `Renderer` instances (same pattern as `styleCollector`).

### Platform notes

- All file-name manipulation uses `path/filepath` for OS portability.
- The `DeriveTagName` function should use `unicode` package functions for PascalCase splitting, not byte-level comparisons, to handle future non-ASCII names gracefully.

---

## 8. Backward Compatibility

### `Component` struct

A new **exported** field `CustomElementScript string` and `CustomElementTag string` are added. This is a backward-compatible addition in Go: existing code that constructs `Component` by field name is unaffected; code that uses positional struct literals (unusual and discouraged) would break at compile time — acceptable given that `Component` is an internal type not intended for direct construction by library consumers.

### `ParseFile` and `ParseDir`

For components without `<script customelement>`, both functions return the same results as today. The only observable behavioural change is that components with a plain `<script>` or `<script setup>` block — which previously silently stored the body in `Component.Script` (a field unused by any render path) — now return an error. This is a **breaking change** for any project that uses such blocks, but since those blocks had no effect on rendering, the only affected case is a misconfigured component that was silently broken. The error message is actionable.

### `RenderPage` / `RenderFragment`

No change for components without `<script customelement>`. For components that do use it, `RenderPage` gains automatic `FlushCustomElements()` injection before `</body>`. `RenderFragment` does not auto-flush (it has no `<body>` tag); callers must invoke `FlushCustomElements()` explicitly.

### `Engine` public API

- New method `FlushCustomElements() template.HTML` — additive, no break.
- No existing methods are removed or have their signatures changed.

### `StyleCollector`

Unchanged. The new `CustomElementCollector` is a parallel type, not a modification.

---

## 9. Alternatives Considered

### A. Top-level `<customelement>` custom block

Vue's SFC spec allows arbitrary custom blocks (e.g. `<docs>`, `<i18n>`). A `<customelement>` block would be syntactically valid in Vue (treated as a no-op custom block) and clearly distinct from `<script>`.

**Rejected** because: Vue's custom block body is not parsed as JavaScript by IDEs or linters — it is treated as opaque text. Authors would lose syntax highlighting, `eslint`, and IDE completions for their element body. Using `<script customelement>` keeps the block recognised as a `<script>` by tooling, which applies JS parsing rules to the body.

### B. A separate `.ce.js` file alongside the `.vue` file

`Counter.vue` + `Counter.ce.js` → `htmlc` combines them automatically.

**Rejected** because: the whole motivation is to keep the component boundary in one file. A companion file reintroduces the synchronisation problem described in §1. It also requires a new file-watching and association mechanism.

### C. Full Vue 3 client-side compilation

Compile the `<script setup>` block (Composition API) to a client-side Vue component, ship the Vue runtime, and mount it on the element.

**Rejected** because: it requires shipping and initialising the Vue runtime (≈50 KB min+gzip), reimplementing a subset of the Vue compiler, and maintaining compatibility with Vue version upgrades. This is out of scope for a server-side rendering engine and contradicts the "no runtime" design principle.

### D. Deriving `observedAttributes` automatically from `Component.Props()`

The existing `Props()` method already extracts all template-referenced variable names. These could be automatically wired as `observedAttributes` and each one's `attributeChangedCallback` could update a corresponding property on `this`.

**Deferred** (not rejected): automatic wiring is appealing but requires deciding on a naming convention (camelCase property names vs. kebab-case attribute names), the semantics of type coercion (all attributes are strings), and whether the author can opt out. Deferring keeps v1 minimal and correct; a follow-up RFC can address reactive attribute binding.

---

## 10. Open Questions

1. **Shadow DOM opt-in attribute** — Should shadow DOM be opt-in via `<script customelement shadowdom>`? If so, should `open` or `closed` mode be the default, and should it be configurable (`shadowdom="closed"`)? *Tentative recommendation*: yes, add `shadowdom` as a boolean attribute defaulting to open mode. **Blocking** before the `shadowdom` sub-feature ships; non-blocking for v1 (which defers shadow DOM).

2. **Tag name override attribute** — The proposed `tag="my-counter"` attribute on `<script customelement>` provides override capability. Should validation enforce that the provided name begins with `htmlc-` (to prevent accidental shadowing of third-party elements), or allow any valid custom element name? *Tentative recommendation*: allow any valid name (contains hyphen, starts with lowercase letter) — the `htmlc-` prefix is a convention, not a requirement. **Non-blocking**.

3. **`FlushCustomElements` placement** — Should this be a method on `Engine`, on `Renderer`, on the new `CustomElementCollector`, or exposed as a Go template function? *Tentative recommendation*: expose as a template function (like `styleBlock`) for `RenderPage`, and as a method on `Engine` for programmatic callers. **Blocking** — the public API surface must be decided before implementation.

4. **Nonce support for inline scripts** — CSP `script-src` policies require a nonce on inline `<script>` tags. Should `FlushCustomElements` accept a nonce string, or should nonce injection be handled by the caller post-hoc (e.g., via `strings.Replace`)? *Tentative recommendation*: add an optional `nonce string` parameter to `FlushCustomElements` and thread it through `emitCustomElement`. **Non-blocking** — can be added without API break if the method accepts variadic options.

5. **Duplicate tag name across namespaced components** — If `blog/Counter.vue` and `admin/Counter.vue` both define `<script customelement>`, they derive the same tag name `htmlc-counter`. Should `htmlc` error at startup (if both are loaded), or at emit time (if both are rendered in the same page), or allow the tag attribute to disambiguate (`tag="htmlc-blog-counter"`)? *Tentative recommendation*: error at startup — load-time detection is far less confusing than a runtime browser error. **Blocking**.

6. **`RenderFragment` and orphaned elements** — If an author calls `RenderFragment` to render a snippet that contains a custom element, and does not call `FlushCustomElements`, the element is in the DOM but never upgraded. Should `RenderFragment` return the collector alongside the fragment so the caller can flush it, or should a combined `RenderFragmentWithElements() (html, scripts template.HTML, err error)` API be introduced? *Tentative recommendation*: expose the collector and let the caller decide. **Non-blocking** for v1.

---

## Test Stage Findings

### [ISSUE] Deduplication does not handle the same tag name derived from different files

The deduplication mechanism (§4.4) keys on `TagName`. If `blog/Counter.vue` and `admin/Counter.vue` both produce `htmlc-counter` and are both rendered in the same page, the first one's script wins silently. Open Question 5 proposes a startup-time error, but the collector itself does not detect which file contributed the entry — so a subtle override is possible if the startup check is not implemented first.

**Recommendation**: `CustomElementEntry` should also store `SourcePath string`. `CustomElementCollector.Add` should log a warning (or return an error) if two different `SourcePath` values are added for the same `TagName`.

---

### [ISSUE] Class-name derivation breaks for single-word components

`Counter.vue` → `HtmlcCounter` / `htmlc-counter`. This is correct. But consider `A.vue`: the algorithm produces `HtmlcA` / `htmlc-a`. The tag name `htmlc-a` is syntactically valid (contains a hyphen, starts with a letter), but it is a poor name. The PascalCase splitter must handle single-segment names gracefully without crashing. The RFC should add a validation rule: the file name (without extension) must contain at least one ASCII letter and must not be a single character.

---

### [ISSUE] `FlushCustomElements` called in `<head>` vs. end-of-`<body>` — ordering hazard

The RFC recommends flushing before `</body>`. If `RenderPage` auto-inserts the flush there, and the author also calls `{{FlushCustomElements}}` manually in `<head>`, both calls emit their scripts (the second call with an empty set, if the collector drains on first flush). The RFC does not specify whether `FlushCustomElements` is idempotent (no double-emit) or destructive (drains the collector). This must be stated explicitly.

**Recommendation**: `FlushCustomElements` should be **non-destructive** (reads entries without clearing them), and `RenderPage`'s auto-flush should be opt-out. Or, state that auto-flush is the only supported mechanism and template-level `{{FlushCustomElements}}` is not exposed for `RenderPage`.

---

### [QUESTION] Is verbatim body insertion safe against XSS?

The `scriptBody` content comes from the `.vue` source file on disk, not from user input. It is therefore trusted. The RFC should state this explicitly so reviewers do not raise a false XSS concern. If `htmlc` ever gains a dynamic template evaluation path where component source could come from user-controlled input, this assumption must be revisited.

---

### [QUESTION] What is the interaction between `CustomElementCollector` and `RenderPage`'s existing `<head>` injection?

`RenderPage` already injects a `<style>` block before `</head>` by searching the rendered HTML for the `</head>` string. If custom element scripts are also injected there, two separate string-search-and-replace operations run on the rendered output. This is fragile if the page contains `</head>` in a comment or string literal. The RFC should acknowledge this fragility and suggest that the injection logic be refactored into a single pass.

---

### [SUGGESTION] Add a `Validate` method to `DeriveTagName`

`DeriveTagName` should return an error (not panic) for invalid inputs: empty string, single-character name, names that produce an invalid custom element tag. This makes the failure surface clear and testable.

---

### [SUGGESTION] Document the `observedAttributes` manual pattern in §6

Since automatic `observedAttributes` wiring is deferred, at least one example in §6 should show how an author manually declares `static observedAttributes` and `attributeChangedCallback` inside the `<script customelement>` body. This prevents confusion for authors who expect the attribute-to-property bridge to happen automatically.

---

### [QUESTION] Should `FlushCustomElements` be on `Engine` or exposed via a context-aware helper?

The current design implies the `CustomElementCollector` is per-render-pass. If `Engine` exposes `FlushCustomElements()` as a method, it must internally retrieve the current render pass's collector. This is non-trivial in a concurrent server: multiple goroutines may be rendering simultaneously. The per-pass collector must live in a request-scoped context, not on `Engine` directly. The RFC should address this concurrency concern explicitly in §7.

---

## Inspection Findings

### Scenario 1 — Progressive Enhancement Island (`<Tabs>`)

**What this enables**: Example 2 in §6 already demonstrates this pattern. With RFC 006, a `<Tabs>` component can render all panels as static HTML (visible to search engines and no-JS users) and simultaneously register a custom element that hides inactive panels and wires click handlers. The server-rendered output is a complete, accessible document; the custom element is a progressive enhancement layer applied in `connectedCallback`.

**What friction remains**:
- The author must ensure the outer element in `<template>` is `<htmlc-tabs>` (the custom element tag name) rather than a generic `<div>`. This is a naming discipline that `htmlc` does not currently enforce — the template can render any tag, including one that does not match the registered custom element name.
- If the author uses `<div class="tabs">` in the template but registers `htmlc-tabs`, the browser never upgrades the element. The RFC should include guidance (or a lint rule) requiring the template's root element to match the derived custom element tag name.
- Slot-based content (`<slot name="overview" />`) is resolved server-side, so the tab panels are fully rendered. The custom element script must use `querySelector` to find panels in the already-rendered DOM, not a VDOM — this is idiomatic for custom elements but unfamiliar to Vue developers.

---

### Scenario 2 — Shared Component, Two Contexts (`Button.vue`)

**Scenario**: `Button.vue` is used on dozens of server-side pages. One page adds a `<script customelement>` block for an analytics ping on click.

**Problem**: The RFC defines `<script customelement>` as a block in the `.vue` file itself. A `<script customelement>` block in `Button.vue` applies to **all** usages of `Button.vue`. There is no per-usage or per-page opt-in mechanism. Every page that uses `<Button>` will trigger `CustomElementCollector.Add(...)` and eventually emit the analytics script, even pages where analytics is unwanted.

**Desired behaviour**: the author likely wants the custom element script only on specific pages. The current design forces an all-or-nothing choice: either every usage gets the script, or no usage does.

**Recommendation for RFC**: add a note that `<script customelement>` is a component-level declaration, not a usage-level opt-in. Authors who need per-page opt-in should create a wrapper component (e.g., `TrackedButton.vue`) that includes `<script customelement>` and composes `<Button>`. This is the idiomatic pattern and should be documented in §6.

**Gap**: the RFC currently has no example of this pattern. An Example 5 covering the "composition-as-opt-in" pattern would address this.

---

### Scenario 3 — Import Map and CDN Delivery

**Scenario**: static assets are served from a CDN; the import map maps custom element specifiers to versioned CDN URLs.

**Current design accommodation**: Option B in §4.5 describes URL-based delivery but defers it. The current proposal (Option A, inline) does not produce URLs and does not emit an import map. A deployment team that wants CDN delivery cannot use v1 for this scenario without post-processing the rendered HTML.

**What would need to be added**: §4.5 Option B describes the mechanism. The key additions are:
1. A configurable `AssetsDir` where generated JS files are written at startup.
2. A hash-based file name (`htmlc-counter.a1b2c3d4.js`) for cache busting.
3. An `ImportMapEmitter` that builds the `<script type="importmap">` JSON from the set of registered custom elements.
4. A CDN base URL configuration option so the import map can reference `https://cdn.example.com/assets/htmlc-counter.a1b2c3d4.js`.

**Gap**: the RFC should explicitly state in §10 that CDN delivery is the primary motivating use case for Option B, and that the open question of when to write asset files (startup vs. first render vs. build step) must be resolved before Option B can be implemented.

---

### Scenario 4 — Multiple Custom Elements on One Page (Dashboard)

**Scenario**: a dashboard with five different custom elements (`<Chart>`, `<Sparkline>`, `<Toggle>`, `<DateRange>`, `<LiveCounter>`).

**Deduplication scale**: `CustomElementCollector` uses a `map[string]struct{}` keyed by tag name. With five distinct elements, five entries accumulate. `FlushCustomElements()` emits five `<script>` blocks. This is correct and scales linearly. There is no inherent performance cliff.

**Where the author calls `FlushCustomElements`**: in `RenderPage`, the auto-flush before `</body>` is the recommended path. For `RenderFragment`, the author calls the returned collector's `FlushCustomElements()` and appends the result to the fragment — or wraps the fragment in a layout template that handles the flush. The RFC's §8 mentions this but does not show a concrete example for the fragment path.

**Gap**: the RFC should add an example showing a `RenderFragment`-based page assembly pattern where the caller is responsible for flushing. This is a common use case for server-side Go applications that compose page sections independently.

---

### Scenario 5 — Nested Custom Elements (`<Modal>` contains `<DatePicker>`)

**Scenario**: `<Modal>` has `<script customelement>`. `<DatePicker>` has `<script customelement>`. `<Modal>` renders `<DatePicker>` in its template.

**Emission order**: during a render of `<Modal>`, the renderer processes `<Modal>`'s template, encounters `<DatePicker>`, and recursively renders it. The recursive `renderElement` call processes `<DatePicker>` first (depth-first), calling `collector.Add(DatePicker entry)`. When the recursion returns, `renderElement` processes `<Modal>` and calls `collector.Add(Modal entry)`. Entries in `collector.entries` are therefore in depth-first order: `[HtmlcDatePicker, HtmlcModal]`.

`FlushCustomElements()` emits `<script>` blocks in insertion order: `DatePicker` first, then `Modal`. This is correct — a browser upgrading `htmlc-modal` may immediately render `<htmlc-date-picker>` inside its shadow DOM (if shadow DOM is used), and the `DatePicker` definition is already registered.

**Shadow DOM nesting**: if `<Modal>` uses shadow DOM and renders `<htmlc-date-picker>` inside its shadow root, the browser needs `HtmlcDatePicker` to be registered before or simultaneously with `HtmlcModal`. Since both definitions are in the same `<head>` or end-of-`<body>` block and are executed in order, this is safe.

**Gap**: the depth-first emission order is a consequence of the rendering algorithm, not an explicit guarantee. The RFC should state that `FlushCustomElements()` emits definitions in the order the collector received them (depth-first, reflecting the component tree), and that this order is safe for nested custom elements because parent elements are registered after their children's definitions are already present.
