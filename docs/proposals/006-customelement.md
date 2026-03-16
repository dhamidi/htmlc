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
5. **Deduplication across the render pass**: the same custom element script is referenced at most once per page, via the importmap and `ScriptsFS`.
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
- **No `RenderFragmentWithElements` convenience method for v1.** Authors use `{{importMap()}}` explicitly in fragment templates; the combined return-value API adds API surface for marginal ergonomic gain and is deferred. See §10 Q3.
- **Shadow DOM is not implemented in v1.** Declarative Shadow DOM requires non-trivial changes to SSR wrapping, `StyleCollector`, and the compiled custom element scripts. The design is outlined in §4.10 for future implementation.

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
// New: require customElements.define in the script body
if comp.CustomElementScript != "" &&
    !strings.Contains(comp.CustomElementScript, "customElements.define") {
    return nil, fmt.Errorf(
        "%s: <script customelement> body must contain a customElements.define() call", path)
}
```

**Evaluation**

- ✅ Reads all script-block variants from a single tokeniser pass — no second parse.
- ✅ The `customelement` attribute is confirmed absent from Vue's SFC spec (`lang`, `src`, `generic`, `setup` are the only recognised attributes on `<script>`). No collision.
- ✅ No `tag` override attribute — the tag name is always derived from the component file path, making it predictable and greppable.
- ⚠️ `sections` map grows one new key; ensure the "duplicate section" guard covers all key combinations.

**Verdict**: extend attribute reading in `extractSections` to detect `customelement`; store verbatim body as the new `CustomElementScript` field.

#### `src` attribute on `<script customelement>`

The `src` attribute on `<script customelement>` is **not supported** and is treated as a compile-time error:

```
path/to/Component.vue: <script customelement src="..."> is not supported;
write the script body inline inside the <script customelement> block
```

**Rationale**: the content hash in §4.6 is computed over the verbatim inline script body. If `src` pointed to an external file, `htmlc` would need to resolve and read it at component-load time, introducing file-system coupling that is out of scope. Authors who maintain a shared JS utility file should import it from within the inline `<script customelement>` block using a standard ES `import` statement (which is emitted verbatim).

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

**Compile-time error**: if a component carries `<script customelement>` and its derived tag name contains no hyphen (e.g. `Counter.vue` → `counter`), `htmlc` returns an error at engine load time: `Counter.vue: custom element tag name "counter" is invalid — tag names must contain at least one hyphen; move the component to a subdirectory (e.g. widgets/Counter.vue → widgets-counter)`

Components without a `<script customelement>` block are unaffected by this restriction regardless of location.

**Collision considerations**: two distinct components that produce the same derived tag name (e.g. `blog/counter.vue` and `blog/Counter.vue` on a case-insensitive filesystem) are a load-time error. The engine detects this during component loading and aborts with a descriptive message.

**Evaluation**

- ✅ Deterministic: same path always produces the same tag name.
- ✅ No synthetic prefix — the tag name is the component's identity, readable at a glance.
- ✅ Directory path encodes namespace — `admin/Card` and `blog/Card` produce distinct tag names automatically.
- ✅ Top-level single-word components with `<script customelement>` are rejected at load time with an actionable error message.
- ⚠️ Acronym sequences (e.g. `XMLParser.vue`) produce `x-m-l-parser` inside a segment. Authors should name files consistently (e.g. `XmlParser.vue` → `xml-parser`).

**Verdict**: derive from the full component path; no prefix; PascalCase segments kebab-cased; directory and file joined with `-`. Single-word derived tag names (no hyphen) are a compile-time error for components that carry `<script customelement>`.

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
- ✅ The browser upgrades the element automatically when the script (referenced via the importmap in `<head>`) executes.
- ⚠️ Authors using `querySelector` in their custom element class must account for the fact that the matched elements are the direct children of the custom element — the same as they are in the rendered DOM.

**Verdict**: automatically wrap rendered template output in the custom element tag when `CustomElementScript` is non-empty.

---

### 4.4 JS Emission — `scriptFor` Template Function

Rather than emitting scripts automatically via a dedicated flush call, `htmlc` exposes a **`scriptFor`** template function that returns the verbatim body of a specific component's custom element script. This lets page authors opt into inline script rendering surgically, placing the `<script>` tag wherever they choose.

#### Usage

```html
<script>{{scriptFor("widgets/Counter")}}</script>
```

The argument is the component path **relative to the root component directory**, without the `.vue` extension. This matches the same path convention used to reference components elsewhere in `htmlc`.

#### Return value and wrapping

`scriptFor` returns the raw script body as a `template.HTML` value. It does **not** wrap the body in `<script>` tags — the page author supplies the surrounding tag. This keeps placement, attributes (e.g. `type`, `nonce`), and ordering under the author's control.

#### Error behaviour

`scriptFor` returns an error (surfaced as a template execution error) in three cases:

1. **Unknown component path**: the argument does not match any component registered with the engine (after stripping the `.vue` extension).
2. **No `<script customelement>` block**: the component exists but has an empty `CustomElementScript`.
3. **Already collected for importmap delivery**: the component's tag name is already in the `CustomElementCollector` for this render pass (i.e. the component has been rendered on this page and its script will be delivered via the importmap). Mixing inline delivery via `scriptFor` with importmap delivery on the same render pass would cause `customElements.define()` to execute twice. Error message: `scriptFor: component %q is already scheduled for importmap delivery on this render pass; use either scriptFor or the importmap, not both`

All errors name the offending path and are emitted at render time.

#### Implementation pseudocode

`scriptFor` is a closure over the per-render `Renderer` so it can access the current render-pass collector:

```go
// pseudo-code — not implementation
func (r *Renderer) scriptFor(relPath string) (template.HTML, error) {
    comp, ok := r.engine.componentsByRelPath[relPath]
    if !ok {
        return "", fmt.Errorf("scriptFor: no component found at path %q", relPath)
    }
    if comp.CustomElementScript == "" {
        return "", fmt.Errorf("scriptFor: component %q has no <script customelement> block", relPath)
    }
    if r.customElementCollector.Has(comp.CustomElementTag) {
        return "", fmt.Errorf(
            "scriptFor: component %q is already scheduled for importmap delivery on this render pass; "+
            "use either scriptFor or the importmap, not both", relPath)
    }
    return template.HTML(comp.CustomElementScript), nil
}
```

`scriptFor` is registered on the engine via `engine.RegisterFunc("scriptFor", ...)`, following the same pattern as other engine-provided expression functions. Call it with JavaScript call syntax in templates: `{{scriptFor("path/to/Component")}}`.

**Evaluation**

- ✅ Authors control placement, tag attributes, and ordering — no hidden magic.
- ✅ Works uniformly for both `RenderPage` and `RenderFragment` without special-casing.
- ✅ Inline opt-in is explicit: a page that does not call `scriptFor` does not get inline scripts.
- ✅ Verbatim emission means any JS pattern is supported: bare functions, IIFE, class bodies, ES module syntax.
- ⚠️ `htmlc` cannot validate JS syntax. Syntax errors surface in the browser console, not at build time.
- ✅ Mixing `scriptFor` and importmap delivery for the same component on the same page is detected at render time and reported as an error.

**Verdict**: expose `scriptFor` as a template function returning the raw script body as `template.HTML`; leave `<script>` tag construction to the page author.

---

### 4.5 Importmap Auto-Injection

When a page uses one or more custom elements, `htmlc` automatically emits a `<script type="importmap">` that maps each element's module specifier to the hashed script URL in `ScriptsFS`. This allows ES module scripts to import from bare specifiers and lets the browser resolve and cache scripts independently.

#### Importmap structure

Each entry maps the custom element's tag name as a bare module specifier to its hashed script URL:

```html
<script type="importmap">
{"imports":{"widgets-counter":"/components/widgets-counter.a1b2c3d4.js","ui-tabs":"/components/ui-tabs.e5f6a7b8.js"}}
</script>
```

The URL is formed by joining the configurable **URL prefix** (default `/components/`) with the hashed filename from `ScriptsFS` (see §4.6).

#### URL prefix configuration

The URL prefix is configurable at engine load time via an option:

```go
// pseudo-code — not implementation
engine, err := htmlc.Load("components/",
    htmlc.WithScriptURLPrefix("/assets/ce/"),
)
```

The default prefix is `/components/`. Callers who serve `ScriptsFS` under a different path (e.g. `/assets/`) set the prefix to match. The prefix must end with `/`.

#### Per-render-pass collection

The importmap is built once per render pass over the set of custom elements actually used on that page. The `CustomElementCollector` (already introduced for per-render deduplication) is reused for this purpose: it tracks which tag names were encountered during the render, and at injection time its entries are used to look up hashed filenames from the engine's `map[tagName]hashedFilename`.

#### Injection point and timing

For **`RenderPage`**: the importmap is injected automatically immediately before `</head>` when at least one custom element was encountered during the render. If `</head>` is absent, injection is skipped and a warning is attached to the render result.

For **`RenderFragment`**: no automatic injection. Authors who need an importmap for a fragment must call an explicit template function:

```html
{{importMap()}}
```

`importMap` is registered as a template function on the engine and emits the importmap for all custom elements used so far in the current render pass.

#### `index.js` loader script

Immediately after the importmap `<script type="importmap">` tag, `RenderPage` injects a module script that loads `index.js`:

```html
<script type="module" src="/components/index.js"></script>
```

Using `type="module"` is the correct approach because:

- ES modules are deferred by default — they do not block HTML parsing.
- `type="module"` scripts execute after the document is parsed, which is appropriate for custom element registration (elements are already in the DOM when `connectedCallback` fires).
- `async` is redundant for modules that have no inline body and no dynamic imports; the browser handles execution ordering automatically.

The bare specifiers in `index.js` (e.g. `import 'widgets-counter'`) are resolved by the importmap to the corresponding hashed URLs. This is why both the `index.js` loader tag and the importmap tag are emitted together — neither is functional without the other.

The `src` value is constructed using `url.JoinPath(scriptURLPrefix, "index.js")`, where `scriptURLPrefix` is the configurable prefix (default `/components/`). Using `url.JoinPath` prevents double-slash issues when the prefix does not end with `/` and correctly handles path escaping — see §4.5 below for the general note on URL construction.

When `RenderPage` injects no importmap (collector is empty), the loader script is also omitted.

#### URL construction

All URL construction in pseudocode uses `net/url` primitives rather than string concatenation. String joining is error-prone: a missing or doubled trailing slash produces an invalid URL that silently misroutes requests. `url.JoinPath` (Go 1.19+) normalises separators and percent-encodes path segments correctly.

#### Nonce support

When a Content Security Policy requires a nonce on inline scripts, the auto-injected `<script type="importmap">` tag must carry a `nonce` attribute. The engine supports this via a `WithNonceFunc` option:

```go
// pseudo-code — not implementation
engine, err := htmlc.Load("components/",
    htmlc.WithNonceFunc(func(ctx context.Context) string {
        return nonceFromContext(ctx) // application-supplied nonce for this request
    }),
)
```

When `WithNonceFunc` is set, `RenderPage` calls it with the render context on each pass and injects the returned value as a `nonce` attribute on both auto-injected script tags (the importmap tag and the module loader tag):

```html
<script type="importmap" nonce="abc123">
{"imports":{"widgets-counter":"/components/widgets-counter.a1b2c3d4.js"}}
</script>
<script type="module" src="/components/index.js" nonce="abc123"></script>
```

If `WithNonceFunc` is not set, no `nonce` attribute is emitted. The `scriptFor` template function returns only the script body (not the tag), so authors supply the nonce themselves on the surrounding `<script nonce="...">` tag — no special engine support is needed for the inline case.

#### Pseudocode

```go
// pseudo-code — not implementation
import (
    "net/url"
)

func (r *Renderer) buildImportMap(ctx context.Context, prefix string) template.HTML {
    if len(r.customElementCollector.entries) == 0 {
        return ""
    }
    type importMapJSON struct {
        Imports map[string]string `json:"imports"`
    }
    m := importMapJSON{Imports: make(map[string]string)}
    for _, e := range r.customElementCollector.entries {
        hashedFile := r.engine.hashedFilename[e.TagName]
        u, _ := url.JoinPath(prefix, hashedFile) // url.JoinPath preferred over string concat
        m.Imports[e.TagName] = u
    }
    raw, _ := json.Marshal(m)

    nonceAttr := ""
    if r.engine.nonceFunc != nil {
        nonceAttr = fmt.Sprintf(` nonce="%s"`, html.EscapeString(r.engine.nonceFunc(ctx)))
    }
    importMapTag := template.HTML(fmt.Sprintf("<script type=\"importmap\"%s>\n%s\n</script>", nonceAttr, raw))
    loaderURL, _ := url.JoinPath(prefix, "index.js")
    loaderTag := template.HTML(fmt.Sprintf("<script type=\"module\" src=\"%s\"%s></script>", loaderURL, nonceAttr))
    return importMapTag + "\n" + loaderTag
}
```

**Evaluation**

- ✅ Zero author burden for `RenderPage`: importmap appears automatically when custom elements are used.
- ✅ Hashed URLs enable aggressive browser caching independent of HTML page caching.
- ✅ Built per render pass from the existing collector — no additional state.
- ✅ Configurable prefix decouples the importmap from the static file serving configuration.
- ⚠️ `RenderFragment` callers must call `{{importMap()}}` explicitly; automatic injection would have no reliable `</head>` anchor.
- ⚠️ Browser support for importmaps is broad (all evergreen browsers as of 2024) but absent in IE11. This is consistent with Custom Elements support requirements and is not a regression.

**Verdict**: inject importmap automatically before `</head>` in `RenderPage`; expose `{{importMap()}}` template function for `RenderFragment`; make URL prefix configurable with default `/components/`.

---

### 4.6 In-Memory Script FS

At engine load time, `htmlc` compiles all custom element scripts into an **in-memory `fs.FS`**. This FS is populated once during startup (when component files are parsed) and is accessible via a method on `Engine`. It provides a stable, request-independent view of all compiled scripts, which the application can use to:

- Serve scripts as static files via `http.FileServer(engine.ScriptsFS())`.
- Write scripts to disk as a build step.
- Embed scripts in `<head>` via the importmap (see §4.5).

#### File naming with content hash

Script files are named with a **short content hash** embedded in the filename for cache busting:

```
<tagName>.<8-char-hex-hash>.js
```

Examples:

```
widgets-counter.a1b2c3d4.js
ui-tabs.e5f6a7b8.js
admin-card.9f3c21aa.js
```

The hash is computed over the **verbatim script body** (not the tag name) using SHA-256 truncated to 4 bytes (8 hex characters). A rename that does not change the script body does not change the hash; a body change always produces a new hash.

#### FS structure

```
widgets-counter.a1b2c3d4.js
ui-tabs.e5f6a7b8.js
admin-card.9f3c21aa.js
admin-date-picker.1c2d3e4f.js
blog-counter.5a6b7c8d.js
index.js
```

#### `index.js` barrel file

In addition to the per-component hashed files, `ScriptsFS` contains a single `index.js` at the root. This file contains one side-effect import per custom element component, using **bare specifiers** (the tag name) so that the importmap is exercised for URL resolution.

**Example generated `index.js`** for a project with three custom element components (`widgets/Counter.vue`, `widgets/Toggle.vue`, `ui/Tabs.vue`):

```js
// generated by htmlc — do not edit
import 'widgets-counter';
import 'widgets-toggle';
import 'ui-tabs';
```

The `// generated by htmlc — do not edit` comment is included to signal that this file should not be modified manually and will be overwritten on the next engine load. The importmap resolves each bare specifier to the corresponding hashed URL (e.g. `widgets-counter` → `/components/widgets-counter.a1b2c3d4.js`). This is why both `index.js` and the importmap are emitted together — `index.js` uses bare specifiers that only the importmap can resolve.

`index.js` is regenerated at engine load time whenever components change. It is intentionally **not** content-hashed because it changes whenever any component's hash changes; it is served with a short `Cache-Control` max-age (see §4.9).

#### Implementation

```go
// pseudo-code — not implementation
import (
    "crypto/sha256"
    "encoding/hex"
    "io/fs"
)

type Engine struct {
    // existing fields ...
    scripts        fs.FS             // in-memory FS populated at load time
    hashedFilename map[string]string // tag name → hashed filename (e.g. "widgets-counter.a1b2c3d4.js")
}

// ScriptsFS returns an fs.FS containing one .js file per component
// that declares <script customelement>. Files are named
// <tagName>.<8-char-hex-hash>.js. The FS is populated at engine
// load time and is safe for concurrent reads.
func (e *Engine) ScriptsFS() fs.FS {
    return e.scripts
}

func contentHash(body string) string {
    sum := sha256.Sum256([]byte(body))
    return hex.EncodeToString(sum[:4]) // 8 hex chars
}

func BuildScriptsFS(components []*Component) (fs.FS, map[string]string, error) {
    // pseudo-code — not implementation
    archive := newInMemoryZip()
    hashedNames := make(map[string]string)
    for _, comp := range components {
        if comp.CustomElementScript == "" {
            continue
        }
        hash := contentHash(comp.CustomElementScript)
        filename := comp.CustomElementTag + "." + hash + ".js"
        archive.WriteFile(filename, []byte(comp.CustomElementScript))
        hashedNames[comp.CustomElementTag] = filename
    }
    // Generate index.js barrel file with one bare-specifier import per component
    var imports []string
    for _, comp := range components {
        if comp.CustomElementScript == "" {
            continue
        }
        imports = append(imports, fmt.Sprintf("import '%s';", comp.CustomElementTag))
    }
    indexBody := strings.Join(imports, "\n") + "\n"
    archive.WriteFile("index.js", []byte(indexBody))
    return archive.FS(), hashedNames, nil
}
```

Callers who need to construct a script URL (e.g. for a `<script src>` tag) should use `fs.ReadDir` or `fs.Glob` to discover filenames rather than constructing them by hand, since the hash component is opaque. The engine's `hashedFilename` map (populated at load time) is the authoritative source for the importmap generator and `scriptFor`.

**Evaluation**

- ✅ `fs.FS` is a standard Go interface — callers can wrap it with `http.FS`, embed it, copy it, or pass it to `io/fs` utilities without depending on `htmlc` internals.
- ✅ Populated at startup — no per-request I/O.
- ✅ Content-addressed filenames enable `Cache-Control: immutable` on script responses.
- ✅ Hash covers script body only: renaming a component subdirectory does not invalidate cached scripts if the body is unchanged.
- ✅ `archive/zip` provides an in-memory `fs.FS`-compatible implementation without requiring a third-party dependency.
- ⚠️ The FS is read-only after load. Reloading requires recreating the engine.

**Verdict**: use an in-memory `archive/zip`-backed `fs.FS` as the compilation output; embed an 8-character SHA-256-derived content hash in each per-component filename; include an unhashed `index.js` barrel file that side-effect-imports all component scripts; expose via `Engine.ScriptsFS()` and `Engine.hashedFilename`.

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

- **`<script customelement src="...">`**: `ParseFile` returns an error:
  ```
  path/to/Component.vue: <script customelement src="..."> is not supported;
  write the script body inline inside the <script customelement> block
  ```
- **`<script customelement>` without `customElements.define`**: `ParseFile` returns an error:
  ```
  path/to/Component.vue: <script customelement> body must contain a customElements.define() call
  ```
  This is a substring check — `htmlc` does not parse JS. Authors whose script calls `customElements.define` via a helper that does not literally contain the substring must include a call-site comment or restructure their registration.

**Verdict**: promote the current silent ignore to a loud compile-time error with an actionable message.

---

### 4.9 HTTP Caching

Content-hashed filenames in `ScriptsFS` are only useful if the application sets appropriate HTTP caching headers. Two files in `ScriptsFS` have different caching semantics:

#### Hashed component files (`<tagName>.<hash>.js`)

These files are **immutable**: their name encodes their content. Serve them with:

```
Cache-Control: public, max-age=31536000, immutable
```

`max-age=31536000` is one year; `immutable` tells the browser the file will never change at this URL so it need not revalidate. When a component's script body changes, its hash changes and a new URL is generated; old cached files become unreachable (no URL points to them).

#### `index.js`

`index.js` changes whenever any component hash changes. It must **not** be cached indefinitely. Serve it with a short max-age, relying on revalidation:

```
Cache-Control: public, max-age=0, must-revalidate
```

Or use `no-cache` for simplicity:

```
Cache-Control: no-cache
```

This ensures browsers always fetch the latest `index.js` on each deployment, while keeping hashed component files cached indefinitely.

#### `NewScriptFSServer` — built-in HTTP handler

`htmlc` provides a convenience constructor that returns an `http.Handler` preconfigured with the correct caching headers, `Content-Type`, and content-encoding negotiation:

```go
// NewScriptFSServer returns an http.Handler that serves ScriptsFS with
// correct Cache-Control headers and Content-Encoding (gzip/br) negotiation.
// Strip the URL prefix before passing requests to this handler, e.g.:
//
//   http.Handle("/components/", http.StripPrefix("/components/", engine.NewScriptFSServer()))
func (e *Engine) NewScriptFSServer() http.Handler
```

The handler applies the following behaviour on each request:

- Parses the request path using `url.Parse` to avoid ambiguity from raw string checks.
- Serves hashed component files (`*.a1b2c3d4.js`) with `Cache-Control: public, max-age=31536000, immutable`.
- Serves `index.js` with `Cache-Control: public, max-age=0, must-revalidate`.
- Sets `Content-Type: text/javascript; charset=utf-8` on all responses.
- Negotiates compressed responses (`Content-Encoding: gzip`, `br`) if pre-compressed variants exist in the FS, or compresses on-the-fly via `compress/gzip`.

**Recommended usage:**

```go
// pseudo-code — not implementation
http.Handle("/components/", http.StripPrefix("/components/", engine.NewScriptFSServer()))
```

#### Manual handler (if you need more control)

Authors who need custom header logic (e.g. additional `Vary` headers, auth checks, or a different compression strategy) can build their own handler. Use `url.Parse` rather than `strings.HasSuffix` to identify `index.js`:

```go
// pseudo-code — not implementation
scriptHandler := http.FileServer(http.FS(engine.ScriptsFS()))
http.Handle("/components/", http.StripPrefix("/components/", withCacheHeaders(scriptHandler)))

func withCacheHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Use url.Parse to avoid ambiguity with raw path string matching
        parsed, err := url.Parse(r.URL.Path)
        if err == nil && path.Base(parsed.Path) == "index.js" {
            w.Header().Set("Cache-Control", "no-cache")
        } else {
            w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
        }
        next.ServeHTTP(w, r)
    })
}
```

`NewScriptFSServer` is the recommended default. The manual pattern is documented for completeness only.

---

### 4.10 Shadow DOM (future)

Shadow DOM is **not implemented in v1**. This section outlines the design for a future opt-in so that reviewers can evaluate feasibility and scope the work.

#### 1. Attachment model

With Shadow DOM, the compiled custom element script would call `attachShadow` in `connectedCallback` rather than operating directly on the element's light-DOM children:

```js
// future example — not v1
connectedCallback() {
    const root = this.attachShadow({ mode: 'open' });
    // ... populate root
}
```

The `mode` would default to `'open'` (inspectable from JS). `'closed'` would be available as a sub-option. The opt-in attribute on the `<script>` block would be `<script customelement shadowdom>` (boolean, open mode) or `<script customelement shadowdom="closed">`.

#### 2. SSR interaction

The server currently renders the component's `<template>` content as **light-DOM children** inside the custom element tag. With Shadow DOM, the rendered HTML would need to be wrapped in a `<template shadowrootmode="open">` element for **Declarative Shadow DOM** (DSD), enabling the browser to attach the shadow root during HTML parsing before JavaScript runs:

```html
<!-- future SSR output with shadowdom opt-in -->
<widgets-counter>
  <template shadowrootmode="open">
    <button>Click me</button>
    <span>0</span>
  </template>
</widgets-counter>
```

DSD is supported in all evergreen browsers as of 2024 (Chrome 90+, Safari 16.4+, Firefox 123+). A progressive-enhancement story would render light DOM as a fallback for older browsers and upgrade to DSD when available.

#### 3. Scoped styles

Today, `<style scoped>` contributions are injected into `<head>` via `StyleCollector`. With Shadow DOM, scoped styles for a shadow-DOM component should instead be injected into the shadow root to achieve true encapsulation. This requires:

- A new `ShadowStyleCollector` (or an extended `StyleCollector`) that distinguishes between head-injected styles and shadow-root-injected styles.
- The SSR output for a shadow-DOM component would include the scoped `<style>` inside the `<template shadowrootmode>` wrapper:

```html
<widgets-counter>
  <template shadowrootmode="open">
    <style>.counter-btn { font-weight: bold; }</style>
    <button class="counter-btn">Click me</button>
    <span>0</span>
  </template>
</widgets-counter>
```

- The existing `StyleCollector` path for light-DOM components is unaffected.

#### 4. Opt-in syntax

The Shadow DOM opt-in is a boolean attribute on `<script customelement>`:

```html
<script customelement shadowdom>
<!-- or, for closed mode: -->
<script customelement shadowdom="closed">
```

This is the same attribute family as `customelement` itself: a boolean attribute (`shadowdom` = open mode) with an optional string value (`shadowdom="closed"` = closed mode). Components without the `shadowdom` attribute continue to use light DOM — this is not a breaking change.

#### 5. Why it is deferred

Shadow DOM is explicitly out of scope for v1 for the following reasons:

- **Complexity**: Declarative Shadow DOM changes the SSR wrapping logic, the style injection pipeline, and requires browsers to support the `<template shadowrootmode>` attribute. These are non-trivial coordinated changes.
- **Declarative Shadow DOM browser support**: While broad in evergreen browsers, DSD is absent in IE11 and was only standardised in 2024. Shipping it without a polyfill story is premature.
- **Scoped-style rework**: The `StyleCollector` refactor to support shadow-root injection is a separate, non-trivial work item.
- **v1 scope**: The primary goal of v1 is light-DOM custom elements with importmap-based script delivery. Shadow DOM can be layered on as a v2 feature without breaking the v1 API.

---

## 5. Syntax Summary

| Block / Function | Attribute / Argument | Meaning in `htmlc` |
|---|---|---|
| `<script customelement>` | *(none)* | Marks component as a custom element. SSR output is wrapped in the derived tag name. Script body is stored verbatim; accessible via `scriptFor` and `ScriptsFS`. Script body must contain `customElements.define`; compile-time error if absent. |
| `<script customelement>` on a component whose derived tag name contains no hyphen | *(n/a)* | **Error at load time**: tag name must contain at least one hyphen |
| `<script>` | *(any)* | **Error**: `<script> blocks are not supported by htmlc` |
| `<script setup>` | `setup` | **Error**: `<script setup> blocks are not supported by htmlc` |
| `<style>` | *(none)* | Global stylesheet contribution; unchanged from today |
| `<style scoped>` | `scoped` | Scoped stylesheet contribution; unchanged from today |
| `<template>` | *(none)* | Server-side render template; unchanged from today |
| `{{scriptFor("path/to/Component")}}` | relative path, no `.vue` extension | Returns the raw script body of the named component as `template.HTML`. Author wraps in `<script>…</script>`. Error if path unknown, component has no `<script customelement>`, or component is already scheduled for importmap delivery on this render pass. |
| `{{importMap()}}` | *(none)* | Emits `<script type="importmap">` for all custom elements used so far in the render pass. Automatic in `RenderPage` (before `</head>`); explicit in `RenderFragment`. |
| `<script customelement src="...">` | `src` | **Error**: `src` attribute not supported on `<script customelement>` |
| `engine.NewScriptFSServer()` | *(none)* | Returns an `http.Handler` that serves `ScriptsFS` with correct `Cache-Control`, `Content-Type`, and compressed-response negotiation. Hashed files get `immutable`; `index.js` gets `no-cache`. |
| `htmlc.WithNonceFunc(f)` | `func(context.Context) string` | Engine option. When set, the returned nonce is injected as `nonce="…"` on both the auto-generated `<script type="importmap">` tag and the `<script type="module" src="…">` loader tag. No effect if not set. |

### Tag name derivation

| File path (relative) | Derived tag name |
|---|---|
| `Counter.vue` | `counter` |
| `DatePicker.vue` | `date-picker` |
| `admin/Card.vue` | `admin-card` |
| `admin/DatePicker.vue` | `admin-date-picker` |
| `ui/form/TextInput.vue` | `ui-form-text-input` |

### ScriptsFS file naming

| Tag name | Example filename |
|---|---|
| `widgets-counter` | `widgets-counter.a1b2c3d4.js` |
| `ui-tabs` | `ui-tabs.e5f6a7b8.js` |
| `admin-card` | `admin-card.9f3c21aa.js` |

Hash is SHA-256 of the script body, truncated to 4 bytes (8 hex chars).

Hashed component files are served with `Cache-Control: immutable`; `index.js` with `Cache-Control: no-cache`. See §4.9.

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

**`Home.vue`** — using importmap (automatic) with external script reference

```html
<template>
  <html>
    <head><title>Home</title></head>
    <body>
      <Counter />
    </body>
  </html>
</template>
```

**Rendered output** — importmap injected automatically before `</head>`

```html
<html>
  <head>
    <title>Home</title>
    <script type="importmap">
{"imports":{"widgets-counter":"/components/widgets-counter.a1b2c3d4.js"}}
</script>
  <script type="module" src="/components/index.js"></script>
  </head>
  <body>
    <widgets-counter>
      <button>Click me</button>
      <span>0</span>
    </widgets-counter>
  </body>
</html>
```

**Alternative — inline script via `scriptFor`**

A page author who prefers inlining the script (e.g. to avoid an extra HTTP request) can use `scriptFor` instead:

```html
<template>
  <html>
    <head><title>Home</title></head>
    <body>
      <Counter />
      <script>{{scriptFor("widgets/Counter")}}</script>
    </body>
  </html>
</template>
```

Rendered output (script section):

```html
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
- The custom element script is referenced via the auto-injected importmap in `<head>`.

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
    </body>
  </html>
</template>
```

**Rendered output — `<head>` with importmap (deduplicated)**

```html
<html>
  <head>
    <title>Dashboard</title>
    <script type="importmap">
{"imports":{"widgets-counter":"/components/widgets-counter.a1b2c3d4.js","widgets-toggle":"/components/widgets-toggle.b2c3d4e5.js"}}
</script>
  <script type="module" src="/components/index.js"></script>
  </head>
  <body>
    <h1>Dashboard</h1>
    <widgets-counter>...</widgets-counter>
    <widgets-counter>...</widgets-counter>
    <widgets-counter>...</widgets-counter>
    <widgets-toggle><input type="checkbox" /><label>Dark mode</label></widgets-toggle>
  </body>
</html>
```

The `CustomElementCollector` tracks `{ "widgets-counter": "widgets/Counter.vue", "widgets-toggle": "widgets/Toggle.vue" }`. The three `<Counter>` renders each call `collector.Add(...)`, but only the first produces an entry. The importmap contains exactly two entries, one per unique element type used on the page.

**Alternative — inline scripts via `scriptFor`**

Authors who prefer inlining (e.g. for a dashboard with a strict no-external-request policy) can use `scriptFor` for each element instead of relying on the importmap:

```html
<script>{{scriptFor("widgets/Counter")}}</script>
<script>{{scriptFor("widgets/Toggle")}}</script>
```

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
- No importmap is injected (collector is empty).
- `ScriptsFS` is empty.
- Output is identical to today.

---

### Example 5 — Serving Scripts via `ScriptsFS`

The application serves compiled custom element scripts as static files, enabling browser caching independent of the HTML page. This example uses the three components from the project in Examples 1–3 (`widgets/Counter.vue`, `widgets/Toggle.vue`, `ui/Tabs.vue`).

**`ScriptsFS` layout** (as described in §4.6):

```
widgets-counter.a1b2c3d4.js
widgets-toggle.b2c3d4e5.js
ui-tabs.e5f6a7b8.js
index.js
```

**Generated `index.js`** (see §4.6 for the full barrel file example):

```js
// generated by htmlc — do not edit
import 'widgets-counter';
import 'widgets-toggle';
import 'ui-tabs';
```

**Serving with `NewScriptFSServer`** (recommended):

```go
// pseudo-code — not implementation
engine, err := htmlc.Load("components/",
    htmlc.WithScriptURLPrefix("/assets/ce/"),
)
if err != nil {
    log.Fatal(err)
}

// NewScriptFSServer sets Cache-Control, Content-Type, and encoding headers automatically.
http.Handle("/assets/ce/", http.StripPrefix("/assets/ce/", engine.NewScriptFSServer()))

// The importmap references hashed URLs automatically:
// {"imports":{"widgets-counter":"/assets/ce/widgets-counter.a1b2c3d4.js"}}
```

To discover the filenames at runtime (e.g. to generate a manifest), use `fs.ReadDir`:

```go
// pseudo-code — not implementation
entries, err := fs.ReadDir(engine.ScriptsFS(), ".")
for _, entry := range entries {
    fmt.Println(entry.Name()) // e.g. "widgets-counter.a1b2c3d4.js"
}
```

Do not construct filenames by hand (e.g. `tagName + ".js"`) — the hash component is opaque and must be read from the FS or the engine's internal map.

---

### Example 6 — Per-Page Opt-In via Wrapper Component

`<script customelement>` is a component-level declaration. If `Button.vue` carries one, every page using `<Button>` emits the script reference via importmap. An author who wants the custom element behaviour on only one page creates a wrapper:

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

### Example 7 — Full `<head>` with Importmap and Scoped Styles

A page using both scoped styles and custom elements, showing the combined `<head>` output.

**`pages/Product.vue`**

```html
<template>
  <html>
    <head>
      <meta charset="UTF-8">
      <title>Product</title>
    </head>
    <body>
      <ui/Tabs>
        <template #overview><p>Overview content</p></template>
      </ui/Tabs>
      <widgets/Counter />
    </body>
  </html>
</template>
```

**Rendered `<head>`**

```html
<head>
  <meta charset="UTF-8">
  <title>Product</title>
  <style>/* scoped styles from ui/Tabs */
.tab-panel[data-v-a1b2c3d4] { display: block; }</style>
  <script type="importmap">
{"imports":{"ui-tabs":"/components/ui-tabs.e5f6a7b8.js","widgets-counter":"/components/widgets-counter.a1b2c3d4.js"}}
</script>
  <script type="module" src="/components/index.js"></script>
</head>
```

The importmap is injected immediately before `</head>`, after any `<style>` contributions injected by `StyleCollector`.

---

## 7. Implementation Sketch

### `component.go`

1. Add two new fields to `Component`: `CustomElementScript string` and `CustomElementTag string`.
2. In `extractSections`, after reading a `<script>` start tag, collect all its attributes into a `map[string]string`.
3. If `attrs["customelement"] != ""` or the attribute name `"customelement"` is present (boolean attribute), store the verbatim body in `sections["script:customelement"]`.
   3a. If `attrs["customelement"] != ""` and `attrs["src"] != ""`, return an error immediately: `<script customelement src="...">` is not supported.
4. In `ParseFile`, populate `CustomElementScript` from `sections["script:customelement"]`.
   4a. After populating `CustomElementScript`, if non-empty and the body does not contain the substring `customElements.define`, return an error as defined in §4.8.
5. Convert the current silent-ignore of `sections["script"]` and `sections["script:setup"]` into explicit error returns with the messages defined in §4.8.

### `customelement.go` (new file)

1. Define `CustomElementEntry` struct with `TagName`, `Script`, `SourcePath string` fields.
2. Define `CustomElementCollector` struct with `seen map[string]string` (tag name → source path) and `entries []CustomElementEntry`.
3. Implement `(c *CustomElementCollector) Add(e CustomElementEntry) error` — no-op if tag already seen from same source; error if tag seen from different source.
4. Add `DeriveTagName(relPath string) string` helper implementing the algorithm in §4.2 (splits on `/`, kebab-cases each PascalCase segment, joins with `-`).
5. Add `contentHash(body string) string` — computes SHA-256 of the body, returns first 4 bytes as 8 hex characters.
6. Add `BuildScriptsFS(components []*Component) (fs.FS, map[string]string, error)` that constructs an in-memory zip-backed `fs.FS` with one entry per component that has `CustomElementScript != ""`, keyed as `<tagName>.<hash>.js`. Returns both the FS and a `map[tagName]hashedFilename` for use by the importmap generator and `scriptFor`.

### `engine.go`

1. Add `scripts fs.FS` and `hashedFilename map[string]string` fields to `Engine`.
2. Add `scriptURLPrefix string` field (set via `WithScriptURLPrefix` option; default `"/components/"`).
3. Add `nonceFunc func(context.Context) string` field (set via `WithNonceFunc` option; nil if not configured).
4. During component loading (in `Load` or equivalent), call `DeriveTagName` for each component and set `component.CustomElementTag`. After calling `DeriveTagName` for each component, if `CustomElementScript != ""` and the derived tag name contains no `-`, return a load-time error: `Counter.vue: custom element tag name "counter" is invalid — tag names must contain at least one hyphen; move the component to a subdirectory (e.g. widgets/Counter.vue → widgets-counter)`.
5. After all components are loaded, call `BuildScriptsFS` and store the result in `Engine.scripts` and `Engine.hashedFilename`.
6. Add `ScriptsFS() fs.FS` public method that returns `Engine.scripts`.
7. Add `NewScriptFSServer() http.Handler` public method: wraps `http.FileServer(http.FS(e.scripts))` with middleware that (a) sets `Content-Type: text/javascript; charset=utf-8`, (b) uses `url.Parse` on the request path to determine whether the file is `index.js` and sets `Cache-Control: public, max-age=0, must-revalidate` for it and `Cache-Control: public, max-age=31536000, immutable` for all other files, and (c) negotiates `Content-Encoding: gzip` / `br` if pre-compressed variants exist or compresses on-the-fly.
8. Register `scriptFor` as a template function: `scriptFor` must close over the per-render `CustomElementCollector` and check it before returning. If the component's tag is already in the collector, return an error as defined in §4.4.
9. Register `importMap` as a template function: delegates to the per-pass renderer's `buildImportMap(ctx, e.scriptURLPrefix)`.
10. In `RenderPage`, after injecting style blocks and before returning, inject the importmap immediately before `</head>` when the collector is non-empty. Pass the render context to `buildImportMap` so the nonce function can be called. Pass `nonceAttr` to both the importmap tag and the module loader tag.
11. In `RenderPage`, immediately after injecting the importmap, inject `<script type="module" src="{prefix}index.js" nonce="..."></script>` before `</head>`, where `prefix` is constructed with `url.JoinPath` and the nonce (if any) is the same value used for the importmap tag.

### `renderer.go`

1. Add `customElementCollector *CustomElementCollector` field to `Renderer`.
2. In `renderElement`, when resolving a component tag: if the resolved `Component.CustomElementScript != ""`:
   a. Call `customElementCollector.Add(CustomElementEntry{...})`.
   b. Wrap the rendered template output in `<tagName>...</tagName>` (see §4.3).
3. Propagate the collector into child `Renderer` instances (same pattern as `styleCollector`).
4. Add `buildImportMap(prefix string) template.HTML` method — iterates `customElementCollector.entries`, maps each tag name to its hashed URL, marshals to JSON importmap format.

### Platform notes

- All file-name manipulation uses `path` (not `path/filepath`) for OS portability, since component paths are relative paths derived from `fs.FS` which always uses forward slashes.
- The `DeriveTagName` function should use `unicode` package functions for PascalCase splitting, not byte-level comparisons, to handle future non-ASCII names gracefully.
- `archive/zip` provides an in-memory `fs.FS` implementation via `zip.NewReader` over a `bytes.Buffer`. No third-party dependency is required.
- `crypto/sha256` and `encoding/hex` are standard library packages; no additional dependency is required for hash computation.

---

## 8. Backward Compatibility

### `Component` struct

New exported fields `CustomElementScript string` and `CustomElementTag string` are added. Backward-compatible in Go: existing code constructing `Component` by field name is unaffected.

### `ParseFile` and `ParseDir`

For components without `<script customelement>`, both functions return the same results as today. The only observable behavioural change is that components with a plain `<script>` or `<script setup>` block — which previously silently stored the body in `Component.Script` (unused) — now return an error. Since those blocks had no effect on rendering, the only affected case is a misconfigured component that was silently broken. The error message is actionable.

### `RenderPage` / `RenderFragment`

No change for components without `<script customelement>`. For components that do use it, `RenderPage` gains automatic SSR wrapping and importmap injection before `</head>`. `RenderFragment` does not auto-inject; callers who need an importmap use `{{importMap()}}` explicitly.

### `FlushCustomElements`

`FlushCustomElements` is not implemented. It was proposed in an earlier draft but never shipped; there are no existing callers. The design is replaced by `scriptFor` (inline opt-in) and automatic importmap injection. No migration path is required.

### `Engine` public API

- New method `ScriptsFS() fs.FS` — additive, no break.
- New method `NewScriptFSServer() http.Handler` — additive, no break. Provides the recommended serving pattern with correct caching headers; authors may continue using `http.FileServer(http.FS(engine.ScriptsFS()))` directly if they need custom header logic.
- New option `htmlc.WithNonceFunc(func(context.Context) string)` — additive, no break. When set, the nonce is applied to both the auto-generated `<script type="importmap">` tag and the `<script type="module" src="…">` loader tag. When not set, behaviour is identical to today (no `nonce` attribute on either tag).
- `scriptFor(path)` and `importMap()` are registered as template functions — additive, no break.
- No existing methods are removed or have their signatures changed.

### Importmap injection

Pages that use custom elements gain a `<script type="importmap">` in `<head>` that was not previously present. This is new behaviour. Pages without custom elements are unaffected — no importmap is injected when the collector is empty.

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

### F. `FlushCustomElements()` for inline script emission

Expose a `FlushCustomElements()` method/template function that emits all used component scripts as inline `<script>` blocks (the original draft design).

**Rejected** because: it conflates two concerns (inline delivery vs. cached delivery) into a single API, forces a placement decision at the call site, and does not integrate with the browser's importmap caching model. `scriptFor` handles the inline case with explicit, per-component control; the importmap handles the external file case automatically.

---

## 10. Open Questions

1. **Shadow DOM opt-in** — *See §4.10* for the full design outline. Shadow DOM is explicitly deferred to a future version; the `shadowdom` attribute is reserved on `<script customelement>`. **Non-blocking** for v1.

2. **`FlushCustomElements` placement** — *Resolved*. The inline flush design is replaced by `scriptFor` (explicit, per-component) and automatic importmap injection (zero author burden for `RenderPage`). No placement decision is required.

3. **`RenderFragment` ergonomics** — *Resolved: no `RenderFragmentWithElements` for v1.* A combined `RenderFragmentWithElements() (html, importmap template.HTML, err error)` API adds public API surface for marginal ergonomic gain; `{{importMap()}}` already covers the case explicitly. The method is deferred — see §3 Non-Goals.

4. **Nonce support for `scriptFor` inline scripts** — *Resolved*. Since `scriptFor` returns only the body (not the tag), authors supply the nonce themselves on the surrounding `<script nonce="...">` tag — no engine support is needed for the inline case. For the auto-injected importmap, the `WithNonceFunc(func(context.Context) string)` engine option is now specified in §4.5 and §5. **Resolved — incorporated into design.**

5. **`ScriptsFS` file naming** — *Resolved*. Hashed filenames (`<tagName>.<8-char-hex-hash>.js`) are included from v1. Content hash uses SHA-256 truncated to 4 bytes. No follow-up needed.

6. **Top-level components and single-word tag names** — *Resolved*: compile-time error (not warning). `htmlc` returns a load-time error when a component with `<script customelement>` derives a tag name with no hyphen (e.g. `Counter.vue` → `counter`). The error message is actionable and directs the author to move the component to a subdirectory. See §4.2. **Resolved.**

7. **CSP and importmap nonce** — *Resolved*. Covered by the `WithNonceFunc` option specified in §4.5 and §5 — the same nonce function is called per render pass and the result is injected as a `nonce` attribute on the auto-generated `<script type="importmap">` tag. **Resolved — incorporated into design.**

---

## 11. Resolved Implementation Concerns

### 11.1 Critical Implementation Gaps (Resolved)

#### a. Importmap entries are never used by the generated scaffolding *(Resolved)*

The proposal injects a `<script type="importmap">` mapping each tag name
(e.g. `"widgets-counter"`) to its hashed URL, and separately loads
`<script type="module" src="/components/index.js">`. The generated `index.js`
barrel file uses **relative** imports:

```js
import "./widgets-counter.a1b2c3d4.js";
```

Relative imports bypass the importmap entirely — the browser resolves them
from the `index.js` URL, not from the importmap registry. The importmap
entries are therefore unreachable through any code the engine itself
generates. They only become useful if an author manually writes
`import "widgets-counter"` (a bare specifier) inside their own `<script
customelement>` body or some other page script. The proposal does not call
this out or provide a worked example of a bare-specifier import.

**Impact**: the importmap is injected on every page that uses custom
elements but serves no function unless the author exploits it manually.
This is dead infrastructure that increases `<head>` size, confuses
future maintainers, and may trigger CSP violations in projects that
whitelist `<script type="importmap">` separately. Either the barrel
`index.js` approach should be dropped in favour of importmap-only loading,
or the `index.js` approach should be the sole mechanism and the importmap
should be removed. The two coexist without coordinating.

- ❌ Two delivery mechanisms (`index.js` barrel + importmap) for the same
  set of scripts, with no clear separation of concerns.
- ❌ `index.js` relative imports never exercise the importmap, making the
  importmap dead code in the default setup.
- ⚠️ Importmap browser support and `<script type="importmap">` CSP
  classification differ across browser versions; injecting it
  unconditionally widens the CSP surface for no gain.

**Verdict**: Resolved — `index.js` uses bare specifiers; importmap is the resolution mechanism. `index.js` imports each component by tag name (e.g. `import 'widgets-counter'`), and the importmap resolves each bare specifier to the corresponding hashed URL. Both tags are necessary and neither is dead code.

#### b. `scriptFor` and importmap can silently double-define custom elements *(Resolved)*

§4.4 notes that authors who use `scriptFor` inline "bypass the importmap"
and "are responsible for not duplicating `customElements.define()` calls."
However, the engine provides no guard. A page that uses `<Counter />` (which
populates the collector and causes the importmap to reference
`widgets-counter.a1b2c3d4.js`) and also calls
`{{scriptFor("widgets/Counter")}}` will cause the browser to execute
`customElements.define('widgets-counter', …)` twice — once from `index.js`
and once inline. The browser throws a `NotSupportedError: already defined`
exception, silently breaking the page.

- ❌ No compile-time or render-time detection of the mix.
- ❌ The error surfaces only at runtime in the browser, far from the
  template that introduced it.
- ⚠️ The two delivery paths are presented as alternatives but nothing
  prevents them from being combined accidentally.

**Verdict**: Resolved — `scriptFor` returns an error if the component is already in the collector. The engine tracks which components have been collected for importmap delivery during the current render pass; calling `scriptFor` for any of them is a render-time error. See §4.4.

#### c. Single-word tag names: blocking open question left unresolved *(Resolved)*

§10 Q6 identifies as **blocking** whether `htmlc` should warn at load time
when a component with `<script customelement>` derives a single-word tag
name (e.g. `Counter.vue` → `counter`). The question is marked blocking but
the proposal provides no resolution — only a tentative recommendation to
warn. A component silently compiled with an invalid custom element tag name
will register without error server-side, produce semantically invalid HTML,
and cause `customElements.define()` to throw a `SyntaxError` in the browser.
This needs a definitive answer before implementation begins.

- ❌ The spec violation is silent at the server and explosive in the
  browser.
- ⚠️ A warning may not be enough: if the tag name cannot be registered, the
  component is functionally broken. An error at load time is safer.

**Verdict**: Resolved — compile-time error. A component with `<script customelement>` whose derived tag name contains no hyphen is rejected at engine load time with an actionable error message. See §4.2 and §10 Q6.

#### d. ES module semantics assumed but not enforced *(Resolved)*

`index.js` loads each component script via a side-effect ES module import.
This works only if the component script does not rely on non-module global
state or use syntax incompatible with strict mode. The proposal says scripts
are emitted **verbatim** with no wrapping or validation. An author who writes
a script using `var` globals, `document.write`, or relies on
`window.onload` will get confusing failures when the script is loaded as an
ES module. There is no guidance on this constraint and no lint step.

- ⚠️ No indication in the SFC block syntax that the script must be
  ES-module-compatible.
- ❌ Syntax errors and module-incompatible patterns are invisible at build
  time.

**Verdict**: Resolved — compile-time substring check for `customElements.define`. `ParseFile` checks that the `<script customelement>` body contains the substring `customElements.define` and returns an error if it does not. See §4.1 and §4.8.

#### e. CSP nonce gap on the module loader `<script>` *(Resolved)*

`WithNonceFunc` covers the auto-injected `<script type="importmap">` tag
but **not** the `<script type="module" src="/components/index.js">` tag
emitted immediately after it. A strict CSP that requires `nonce-*` on all
`<script>` elements will block the module loader silently, with no hint
from `htmlc`. This is an incomplete CSP integration.

- ❌ The nonce is applied to one of two auto-injected tags; the other is
  unprotected.
- ⚠️ Authors relying on `WithNonceFunc` for CSP compliance will receive a
  false sense of security.

**Verdict**: Resolved — nonce applied to both injected tags. `WithNonceFunc` is applied to both the `<script type="importmap">` tag and the `<script type="module" src="…">` loader tag. See §4.5 and §8.

---

### 11.2 Alignment with htmlc's Philosophy

`htmlc` is explicitly described as a **server-side-only renderer** that
produces static HTML with no JavaScript runtime, no reactivity, and no
client-side execution. RFC 006 introduces client-side delivery machinery
(script hashing, HTTP handlers, importmap injection, CSP nonce hooks) that
is architecturally distinct from the server-rendering core.

**Alignment — what fits well**:

- ✅ The `<script customelement>` block is an opt-in: components without it
  behave exactly as today. The "zero impact" guarantee (§2 goal 7) is strong.
- ✅ Verbatim script emission means no new language features, no runtime,
  and no compiler to maintain — consistent with the library's avoidance of
  heavy tooling.
- ✅ `fs.FS` as the compilation output is idiomatic Go and composes cleanly
  with `http.FileServer`, `embed.FS`, and `io/fs` utilities.
- ✅ Tag-name derivation from file paths is deterministic and consistent with
  how `htmlc` already derives component identities from directory structure.
- ✅ Compile-time errors for `<script>` and `<script setup>` (§4.8) improve
  authoring safety without adding runtime complexity.

**Alignment — what fits poorly**:

- ⚠️ `NewScriptFSServer()` adds an `net/http` dependency to what has been a
  pure rendering library. HTTP serving is outside the library's stated scope.
  The library grows from "renders .vue files" to "also manages how scripts are
  served." This couples the rendering engine to deployment topology decisions.
- ⚠️ The importmap injection, content-hash management, and barrel-file
  generation constitute a mini build pipeline embedded in the rendering
  engine. This complexity lives far from the library's rendering core.
- ❌ The proposal introduces a `WithNonceFunc` option that injects per-request
  security material into an engine-level option. The engine is currently
  request-agnostic (stateless after `Load`); `WithNonceFunc` adds a
  request-scoped callback to an otherwise startup-time-configured object.
  This is an impedance mismatch with the current architecture.

**Overall verdict**: the core of RFC 006 — `<script customelement>` parsing,
tag-name derivation, SSR wrapping, and `ScriptsFS` — aligns well with
`htmlc`'s philosophy. The peripheral machinery (`NewScriptFSServer`,
importmap injection, nonce hooks) expands the library into HTTP-serving
territory and should be evaluated carefully. Consider whether
`NewScriptFSServer` belongs in a separate `htmlchttp` package rather than
the core library.

---

### 11.3 Feature Maintenance Burden

#### Is this a "wicked feature"?

A "wicked feature" is one where implementing it makes future development
harder — either because it entangles the codebase, creates long-lived
conceptual debt, or spawns a cascade of follow-on work that can never quite
be closed.

RFC 006 has characteristics of a wicked feature in the following respects:

**Shadow DOM deferral (§4.10)**

The proposal explicitly defers Shadow DOM to a future version, outlining a
design that requires coordinated changes to SSR wrapping, `StyleCollector`,
and the compiled component scripts. Once v1 ships, any author who uses
`<style scoped>` with a custom element will observe that their scoped styles
are stamped onto the custom element's light-DOM children but are *not*
encapsulated. When Shadow DOM arrives as a v2 feature, the SSR output format
changes (the `<template shadowrootmode>` wrapper is inserted), which is a
**breaking change** to the rendered HTML structure of any component that
opts in. Authors must update their JS `connectedCallback` implementations
accordingly. This deferred work will generate a meaningful migration burden.

- ⚠️ Every v1 custom element component that later opts into Shadow DOM
  requires a manual update to its JS body.
- ⚠️ `StyleCollector` will need to understand a new injection path (shadow
  root vs. `<head>`), adding branching logic to an existing well-defined
  component.

**Two delivery paths (importmap + `scriptFor`) create permanent complexity**

The proposal maintains two ways to deliver the same script: importmap
(external, cached) and `scriptFor` (inline). Every future change to script
delivery — compression, sub-resource integrity, preloading — must be
considered for both paths. The existence of two paths also means that
documentation, tutorials, and error messages must address both, doubling
the authoring surface.

**`index.js` cache invalidation on every component change**

The unhashed `index.js` barrel file is invalidated whenever any component
script changes (because the hashed filename of the changed component
changes). In a project with many custom element components, every deployment
regenerates `index.js` and forces clients to re-fetch it, even if they only
use a subset of components. A page-scoped importmap (only the elements used
on that page) would avoid this, but the current design opts for a global
barrel file.

- ⚠️ As the number of custom element components grows, `index.js` becomes
  a progressively larger global invalidation surface.

**Assessment**

RFC 006 is **not fully wicked** but has wicked tendencies in the Shadow DOM
deferral and the dual-delivery-path design. The core feature (verbatim script
emission + SSR wrapping + `ScriptsFS`) is well-bounded and maintainable. The
peripheral concerns (Shadow DOM deferred design, importmap + barrel file
coexistence, HTTP server methods on the engine) are where the long-term
maintenance cost will accumulate.

**Recommendations before implementation**:

1. Resolve the importmap vs. `index.js` tension: pick one primary delivery
   mechanism and remove or clearly relegate the other.
2. Decide now whether `NewScriptFSServer` and importmap injection belong in
   the core `htmlc` package or in a thin `htmlchttp` companion package.
   Deferring this decision makes it harder to refactor later once public API
   surface exists.
3. Resolve Q6 (single-word tag names) as an error, not a warning, before
   implementation begins.
4. Fix the CSP nonce gap: apply `WithNonceFunc` to both auto-injected
   `<script>` tags.
5. Add a guard in the renderer to make the `scriptFor` + importmap
   double-define scenario a render-time error rather than a silent browser
   failure.
