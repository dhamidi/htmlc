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
2. **Author interactivity inside the `.vue` file**: provide a single block (`<script customelement>`) where the author writes any client-side JavaScript they need; `htmlc` emits it verbatim into the page.
3. **Automatic SSR wrapping**: when a component carries `<script customelement>`, `htmlc` automatically wraps the rendered template output in the component's derived tag name (e.g. `<counter>`, `<admin-card>`), so the browser's Custom Elements registry can upgrade it.
4. **Error on `<script>` and `<script setup>`**: emit a descriptive compile-time error when these blocks appear, preventing silent failures for authors who accidentally write standard Vue script blocks.
5. **Deduplication across the render pass**: the same custom element script is emitted at most once per page, even if the component is used multiple times.
6. **In-memory script FS**: compiled scripts are collected into an in-memory `fs.FS` accessible on the engine, enabling the application to serve, cache-bust, or embed them however it chooses.
7. **Zero impact on components without `<script customelement>`**: server-side rendering behaviour is identical to today for all existing components.

---

## 3. Non-Goals

- **Implementing Vue 3 reactivity, the Options API, or the Composition API on the client.** The emitted JS is whatever the author writes; no reactivity primitives are provided or planned.
- **Supporting `<script>` or `<script setup>` blocks.** These are intentional compile-time errors.
- **SSR hydration or resumability.** The server renders static HTML; the Custom Element enhances it after the fact. There is no serialised component state passed from server to client.
- **Bundling or tree-shaking.** Each custom element script is independent. There is no module bundler integration.
- **Dynamic imports or lazy loading.** All emitted scripts are eager.
- **Customised built-in elements (`is="..."` syntax).** Safari does not support them; autonomous custom elements are the only viable cross-browser target.
- **Generating class boilerplate or autoregistering elements.** The compiler emits the author's script verbatim. Authors who want a `class extends HTMLElement` pattern write it themselves.

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
    CustomElementScript string   // new: verbatim body of <script customelement>, empty if absent
    CustomElementTag    string   // new: derived tag name (set during load, not parsing)
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
    // new: record as custom element body (verbatim)
    sections["script:customelement"] = rawBody(tokenizer)
default:
    // existing: plain <script> — stored; will be rejected later
    sections["script"] = rawBody(tokenizer)
}
```

`ParseFile` then populates the struct:

```go
// pseudo-code — not implementation
comp.CustomElementScript = sections["script:customelement"]
// CustomElementTag is set by the engine after load, derived from the component path

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
- ✅ No `tag` override attribute — the tag name is always derived from the component file path, making it predictable and greppable.
- ⚠️ `sections` map grows one new key; ensure the "duplicate section" guard covers all key combinations.

**Verdict**: extend attribute reading in `extractSections` to detect `customelement`; store verbatim body as the new `CustomElementScript` field.

---

### 4.2 Tag-Name Derivation

The HTML custom element tag name is derived deterministically from the component's file path relative to the component root, using every path segment:

**Algorithm**:

1. Split the relative path into directory segments and the file name (without extension).
2. For each segment: convert PascalCase or CamelCase to kebab-case by inserting a hyphen before each uppercase letter that follows a lowercase letter or digit, then lowercase the whole string.
3. Join all kebab-cased segments with `-`.

Examples:

| File path (relative to component root) | Derived tag name |
|-----------------------------------------|-----------------|
| `Counter.vue`                           | `counter`        |
| `DatePicker.vue`                        | `date-picker`    |
| `admin/Card.vue`                        | `admin-card`     |
| `admin/DatePicker.vue`                  | `admin-date-picker` |
| `blog/Counter.vue`                      | `blog-counter`   |
| `ui/form/TextInput.vue`                 | `ui-form-text-input` |

**Note on the Custom Elements specification**: the browser's `customElements.define()` API requires tag names to contain at least one hyphen. A top-level component such as `Counter.vue` derives the tag name `counter`, which does not satisfy this requirement and cannot be registered as a browser custom element. Authors who want to register a component as a browser custom element must place it in at least one subdirectory (e.g. `widgets/Counter.vue` → `widgets-counter`). Components without a `<script customelement>` block are unaffected by this restriction regardless of location.

**Collision considerations**: two distinct components that produce the same derived tag name (e.g. `blog/counter.vue` and `blog/Counter.vue` on a case-insensitive filesystem) are a load-time error. The engine detects this during component loading and aborts with a descriptive message.

**Evaluation**

- ✅ Deterministic: same path always produces the same tag name.
- ✅ No synthetic prefix — the tag name is the component's identity, readable at a glance.
- ✅ Directory path encodes namespace — `admin/Card` and `blog/Card` produce distinct tag names automatically.
- ⚠️ Top-level single-word components (`Counter.vue` → `counter`) cannot be registered as browser custom elements. Documented above; authors are expected to namespace components in subdirectories.
- ⚠️ Acronym sequences (e.g. `XMLParser.vue`) produce `x-m-l-parser` inside a segment. Authors should name files consistently (e.g. `XmlParser.vue` → `xml-parser`).

**Verdict**: derive from the full component path; no prefix; PascalCase segments kebab-cased; directory and file joined with `-`.

---

### 4.3 SSR Wrapping

When `Component.CustomElementScript` is non-empty, `htmlc` wraps the fully rendered template output in the component's derived tag name. This happens automatically — the author does not need to place the custom element tag in their `<template>` root.

#### Wrapping pseudocode

```go
// pseudo-code — not implementation
func wrapInCustomElement(tagName string, renderedHTML template.HTML) template.HTML {
    return template.HTML(fmt.Sprintf("<%s>%s</%s>", tagName, renderedHTML, tagName))
}
```

The author's `<template>` block contains only the component's inner content:

```html
<!-- widgets/Counter.vue -->
<template>
  <button>Click me</button>
  <span>0</span>
</template>

<script customelement>
customElements.define('widgets-counter', class extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
});
</script>
```

This produces the following SSR output when `<Counter />` is used in a parent template:

```html
<widgets-counter>
  <button>Click me</button>
  <span>0</span>
</widgets-counter>
```

**Evaluation**

- ✅ Authors do not duplicate the tag name inside the template — the compiler derives and applies it.
- ✅ Template content is identical to a component without `<script customelement>`, keeping the mental model consistent.
- ✅ The browser upgrades the element automatically when the script (emitted separately via `FlushCustomElements`) executes.
- ⚠️ Authors using `querySelector` in their custom element class must account for the fact that the matched elements are the direct children of the custom element — the same as they are in the rendered DOM.

**Verdict**: automatically wrap rendered template output in the custom element tag when `CustomElementScript` is non-empty.

---

### 4.4 JS Emission — Verbatim Script

When `Component.CustomElementScript` is non-empty, `htmlc` emits the author's script **verbatim** inside a `<script>` tag. No class boilerplate is added, no `customElements.define()` call is generated. The script content is the author's complete, standalone JavaScript.

#### Emission pseudocode

```go
// pseudo-code — not implementation
func emitCustomElementScript(scriptBody string) template.HTML {
    return template.HTML("<script>\n" + scriptBody + "\n</script>")
}
```

The author writes the full script — including any `customElements.define()` call if browser-side upgrade is desired:

```html
<!-- widgets/Counter.vue -->
<script customelement>
customElements.define('widgets-counter', class extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
});
</script>
```

**Evaluation**

- ✅ No runtime dependency: the emitted script is exactly what the author wrote.
- ✅ Verbatim emission means the author can use any JS pattern: bare functions, IIFE, `class extends HTMLElement`, ES module syntax — no wrapper imposes a structure.
- ✅ Authors who do not need browser-side upgrade can use `<script customelement>` purely to signal SSR wrapping and emit a no-op script (or one that does unrelated page initialisation).
- ⚠️ `htmlc` cannot validate JS syntax. Syntax errors surface in the browser console, not at build time.

**Verdict**: verbatim emission only. No class wrapper, no autoregistration. The `<script customelement>` block informs compilation (SSR wrapping) and supplies the page script; its content is the author's responsibility.

---

### 4.5 Deduplication

The same component may be rendered many times on a single page (e.g. a `<Counter>` inside a list of 50 items). The script must appear **exactly once** per page.

#### Current state

`StyleCollector` (in `style.go`) already implements deduplication for CSS: it accumulates `StyleContribution` values keyed by `ScopeID+"\x00"+CSS`, and each unique contribution is flushed once via `styleBlock(sc)`.

#### Proposed extension

Introduce a parallel **`CustomElementCollector`** type:

```go
// pseudo-code — not implementation
type CustomElementEntry struct {
    TagName    string
    Script     string // verbatim body from <script customelement>
    SourcePath string // component file path, for collision detection
}

type CustomElementCollector struct {
    seen    map[string]string    // tag name → source path of first registration
    entries []CustomElementEntry
}

func (c *CustomElementCollector) Add(e CustomElementEntry) error {
    if prior, ok := c.seen[e.TagName]; ok {
        if prior != e.SourcePath {
            return fmt.Errorf(
                "custom element tag %q is defined by both %s and %s",
                e.TagName, prior, e.SourcePath)
        }
        return nil   // same component rendered again — deduplicate silently
    }
    c.seen[e.TagName] = e.SourcePath
    c.entries = append(c.entries, e)
    return nil
}

func (c *CustomElementCollector) FlushCustomElements() template.HTML {
    var b strings.Builder
    for _, e := range c.entries {
        b.WriteString(string(emitCustomElementScript(e.Script)))
    }
    return template.HTML(b.String())
}
```

The `Renderer` receives a `*CustomElementCollector` (analogous to `*StyleCollector`). Whenever `renderElement` processes a component tag whose `Component.CustomElementScript` is non-empty, it calls `collector.Add(...)`. The collector is allocated per render pass (per `RenderPage` / `RenderFragment` call) and is not shared across concurrent requests.

`FlushCustomElements()` is called by the page author (or by `RenderPage` automatically before `</body>`) to emit the deduplicated script tags. `FlushCustomElements()` is **non-destructive**: it reads entries without clearing them, so calling it multiple times produces the same output.

**Evaluation**

- ✅ Exact mirror of the existing style deduplication pattern — low conceptual overhead.
- ✅ Per-render-pass allocation means no cross-request state leakage.
- ✅ `SourcePath` tracking surfaces collisions immediately rather than silently shadowing.
- ✅ Non-destructive flush means `{{FlushCustomElements}}` can be called from a template without ordering constraints.
- ⚠️ `FlushCustomElements` placement (end of `<body>` vs. `<head>`) affects when the script executes. Deferring to end-of-`<body>` is the safest default because the element's HTML is already in the DOM when the script executes.

**Verdict**: introduce `CustomElementCollector` mirroring `StyleCollector`; non-destructive `FlushCustomElements()` returns a `template.HTML` string of deduplicated `<script>` tags.

---

### 4.6 In-Memory Script FS

At engine load time, `htmlc` compiles all custom element scripts into an **in-memory `fs.FS`**. This FS is populated once during startup (when component files are parsed) and is accessible via a method on `Engine`. It provides a stable, request-independent view of all compiled scripts, which the application can use to:

- Serve scripts as static files via `http.FileServer(engine.ScriptsFS())`.
- Write scripts to disk as a build step.
- Generate content-addressed URLs for cache busting.
- Embed scripts in `<head>` independently of the per-render-pass collector.

#### FS structure

Scripts are stored under their tag name with a `.js` extension:

```
counter.js
date-picker.js
admin-card.js
admin-date-picker.js
blog-counter.js
```

#### Implementation

```go
// pseudo-code — not implementation
import "archive/zip"
import "io/fs"

type Engine struct {
    // existing fields ...
    scripts fs.FS   // in-memory FS populated at load time
}

// ScriptsFS returns an fs.FS containing one .js file per component
// that declares <script customelement>. The FS is populated at engine
// load time and is safe for concurrent reads.
func (e *Engine) ScriptsFS() fs.FS {
    return e.scripts
}
```

The FS is constructed using an in-memory zip archive (`archive/zip`) or an equivalent in-memory `fs.FS` implementation. During `Engine.Load()`, for each `Component` with a non-empty `CustomElementScript`, a file is written to the archive at `<tagName>.js` containing the verbatim script body.

**Evaluation**

- ✅ `fs.FS` is a standard Go interface — callers can wrap it with `http.FS`, embed it, copy it, or pass it to `io/fs` utilities without depending on `htmlc` internals.
- ✅ Populated at startup — no per-request I/O.
- ✅ `archive/zip` provides an in-memory `fs.FS`-compatible implementation without requiring a third-party dependency.
- ⚠️ The FS is read-only after load. Reloading requires recreating the engine.

**Verdict**: use an in-memory `archive/zip`-backed `fs.FS` as the compilation output for custom element scripts; expose it via `Engine.ScriptsFS()`.

---

### 4.7 Interaction with `<style scoped>` and `<style>`

A component that declares `<script customelement>` may also declare a `<style>` block. The style is handled exactly as today by `StyleCollector`. The SSR-wrapped element (e.g. `<admin-card>`) inherits any scoped attribute (e.g. `data-v-a1b2c3d4`) stamped on it by the renderer. No change required.

---

### 4.8 Error Behaviour for `<script>` and `<script setup>`

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
| `<script customelement>` | *(none)* | Marks component as a custom element. SSR output is wrapped in the derived tag name. Script body is emitted verbatim (no class wrapper, no autoregistration generated). |
| `<script>` | *(any)* | **Error**: `<script> blocks are not supported by htmlc` |
| `<script setup>` | `setup` | **Error**: `<script setup> blocks are not supported by htmlc` |
| `<style>` | *(none)* | Global stylesheet contribution; unchanged from today |
| `<style scoped>` | `scoped` | Scoped stylesheet contribution; unchanged from today |
| `<template>` | *(none)* | Server-side render template; unchanged from today |

### Tag name derivation

| File path (relative) | Derived tag name |
|---|---|
| `Counter.vue` | `counter` |
| `DatePicker.vue` | `date-picker` |
| `admin/Card.vue` | `admin-card` |
| `admin/DatePicker.vue` | `admin-date-picker` |
| `ui/form/TextInput.vue` | `ui-form-text-input` |

---

## 6. Examples

### Example 1 — Namespaced Interactive Counter

A counter component under `widgets/` so its tag name contains a hyphen and can be registered as a browser custom element.

**Directory tree**

```
components/
  widgets/
    Counter.vue
pages/
  Home.vue
```

**`widgets/Counter.vue`**

```html
<template>
  <button>Click me</button>
  <span>0</span>
</template>

<script customelement>
customElements.define('widgets-counter', class extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
});
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
    <widgets-counter>
      <button>Click me</button>
      <span>0</span>
    </widgets-counter>
    <script>
customElements.define('widgets-counter', class extends HTMLElement {
  connectedCallback() {
    this.count = 0;
    this.button = this.querySelector('button');
    this.display = this.querySelector('span');
    this.button.addEventListener('click', () => {
      this.count++;
      this.display.textContent = this.count;
    });
  }
});
</script>
  </body>
</html>
```

---

### Example 2 — Progressive Enhancement (Template + Custom Element)

A `<Tabs>` component under `ui/` that renders all tab panels statically for SEO and no-JS users, and uses `<script customelement>` to add client-side tab switching.

**`ui/Tabs.vue`**

```html
<template>
  <nav class="tab-bar">
    <button data-tab="0">Overview</button>
    <button data-tab="1">Details</button>
    <button data-tab="2">Reviews</button>
  </nav>
  <div class="tab-panel" data-panel="0"><slot name="overview" /></div>
  <div class="tab-panel" data-panel="1"><slot name="details" /></div>
  <div class="tab-panel" data-panel="2"><slot name="reviews" /></div>
</template>

<style scoped>
.tab-panel { display: block; }
</style>

<script customelement>
customElements.define('ui-tabs', class extends HTMLElement {
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
});
</script>
```

**Rendered SSR output for `<Tabs>` usage**

```html
<ui-tabs data-v-a1b2c3d4>
  <nav class="tab-bar" data-v-a1b2c3d4>
    <button data-tab="0" data-v-a1b2c3d4>Overview</button>
    ...
  </nav>
  <div class="tab-panel" data-panel="0" data-v-a1b2c3d4>...</div>
  ...
</ui-tabs>
```

**Behaviour**

- Without JS: all three panels are visible; users see all content (SEO-friendly, accessible).
- With JS: the Custom Element's `connectedCallback` hides panels 1 and 2 and activates tab-bar button events.
- The `<style scoped>` contribution is flushed into `<head>` by the existing `StyleCollector` path.
- The custom element script is flushed by `FlushCustomElements()` before `</body>`.

---

### Example 3 — Multiple Custom Elements on One Page (Deduplication)

A dashboard page that uses `<Counter>` three times and `<Toggle>` once (both under `widgets/`).

**`widgets/Toggle.vue`** (abbreviated)

```html
<template>
  <input type="checkbox" /><label><slot /></label>
</template>

<script customelement>
customElements.define('widgets-toggle', class extends HTMLElement {
  connectedCallback() {
    this.input = this.querySelector('input');
    this.input.addEventListener('change', () => {
      this.dispatchEvent(new CustomEvent('toggle', { detail: this.input.checked }));
    });
  }
});
</script>
```

**`Dashboard.vue`** (abbreviated)

```html
<template>
  <html>
    <head><title>Dashboard</title></head>
    <body>
      <h1>Dashboard</h1>
      <Counter />
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
<!-- Only ONE script block per element, regardless of how many times used -->
<script>
customElements.define('widgets-counter', class extends HTMLElement {
  connectedCallback() { /* ... */ }
});
</script>
<script>
customElements.define('widgets-toggle', class extends HTMLElement {
  connectedCallback() { /* ... */ }
});
</script>
```

The `CustomElementCollector` tracks `{ "widgets-counter": "widgets/Counter.vue", "widgets-toggle": "widgets/Toggle.vue" }`. The three `<Counter>` renders each call `collector.Add(...)`, but only the first produces an entry. `FlushCustomElements()` emits exactly two `<script>` blocks.

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
- No SSR wrapping is applied — output is a plain `<div class="card">` as today.
- `CustomElementCollector.Add` is never called.
- `FlushCustomElements()` returns `template.HTML("")`.
- Output is identical to today.

---

### Example 5 — Serving Scripts via `ScriptsFS`

The application serves compiled custom element scripts as static files, enabling browser caching independent of the HTML page.

```go
// pseudo-code — not implementation
engine, err := htmlc.Load("components/")
if err != nil {
    log.Fatal(err)
}

// Serve all custom element scripts under /assets/
http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(engine.ScriptsFS()))))

// In the page template, reference the script by URL instead of inlining:
// <script src="/assets/widgets-counter.js"></script>
```

This pattern is appropriate for production deployments where scripts must be independently cacheable. For development or low-traffic sites, inline emission via `FlushCustomElements()` (§4.5) requires no additional setup.

---

### Example 6 — Per-Page Opt-In via Wrapper Component

`<script customelement>` is a component-level declaration. If `Button.vue` carries one, every page using `<Button>` emits the script. An author who wants the custom element behaviour on only one page creates a wrapper:

**Directory tree**

```
components/
  Button.vue          ← no <script customelement>; pure SSR
  tracked/
    Button.vue        ← wraps Button, adds analytics custom element
```

**`tracked/Button.vue`**

```html
<template>
  <Button v-bind="$props"><slot /></Button>
</template>

<script customelement>
customElements.define('tracked-button', class extends HTMLElement {
  connectedCallback() {
    this.querySelector('button').addEventListener('click', () => {
      navigator.sendBeacon('/analytics', JSON.stringify({ event: 'click', component: 'button' }));
    });
  }
});
</script>
```

Pages that need analytics use `<tracked/Button>` (or the namespaced alias). Pages that do not want analytics continue to use `<Button>`. This is the idiomatic `htmlc` pattern for usage-scoped opt-in.

---

## 7. Implementation Sketch

### `component.go`

1. Add two new fields to `Component`: `CustomElementScript string` and `CustomElementTag string`.
2. In `extractSections`, after reading a `<script>` start tag, collect all its attributes into a `map[string]string`.
3. If `attrs["customelement"] != ""` or the attribute name `"customelement"` is present (boolean attribute), store the verbatim body in `sections["script:customelement"]`.
4. In `ParseFile`, populate `CustomElementScript` from `sections["script:customelement"]`.
5. Convert the current silent-ignore of `sections["script"]` and `sections["script:setup"]` into explicit error returns with the messages defined in §4.8.

### `customelement.go` (new file)

1. Define `CustomElementEntry` struct with `TagName`, `Script`, `SourcePath string` fields.
2. Define `CustomElementCollector` struct with `seen map[string]string` (tag name → source path) and `entries []CustomElementEntry`.
3. Implement `(c *CustomElementCollector) Add(e CustomElementEntry) error` — no-op if tag already seen from same source; error if tag seen from different source.
4. Implement `(c *CustomElementCollector) FlushCustomElements() template.HTML` — iterates `entries`, calls `emitCustomElementScript` for each, concatenates results. Non-destructive.
5. Add standalone `emitCustomElementScript(scriptBody string) template.HTML` helper.
6. Add `DeriveTagName(relPath string) string` helper implementing the algorithm in §4.2 (splits on `/`, kebab-cases each PascalCase segment, joins with `-`).
7. Add `BuildScriptsFS(components []*Component) (fs.FS, error)` that constructs an in-memory zip-backed `fs.FS` with one entry per component that has `CustomElementScript != ""`, keyed as `<tagName>.js`.

### `engine.go`

1. Add `scripts fs.FS` field to `Engine`.
2. During component loading (in `Load` or equivalent), call `DeriveTagName` for each component and set `component.CustomElementTag`.
3. After all components are loaded, call `BuildScriptsFS` and store the result in `Engine.scripts`.
4. Add `ScriptsFS() fs.FS` public method that returns `Engine.scripts`.
5. In `renderComponent` (and its callers), allocate a `*CustomElementCollector` per render pass alongside the existing `*StyleCollector`.
6. In `RenderPage`, after injecting the style block, call `collector.FlushCustomElements()` and inject the result before `</body>`. Alternatively, expose `FlushCustomElements` as a template function so page authors control placement.
7. Add `FlushCustomElements() template.HTML` as a public method on `Engine` (delegating to the per-pass collector) for programmatic callers.

### `renderer.go`

1. Add `customElementCollector *CustomElementCollector` field to `Renderer`.
2. In `renderElement`, when resolving a component tag: if the resolved `Component.CustomElementScript != ""`:
   a. Call `customElementCollector.Add(CustomElementEntry{...})`.
   b. Wrap the rendered template output in `<tagName>...</tagName>` (see §4.3).
3. Propagate the collector into child `Renderer` instances (same pattern as `styleCollector`).

### Platform notes

- All file-name manipulation uses `path` (not `path/filepath`) for OS portability, since component paths are relative paths derived from `fs.FS` which always uses forward slashes.
- The `DeriveTagName` function should use `unicode` package functions for PascalCase splitting, not byte-level comparisons, to handle future non-ASCII names gracefully.
- `archive/zip` provides an in-memory `fs.FS` implementation via `zip.NewReader` over a `bytes.Buffer`. No third-party dependency is required.

---

## 8. Backward Compatibility

### `Component` struct

New exported fields `CustomElementScript string` and `CustomElementTag string` are added. Backward-compatible in Go: existing code constructing `Component` by field name is unaffected.

### `ParseFile` and `ParseDir`

For components without `<script customelement>`, both functions return the same results as today. The only observable behavioural change is that components with a plain `<script>` or `<script setup>` block — which previously silently stored the body in `Component.Script` (unused) — now return an error. Since those blocks had no effect on rendering, the only affected case is a misconfigured component that was silently broken. The error message is actionable.

### `RenderPage` / `RenderFragment`

No change for components without `<script customelement>`. For components that do use it, `RenderPage` gains automatic SSR wrapping and `FlushCustomElements()` injection before `</body>`. `RenderFragment` does not auto-flush; callers must invoke `FlushCustomElements()` explicitly.

### `Engine` public API

- New method `ScriptsFS() fs.FS` — additive, no break.
- New method `FlushCustomElements() template.HTML` — additive, no break.
- No existing methods are removed or have their signatures changed.

### `StyleCollector`

Unchanged. The new `CustomElementCollector` is a parallel type, not a modification.

---

## 9. Alternatives Considered

### A. Top-level `<customelement>` custom block

Vue's SFC spec allows arbitrary custom blocks (e.g. `<docs>`, `<i18n>`). A `<customelement>` block would be syntactically valid in Vue and clearly distinct from `<script>`.

**Rejected** because: Vue's custom block body is not parsed as JavaScript by IDEs or linters — it is treated as opaque text. Authors would lose syntax highlighting, `eslint`, and IDE completions. Using `<script customelement>` keeps the block recognised as a `<script>` by tooling.

### B. A separate `.ce.js` file alongside the `.vue` file

`Counter.vue` + `Counter.ce.js` → `htmlc` combines them automatically.

**Rejected** because: the whole motivation is to keep the component boundary in one file. A companion file reintroduces the synchronisation problem described in §1.

### C. Auto-generate class boilerplate and `customElements.define()`

`htmlc` wraps the script body in `class extends HTMLElement { … }` and appends `customElements.define(tagName, ClassName)`.

**Rejected** because: the class wrapper is limiting (authors may prefer an IIFE, a factory function, or a class with a custom base) and confusing (the generated class name is not visible in the source file). Verbatim emission gives authors full control with no hidden indirection.

### D. `htmlc-` prefix for tag names

Use `htmlc-counter` instead of `counter` / `widgets-counter` to avoid collisions with HTML elements.

**Rejected** because: the directory-based namespacing already provides collision avoidance. The `htmlc-` prefix is a synthetic namespace that does not reflect any meaningful structure in the project, whereas directory paths do. Authors who want a custom prefix can use their own directory names.

### E. Full Vue 3 client-side compilation

Compile the `<script setup>` block (Composition API) to a client-side Vue component, ship the Vue runtime, and mount it on the element.

**Rejected** because: requires shipping the Vue runtime (≈50 KB min+gzip), reimplementing the Vue compiler, and maintaining compatibility with Vue version upgrades. Out of scope for a server-side rendering engine.

---

## 10. Open Questions

1. **Shadow DOM opt-in** — Should shadow DOM be opt-in via `<script customelement shadowdom>`? If so, should `open` or `closed` mode be the default? *Tentative recommendation*: reserve the `shadowdom` attribute for a future RFC. **Non-blocking** for v1.

2. **`FlushCustomElements` placement** — Should auto-flush in `RenderPage` be opt-out (on by default, author can disable) or opt-in (off by default, author must call `{{FlushCustomElements}}`)? *Tentative recommendation*: opt-out — the overwhelming common case is to emit scripts before `</body>`. **Blocking** — the default must be decided before implementation.

3. **`RenderFragment` ergonomics** — If an author calls `RenderFragment` to render a snippet containing a custom element, the element is in the DOM but no script is emitted. Should a combined `RenderFragmentWithElements() (html, scripts template.HTML, err error)` API be added? *Tentative recommendation*: yes, as a convenience method. **Non-blocking** for v1.

4. **Nonce support for inline scripts** — CSP `script-src` policies require a nonce on inline `<script>` tags. Should `FlushCustomElements` accept a nonce string? *Tentative recommendation*: yes, as a variadic option so the call site without a nonce is unchanged. **Non-blocking** — can be added without API break.

5. **`ScriptsFS` file naming** — Should script files be named `<tagName>.js` (e.g. `widgets-counter.js`) or include a content hash for cache busting (e.g. `widgets-counter.a1b2c3d4.js`)? *Tentative recommendation*: plain `<tagName>.js` for v1; content-addressed naming can be added as an engine option in a follow-up. **Non-blocking**.

6. **Top-level components and single-word tag names** — `Counter.vue` derives `counter`, which is not a valid browser custom element name. Should `htmlc` emit a warning (not an error) when a component with `<script customelement>` derives a single-word tag name? *Tentative recommendation*: yes, warn at load time. **Blocking** — authors need to know this constraint.
