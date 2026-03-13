# v-syntax-highlight

`v-syntax-highlight` is an [htmlc](../../README.md) external directive that
syntax-highlights source code using the
[chroma](https://github.com/alecthomas/chroma) library.

---

## Table of contents

1. [Tutorial — highlight code in 5 minutes](#tutorial--highlight-code-in-5-minutes)
2. [How-to guides](#how-to-guides)
   - [Generate the CSS stylesheet](#generate-the-css-stylesheet)
   - [Use a different style](#use-a-different-style)
   - [Install into a component tree](#install-into-a-component-tree)
   - [Highlight code with a static language](#highlight-code-with-a-static-language)
3. [Reference](#reference)
   - [Flags](#flags)
   - [Directive protocol](#directive-protocol)
   - [Response fields](#response-fields)
4. [Explanation](#explanation)
   - [Why a separate binary?](#why-a-separate-binary)
   - [Dynamic code content](#dynamic-code-content)
   - [CSS strategies](#css-strategies)

---

## Tutorial — highlight code in 5 minutes

**Prerequisites:** Go installed; the `htmlc` CLI built.

**1. Build the binary.**

```sh
go build -o bin/v-syntax-highlight ./cmd/v-syntax-highlight
```

**2. Install into your component tree.**

```sh
cp bin/v-syntax-highlight ./templates/v-syntax-highlight
chmod +x ./templates/v-syntax-highlight
```

**3. Generate the CSS stylesheet.**

```sh
./templates/v-syntax-highlight -print-css -style monokai > assets/highlight.css
```

**4. Include the stylesheet in your layout component.**

```vue
<!-- templates/AppLayout.vue -->
<template>
  <html>
    <head>
      <link rel="stylesheet" href="/assets/highlight.css" />
    </head>
    <body><slot /></body>
  </html>
</template>
```

**5. Use the directive in a template.**

```vue
<!-- templates/Article.vue -->
<template>
  <article>
    <pre v-syntax-highlight="'go'">func main() {
    fmt.Println("hello, world")
}</pre>
  </article>
</template>
```

**6. Build the site.**

```sh
htmlc build -dir ./templates -pages ./pages -out ./dist
```

The `<pre>` element is now syntax-highlighted in the output HTML.

---

## How-to guides

### Generate the CSS stylesheet

Run `v-syntax-highlight` with `-print-css` to emit the chroma stylesheet for
the chosen style to stdout:

```sh
v-syntax-highlight -print-css -style monokai > assets/highlight.css
```

The stylesheet uses CSS classes (`.chroma`, `.kd`, `.s`, etc.) that match the
HTML produced during highlighting.  Include the file in every page that
displays highlighted code.

### Use a different style

Pass `-style` with any chroma style name (see the
[chroma style gallery](https://xyproto.github.io/splash/docs/)):

```sh
v-syntax-highlight -print-css -style github > assets/highlight.css
```

Use the same `-style` value when running `htmlc build` so the CSS and HTML
classes agree.  To configure the style at build time, place the binary in the
component tree under a wrapper script that passes the flag:

```sh
#!/bin/sh
exec /usr/local/bin/v-syntax-highlight -style github "$@"
```

### Install into a component tree

**Copy (static builds):**

```sh
cp bin/v-syntax-highlight ./templates/v-syntax-highlight
chmod +x ./templates/v-syntax-highlight
```

**Symlink (development):**

```sh
ln -s "$(pwd)/bin/v-syntax-highlight" ./templates/v-syntax-highlight
```

`htmlc build` discovers any executable file named `v-<name>` in the component
directory and registers it as an external directive automatically.

### Highlight code with a static language

```vue
<template>
  <article>
    <pre v-syntax-highlight="'python'">def greet(name):
    return f"Hello, {name}"</pre>
  </article>
</template>
```

The string literal passed to the directive is the language name recognised by
chroma (e.g. `"go"`, `"python"`, `"javascript"`, `"bash"`, `"yaml"`).

---

## Reference

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-style` | `monokai` | Chroma style name |
| `-print-css` | false | Print CSS for the chosen style to stdout and exit |
| `-formatter` | `html` | Chroma formatter (`html` or `terminal256`) — reserved for future use |
| `-inline` | false | Reserved for future use |

### Directive protocol

`v-syntax-highlight` implements the [htmlc external directive protocol](../../docs/external-directive-protocol.md).

It reads newline-delimited JSON from stdin and writes newline-delimited JSON
to stdout.  One request line produces exactly one response line.

**Request fields used:**

| Field | Type | Description |
|-------|------|-------------|
| `hook` | string | `"created"` or `"mounted"` |
| `id` | string | Echoed in the response |
| `tag` | string | HTML tag name of the element |
| `attrs` | object | Current element attributes |
| `text` | string | Text content of the element (code to highlight) |
| `binding.value` | string | Language name (e.g. `"go"`) |

### Response fields

**`created` hook:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Echoed request id |
| `attrs` | object | Updated attributes (includes `class: "language-<lang>"`) |
| `inner_html` | string | Highlighted HTML to replace element children |
| `error` | string | Set if highlighting failed |

**`mounted` hook:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Echoed request id |
| `html` | string | Always empty — no content injected after the element |

---

## Explanation

### Why a separate binary?

`htmlc` intentionally has no external dependencies beyond the standard library.
`v-syntax-highlight` is kept as a separate Go module so that the chroma library
is an opt-in dependency — projects that do not need syntax highlighting are not
affected.

### Dynamic code content

The `text` field in the `created` request contains the *text content* of the
element as the template engine sees it.  When the element body is a template
expression such as `{{ snippet.code }}`, the text contains the literal
expression text, not the evaluated value, because `htmlc` resolves directives
before evaluating expressions.

For dynamic code, pass the code as a prop to a reusable component:

```vue
<!-- templates/CodeBlock.vue -->
<template>
  <pre v-syntax-highlight="lang">{{ code }}</pre>
</template>
<script>
export default { props: ['lang', 'code'] }
</script>
```

The calling template passes the already-evaluated string:

```vue
<CodeBlock lang="go" :code="snippet.code" />
```

### CSS strategies

**Static builds (recommended):** Generate the stylesheet once with
`-print-css` and serve it as a static asset.  This is the simplest approach
and works with any hosting setup.

**Inline styles:** If you prefer a self-contained page with no external
stylesheet, copy the CSS output into a `<style>` block in your layout
component.
