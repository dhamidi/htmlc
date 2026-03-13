<template>
  <Layout pageTitle="Tutorial — htmlc.sh" description="Step-by-step tutorial: build your first htmlc component from scratch in Go." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-label">Steps</div>
          <a href="#step-1" class="sidebar-link">1 — Install</a>
          <a href="#step-2" class="sidebar-link">2 — Write a component</a>
          <a href="#step-3" class="sidebar-link">3 — Create an engine</a>
          <a href="#step-4" class="sidebar-link">4 — Render with props</a>
          <a href="#step-5" class="sidebar-link">5 — Use slots</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">See also</div>
          <a href="/docs/components.html" class="sidebar-link">Component system</a>
          <a href="/docs/go-api.html" class="sidebar-link">Go API reference</a>
          <a href="/docs/howto.html" class="sidebar-link">How-to guides</a>
        </div>
      </aside>

      <div class="docs-content">
        <h1>Tutorial</h1>
        <p class="lead">Build your first htmlc component from scratch. This walkthrough takes you from installation to rendering a component with props and slots in about five minutes.</p>

        <!-- ═══════════════════════════════════════════════ Step 1 -->
        <h2 id="step-1">Step 1 — Install htmlc</h2>
        <p>Add the package to your Go module:</p>
        <pre><code>go get github.com/dhamidi/htmlc</code></pre>

        <p>The CLI is optional but handy for testing components without writing Go code:</p>
        <pre><code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

        <!-- ═══════════════════════════════════════════════ Step 2 -->
        <h2 id="step-2">Step 2 — Write a component</h2>
        <p>Create a directory called <code>components/</code> and add a file named <code>Card.vue</code>:</p>
        <pre><code>&lt;!-- components/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ "{{" }} title }}&lt;/h2&gt;
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
        <pre><code>package main

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
        <pre><code>html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello, htmlc!",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println(html)</code></pre>

        <p>Expected output (style block prepended by the engine):</p>
        <pre><code>&lt;style&gt;
.card[data-v-…]{border:1px solid #ccc;border-radius:8px;padding:1rem}
&lt;/style&gt;
&lt;div class="card" data-v-…&gt;
  &lt;h2&gt;Hello, htmlc!&lt;/h2&gt;
  No content provided.
&lt;/div&gt;</code></pre>

        <p>The fallback text <em>"No content provided."</em> is rendered because no slot content was passed. Step 5 shows how to supply it.</p>

        <!-- ═══════════════════════════════════════════════ Step 5 -->
        <h2 id="step-5">Step 5 — Use slots</h2>
        <p>Slot content is supplied through <strong>component composition in a <code>.vue</code> template</strong>. There is no <code>$slots</code> key in the Go props map; htmlc does not support injecting raw HTML into slots via the data map.</p>

        <p>Create a wrapper component that uses <code>Card</code> with slot content:</p>
        <pre><code>&lt;!-- components/WelcomeCard.vue --&gt;
&lt;template&gt;
  &lt;Card title="Welcome"&gt;
    &lt;p&gt;This paragraph is rendered inside the Card's slot.&lt;/p&gt;
  &lt;/Card&gt;
&lt;/template&gt;</code></pre>

        <p>Then render the wrapper from Go:</p>
        <pre><code>html, err := engine.RenderFragmentString("WelcomeCard", nil)
if err != nil {
    log.Fatal(err)
}
fmt.Println(html)</code></pre>

        <p>Expected output:</p>
        <pre><code>&lt;div class="card" data-v-…&gt;
  &lt;h2&gt;Welcome&lt;/h2&gt;
  &lt;p&gt;This paragraph is rendered inside the Card's slot.&lt;/p&gt;
&lt;/div&gt;</code></pre>

        <p>The same pattern applies to named and scoped slots — the parent component uses <code>&lt;template #name&gt;</code> syntax in the <code>.vue</code> file to target specific slots. See the <a href="/docs/components.html#slots">component system reference</a> for named and scoped slot examples.</p>

        <Callout>
          <p><strong>Dynamic slot content from Go</strong><br>
          If you need to inject a dynamic HTML string into a component from Go, use a regular prop with <code>v-html</code> instead of a slot:</p>
          <pre><code>&lt;!-- components/Card.vue --&gt;
&lt;div class="card"&gt;
  &lt;h2&gt;{{ "{{" }} title }}&lt;/h2&gt;
  &lt;div v-html="body"&gt;&lt;/div&gt;
&lt;/div&gt;</code></pre>
          <pre><code>html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
    "body":  "&lt;p&gt;Dynamic content from Go&lt;/p&gt;",
})</code></pre>
        </Callout>

      </div>
    </div>

  </Layout>
</template>

<script>
export default {
  props: ['siteTitle']
}
</script>

<style scoped>
.docs-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 3rem;
  max-width: 1100px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.docs-sidebar {
  position: sticky;
  top: 2rem;
  align-self: start;
  max-height: calc(100vh - 4rem);
  overflow-y: auto;
}

.sidebar-section {
  margin-bottom: 1.25rem;
}

.sidebar-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #8b8fa8;
  margin-bottom: 0.35rem;
  padding: 0 0.5rem;
}

.sidebar-link {
  display: block;
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
  color: #8b8fa8;
  border-radius: 4px;
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}

.sidebar-link:hover {
  color: #e2e4f0;
  background: rgba(255,255,255,0.06);
  text-decoration: none;
}

.docs-content {
  min-width: 0;
}

.docs-content h2 {
  margin-top: 2.5rem;
  padding-top: 2.5rem;
  border-top: 1px solid #2a2d3e;
  font-size: 1.35rem;
  color: #e2e4f0;
}

.docs-content h2:first-of-type {
  border-top: none;
  padding-top: 0;
  margin-top: 0;
}

.lead {
  font-size: 1.1rem;
  color: #c4c8e2;
  margin-bottom: 2rem;
}

@media (max-width: 700px) {
  .docs-layout {
    grid-template-columns: 1fr;
  }
  .docs-sidebar {
    position: static;
  }
}
</style>
