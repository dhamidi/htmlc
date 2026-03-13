<template>
  <Layout pageTitle="Component System — htmlc.sh" description="htmlc component system: SFC format, props, slots, scoped styles, Go API." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-label">Components</div>
          <a href="#sfc-format" class="sidebar-link">SFC format</a>
          <a href="#registration" class="sidebar-link">Registration</a>
          <a href="#composition" class="sidebar-link">Composition</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Data</div>
          <a href="#props" class="sidebar-link">Props</a>
          <a href="#slots" class="sidebar-link">Slots</a>
          <a href="#scoped-styles" class="sidebar-link">Scoped styles</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Go API</div>
          <a href="#go-api" class="sidebar-link">Engine</a>
          <a href="#rendering" class="sidebar-link">Rendering</a>
          <a href="#custom-directives" class="sidebar-link">Custom directives</a>
        </div>
      </aside>

      <div class="docs-content">
        <h1>Component system</h1>
        <p class="lead">htmlc components are Vue Single File Components — <code>.vue</code> files with template, optional script, and optional style sections.</p>

        <h2 id="sfc-format">SFC format</h2>
        <p>A component file has up to three sections:</p>
        <pre><code>&lt;!-- components/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ "{{" }} title }}&lt;/h2&gt;
    &lt;slot&gt;No content provided.&lt;/slot&gt;
  &lt;/div&gt;
&lt;/template&gt;

&lt;!-- Optional: preserved verbatim in output, never executed --&gt;
&lt;script&gt;
export default { props: ['title'] }
&lt;/script&gt;

&lt;!-- Optional: global or scoped CSS --&gt;
&lt;style scoped&gt;
.card {
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 1rem;
}
&lt;/style&gt;</code></pre>

        <ul>
          <li><code>&lt;template&gt;</code> — required; contains the HTML template with directives</li>
          <li><code>&lt;script&gt;</code> — optional; preserved verbatim but never executed by the engine</li>
          <li><code>&lt;style&gt;</code> — optional; add <code>scoped</code> attribute to scope styles to this component</li>
        </ul>

        <h2 id="registration">Component registration</h2>
        <p>The engine automatically discovers all <code>.vue</code> files in the component directory. Components are referenced by their filename without the extension.</p>
        <pre><code>// Go API
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
})

// Register an additional component explicitly
engine.Register("MyCard", "/path/to/MyCard.vue")</code></pre>

        <p>In templates, component names follow PascalCase:</p>
        <pre><code>&lt;!-- Card.vue in the component dir --&gt;
&lt;Card :title="post.title"&gt;
  &lt;p&gt;{{ "{{" }} post.body }}&lt;/p&gt;
&lt;/Card&gt;</code></pre>

        <h2 id="composition">Component composition</h2>
        <p>Components can nest other components from the same registry. Props are passed as attributes; expressions use <code>:</code> shorthand.</p>
        <pre><code>&lt;!-- templates/PostPage.vue --&gt;
&lt;template&gt;
  &lt;Layout :title="title"&gt;
    &lt;Card :title="post.title"&gt;
      &lt;p&gt;{{ "{{" }} post.body }}&lt;/p&gt;
    &lt;/Card&gt;
    &lt;Card v-for="related in relatedPosts" :title="related.title" /&gt;
  &lt;/Layout&gt;
&lt;/template&gt;</code></pre>

        <h2 id="props">Props</h2>
        <p>Props are any data passed to a component. In templates, static props are strings; dynamic props use <code>:</code>.</p>
        <pre><code>&lt;!-- Static: value is the literal string "Hello" --&gt;
&lt;Card title="Hello" /&gt;

&lt;!-- Dynamic: value is the result of the expression --&gt;
&lt;Card :title="post.title" /&gt;

&lt;!-- Spread all props --&gt;
&lt;Card v-bind="post" /&gt;</code></pre>

        <p>Discover what props a component uses:</p>
        <pre><code>$ htmlc props -dir ./templates Card
title
author
body</code></pre>

        <h2 id="slots">Slots</h2>

        <h3>Default slot</h3>
        <pre><code>&lt;!-- In Card.vue --&gt;
&lt;div class="card"&gt;
  &lt;slot&gt;Fallback when no content is provided&lt;/slot&gt;
&lt;/div&gt;

&lt;!-- Usage --&gt;
&lt;Card title="Hello"&gt;
  &lt;p&gt;This renders inside the slot.&lt;/p&gt;
&lt;/Card&gt;</code></pre>

        <h3>Named slots</h3>
        <pre><code>&lt;!-- In Layout.vue --&gt;
&lt;header&gt;&lt;slot name="header" /&gt;&lt;/header&gt;
&lt;main&gt;&lt;slot /&gt;&lt;/main&gt;
&lt;footer&gt;&lt;slot name="footer" /&gt;&lt;/footer&gt;

&lt;!-- Usage --&gt;
&lt;Layout&gt;
  &lt;template #header&gt;
    &lt;nav&gt;&lt;a href="/"&gt;Home&lt;/a&gt;&lt;/nav&gt;
  &lt;/template&gt;
  &lt;article&gt;Main content&lt;/article&gt;
  &lt;template #footer&gt;&lt;p&gt;&amp;copy; 2024&lt;/p&gt;&lt;/template&gt;
&lt;/Layout&gt;</code></pre>

        <h3>Scoped slots</h3>
        <pre><code>&lt;!-- In List.vue --&gt;
&lt;ul&gt;
  &lt;li v-for="item in items"&gt;
    &lt;slot :item="item"&gt;{{ "{{" }} item }}&lt;/slot&gt;
  &lt;/li&gt;
&lt;/ul&gt;

&lt;!-- Usage: destructure slot props --&gt;
&lt;List :items="posts"&gt;
  &lt;template #default="{ item }"&gt;
    &lt;a :href="item.url"&gt;{{ "{{" }} item.title }}&lt;/a&gt;
  &lt;/template&gt;
&lt;/List&gt;</code></pre>

        <h2 id="scoped-styles">Scoped styles</h2>
        <p>Add <code>scoped</code> to <code>&lt;style&gt;</code> to confine styles to the component. The engine rewrites selectors and adds a unique scope attribute to matching elements.</p>
        <pre><code>&lt;style scoped&gt;
.card   { background: white; border-radius: 8px; }
h2      { color: #333; }
&lt;/style&gt;</code></pre>
        <p>Output (approximately):</p>
        <pre><code>&lt;style&gt;
.card[data-v-a1b2c3]   { background: white; border-radius: 8px; }
h2[data-v-a1b2c3]      { color: #333; }
&lt;/style&gt;</code></pre>

        <h2 id="go-api">Go API</h2>
        <pre><code>import "github.com/dhamidi/htmlc"

// Create an engine that loads components from a directory
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
    Debug:        false,
})
if err != nil {
    log.Fatal(err)
}</code></pre>

        <h2 id="rendering">Rendering</h2>
        <pre><code>// Render a fragment (no &lt;!DOCTYPE&gt;)
html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
    "body":  "World",
})

// Render a full page (&lt;!DOCTYPE html&gt;)
err = engine.RenderPage(w, "HomePage", map[string]any{
    "title": "My site",
})</code></pre>

        <h2 id="custom-directives">Custom directives</h2>
        <pre><code>engine.RegisterDirective("v-highlight", func(ctx *htmlc.DirectiveContext) error {
    // ctx.Node  — the HTML node being rendered
    // ctx.Value — the directive value expression result
    // ctx.Scope — the current render scope
    ctx.Node.Attr = append(ctx.Node.Attr, html.Attribute{
        Key: "class", Val: "highlighted",
    })
    return nil
})</code></pre>
      </div>
    </div>

  </Layout>
</template>

<style>
  p { margin: 1rem 0; }
  ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
  li { margin: 0.25rem 0; }

  .docs-layout { display: grid; grid-template-columns: 220px 1fr; gap: 0; max-width: 1200px; margin: 0 auto; }
  @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } }
  .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
  .sidebar-section { margin-bottom: 1.5rem; }
  .sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--muted); margin-bottom: 0.5rem; padding: 0 0.5rem; }
  .sidebar-link { display: block; padding: 0.3rem 0.5rem; font-size: 0.875rem; color: var(--muted); border-radius: 4px; }
  .sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.05); text-decoration: none; }
  .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }
  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.05rem; margin: 1.5rem 0 0.5rem; }
  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }
  .callout { background: rgba(124,106,247,0.08); border: 1px solid rgba(124,106,247,0.25); border-radius: 8px; padding: 1rem 1.25rem; margin: 1.5rem 0; }
  .callout p { margin: 0; font-size: 0.9rem; color: #c9ccf5; }
</style>
