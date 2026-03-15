# How-to: Template Bridge Recipes

Each section is a self-contained recipe. Goal first, minimal working snippet, no design rationale (see `docs/explanation-template-bridge.md` for that).

---

## 2.1 How to compile a `.vue` component to `*html/template.Template`

```go
engine, err := htmlc.New(htmlc.Options{ComponentDir: "./templates"})
if err != nil {
    log.Fatal(err)
}

tmpl, err := engine.CompileToTemplate("EmailLayout")
if err != nil {
    log.Fatal(err)
}

var buf strings.Builder
if err := tmpl.Execute(&buf, data); err != nil {
    log.Fatal(err)
}
```

The returned template name is the lowercased component name: `"EmailLayout"` → `"emaillayout"`. Sub-components referenced by `EmailLayout` are included as named `{{ define }}` blocks in the same template set. Scoped `<style>` blocks are stripped from the output.

---

## 2.2 How to register a legacy `html/template` as an htmlc component

```go
legacy, err := template.ParseFiles("partials/header.html")
if err != nil {
    log.Fatal(err)
}

if err := engine.RegisterTemplate("Header", legacy); err != nil {
    log.Fatal(err)
}
// Now use <Header /> in any .vue file.
```

Once registered, `<Header />` in any `.vue` component resolves to the converted legacy template. All named `{{ define }}` blocks within `legacy` are also registered as separate components under their own names. If conversion fails (unsupported template constructs), `RegisterTemplate` returns a `*bridge.ConversionError` and nothing is registered.

---

## 2.3 How to convert a `.vue` file to `html/template` on the command line

```bash
htmlc template vue-to-tmpl -dir ./templates PostPage > post.html
```

The output is one `{{ define }}` block per component (root last, sub-components first), suitable for `html/template.New("").ParseFiles("post.html")` or `//go:embed`:

```html
{{define "postmeta"}}
  <span class="date">{{.date}}</span>
{{end}}
{{define "postpage"}}
  <article>
    <h1>{{.title}}</h1>
    <postmeta></postmeta>
    {{.body}}
  </article>
{{end}}
```

Scoped `<style>` blocks are stripped. Conversion warnings go to stderr; add `-quiet` to suppress them.

---

## 2.4 How to convert an `html/template` file to `.vue` on the command line

```bash
htmlc template tmpl-to-vue -name PostPage < legacy/post.html > PostPage.vue
```

The converter reads `html/template` source from stdin and writes a `.vue` component to stdout. Conversion is best-effort: the output compiles as a `.vue` file that renders identically to the original template for all supported constructs.

Constructs that cause an error and halt conversion:
- `{{ .field | funcname }}` — multi-command pipelines
- `{{ with .x }}` — `with` blocks
- `$x := .field` — variable assignments
- `{{ template "Name" expr }}` where `expr` is not `.`

If any of these appear, the converter exits with status 1 and prints a `ConversionError` message to stderr. Manually translate those sections, then retry.

---

## 2.5 How to satisfy the `v-html` data contract

When a `.vue` component uses `v-html`, the compiled `html/template` requires the corresponding data field to be `html/template.HTML`, not a plain `string`.

```go
import htmltmpl "html/template"

data := map[string]any{
    "body": htmltmpl.HTML("<p>Safe, pre-sanitised content</p>"),
}
if err := tmpl.Execute(w, data); err != nil {
    log.Fatal(err)
}
```

**Why this matters**: `html/template` HTML-escapes plain `string` values in HTML body context. A field declared as `template.HTML` is passed through verbatim. If you pass a `string` instead, the angle brackets in your markup are escaped to `&lt;` and `&gt;`, producing visible escaped text rather than rendered HTML — silently diverging from htmlc's behaviour.

The compiled template source includes a comment documenting the requirement:

```html
{{/* .body must be html/template.HTML; plain strings are auto-escaped */}}
<div>{{ .body }}</div>
```

---

## 2.6 How to handle the `v-for` outer-scope limitation

`v-for` in the bridge only has access to the loop item (`.`). References to variables from the outer scope produce a `ConversionError`. The workaround is to embed all needed outer-scope data into each item before passing it to the engine:

```go
type PostItem struct {
    // Fields from the outer scope that the loop body needs.
    SiteTitle string

    // Fields that belong to the item.
    Title   string
    Content string
}

// Build the slice with outer-scope data embedded in each element.
items := make([]PostItem, len(posts))
for i, p := range posts {
    items[i] = PostItem{
        SiteTitle: siteConfig.Title, // outer-scope value copied in
        Title:     p.Title,
        Content:   p.Content,
    }
}

data := map[string]any{"items": items}
```

The `.vue` component can then access `{{ item.siteTitle }}` without needing a reference outside the loop.

---

## 2.7 How to suppress conversion warnings

**On the command line:**

```bash
htmlc template vue-to-tmpl -quiet -dir ./templates Card
```

The `-quiet` flag silences all non-fatal warnings on stderr (for example, the `v-html` data contract notice).

**Programmatically:**

```go
text, warnings, err := engine.TemplateText("Card")
if err != nil {
    log.Fatal(err)
}
_ = warnings // discard warnings; proceed with text
```

`TemplateText` separates warnings from errors. Non-fatal warnings (data contract notices) are returned in the `warnings` slice; hard errors return a non-nil `err`. Inspect or log `warnings` when you want visibility; assign to `_` to discard them explicitly.
