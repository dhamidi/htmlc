# htmlc CLI

`htmlc` renders Vue Single File Components (`.vue`) to HTML entirely in Go — no Node.js, no browser, no JavaScript runtime.

This document covers the command-line interface.
For template syntax, directives, the Go API, and the expression language, see the [repository README](../../README.md).

---

## Table of contents

1. [Tutorial — render your first component](#tutorial--render-your-first-component)
2. [How-to guides](#how-to-guides)
   - [Render a component fragment](#render-a-component-fragment)
   - [Render a full HTML page](#render-a-full-html-page)
   - [Wrap a page in a layout using a slot (manual composition)](#wrap-a-page-in-a-layout-using-a-slot-manual-composition)
   - [Use the `-layout` flag to apply a layout at render time](#use-the--layout-flag-to-apply-a-layout-at-render-time)
   - [Build a static site from a page tree](#build-a-static-site-from-a-page-tree)
   - [Apply a shared layout across all pages in a build](#apply-a-shared-layout-across-all-pages-in-a-build)
   - [Pass props to a component](#pass-props-to-a-component)
   - [Pipe props from stdin](#pipe-props-from-stdin)
   - [Inspect a component's props](#inspect-a-components-props)
   - [Export props as shell variables](#export-props-as-shell-variables)
   - [Debug a rendering problem](#debug-a-rendering-problem)
   - [Inspect the parsed template AST](#inspect-the-parsed-template-ast)
   - [Use components from a different directory](#use-components-from-a-different-directory)
   - [Add an external directive to the build](#add-an-external-directive-to-the-build)
3. [Reference](#reference)
   - [Subcommands at a glance](#subcommands-at-a-glance)
   - [render](#render)
   - [page](#page)
   - [build](#build)
   - [External directives](#external-directives)
   - [props](#props)
   - [ast](#ast)
   - [help](#help)
4. [Explanation](#explanation)
   - [Fragment vs page rendering](#fragment-vs-page-rendering)
   - [Component name resolution](#component-name-resolution)
   - [Scoped styles in CLI output](#scoped-styles-in-cli-output)
   - [Debug mode](#debug-mode)
   - [Page-centric build](#page-centric-build)
5. [External Directives](#external-directives)

---

## Tutorial — render your first component

This walkthrough takes you from zero to a rendered HTML snippet in under five minutes.

**Prerequisites:** Go 1.21+ installed; `htmlc` built or installed via `go install github.com/dhamidi/htmlc/cmd/htmlc@latest`.

**1. Create a component file.**

```vue
<!-- templates/Greeting.vue -->
<template>
  <p>Hello, {{ name }}!</p>
</template>
```

**2. Render it.**

```
$ htmlc render -dir ./templates Greeting -props '{"name":"world"}'
<p>Hello, world!</p>
```

That is the complete output — a plain HTML fragment written to stdout.

**3. Render a full page.**

```vue
<!-- templates/HomePage.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body><h1>{{ title }}</h1></body>
  </html>
</template>
```

```
$ htmlc page -dir ./templates HomePage -props '{"title":"My site"}'
<!DOCTYPE html>
<html>
  <head><title>My site</title></head>
  <body><h1>My site</h1></body>
</html>
```

**4. Build a whole site.**

```
htmlc build -dir ./templates -pages ./pages -out ./dist
```

Every `.vue` file in `pages/` is rendered to a matching `.html` file in `dist/`.  Props come from sibling `.json` files.

**5. Discover what props a component expects.**

```
$ htmlc props -dir ./templates Greeting
name
```

You now know the four most-used subcommands: `render`, `page`, `build`, and `props`.

---

## How-to guides

### Render a component fragment

Use `render` when you want an HTML fragment suitable for embedding inside a larger page — for example, a partial returned by an HTMX endpoint.

```
htmlc render -dir ./templates Card -props '{"title":"Intro","body":"Hello"}'
```

Scoped styles (if any) are prepended as a `<style>` block before the element output.

---

### Render a full HTML page

Use `page` when the component represents a complete document.
The output always starts with `<!DOCTYPE html>`.
Any scoped styles are injected into the document `<head>` automatically.

```
htmlc page -dir ./templates PostPage -props '{"slug":"intro","title":"Introduction"}'
```

---

### Wrap a page in a layout using a slot (manual composition)

In this pattern the page component itself references the layout component directly in its template and passes its content through named slots.  No extra CLI flag is needed.

```vue
<!-- templates/AppLayout.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body>
      <header><slot name="header" /></header>
      <main><slot /></main>
      <footer><slot name="footer" /></footer>
    </body>
  </html>
</template>
```

```vue
<!-- templates/PostPage.vue -->
<template>
  <AppLayout :title="title">
    <template #header>
      <nav><a href="/">Home</a></nav>
    </template>

    <article>{{ body }}</article>

    <template #footer>
      <p>&copy; 2024 My Site</p>
    </template>
  </AppLayout>
</template>
```

```
htmlc page -dir ./templates PostPage -props '{"title":"Hello","body":"World"}'
```

Named slots let different pages supply different header and footer content while sharing the same outer structure.  Use this approach when the layout is an inherent part of the component's design, or when pages need to customise individual regions independently.

---

### Use the `-layout` flag to apply a layout at render time

With `-layout` the page component does not need to know about the layout at all.  `htmlc` renders the page as a fragment, then passes the resulting HTML to the layout component as a `content` prop.

Named slots can still structure the layout's static regions (header, footer) while the injected page HTML occupies the main area via `v-html="content"`.

```vue
<!-- templates/AppLayout.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body>
      <header>
        <slot name="header"><nav><a href="/">Home</a></nav></slot>
      </header>
      <main v-html="content"></main>
      <footer>
        <slot name="footer"><p>&copy; 2024 My Site</p></slot>
      </footer>
    </body>
  </html>
</template>
```

```vue
<!-- templates/PostPage.vue -->
<template>
  <article>{{ body }}</article>
</template>
```

```
htmlc page -dir ./templates -layout AppLayout PostPage \
  -props '{"title":"Hello","body":"World"}'
```

The layout receives:

- `content` — the rendered HTML of the page component, injected into the `<main>` element via `v-html`.
- all top-level props from `-props` (e.g. `title`) so the layout can use them directly.
- named slots (`header`, `footer`) fall back to their default content because the CLI does not supply slot children when applying a layout.

Use this approach when the layout is a deployment-time concern (a shared shell applied to every page) and you want page components to remain independent of it.

---

### Build a static site from a page tree

Create a `pages/` directory with `.vue` page components and optional `.json` data files:

```
pages/
  index.vue
  index.json
  about.vue
```

```json
// pages/index.json
{"title": "Home", "body": "Welcome!"}
```

```vue
<!-- pages/index.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body><h1>{{ title }}</h1><p>{{ body }}</p></body>
  </html>
</template>
```

Run the build:

```
$ htmlc build -dir ./templates -pages ./pages -out ./dist
Build complete: 2 pages, 0 errors.
```

The output directory mirrors the page tree:

```
dist/
  index.html
  about.html
```

Spot-check the result:

```
$ grep -i title dist/index.html
    <title>Home</title>
```

Files whose base name starts with `_` are skipped.  Use `_partial.vue` or `_data.json` for shared fragments and default data.

---

### Apply a shared layout across all pages in a build

Create a layout component in your shared components directory that uses `v-html="content"` to inject the page HTML:

```vue
<!-- templates/AppLayout.vue -->
<template>
  <html>
    <head><title>{{ title }}</title></head>
    <body>
      <header><nav><a href="/">Home</a></nav></header>
      <main v-html="content"></main>
      <footer><p>&copy; 2024 My Site</p></footer>
    </body>
  </html>
</template>
```

Page components do not reference the layout at all:

```vue
<!-- pages/index.vue -->
<template>
  <article><h1>{{ title }}</h1><p>{{ body }}</p></article>
</template>
```

Run the build with `-layout`:

```
$ htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout
Build complete: 2 pages, 0 errors.
```

Every output file is wrapped in the layout shell.

**Data flow:** page props (loaded from `.json` files) are available in both the page template and the layout template.  The `content` key is the only injected value — it holds the rendered HTML of the page component.  If a page data file contains a key named `content`, it will be overridden by the injected HTML.

---

### Pass props to a component

Props are provided as a JSON object with `-props`.

```
htmlc render -dir ./templates Button -props '{"label":"Click me","disabled":true}'
```

Flags may appear before or after the component name — both orderings are accepted:

```
# equivalent
htmlc render -dir ./templates -props '{"label":"OK"}' Button
htmlc render Button -dir ./templates -props '{"label":"OK"}'
```

---

### Pipe props from stdin

Pass `-props -` to read the JSON object from standard input.

```
echo '{"post":{"title":"Hello","body":"World"}}' \
  | htmlc render -dir ./templates PostCard -props -
```

This is useful when props are generated by another program or when the JSON is too large for a shell argument.

---

### Inspect a component's props

`props` lists every prop name declared in a component.

```
# plain text (one prop per line, sorted)
htmlc props -dir ./templates Card

# JSON — includes component name and full prop definitions
htmlc props -dir ./templates Card -format json

# shell-variable format (SCREAMING_SNAKE_CASE)
htmlc props -dir ./templates Card -format env
```

You can also pass a file path directly instead of a component name:

```
htmlc props ./templates/Card.vue
```

---

### Export props as shell variables

The `env` format prints each prop as `PROP_NAME=` lines suitable for sourcing or inspection in a shell script.

```
$ htmlc props -dir ./templates PostCard -format env
POST=
AUTHOR=
```

camelCase prop names are converted to SCREAMING_SNAKE_CASE automatically (`authorName` → `AUTHOR_NAME`).

---

### Debug a rendering problem

Add `-debug` to `render` or `page`.  Each component's root element gains three `data-htmlc-*` attributes carrying the component name, source file, and props.

```
htmlc render -debug -dir ./templates Card -props '{"title":"Test"}'
```

Example annotated output:

```html
<div data-htmlc-component="Card"
     data-htmlc-file="templates/Card.vue"
     data-htmlc-props="{&quot;title&quot;:&quot;Test&quot;}">
  ...
</div>
```

The attributes are standard HTML `data-*` attributes and are visible in browser DevTools.  Inspect all rendered components with JavaScript:

```javascript
document.querySelectorAll('[data-htmlc-component]').forEach(el => {
  console.log(el.dataset.htmlcComponent, JSON.parse(el.dataset.htmlcProps));
});
```

Avoid `-debug` in production — the attributes increase output size and expose prop values.

---

### Inspect the parsed template AST

`ast` prints the component's template as an indented pseudo-XML tree.  Use it to verify that the parser sees your template the way you expect.

```
htmlc ast -dir ./templates Card
```

No rendering happens; the output shows nodes, directives, and attributes as the engine parsed them.

---

### Use components from a different directory

All subcommands accept `-dir` to set the component search path.  The default is the current working directory.

```
htmlc render -dir /var/www/templates/components Hero -props '{}'
```

---

### Add an external directive to the build

Use external directives to transform specific elements during a `build` run
without modifying `htmlc` itself — for example, to apply syntax highlighting,
generate table-of-contents entries, or call an external tool.

**Prerequisites:** `htmlc build` is working for your project.

---

**Step 1 — Create a directive executable.**

Place an executable file whose base name (without extension) matches
`v-<directive-name>` anywhere inside the component directory (`-dir`).
The directive name must be lower-kebab-case.

```
templates/
  directives/
    v-syntax-highlight        ← registered as "syntax-highlight"
```

A minimal shell skeleton (replace the highlighted logic with your own):

```bash
#!/usr/bin/env bash
set -euo pipefail
while IFS= read -r line; do
    hook=$(printf '%s' "$line" | python3 -c "import sys,json; print(json.load(sys.stdin)['hook'])")
    id=$(printf '%s'   "$line" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
    if [ "$hook" = "created" ]; then
        printf '{"id":"%s","inner_html":"<b>replaced</b>"}\n' "$id"
    else
        printf '{"id":"%s","html":""}\n' "$id"
    fi
done
```

A minimal Node.js skeleton:

```js
#!/usr/bin/env node
const rl = require('readline').createInterface({ input: process.stdin, terminal: false });
rl.on('line', line => {
    const req = JSON.parse(line);
    if (req.hook === 'created') {
        process.stdout.write(JSON.stringify({
            id: req.id,
            inner_html: `<b>${req.text}</b>`,
        }) + '\n');
    } else {
        process.stdout.write(JSON.stringify({ id: req.id }) + '\n');
    }
});
```

Make the file executable:

```
chmod +x templates/directives/v-syntax-highlight
```

---

**Step 2 — Reference the directive in a template.**

Use the directive as a Vue-style attribute on any element:

```vue
<template>
  <pre v-syntax-highlight="'go'">
func main() {
    fmt.Println("hello")
}
  </pre>
</template>
```

The expression (`'go'`) becomes `binding.value` in the request JSON.

---

**Step 3 — Run the build.**

```
htmlc build -dir ./templates -pages ./pages -out ./dist
```

`htmlc` discovers, starts, and communicates with the directive automatically.
Any text written by the directive to stderr is forwarded to the terminal.

---

**Step 4 — Verify the output.**

```
grep -A3 'syntax-highlight' dist/index.html
```

---

**Troubleshooting**

| Symptom | Likely cause |
|---------|--------------|
| Directive is silently skipped | File is not executable (`chmod +x`) |
| Directive is not discovered | Name does not match `v-[a-z][a-z0-9-]*` |
| Warning: `invalid JSON from directive` | Response line is malformed; check stderr |
| Element unchanged | Directive returned empty `inner_html` or `html` |

---

## Reference

### Subcommands at a glance

| Subcommand | Output | Use when |
|---|---|---|
| `render` | HTML fragment | Embedding a component inside a larger page |
| `page` | Full HTML document | Component represents a complete page |
| `build` | HTML files on disk | Rendering an entire pages directory tree |
| `props` | Prop list (text / JSON / env) | Discovering what data a component needs |
| `ast` | Indented parse tree | Debugging parser behaviour |
| `help` | Help text | Learning subcommand flags |

---

### render

```
htmlc render [-dir <path>] [-props <json|->] [-debug] <component>
```

| Flag | Default | Description |
|---|---|---|
| `-dir` | `.` | Directory that contains `.vue` files |
| `-props` | _(empty)_ | Props as a JSON object literal, or `-` to read from stdin |
| `-debug` | false | Annotate output with `data-htmlc-*` attributes on each component's root element |

Exits non-zero and prints an error to stderr when the component is not found, props JSON is malformed, or rendering fails.

---

### page

```
htmlc page [-dir <path>] [-props <json|->] [-debug] [-layout <component>] <component>
```

| Flag | Default | Description |
|---|---|---|
| `-dir` | `.` | Directory that contains `.vue` files |
| `-props` | _(empty)_ | Props as a JSON object literal, or `-` to read from stdin |
| `-debug` | false | Annotate output with `data-htmlc-*` attributes on each component's root element |
| `-layout` | _(none)_ | Wrap the rendered page inside this layout component. The layout receives the rendered HTML as a `content` prop. |

Differences from `render`:

- Output begins with `<!DOCTYPE html>`.
- Scoped styles are moved into the document `<head>`.

When `-layout` is given, the page component is rendered as a fragment first, then its HTML is passed to the layout as the `content` prop.  All top-level `-props` values are forwarded to both renders.

---

### build

```
htmlc build [-dir <path>] [-pages <path>] [-out <path>] [-layout <name>] [-debug]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | `.` | Shared component directory |
| `-pages` | `./pages` | Page tree root |
| `-out` | `./out` | Output directory (created if absent) |
| `-layout` | _(none)_ | Layout component to wrap every page |
| `-debug` | false | Annotate output with `data-htmlc-*` attributes on each component's root element |

Files in the pages directory whose base name starts with `_` are skipped — they are treated as shared partials, not pages.

Props for each page are loaded by shallow-merging JSON data files in ascending directory order:

1. `pages/_data.json` — root-level defaults inherited by every page.
2. `pages/subdir/_data.json` — sub-directory defaults (one per directory level).
3. `pages/subdir/hello.json` — page-level props with the same base name as the `.vue` file (highest priority).

Missing files are silently skipped.  A page-level key always overrides the same key from an ancestor `_data.json`.  If no data files exist the page is rendered with no props.

The directory hierarchy is preserved: `pages/posts/hello.vue` becomes `out/posts/hello.html`.

```
pages/
  _data.json         ← defaults for all pages
  index.vue          → out/index.html
  index.json         ← props for index.vue (merged on top of _data.json)
  about.vue          → out/about.html
  posts/
    _data.json       ← defaults for pages under posts/
    hello.vue        → out/posts/hello.html
    hello.json       ← props for posts/hello.vue
```

**Progress and summary output**

When stdout is a terminal, `build` prints one status line per page:

```
  built  posts/hello.html
  ERROR  posts/broken.html  (reason)
```

A summary is always printed:

```
Build complete: 5 pages, 0 errors.
```

**Exit code**

`build` exits `0` when all pages are rendered successfully.  It exits `1` when one or more pages fail (parse error, data error, or render error).  Failed pages do not abort the build — all remaining pages are still attempted.

**Output directory**

The `-out` directory is created automatically with `mkdir -p` semantics if it does not already exist.  Intermediate subdirectories for nested pages are also created as needed.

---

### External directives

External directives extend `htmlc build` with custom element transformations.
They are independent executables that communicate with the build via
newline-delimited JSON (NDJSON) over stdin/stdout.

---

#### Discovery

During `build`, `htmlc` walks the entire component directory tree (`-dir`)
and registers every file that satisfies all three conditions:

| Condition | Rule |
|-----------|------|
| Name | Base name without extension matches `v-<directive-name>` |
| Directive name format | Lower-kebab-case: `[a-z][a-z0-9-]*` |
| Executable | File mode has at least one executable bit set (`mode & 0111 != 0`) |

Hidden directories (names starting with `.`) are skipped.  Extensions are
ignored, so `v-foo`, `v-foo.sh`, and `v-foo.py` all register as `foo`.
If multiple files resolve to the same directive name, the last one found
in the directory walk wins.

**Examples of valid file names**

```
v-syntax-highlight      → directive name: syntax-highlight
v-upper.sh              → directive name: upper
v-toc-builder.py        → directive name: toc-builder
```

**Examples of invalid file names** (not registered)

```
syntax-highlight        ← missing "v-" prefix
v-Syntax-Highlight      ← uppercase letters not allowed
v-123                   ← must start with a letter
```

---

#### Lifecycle

Each discovered directive is started **once** at the beginning of `build`
and stopped (stdin closed, process awaited) at the end.  If a directive
fails to start, a warning is printed and the build continues without it.

A non-zero exit code from the directive process is treated as a warning;
the build is not aborted.

---

#### Protocol

Communication is newline-delimited JSON (NDJSON): one JSON object per line,
with no pretty-printing.  Requests flow from `htmlc` to the directive on
stdin; responses flow from the directive to `htmlc` on stdout.  Requests are
sent sequentially — `htmlc` sends the next request only after receiving a
valid response for the current one.

The directive's stderr is forwarded verbatim to `htmlc`'s stderr.

---

#### Request envelope

Sent by `htmlc` for every element that carries the directive's attribute.

```json
{
  "hook":    "created" | "mounted",
  "id":      "<opaque string>",
  "tag":     "<element tag name>",
  "attrs":   { "<name>": "<value>", ... },
  "text":    "<concatenated text content of all child text nodes>",
  "binding": {
    "value":     <evaluated expression>,
    "raw_expr":  "<unevaluated expression string>",
    "arg":       "<directive argument, or empty string>",
    "modifiers": { "<modifier>": true, ... }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `hook` | string | `"created"` (before render) or `"mounted"` (after closing tag) |
| `id` | string | Opaque unique identifier; **must** be echoed in the response |
| `tag` | string | HTML element tag name, e.g. `"pre"`, `"div"` |
| `attrs` | object | All HTML attributes present on the element |
| `text` | string | Concatenated text content of all descendant text nodes |
| `binding.value` | any | Result of evaluating the directive expression |
| `binding.raw_expr` | string | Unevaluated expression string as written in the template |
| `binding.arg` | string | Directive argument after `:` (e.g. `"href"` from `v-bind:href`), or `""` |
| `binding.modifiers` | object | Modifier flags (e.g. `{"prevent": true}` from `.prevent`) |

---

#### `created` hook response

Called **before** the element is rendered.  The response may mutate the
element's tag, attributes, or inner content.

```json
{
  "id":         "<same id as request>",
  "tag":        "<optional: replacement tag name>",
  "attrs":      { "<name>": "<value>", ... },
  "inner_html": "<optional: verbatim HTML to use as element content>",
  "error":      "<optional: non-empty string aborts rendering of this element>"
}
```

| Field | Type | Effect |
|-------|------|--------|
| `id` | string | **Required.** Must match the request `id`; mismatches are ignored with a warning. |
| `tag` | string | If non-empty, replaces the element's tag name. |
| `attrs` | object | If present, replaces all element attributes with this map. |
| `inner_html` | string | If non-empty, replaces the element's children with this HTML verbatim (not escaped). Template children are discarded. |
| `error` | string | If non-empty, aborts rendering of this element and logs the message. |

When `inner_html` is provided, it takes precedence over `v-text`, `v-html`,
and all template children.

---

#### `mounted` hook response

Called **after** the element's closing tag has been written.

```json
{
  "id":    "<same id as request>",
  "html":  "<optional: HTML injected immediately after the closing tag>",
  "error": "<optional: non-empty string aborts rendering>"
}
```

| Field | Type | Effect |
|-------|------|--------|
| `id` | string | **Required.** Must match the request `id`. |
| `html` | string | If non-empty, this HTML is written verbatim after the closing tag. |
| `error` | string | If non-empty, aborts rendering and logs the message. |

---

#### Error handling summary

| Situation | Behaviour |
|-----------|-----------|
| Response is not valid JSON | Warning printed to stderr; request treated as no-op |
| Response `id` does not match request `id` | Warning printed to stderr; request treated as no-op |
| `error` field is non-empty | Element rendering aborted; error logged |
| Directive fails to start | Warning printed; build continues without that directive |
| Directive exits non-zero | Warning printed; does not abort the build |

---

### props

```
htmlc props [-dir <path>] [-format <fmt>] <component>
```

| Flag | Default | Accepted values |
|---|---|---|
| `-dir` | `.` | Path to component directory |
| `-format` | `text` | `text`, `json`, `env` |

`<component>` may be a bare name (`Card`) or a file path (`./templates/Card.vue`).

**text** output — one prop name per line, sorted alphabetically.

**json** output structure:

```json
{
  "component": "Card",
  "props": [
    {"name": "title", "expression": "\"\""},
    {"name": "count", "expression": "0"}
  ]
}
```

**env** output — one assignment per line in `NAME=` format, suitable for `export` or `env(1)`.

---

### ast

```
htmlc ast [-dir <path>] <component>
```

| Flag | Default | Description |
|---|---|---|
| `-dir` | `.` | Directory that contains `.vue` files |

Prints a human-readable tree of the parsed template.  No rendering is performed.

---

### help

```
htmlc help [subcommand]
```

Prints the top-level help when called with no argument, or the detailed help for the named subcommand.

```
htmlc help render
htmlc help props
```

---

## Explanation

### Fragment vs page rendering

`render` and `page` differ only in how they wrap their output.

`render` produces a raw HTML fragment.  Any `<style scoped>` block in the component is prepended as a `<style>` element so the styles take effect when the fragment is inserted into an existing document.

`page` wraps the component output in a minimal document shell beginning with `<!DOCTYPE html>`, and inserts scoped styles into the `<head>` element.  Use it when the `.vue` file represents an entire page.

Both subcommands write to stdout.  Redirect output or capture it in a shell pipeline as needed.

---

### Component name resolution

When a template contains a component tag such as `<Card>`, the engine resolves
it using **proximity-based resolution**:

1. Starting from the directory of the calling component, search for a matching
   `.vue` file using four name-folding strategies (exact match, first letter
   capitalised, kebab-to-PascalCase, case-insensitive).
2. If no match is found, move one level up toward the root (`-dir`) and repeat.
3. Continue until the root is reached.
4. If still not found, fall back to the flat registry (backward compatible with
   single-directory projects).

**Example:**

```
templates/
  Card.vue          ← generic card
  blog/
    Card.vue        ← blog-specific card
    PostPage.vue    ← <Card> resolves to blog/Card.vue
  admin/
    Dashboard.vue   ← <Card> resolves to Card.vue (walk-up to root)
```

`blog/PostPage.vue` uses `<Card>` → resolves to `blog/Card.vue` (same directory).
`admin/Dashboard.vue` uses `<Card>` → no `Card` in `admin/`, walks up, finds `Card.vue` at root.

**Explicit path references**

To bypass proximity resolution and target a specific component, use a
path-qualified `is` attribute on `<component>`:

```vue
<!-- always resolves to blog/Card.vue regardless of where this template lives -->
<component is="blog/Card" />

<!-- root-relative: resolves to Card.vue at the -dir root -->
<component is="/Card" />
```

Path-based references do not apply name-folding and return an error if the
component is not found.

If a name is not resolved through any path, the CLI lists up to ten available
component names as a hint.

Supplying a file path (`./Card.vue`, `/abs/path/Card.vue`, or anything
containing a path separator or ending in `.vue`) as the component *argument*
is only supported by the `props` subcommand.  The other subcommands require a
bare component name.

---

### Scoped styles in CLI output

When a component contains `<style scoped>`, the engine:

1. Generates a unique scope identifier (e.g. `data-v-a1b2c3`).
2. Rewrites every CSS selector in the `<style>` block to include an attribute selector for that identifier.
3. Adds the scope attribute to every element in the rendered HTML.

The `render` subcommand prepends the rewritten `<style>` block to the output.  The `page` subcommand injects it into the document `<head>`.  In both cases the scope attribute appears only on elements rendered from that specific component, so styles do not leak across component boundaries.

---

### Debug mode

Passing `-debug` injects three `data-htmlc-*` attributes onto the root element of every rendered component:

| Attribute | Value |
|---|---|
| `data-htmlc-component` | Component name (e.g. `"Card"`) |
| `data-htmlc-file` | Relative path to the `.vue` source file |
| `data-htmlc-props` | HTML-escaped JSON object of the props passed to the component |

If the props cannot be JSON-serialised (e.g. a non-marshallable Go type), `data-htmlc-props` is replaced by `data-htmlc-props-error` containing the error message.

Components whose template has no single root element (fragment templates) are not annotated — there is no element to carry the attributes.

The attributes are standard HTML `data-*` attributes.  They are visible in browser DevTools and accessible via the `dataset` API.  They are intended solely for development use.

---

### Page-centric build

`htmlc build` treats the pages directory as a source tree where every `.vue` file is an independently renderable page.  The output hierarchy mirrors the input: `pages/posts/hello.vue` becomes `out/posts/hello.html`.

Props are loaded from JSON files that sit next to the `.vue` files.  This keeps templates free of data concerns and allows the same data to be reused by other tools (APIs, tests, etc.).  Directory-level `_data.json` files provide shared defaults that are shallow-merged with page-level files.

Layout wrapping is additive: a layout component receives the rendered page HTML as a `content` prop, enabling a single shell to be applied to every page without modifying any page component.

Files whose base name starts with `_` are skipped during page discovery.  Use this convention for shared partials that are referenced by page components but should not produce standalone output files.

---

## External Directives

`htmlc build` automatically discovers executable files named `v-<name>` in the
`-dir` component tree and registers them as **external directives**.

An external directive is a standalone executable that speaks the
[External Directive Protocol](../../docs/external-directive-protocol.md) over
stdin/stdout using newline-delimited JSON.  It is spawned once per build and
handles all invocations of the directive for that run.

### Protocol

Each directive executable is spawned once at build start.  `htmlc` sends one
JSON object per directive invocation and reads one JSON response, both
separated by newlines.  See the
[External Directive Protocol](../../docs/external-directive-protocol.md) for
the full specification.

### Example: v-syntax-highlight

[`v-syntax-highlight`](../v-syntax-highlight/README.md) is a ready-made
external directive that syntax-highlights source code using the
[chroma](https://github.com/alecthomas/chroma) library.

```sh
# Build
go build -o bin/v-syntax-highlight ./cmd/v-syntax-highlight

# Install into the component tree
cp bin/v-syntax-highlight ./templates/v-syntax-highlight
chmod +x ./templates/v-syntax-highlight

# Generate CSS (include in your layout's <head>)
./templates/v-syntax-highlight -print-css -style monokai > assets/highlight.css

# Use in templates
# <pre v-syntax-highlight="'go'">…</pre>
htmlc build -dir ./templates -pages ./pages -out ./dist
```
