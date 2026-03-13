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
   - [Pass props to a component](#pass-props-to-a-component)
   - [Pipe props from stdin](#pipe-props-from-stdin)
   - [Inspect a component's props](#inspect-a-components-props)
   - [Export props as shell variables](#export-props-as-shell-variables)
   - [Debug a rendering problem](#debug-a-rendering-problem)
   - [Inspect the parsed template AST](#inspect-the-parsed-template-ast)
   - [Use components from a different directory](#use-components-from-a-different-directory)
3. [Reference](#reference)
   - [Subcommands at a glance](#subcommands-at-a-glance)
   - [render](#render)
   - [page](#page)
   - [build](#build)
   - [props](#props)
   - [ast](#ast)
   - [help](#help)
4. [Explanation](#explanation)
   - [Fragment vs page rendering](#fragment-vs-page-rendering)
   - [Component name resolution](#component-name-resolution)
   - [Scoped styles in CLI output](#scoped-styles-in-cli-output)
   - [Debug mode](#debug-mode)

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

**4. Discover what props a component expects.**

```
$ htmlc props -dir ./templates Greeting
name
```

You now know the three most-used subcommands: `render`, `page`, and `props`.

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

Add `-debug` to `render` or `page`.  The output is still valid HTML, but HTML comments are injected that describe what the engine did at each step.

```
htmlc render -debug -dir ./templates Card -props '{"title":"Test"}'
```

Example comment annotations in the output:

```html
<!-- [htmlc:debug] component=Card file=templates/Card.vue -->
<!-- [htmlc:debug] expr="title" value="Test" -->
<!-- [htmlc:debug] v-if="showBadge" → false: node skipped -->
```

Pipe the output through `grep htmlc:debug` to see only the diagnostic lines:

```
htmlc render -debug -dir ./templates Card -props '{"title":"Test"}' \
  | grep htmlc:debug
```

Never use `-debug` in production — the comments expose internal expression values.

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
| `-debug` | false | Annotate output with diagnostic HTML comments |

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
| `-debug` | false | Annotate output with diagnostic HTML comments |
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
|---|---|---|
| `-dir` | `.` | Directory containing shared `.vue` component files |
| `-pages` | `./pages` | Root of the page tree. All `.vue` files found recursively are treated as pages. |
| `-out` | `./out` | Output directory. The page tree hierarchy is reproduced here as `.html` files. Created if it does not exist. |
| `-layout` | _(empty)_ | Optional layout component name (resolved from `-dir`) that wraps every page. |
| `-debug` | false | Annotate output with diagnostic HTML comments |

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

When you supply a bare name such as `Card`, the engine looks for a file named `card.vue` (case-insensitive) inside the `-dir` directory.  It does not search subdirectories.

If the name is not found, the CLI lists up to ten available component names as a hint.

Supplying a path (`./Card.vue`, `/abs/path/Card.vue`, or anything containing a path separator or ending in `.vue`) is only supported by the `props` subcommand.  The other subcommands require a bare name.

---

### Scoped styles in CLI output

When a component contains `<style scoped>`, the engine:

1. Generates a unique scope identifier (e.g. `data-v-a1b2c3`).
2. Rewrites every CSS selector in the `<style>` block to include an attribute selector for that identifier.
3. Adds the scope attribute to every element in the rendered HTML.

The `render` subcommand prepends the rewritten `<style>` block to the output.  The `page` subcommand injects it into the document `<head>`.  In both cases the scope attribute appears only on elements rendered from that specific component, so styles do not leak across component boundaries.

---

### Debug mode

Passing `-debug` activates a secondary rendering pass that inserts HTML comments before or inside affected nodes.  The comment format is:

```
<!-- [htmlc:debug] key=value key2=value2 -->
```

Common keys:

| Key | Meaning |
|---|---|
| `component` | Component name being rendered |
| `file` | Source file path |
| `expr` | Expression text |
| `value` | Evaluated expression value |
| `v-if` / `v-for` / … | Directive and its outcome |
| `slot` | Slot name and node count |

Because comments are part of the HTML output, they survive round-trips through most HTML parsers and are visible in browser DevTools.  They are intended solely for development use.
