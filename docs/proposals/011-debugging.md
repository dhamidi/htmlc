# RFC 011: Attribute-Based Debug Annotations

- **Status**: Accepted
- **Date**: 2026-03-17
- **Author**: TBD

---

## 1. Motivation

The current debug mode emits `<!-- [htmlc:debug] ... -->` HTML comments to annotate rendered output. This approach has two concrete failure modes that make it both incorrect and misleading.

**Failure mode 1: Nested comment corruption.** When an expression value contains `-->`, the string prematurely closes the outer debug comment. For example, if a component receives a `code` prop containing `<!-- Greeting.vue -->`, the rendered output is:

```html
<!-- [htmlc:debug] expr="code" value=<!-- Greeting.vue -->
```

The `-->` inside the value string terminates the HTML comment. Everything after that point — the remainder of the value and the closing ` -->` — becomes raw HTML text, corrupting the document. The failure is silent: the browser renders the page without error, but the visible text now includes debug annotation fragments.

**Failure mode 2: Wrong DOM position.** Debug comments accumulate at the top of the document rather than surrounding the component subtrees they annotate. This happens because the `debugWriter` writes to the outer writer `w` at the time a component render begins, outside the buffered sub-render. The resulting comment appears wherever the outer writer is positioned at call time — typically before any component output.

**Why escaping is not the fix.** The most obvious alternative is to encode `-->` as something safe, such as `-- >` or a base64 blob. This does not work. The HTML5 specification (§13.1.2) forbids `--` anywhere inside an HTML comment. Browsers are permitted to treat `<!--foo--bar-->` as ending at the first `-->`, which is exactly the parse behaviour that causes the bug. There is no encoding that makes arbitrary string values safe inside HTML comments. The only correct fix is to abandon the HTML-comment format entirely.

---

## 2. Goals

1. **No-corruption guarantee**: debug annotations are valid HTML regardless of expression value content — arbitrary strings, code snippets, HTML fragments, or binary data.
2. **Correct DOM position**: debug information for a component appears on the root element the component renders, not at a different location in the document.
3. **Standard mechanism**: use standard HTML `data-*` attributes; no custom parser, comment format, or post-processing step required.
4. **Zero cost when disabled**: when `Options.Debug` is false, no extra attributes are emitted and no allocation occurs.
5. **Component identity**: the rendered root element carries the component name and source file, enabling DevTools inspection without source-map lookups.
6. **Prop visibility**: the props passed to the component are available as a structured JSON attribute on the root element, enabling round-trip inspection and snapshot testing.

---

## 3. Non-Goals

1. **Expression-level tracing**: individual `{{ expr }}` values are not annotated. The `data-*` attribute approach only applies to element nodes; text nodes produced by expression interpolation cannot carry attributes. Expression tracing is deferred to a future RFC.
2. **v-if skip annotation**: skipped conditional branches emit no element, so there is no element to annotate. This case is deferred.
3. **Slot boundary markers**: slot boundaries are positional concepts that do not map cleanly to a single element. Deferred.
4. **Runtime DevTools integration**: a browser extension or DevTools panel that reads `data-htmlc-*` attributes and presents a component tree is out of scope for this RFC.

---

## 4. Proposed Design

### 4.1 Attribute schema

Three `data-htmlc-*` attributes are placed on the **root element** of the template rendered by a component. "Root element" means the first and only direct child element of `<template>` in the component's `.vue` file.

| Attribute               | Value                                      | Example                            |
|-------------------------|--------------------------------------------|------------------------------------|
| `data-htmlc-component`  | Component name (from registry key)         | `"HeroBanner"`                     |
| `data-htmlc-file`       | Relative path to the `.vue` source file    | `"components/HeroBanner.vue"`      |
| `data-htmlc-props`      | HTML-escaped JSON-encoded props map        | `"{&quot;headline&quot;:&quot;Hello&quot;}"` |

Example output for `<HeroBanner headline="Hello">` with debug mode enabled:

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

### 4.2 Where attributes are injected

**Current state** (`renderer.go`): `renderComponentElement` constructs a child `Renderer` and calls `childRenderer.Render(w, childScope)`. The child renderer walks the template's `<template>` node and emits its root element via `renderElement`.

**Proposed extension**: Before calling `childRenderer.Render`, populate a new `debugAttrs map[string]string` field on the child renderer with the three key-value pairs. Inside `renderElement`, after writing all existing attributes of an element, check whether `r.debugAttrs != nil` AND `n.Parent == r.component.Template` (meaning this is the root element of the component template). If both conditions hold, emit the debug attributes in attribute order and set `r.debugAttrs = nil`. The parent-pointer check is allocation-free and scoped to each child renderer's own `component` pointer, so it is unaffected by slot content rendered at deeper nesting levels.

Two implementation strategies for injecting the attributes:

| Option | Description                                                                                                                       | Verdict |
|--------|-----------------------------------------------------------------------------------------------------------------------------------|---------|
| A      | Add `debugAttrs map[string]string` field to `Renderer`; `renderElement` checks `debugAttrs != nil && n.Parent == r.component.Template`, injects, then sets `debugAttrs = nil`. | ✅ Clean, no structural change to the render loop. One field, one site. |
| B      | Post-process rendered bytes using `html.Tokenizer` to inject attributes after the fact.                                           | ❌ Fragile, doubles allocations, requires special handling for void elements, reintroduces a second parse pass. |

**Verdict**: Option A.

### 4.3 Fragment templates (no element root)

If a component's `<template>` has no element root — only text nodes or multiple sibling elements — there is no single element to carry the debug attributes. Three options:

| Option | Description                                                                                   | Verdict |
|--------|-----------------------------------------------------------------------------------------------|---------|
| A      | Wrap output in `<htmlc-debug>` (a custom element) carrying the attributes.                    | ⚠️ Adds an extra DOM node; may break CSS layout rules that depend on direct-child selectors. |
| B      | Emit `<!-- htmlc-debug component="..." -->` comment only for this specific case.              | ⚠️ Reintroduces comments as a limited fallback; inconsistent with the overall design. |
| C      | Skip annotation for fragment templates; document the limitation with a `TODO` comment.        | ✅ Simplest; fragment templates (multiple root elements or text-only roots) are uncommon in practice. |

**Verdict**: Option C for now. Revisit if fragment templates become frequent enough to warrant a dedicated solution.

### 4.4 Props serialisation

Props cannot be serialised with a direct `json.Marshal(childScope.props)` call because `childScope.props` is a `Props` interface value (`Keys() []string`, `Get(key string) (any, bool)`). The `encoding/json` package has no knowledge of the `Props` interface and would either produce `{}` for an unexported concrete type or panic on an interface value. Serialisation must be performed through the `Props` interface itself, with recursive handling for nested `Props` values.

The serialisation algorithm is:

1. If `props` is nil, emit `data-htmlc-props="{}"`.
2. Otherwise, call `props.Keys()` to enumerate key names, then `props.Get(key)` for each key to retrieve its value.
3. For each value returned by `Get`, check whether it implements `Props`. If it does, recurse into step 1 with that value, producing a nested JSON object (stored as `json.RawMessage`). If it does not, pass the value to `json.Marshal` directly (scalar types, slices, maps, plain structs with exported fields, etc.).
4. Collect all key-value pairs into a `map[string]any`, then call `json.Marshal` on the map to produce the final JSON bytes.

The resulting JSON string is then passed through the existing attribute-value escaping that the renderer applies to all attribute values (which HTML-encodes `"`, `<`, `>`, `&`, and `'`). This means:

- No special-casing of single vs. double quote delimiters is required.
- Values containing `<!--`, `-->`, `"`, or any other HTML-special character are safe.
- The attribute is always wrapped in double quotes by the renderer's existing attribute emitter.

```go
// pseudo-code — not implementation
func propsToJSON(p Props) ([]byte, error) {
    if p == nil {
        return []byte("{}"), nil
    }
    m := make(map[string]any, len(p.Keys()))
    for _, k := range p.Keys() {
        v, _ := p.Get(k)
        if nested, ok := v.(Props); ok {
            raw, err := propsToJSON(nested)
            if err != nil {
                return nil, err
            }
            m[k] = json.RawMessage(raw)
        } else {
            m[k] = v
        }
    }
    return json.Marshal(m)
}
```

If `propsToJSON` returns an error (for example, because a leaf prop value contains a Go channel or an un-marshallable struct), the `data-htmlc-props` attribute is omitted and a `data-htmlc-props-error` attribute is emitted containing the error message string. This keeps the output valid and surfaces the problem without causing a render failure.

### 4.5 `debugWriter` removal

The existing `debug.go` file and `debugWriter` struct become dead code once RFC 011 is implemented. The `debugW *debugWriter` field on `Renderer` is replaced by `debugAttrs map[string]string`. The `exprValue`, `vifSkipped`, `slotStart`, and `slotEnd` methods on `debugWriter` have no equivalent in the new design (per §3) and are deleted. The `withDebug(dw *debugWriter)` builder method on `Renderer` is replaced by population of `debugAttrs` directly in `renderComponentElement`.

### 4.6 Interactive inspector (debug mode)

The interactive inspector is not part of this RFC's implementation scope — it remains a Non-Goal per §3.4. This section documents the intended design so that a future RFC or PR can implement it without re-litigating the architecture.

#### 4.6.1 Injection mechanism

When `Options.Debug` is true and an inspector script path (or inline flag) is configured, the engine appends a `<script>` element as the last child of `<body>`. The script is a self-contained, no-dependency JS module. It must not be injected when `Options.Debug` is false.

#### 4.6.2 Runtime behaviour

**Component tree construction**: on `DOMContentLoaded`, the inspector walks `document.querySelectorAll('[data-htmlc-component]')` and builds a tree by testing containment — an ancestor component contains a descendant if `ancestor.contains(descendant)`. The resulting tree mirrors the server-side component nesting order.

**Overlay panel**: the inspector renders a collapsible `<aside>` panel (injected into `<body>`) listing the component tree. Each entry shows the component name, its source file path, and a collapsed JSON props viewer.

**Hover highlight**: hovering a component entry in the panel adds a CSS outline and a tooltip showing the component name on the corresponding DOM element. Hovering a DOM element that has `data-htmlc-component` highlights the panel entry.

**Props viewer**: clicking a component entry expands an inline `<pre>` showing the JSON-pretty-printed props (via `JSON.stringify(props, null, 2)`). If `data-htmlc-props-error` is present, a red error badge is shown instead.

**Keyboard shortcut**: pressing `Alt+Shift+D` toggles the panel visibility.

#### 4.6.3 Isolation requirements

- The panel is rendered inside a `<shadow-root>` attached to a `<htmlc-inspector>` custom element to avoid CSS leakage from the page's own styles.
- The inspector must not modify any existing DOM nodes — only read `data-htmlc-*` attributes and append its own elements.
- The inspector script must be idempotent: calling it twice (e.g. via HMR reload) must not add a second panel.

#### 4.6.4 Attribute contract

The inspector relies on exactly the three attributes defined in §4.1: `data-htmlc-component`, `data-htmlc-file`, `data-htmlc-props` (and the fallback `data-htmlc-props-error`). No additional attributes are required.

### 4.7 Attribute contract

The three `data-htmlc-*` attributes form an atomic unit: they are always emitted together or not at all. The contract is:

1. **All-or-nothing**: if `debugAttrs != nil` at injection time, all three attributes from the map are emitted. No partial emission.
2. **Encoding**: attribute names are literal ASCII lowercase strings. Attribute values are passed through the renderer's existing HTML attribute-value escaper, which encodes `"`, `<`, `>`, `&`, and `'`. The JSON produced by `json.Marshal` is a valid UTF-8 string; after attribute-value escaping, it is safe inside a double-quoted HTML attribute.
3. **Injection point**: debug attributes are appended after all attributes already present on the root element. Existing attributes are not reordered.
4. **Deterministic order**: the three debug attributes are always emitted in the fixed order `data-htmlc-component`, `data-htmlc-file`, `data-htmlc-props` (or `data-htmlc-props-error`). A package-level `debugAttrOrder []string` slice defines this order.
5. **Single injection**: once the attributes are emitted for a renderer's root element, `r.debugAttrs` is set to `nil`. Nested elements within the same component template do not receive the attributes.

---

## 5. Syntax Summary

No new `.vue` template syntax is introduced. The following HTML attributes appear in rendered output when debug mode is active:

| HTML attribute          | Present when                                        | Value                                      |
|-------------------------|-----------------------------------------------------|--------------------------------------------|
| `data-htmlc-component`  | `Options.Debug` true, component has element root    | Component registry key (original casing)   |
| `data-htmlc-file`       | `Options.Debug` true, component has element root    | Relative path to `.vue` source file        |
| `data-htmlc-props`      | `Options.Debug` true, component has element root, props are JSON-serialisable | HTML-escaped JSON object |
| `data-htmlc-props-error`| `Options.Debug` true, component has element root, props are **not** JSON-serialisable | `json.Marshal` error message |

---

## 6. Examples

### Example 1: Simple component

Template `components/Greeting.vue`:

```html
<template>
  <p>Hello, {{ name }}!</p>
</template>
```

Rendered with `Options.Debug = true` and props `{"name": "world"}`:

```html
<p data-htmlc-component="Greeting" data-htmlc-file="components/Greeting.vue" data-htmlc-props="{&quot;name&quot;:&quot;world&quot;}">Hello, world!</p>
```

No nested comment issues. No position drift. The attribute sits on the element it annotates.

### Example 2: Nested components

`HomePage` renders `<NavBar>` and `<HeroBanner>`:

```html
<div data-htmlc-component="HomePage" data-htmlc-file="pages/HomePage.vue" data-htmlc-props="{}">
  <nav data-htmlc-component="NavBar" data-htmlc-file="components/NavBar.vue" data-htmlc-props="{&quot;links&quot;:[...]}">
    ...
  </nav>
  <section data-htmlc-component="HeroBanner" data-htmlc-file="components/HeroBanner.vue" data-htmlc-props="{&quot;headline&quot;:&quot;Hello&quot;}">
    ...
  </section>
</div>
```

Each component annotates exactly its own root element. Nesting depth is correct and determined by the DOM structure, not by write ordering.

### Example 3: Code snippet as prop value (the original failure case)

`CodeStep` receives `code="<!-- Greeting.vue -->\n<template>..."`:

```html
<div data-htmlc-component="CodeStep"
     data-htmlc-file="components/CodeStep.vue"
     data-htmlc-props="{&quot;code&quot;:&quot;&lt;!-- Greeting.vue --&gt;\n&lt;template&gt;...&quot;}">
  <pre><!-- Greeting.vue -->
&lt;template&gt;...</pre>
</div>
```

The `<!--` and `-->` sequences inside the JSON string value are HTML-escaped and are inert inside an attribute value. The HTML structure is valid. Compare with the previous broken output:

```html
<!-- [htmlc:debug] expr="code" value=<!-- Greeting.vue -->
```

### Example 4: Debug disabled (zero output change)

When `Options.Debug` is false (the default), `debugAttrs` is nil on every renderer. The `renderElement` check `r.debugAttrs != nil && n.Parent == r.component.Template` is false at every call site. No `data-htmlc-*` attributes are emitted. The rendered HTML is byte-for-byte identical to today's non-debug output. No allocations are introduced on the hot path.

### Example 5: Fragment template (limitation)

Template `components/Pair.vue`:

```html
<template>
  <dt>{{ key }}</dt>
  <dd>{{ value }}</dd>
</template>
```

With debug mode enabled, no attributes are injected because there is no single root element. The rendered output is:

```html
<dt>name</dt>
<dd>world</dd>
```

A `// TODO(RFC-011): fragment template debug annotation not supported` comment is placed at the injection site in `renderComponentElement` to mark the limitation.

### Example 6: Accessing debug information in JavaScript

The `data-htmlc-*` attributes are standard HTML `data-*` attributes and are accessible via the browser's `dataset` API. The `dataset` property maps hyphenated attribute names to camelCase property names: `data-htmlc-props` becomes `dataset.htmlcProps`, `data-htmlc-component` becomes `dataset.htmlcComponent`, and so on. `JSON.parse` is safe here because the attribute value is server-emitted JSON — not arbitrary user input at parse time. Component prop values may originate from user data, however, and should not be trusted blindly in production contexts.

**Pattern A — reading a single component's props**

```javascript
// Find the rendered HeroBanner element and read its props
const el = document.querySelector('[data-htmlc-component="HeroBanner"]');
const props = JSON.parse(el.dataset.htmlcProps);
console.log(props.headline); // "Hello"
```

**Pattern B — enumerating all debug-annotated components on the page**

```javascript
// Build a list of every component rendered on the page
const components = [...document.querySelectorAll('[data-htmlc-component]')]
  .map(el => ({
    name:  el.dataset.htmlcComponent,
    file:  el.dataset.htmlcFile,
    props: JSON.parse(el.dataset.htmlcProps ?? '{}'),
    el,
  }));
console.table(components.map(c => ({ name: c.name, file: c.file })));
```

**Pattern C — detecting prop-serialisation errors**

```javascript
// Surface any components whose props could not be serialised
document.querySelectorAll('[data-htmlc-props-error]').forEach(el => {
  console.warn(
    `[htmlc] props serialisation failed for ${el.dataset.htmlcComponent}:`,
    el.dataset.htmlcPropsError
  );
});
```

### Example 7: Non-serialisable prop value

`StreamWidget` receives a prop `reader` of type `io.Reader`, which `encoding/json` cannot marshal:

```html
<div data-htmlc-component="StreamWidget"
     data-htmlc-file="components/StreamWidget.vue"
     data-htmlc-props-error="json: unsupported type: *os.File">
  ...
</div>
```

The `data-htmlc-props` attribute is absent. The `data-htmlc-props-error` attribute surfaces the marshalling failure without aborting the render. The component output itself is unaffected — debug annotations are best-effort and never cause a render failure.

---

## 7. Implementation Sketch

### `debug.go`

Remove the file entirely once the implementation is complete. While the current no-op stub exists (post-silencing), it can remain until the new design is wired in and all call sites are updated.

### `renderer.go`

1. Add one field to `Renderer`:
   - `debugAttrs map[string]string` — nil when debug is disabled or after injection. (~1 field)
2. Remove the `withDebug(dw *debugWriter) *Renderer` builder method. (~3 lines deleted)
3. In `renderComponentElement`, after constructing the child renderer and before calling `childRenderer.Render`, populate `childRenderer.debugAttrs` when `r.debug` is true:
   ```go
   // pseudo-code — not implementation
   if e.opts.Debug {
       propsJSON, err := propsToJSON(childScope.props)
       if err != nil {
           childRenderer.debugAttrs = map[string]string{
               "data-htmlc-component":    comp.Name,
               "data-htmlc-file":         comp.Path,
               "data-htmlc-props-error":  err.Error(),
           }
       } else {
           childRenderer.debugAttrs = map[string]string{
               "data-htmlc-component": comp.Name,
               "data-htmlc-file":      comp.Path,
               "data-htmlc-props":     string(propsJSON),
           }
       }
   }
   ```
   (~12 lines)
4. In `renderElement`, after writing all existing attributes of the element's opening tag, add:
   ```go
   // pseudo-code — not implementation
   if r.debugAttrs != nil && n.Parent == r.component.Template {
       for _, key := range debugAttrOrder {
           if val, ok := r.debugAttrs[key]; ok {
               writeAttr(w, key, html.EscapeString(val))
           }
       }
       r.debugAttrs = nil
   }
   ```
   (~7 lines; `debugAttrOrder` is a package-level `[]string` defining deterministic attribute output order)
5. Remove all `if r.debug { r.debugW.exprValue(...) }`, `r.debugW.vifSkipped(...)`, `r.debugW.slotStart(...)`, `r.debugW.slotEnd(...)` call sites. (~8 deletions)

### `engine.go`

1. Remove the call to `withDebug(newDebugWriter(w))`. The `debugAttrs` field is now populated per component render in `renderComponentElement` when `e.opts.Debug` is true. (~2 lines changed)

### `doc.go`

1. Update the `Debug` field documentation to describe attribute injection rather than HTML-comment emission.
2. Update `SetDebug` documentation to match.

### `README.md`

1. Replace the Debug Mode section with the new attribute schema table and an example of `data-htmlc-*` output.
2. Remove or soften the "never enable in production" warning — attribute annotations are valid HTML and do not corrupt the document. Replace with a note that debug mode adds extra attributes and increases output size, so it should not be used in production for performance reasons.

---

## 8. Backward Compatibility

### `Options.Debug` field

Unchanged. The field exists, is accepted, and now controls attribute injection instead of comment injection. No API break.

### `SetDebug(bool)` method

Unchanged signature and semantics (enables/disables debug mode at the engine level).

### Rendered HTML output — non-debug mode

Byte-for-byte identical to today. The `debugAttrs == nil` fast path adds no output and no allocation.

### Rendered HTML output — debug mode

**Breaking change for debug-mode consumers.** The output format changes from `<!-- [htmlc:debug] ... -->` HTML comments to `data-htmlc-*` attributes on component root elements. Any tooling that parses `<!-- [htmlc:debug] -->` comments must be updated to read `data-htmlc-*` attributes instead.

This break is acceptable for two reasons: (1) the comment format was never documented as stable public API, and (2) the comment format is currently emitting structurally invalid HTML that corrupts document parsing. Preserving the broken format is not an option.

### `debugWriter` type (unexported)

Removed. This is an unexported type; there is no public API break.

---

## 9. Alternatives Considered

### A. Fix HTML-comment escaping

Replace `-->` with `-- >` or encode values in base64 inside the comment.

**Rejected**: The HTML5 specification (§13.1.2) forbids `--` inside HTML comments. A browser is permitted to parse `<!--foo--bar-->` as ending at the first `-->`, which is the exact behaviour causing the bug. Base64 encoding would prevent corruption but makes the output unreadable in DevTools and does not address the position problem.

### B. Keep HTML comments, fix position by threading the writer correctly

Buffer the entire rendered subtree, then inject comments adjacent to each component's output using an `html.Tokenizer` pass over the buffer.

**Rejected**: Two-pass rendering doubles allocations and latency. The architecture change required to thread the writer correctly is more invasive than the `debugAttrs` field approach. Once the writer is correctly threaded, `data-*` attributes are strictly simpler than comments and do not require a tokeniser pass.

### C. Use `<script type="application/json+htmlc-debug">` blocks

Emit a `<script>` element immediately after each component's root element containing a JSON summary of the component render.

**Rejected**: `<script>` elements are real DOM nodes. They affect `querySelector`, `children`, `childElementCount`, and CSS sibling selectors. They are also parsed and potentially executed by JavaScript runtimes that do not recognise the MIME type. `data-*` attributes on the existing element are the standard HTML mechanism for per-element metadata and are invisible to layout.

### D. Use a `<template>` wrapper element

Wrap each component's output in `<template data-htmlc-component="...">`.

**Rejected**: `<template>` elements in HTML are inert. Their children live in a document fragment detached from the live DOM. CSS selectors, JavaScript queries, and browser DevTools treat `<template>` content differently from normal elements. The debug attributes would not be inspectable via standard DevTools element inspection.

---

## 10. Resolved Design Decisions

All design questions were resolved during RFC authoring. The decisions are recorded here for traceability.

1. **Props serialisation of non-JSON-serialisable values**
   If a prop value is an `io.Reader`, a Go channel, or any type that `encoding/json` cannot marshal, `json.Marshal` returns an error. **Decision**: omit `data-htmlc-props` and emit `data-htmlc-props-error` containing the error string. The render is never failed for a debug-only annotation.

2. **Root-element detection and slot rendering**
   The `templateDepth int` counter originally proposed for this purpose has been removed. The root element is identified by the parent-pointer check `n.Parent == r.component.Template`. This check is scoped to each child renderer's own `component` pointer and is unaffected by slot content rendered at deeper nesting levels — slot content is walked by its own renderer whose `component` points to the slot's defining component, not the host. No counter to maintain; no zeroing required on renderer construction.

3. **Attribute order**
   **Decision**: `data-htmlc-*` attributes are appended after all attributes already present on the root element. This minimises diff noise when comparing debug vs. non-debug output.

4. **`data-htmlc-component` casing**
   **Decision**: use the component registry key with its original casing (e.g., `"HeroBanner"`). The lowercase tag name is already visible in the element tag itself; the registry key provides the canonical identifier used throughout the codebase.

5. **Interaction with scoped styles**
   **Decision**: `data-htmlc-*` attributes are injected after any existing `data-v-*` scope attribute, consistent with the "inject last" rule from decision 3. Verify during implementation that snapshot output is stable across both code paths.
