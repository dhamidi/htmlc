<template>
  <DocsPage
    pageTitle="Tutorial — htmlc.sh"
    description="Step-by-step tutorial: build your first htmlc component from scratch in Go."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Steps'},
      {href: '#step-1', label: '1 — Install htmlc'},
      {href: '#step-2', label: '2 — Write a component'},
      {href: '#step-3', label: '3 — Create an engine'},
      {href: '#step-4', label: '4 — Render with props'},
      {href: '#step-4b', label: '4b — Pass a struct as props'},
      {href: '#step-5', label: '5 — Layouts with slots'},
      {href: '#step-6', label: '6 — Reuse existing templates'},
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

    <p>The CLI is optional but handy for testing components without writing Go code. CLI equivalents are shown alongside each step below.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

    <!-- ═══════════════════════════════════════════════ Step 2 -->
    <h2 id="step-2">Step 2 — Write a component</h2>
    <p>Create a directory called <code>components/</code> and add a file named <code>Card.vue</code>:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ title }}&lt;/h2&gt;
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

    <p>The <code v-pre>{{ title }}</code> interpolation reads the <code>title</code> prop. The <code>&lt;slot&gt;</code> element is a placeholder for content supplied by a parent component; its children are the fallback rendered when no content is provided.</p>

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

    <Callout>
      <p><strong>CLI equivalent</strong><br>
      There is no explicit "create engine" step from the CLI — <code>htmlc render</code> discovers components from a directory automatically. Use it as a quick smoke-test that a component is parseable:</p>
      <pre v-syntax-highlight="'bash'"><code v-pre>htmlc render -dir ./components Card</code></pre>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Step 4 -->
    <h2 id="step-4">Step 4 — Render with props</h2>
    <p>Call <a href="/docs/go-api.html#render-fragment"><code>RenderFragment</code></a> to write a component directly to an <code>io.Writer</code>. Prefer this over <code>RenderFragmentString</code> — it writes directly to an <code>io.Writer</code> and avoids allocating a full string copy. Pass props as a <code>map[string]any</code>.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>err := engine.RenderFragment(os.Stdout, "Card", map[string]any{
    "title": "Hello, htmlc!",
})
if err != nil {
    log.Fatal(err)
}</code></pre>

    <p>Expected output (style block prepended by the engine):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;style&gt;
.card[data-v-…]{border:1px solid #ccc;border-radius:8px;padding:1rem}
&lt;/style&gt;
&lt;div class="card" data-v-…&gt;
  &lt;h2&gt;Hello, htmlc!&lt;/h2&gt;
  No content provided.
&lt;/div&gt;</code></pre>

    <p>The fallback text <em>"No content provided."</em> is rendered because no slot content was passed. Step 5 shows how to supply it.</p>

    <Callout>
      <p><strong>CLI equivalent</strong></p>
      <pre v-syntax-highlight="'bash'"><code v-pre>htmlc render -dir ./components Card -props '{"title":"Hello, htmlc!"}'</code></pre>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Step 4b -->
    <h2 id="step-4b">Step 4b — Pass a struct as props</h2>
    <p>Instead of building a <code>map[string]any</code> by hand you can pass any Go struct directly. The engine reads exported fields using their <code>json</code> struct tag (if present) and the Go field name otherwise.</p>
    <p>Define a struct that mirrors the props your component expects:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>type CardData struct {
    Title string `json:"title"`
}

data := CardData{Title: "Hello from a struct!"}

err := engine.RenderFragment(os.Stdout, "Card", data)
if err != nil {
    log.Fatal(err)
}</code></pre>

    <p>The <code>Card</code> component template accesses <code v-pre>{{ title }}</code> exactly as before — nothing changes on the template side. Structs and maps are interchangeable from the template's point of view.</p>

    <p>You can also spread a struct onto a child component using <code>v-bind</code> in a parent template:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/PostPage.vue --&gt;
&lt;template&gt;
  &lt;!-- The struct's fields become individual props of PostCard --&gt;
  &lt;PostCard v-bind="post" /&gt;
&lt;/template&gt;</code></pre>

    <p>The engine accepts any struct or <code>map[string]any</code> as the right-hand side of <code>v-bind</code>. Embedded struct fields are promoted and resolved as if they were declared directly on the outer struct.</p>

    <Callout>
      <p><strong>CLI equivalent</strong><br>
      Props are always a JSON object from the CLI — the distinction between map and struct is only relevant in Go:</p>
      <pre v-syntax-highlight="'bash'"><code v-pre>htmlc render -dir ./components Card -props '{"title":"Hello from a struct!"}'</code></pre>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Step 5 -->
    <h2 id="step-5">Step 5 — Layouts with slots</h2>
    <p>Slots let a component own its <em>structure</em> while the caller supplies the <em>content</em>. The classic use case is a layout component: one component that provides the HTML skeleton (header, main, footer wrappers), reused across every page.</p>

    <p>We'll build a <code>PageLayout</code> component with three slots, then use it in a <code>HomePage</code> component.</p>

    <h3>Step A — Define the layout component</h3>
    <p>Create <code>components/PageLayout.vue</code>. It declares a named <code>header</code> slot, the default (unnamed) slot for main content, and a named <code>footer</code> slot:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/PageLayout.vue --&gt;
&lt;template&gt;
  &lt;div class="page"&gt;
    &lt;header&gt;
      &lt;slot name="header"&gt;&lt;/slot&gt;
    &lt;/header&gt;
    &lt;main&gt;
      &lt;slot&gt;&lt;/slot&gt;
    &lt;/main&gt;
    &lt;footer&gt;
      &lt;slot name="footer"&gt;&lt;/slot&gt;
    &lt;/footer&gt;
  &lt;/div&gt;
&lt;/template&gt;</code></pre>

    <p>Named slots use <code>&lt;slot name="…"&gt;</code>. The unnamed <code>&lt;slot&gt;</code> is the default slot — it receives any content not assigned to a named slot. You can place fallback content between the <code>&lt;slot&gt;</code> tags; it renders when the caller provides nothing for that slot.</p>

    <h3>Step B — Fill the slots from a page component</h3>
    <p>Create <code>components/HomePage.vue</code>. Use <code>&lt;template #name&gt;</code> to target each named slot; content outside any <code>&lt;template #…&gt;</code> goes to the default slot:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/HomePage.vue --&gt;
&lt;template&gt;
  &lt;PageLayout&gt;
    &lt;template #header&gt;
      &lt;nav&gt;&lt;a href="/"&gt;Home&lt;/a&gt; · &lt;a href="/about"&gt;About&lt;/a&gt;&lt;/nav&gt;
    &lt;/template&gt;

    &lt;h1&gt;Welcome&lt;/h1&gt;
    &lt;p&gt;This is the main content area.&lt;/p&gt;

    &lt;template #footer&gt;
      &lt;p&gt;&amp;copy; {{ year }} My Site&lt;/p&gt;
    &lt;/template&gt;
  &lt;/PageLayout&gt;
&lt;/template&gt;</code></pre>

    <p>The <code>#header</code> shorthand is equivalent to <code>v-slot:header</code>. See the <a href="/docs/components.html#slots">component system reference</a> for full named and scoped slot details.</p>

    <h3>Step C — Render from Go</h3>
    <p>Render <code>HomePage</code> the same way you would any other component. Pass the <code>year</code> prop to show data flowing from Go into the nested layout:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>err := engine.RenderFragment(os.Stdout, "HomePage", map[string]any{
    "year": 2024,
})
if err != nil {
    log.Fatal(err)
}</code></pre>

    <Callout>
      <p><strong>CLI equivalent</strong></p>
      <pre v-syntax-highlight="'bash'"><code v-pre>htmlc render -dir ./components HomePage -props '{"year":2024}'</code></pre>
    </Callout>

    <h3>Step D — Expected output</h3>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;div class="page" data-v-…&gt;
  &lt;header&gt;
    &lt;nav&gt;&lt;a href="/"&gt;Home&lt;/a&gt; · &lt;a href="/about"&gt;About&lt;/a&gt;&lt;/nav&gt;
  &lt;/header&gt;
  &lt;main&gt;
    &lt;h1&gt;Welcome&lt;/h1&gt;
    &lt;p&gt;This is the main content area.&lt;/p&gt;
  &lt;/main&gt;
  &lt;footer&gt;
    &lt;p&gt;&amp;copy; 2024 My Site&lt;/p&gt;
  &lt;/footer&gt;
&lt;/div&gt;</code></pre>

    <p><code>PageLayout</code> owns the skeleton; <code>HomePage</code> owns the content. Now we can create any number of page components — <code>AboutPage</code>, <code>BlogPage</code>, and so on — all sharing the same layout without duplicating the HTML structure.</p>

    <p><strong>One sharp edge:</strong> slots are a template-composition mechanism — they are filled by parent <code>.vue</code> components, not by Go code. You cannot pass slot content through the props map or inject it via <code>RenderFragment</code>; that is by design. (If you are coming from Vue: there is no <code>$slots</code> key in the Go props map.)</p>

    <Callout>
      <p><strong>Dynamic slot content from Go</strong><br>
      If you need to inject a dynamic HTML string into a component from Go, use a regular prop with <code>v-html</code> instead of a slot:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/Card.vue --&gt;
&lt;div class="card"&gt;
  &lt;h2&gt;{{ title }}&lt;/h2&gt;
  &lt;div v-html="body"&gt;&lt;/div&gt;
&lt;/div&gt;</code></pre>
      <pre v-syntax-highlight="'go'"><code v-pre>html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
    "body":  "&lt;p&gt;Dynamic content from Go&lt;/p&gt;",
})</code></pre>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Step 6 -->
    <h2 id="step-6">Step 6 — Reuse existing templates</h2>
    <p>If you have existing <code>html/template</code> partials — headers, footers, shared snippets — <code>RegisterTemplate</code> lets you use them as component tags in <code>.vue</code> files without rewriting anything.</p>

    <p>Register an existing template with the engine after creating it:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// Existing html/template code — no changes needed.
headerTmpl := html.template.Must(
    html.template.New("site-header").Parse(
        `&lt;header&gt;&lt;h1&gt;&#123;&#123;.title}}&lt;/h1&gt;&lt;/header&gt;`,
    ),
)

engine, err := htmlc.New(htmlc.Options{ComponentDir: "./components"})
if err != nil {
    log.Fatal(err)
}

if err := engine.RegisterTemplate("site-header", headerTmpl); err != nil {
    log.Fatal(err)
}</code></pre>

    <p>After registration, the template is available as a component tag in any <code>.vue</code> file — no <code>.vue</code> file is needed for the old template itself:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- pages/home.vue --&gt;
&lt;template&gt;
  &lt;site-header :title="pageTitle"&gt;&lt;/site-header&gt;
  &lt;main&gt;…&lt;/main&gt;
&lt;/template&gt;</code></pre>

    <p>If your template file contains <code v-pre>{{define}}</code> blocks, each block is automatically registered as its own component under its block name. A multi-partial template file just works — you don't need to register each block separately.</p>

    <Callout>
      <p><strong>Conversion limits</strong><br>
      <code>RegisterTemplate</code> converts common Go template constructs to their <code>.vue</code> equivalents, but <code v-pre>{{with}}</code>, variable assignments (<code>$x :=</code>), and multi-command pipelines are not supported and will return an error. Nothing is registered if any conversion fails.
      See the <a href="/docs/go-api.html">Go API reference</a> for the full list of supported constructs.</p>
    </Callout>

  </DocsPage>
</template>

