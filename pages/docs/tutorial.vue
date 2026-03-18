<template>
  <DocsPage
    pageTitle="Tutorial — htmlc.sh"
    description="Step-by-step tutorial: build your first htmlc component from scratch in Go."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Steps'},
      {href: '#step-1', label: '1 — Install'},
      {href: '#step-2', label: '2 — Write a component'},
      {href: '#step-3', label: '3 — Create an engine'},
      {href: '#step-4', label: '4 — Render with props'},
      {href: '#step-4b', label: '4b — Props from a struct'},
      {href: '#step-5', label: '5 — Use slots'},
      {href: '#step-6', label: '6 — Export to html/template'},
      {label: 'See also'},
      {href: '/docs/components.html', label: 'Component system'},
      {href: '/docs/go-api.html', label: 'Go API reference'},
      {href: '/docs/howto.html', label: 'How-to guides'}
    ]"
  >
    <h1>Tutorial</h1>
    <p class="lead">Build your first htmlc component from scratch. This walkthrough takes you from installation to rendering a component with props and slots in about five minutes.</p>

    <!-- ═══════════════════════════════════════════════ Step 1 -->
    <h2 id="step-1">Step 1 — Install htmlc</h2>
    <p>Add the package to your Go module:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>go get github.com/dhamidi/htmlc</code></pre>

    <p>The CLI is optional but handy for testing components without writing Go code:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

    <!-- ═══════════════════════════════════════════════ Step 2 -->
    <h2 id="step-2">Step 2 — Write a component</h2>
    <p>Create a directory called <code>components/</code> and add a file named <code>Card.vue</code>:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;&#123;&#123;<!---><!----> title }}&lt;/h2&gt;
    &lt;slot&gt;No content provided.&lt;/slot&gt;
  &lt;/div&gt;
&lt;/template&gt;

&lt;style scoped&gt;
.card {
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 1rem;
}
&lt;/style&gt;</code></pre>

    <p>The <code>{{ "{{" }} title }}</code> interpolation reads the <code>title</code> prop. The <code>&lt;slot&gt;</code> element is a placeholder for content supplied by a parent component; its children are the fallback rendered when no content is provided.</p>

    <!-- ═══════════════════════════════════════════════ Step 3 -->
    <h2 id="step-3">Step 3 — Create an engine</h2>
    <p>Call <code>htmlc.New</code> with the directory that contains your <code>.vue</code> files. The engine discovers and registers every component automatically.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    "log"

    "github.com/dhamidi/htmlc"
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: "./components",
    })
    if err != nil {
        log.Fatal(err)
    }
    _ = engine
}</code></pre>

    <!-- ═══════════════════════════════════════════════ Step 4 -->
    <h2 id="step-4">Step 4 — Render with props</h2>
    <p>Call <code>RenderFragmentString</code> to render a component to a string. Pass props as a <code>map[string]any</code>.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello, htmlc!",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(html)</code></pre>

    <p>Expected output (style block prepended by the engine):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;style&gt;
.card[data-v-…]{border:1px solid #ccc;border-radius:8px;padding:1rem}
&lt;/style&gt;
&lt;div class="card" data-v-…&gt;
  &lt;h2&gt;Hello, htmlc!&lt;/h2&gt;
  No content provided.
&lt;/div&gt;</code></pre>

    <p>The fallback text <em>"No content provided."</em> is rendered because no slot content was passed. Step 5 shows how to supply it.</p>

    <!-- ═══════════════════════════════════════════════ Step 4b -->
    <h2 id="step-4b">Step 4b — Pass a struct as props</h2>
    <p>Instead of building a <code>map[string]any</code> by hand you can pass any Go struct directly. The engine reads exported fields using their <code>json</code> struct tag (if present) and the Go field name otherwise.</p>
    <p>Define a struct that mirrors the props your component expects:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>type CardData struct {
    Title string `json:"title"`
}

data := CardData{Title: "Hello from a struct!"}

html, err := engine.RenderFragmentString("Card", data)
if err != nil {
    log.Fatal(err)
}
fmt.Println(html)</code></pre>

    <p>The <code>Card</code> component template accesses <code>{{ "{{" }} title }}</code> exactly as before — nothing changes on the template side. Structs and maps are interchangeable from the template's point of view.</p>

    <p>You can also spread a struct onto a child component using <code>v-bind</code> in a parent template:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/PostPage.vue --&gt;
&lt;template&gt;
  &lt;!-- The struct's fields become individual props of PostCard --&gt;
  &lt;PostCard v-bind="post" /&gt;
&lt;/template&gt;</code></pre>

    <p>The engine accepts any struct or <code>map[string]any</code> as the right-hand side of <code>v-bind</code>. Embedded struct fields are promoted and resolved as if they were declared directly on the outer struct.</p>

    <!-- ═══════════════════════════════════════════════ Step 5 -->
    <h2 id="step-5">Step 5 — Use slots</h2>
    <p>Slot content is supplied through <strong>component composition in a <code>.vue</code> template</strong>. There is no <code>$slots</code> key in the Go props map; htmlc does not support injecting raw HTML into slots via the data map.</p>

    <p>Create a wrapper component that uses <code>Card</code> with slot content:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/WelcomeCard.vue --&gt;
&lt;template&gt;
  &lt;Card title="Welcome"&gt;
    &lt;p&gt;This paragraph is rendered inside the Card's slot.&lt;/p&gt;
  &lt;/Card&gt;
&lt;/template&gt;</code></pre>

    <p>Then render the wrapper from Go:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>html, err := engine.RenderFragmentString("WelcomeCard", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(html)</code></pre>

    <p>Expected output:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;div class="card" data-v-…&gt;
  &lt;h2&gt;Welcome&lt;/h2&gt;
  &lt;p&gt;This paragraph is rendered inside the Card's slot.&lt;/p&gt;
&lt;/div&gt;</code></pre>

    <p>The same pattern applies to named and scoped slots — the parent component uses <code>&lt;template #name&gt;</code> syntax in the <code>.vue</code> file to target specific slots. See the <a href="/docs/components.html#slots">component system reference</a> for named and scoped slot examples.</p>

    <Callout>
      <p><strong>Dynamic slot content from Go</strong><br>
      If you need to inject a dynamic HTML string into a component from Go, use a regular prop with <code>v-html</code> instead of a slot:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/Card.vue --&gt;
&lt;div class="card"&gt;
  &lt;h2&gt;&#123;&#123;<!---><!----> title }}&lt;/h2&gt;
  &lt;div v-html="body"&gt;&lt;/div&gt;
&lt;/div&gt;</code></pre>
      <pre v-syntax-highlight="'go'"><code v-pre>html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
    "body":  "&lt;p&gt;Dynamic content from Go&lt;/p&gt;",
})</code></pre>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Step 6 -->
    <h2 id="step-6">Step 6 — Export to html/template</h2>
    <p>If your application already uses Go's standard <code>html/template</code> package, you can compile any <code>.vue</code> component into a <code>*html/template.Template</code> and drop it straight into your existing template set. No rewrite required.</p>

    <p>Call <code>CompileToTemplate</code> right after creating the engine:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{ComponentDir: "./components"})
if err != nil {
    log.Fatal(err)
}

// Compile Card.vue → *html/template.Template.
// The template name is the lowercased component name: "card".
cardTmpl, err := engine.CompileToTemplate("Card")
if err != nil {
    log.Fatal(err)
}</code></pre>

    <p>The compiled template can be executed directly or merged into a larger template set. To call it from an existing page template, add its parse tree and invoke it with <code>{{ "{{" }}template "card" .}}</code>:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// pageTmpl is your existing *html/template.Template.
if _, err := pageTmpl.AddParseTree("card", cardTmpl.Tree); err != nil {
    log.Fatal(err)
}

// Now page.html can use: &#123;&#123;<!---><!--->"card" .}}</code></pre>

    <p>The <code>.vue</code> expression syntax maps directly to Go template syntax. A component that uses <code>{{ "{{" }} title }}</code> or <code>{{ "{{" }} user.name }}</code> compiles to <code>{{ "{{" }}.title}}</code> and <code>{{ "{{" }}.user.name}}</code> respectively, so you pass data the same way you would for any other Go template:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>var buf strings.Builder
err = cardTmpl.Execute(&buf, map[string]any{
    "title": "Hello, html/template!",
})
fmt.Println(buf.String())</code></pre>

    <p>Expected output (no scoped style prefix — <code>html/template</code> knows nothing about htmlc's style scoping):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;div class="card"&gt;
  &lt;h2&gt;Hello, html/template!&lt;/h2&gt;
&lt;/div&gt;</code></pre>

    <Callout>
      <p><strong>Supported expressions</strong><br>
      <code>CompileToTemplate</code> only supports simple identifiers (<code>{{ "{{" }} title }}</code>)
      and dot-path expressions (<code>{{ "{{" }} user.name }}</code>). Complex expressions such as
      function calls, arithmetic, or index access are not converted and will return an error.
      See the <a href="/docs/go-api.html">Go API reference</a> for the full list of supported constructs.</p>
    </Callout>

  </DocsPage>
</template>

