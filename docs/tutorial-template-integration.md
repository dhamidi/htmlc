# Tutorial: Incrementally adopting htmlc in a Go web application

This tutorial walks you through adding a single htmlc `.vue` component to an existing Go web application that already uses `html/template`. You will not rewrite anything — the two template systems run side-by-side, and you call into htmlc only for the new component.

## What you'll build

You will have a working Go HTTP server whose main page is rendered by a legacy `html/template`. One section of that page — a user profile card — is rendered by a new `.vue` component compiled with htmlc's `CompileToTemplate`. By the end the legacy handler calls into the compiled template exactly as if it were an ordinary `*html/template.Template`, with no knowledge of `.vue` files.

## Before you start

- Go 1.21 or later
- A working Go module (`go mod init` is enough; no web framework required)
- An existing `html/template`-based handler, or follow the minimal example in Step 1

## Step 1 — Install htmlc

```bash
go get github.com/dhamidi/htmlc@latest
```

Add the import to the file where you set up your engine:

```go
import "github.com/dhamidi/htmlc"
```

## Step 2 — Create your first .vue component

Create a file `templates/ProfileCard.vue`:

```html
<template>
  <div class="profile-card">
    <h2>{{ user.name }}</h2>
    <p>{{ user.email }}</p>
  </div>
</template>
```

- `{{ user.name }}` and `{{ user.email }}` are dot-path expressions. They map directly to `{{ .user.name }}` and `{{ .user.email }}` in Go template syntax.
- There is no `<script>` section and no reactivity. This component renders once per call and produces static HTML.

## Step 3 — Register it alongside your existing templates

In your application setup, create an htmlc engine pointing at your templates directory. Then compile the `ProfileCard` component into a `*html/template.Template`:

```go
package main

import (
    "html/template"
    "log"
    "net/http"

    "github.com/dhamidi/htmlc"
)

func main() {
    // --- Legacy template (unchanged) ---
    pageTmpl := template.Must(template.ParseFiles("templates/page.html"))

    // --- htmlc engine for the new component ---
    engine, err := htmlc.New(htmlc.Options{ComponentDir: "./templates"})
    if err != nil {
        log.Fatal(err)
    }

    // Compile ProfileCard.vue to a *html/template.Template.
    // The template name is the lowercased component name: "profilecard".
    cardTmpl, err := engine.CompileToTemplate("ProfileCard")
    if err != nil {
        log.Fatal(err)
    }

    // Register the compiled card template with the legacy template set so
    // {{template "profilecard" .}} works in page.html.
    if _, err := pageTmpl.AddParseTree("profilecard", cardTmpl.Tree); err != nil {
        log.Fatal(err)
    }

    http.HandleFunc("/", makeHandler(pageTmpl))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

`templates/page.html` can now call the compiled component by its lowercased name:

```html
<!DOCTYPE html>
<html>
<body>
  <h1>My App</h1>
  {{template "profilecard" .}}
</body>
</html>
```

## Step 4 — Render from your handler

The handler is unchanged from what you would write for any `html/template` page:

```go
type PageData struct {
    User struct {
        Name  string
        Email string
    }
}

func makeHandler(tmpl *template.Template) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        data := PageData{}
        data.User.Name = "Ada Lovelace"
        data.User.Email = "ada@example.com"

        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        if err := tmpl.Execute(w, data); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    }
}
```

Start the server with `go run .` and open `http://localhost:8080`. The page renders the legacy template with the profile card slot filled in by the compiled htmlc component.

## Step 5 — Export a component for a library

Some Go libraries (email senders, PDF renderers) accept a `*html/template.Template` directly. You can export any htmlc component to them without a web server in the loop:

```go
import (
    "bytes"
    "html/template"
    "log"

    "github.com/dhamidi/htmlc"
)

func sendWelcomeEmail(name, email string) {
    engine, err := htmlc.New(htmlc.Options{ComponentDir: "./templates"})
    if err != nil {
        log.Fatal(err)
    }

    tmpl, err := engine.CompileToTemplate("WelcomeEmail")
    if err != nil {
        log.Fatal(err)
    }

    type EmailData struct {
        Name  string
        Email string
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, EmailData{Name: name, Email: email}); err != nil {
        log.Fatal(err)
    }

    // Pass buf.String() to your email library.
    _ = buf.String()
}
```

The library receives a rendered HTML string. It never sees the `.vue` source.

## What's next

- The Go API reference is in `doc.go` and inline go doc comments.
- The bridge design rationale is covered in the README under *html/template Integration*.
