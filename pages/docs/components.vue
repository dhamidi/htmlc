<template>
  <Layout pageTitle="Component System — htmlc.sh" description="htmlc component system: SFC format, props, slots, scoped styles, Go API." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <SidebarSection label="Components">
          <a href="#sfc-format" class="sidebar-link">SFC format</a>
          <a href="#registration" class="sidebar-link">Registration</a>
          <a href="#composition" class="sidebar-link">Composition</a>
        </SidebarSection>
        <SidebarSection label="Data">
          <a href="#props" class="sidebar-link">Props</a>
          <a href="#slots" class="sidebar-link">Slots</a>
          <a href="#scoped-styles" class="sidebar-link">Scoped styles</a>
        </SidebarSection>
        <SidebarSection label="Go API">
          <a href="#go-api" class="sidebar-link">Engine</a>
          <a href="#rendering" class="sidebar-link">Rendering</a>
          <a href="#http-handlers" class="sidebar-link">HTTP handlers</a>
          <a href="#validate" class="sidebar-link">ValidateAll</a>
          <a href="#missing-props" class="sidebar-link">Missing props</a>
          <a href="#register-func" class="sidebar-link">RegisterFunc</a>
          <a href="#advanced-options" class="sidebar-link">Hot-reload / FS</a>
          <a href="#errors" class="sidebar-link">Error handling</a>
          <a href="#scope-rules" class="sidebar-link">Scope rules</a>
          <a href="#custom-directives" class="sidebar-link">Custom directives</a>
        </SidebarSection>
      </aside>

      <details class="mobile-nav">
        <summary>On this page</summary>
        <div class="sidebar-label">Components</div>
        <a href="#sfc-format" class="sidebar-link">SFC format</a>
        <a href="#registration" class="sidebar-link">Registration</a>
        <a href="#composition" class="sidebar-link">Composition</a>
        <div class="sidebar-label">Data</div>
        <a href="#props" class="sidebar-link">Props</a>
        <a href="#slots" class="sidebar-link">Slots</a>
        <a href="#scoped-styles" class="sidebar-link">Scoped styles</a>
        <div class="sidebar-label">Go API</div>
        <a href="#go-api" class="sidebar-link">Engine</a>
        <a href="#rendering" class="sidebar-link">Rendering</a>
        <a href="#http-handlers" class="sidebar-link">HTTP handlers</a>
        <a href="#validate" class="sidebar-link">ValidateAll</a>
        <a href="#missing-props" class="sidebar-link">Missing props</a>
        <a href="#register-func" class="sidebar-link">RegisterFunc</a>
        <a href="#advanced-options" class="sidebar-link">Hot-reload / FS</a>
        <a href="#errors" class="sidebar-link">Error handling</a>
        <a href="#scope-rules" class="sidebar-link">Scope rules</a>
        <a href="#custom-directives" class="sidebar-link">Custom directives</a>
      </details>

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

        <h2 id="http-handlers">HTTP handlers</h2>

        <h3>ServeComponent</h3>
        <p>
          Returns an <code>http.HandlerFunc</code> that renders a component as an HTML
          fragment and writes it with <code>Content-Type: text/html; charset=utf-8</code>.
          The data function is called on every request; pass <code>nil</code> if no data
          is needed.
        </p>
        <pre><code>http.Handle("/widget", engine.ServeComponent("Widget", func(r *http.Request) map[string]any {
    return map[string]any{"id": r.URL.Query().Get("id")}
}))</code></pre>

        <h3>ServePageComponent</h3>
        <p>
          Like <code>ServeComponent</code> but renders a full HTML page (injecting scoped
          styles into <code>&lt;/head&gt;</code>) and lets the data function return an HTTP
          status code alongside the data map. A status code of 0 is treated as 200.
        </p>
        <pre><code>http.Handle("/post", engine.ServePageComponent("PostPage",
    func(r *http.Request) (map[string]any, int) {
        post, err := db.GetPost(r.URL.Query().Get("slug"))
        if err != nil {
            return nil, http.StatusNotFound
        }
        return map[string]any{"post": post}, http.StatusOK
    },
))</code></pre>

        <h3>Mount</h3>
        <p>
          Registers multiple component routes on an <code>http.ServeMux</code> in one
          call. Each component is served as a full HTML page. Keys are
          <code>http.ServeMux</code> patterns (e.g. <code>"GET /{$}"</code>).
        </p>
        <pre><code>engine.Mount(mux, map[string]string{
    "GET /{$}":   "HomePage",
    "GET /about": "AboutPage",
    "GET /posts": "PostsPage",
})</code></pre>

        <h3>WithDataMiddleware</h3>
        <p>
          Adds a function that enriches the data map on every HTTP-triggered render.
          Multiple middleware functions are applied in registration order. Use this to
          inject values shared across all routes — current user, CSRF token, etc.
        </p>
        <Callout>
          <p>
            <strong>Scope note:</strong> Middleware values are available only in the
            top-level page scope. If a child component needs a middleware-supplied value,
            pass it down as an explicit prop or register it with <code>RegisterFunc</code>
            instead.
          </p>
        </Callout>
        <pre><code>engine.WithDataMiddleware(func(r *http.Request, data map[string]any) map[string]any {
    data["currentUser"] = sessionUser(r)
    data["csrfToken"]   = csrf.Token(r)
    return data
})</code></pre>

        <h2 id="validate">Startup validation</h2>

        <h3>ValidateAll</h3>
        <p>
          Checks every registered component for unresolvable child component references.
          Returns a slice of <code>ValidationError</code> (one per problem). Call once
          at startup to surface missing-component problems before the first request.
        </p>
        <pre><code>if errs := engine.ValidateAll(); len(errs) > 0 {
    for _, e := range errs {
        log.Printf("component error: %v", e)
    }
    os.Exit(1)
}</code></pre>

        <h2 id="missing-props">Missing prop handling</h2>
        <p>
          By default a missing prop renders a visible
          <code>[missing: propName]</code> placeholder so the page still loads and the
          absent prop is immediately obvious. Override this behaviour with
          <code>WithMissingPropHandler</code>:
        </p>
        <pre><code>// Abort the render with an error on any missing prop
engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)

// Silently substitute an empty string
engine.WithMissingPropHandler(func(name string) (any, error) {
    return "", nil
})</code></pre>

        <h2 id="register-func">Template functions</h2>

        <h3>RegisterFunc</h3>
        <p>
          Registers a Go function callable from any template expression rendered by
          this engine. Unlike props, registered functions are available in
          <em>every</em> component at every nesting depth — no prop threading needed.
          Engine functions act as lower-priority builtins: the render data scope
          overrides them.
        </p>
        <pre><code>engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    t, _ := args[0].(time.Time)
    return t.Format("2 Jan 2006"), nil
})

engine.RegisterFunc("url", func(args ...any) (any, error) {
    name, _ := args[0].(string)
    return router.URLFor(name), nil
})</code></pre>

        <p>Use them directly in templates:</p>
        <pre><code>&lt;span&gt;{{ "{{" }} formatDate(post.CreatedAt) }}&lt;/span&gt;
&lt;a :href="url('home')"&gt;Home&lt;/a&gt;</code></pre>

        <h2 id="advanced-options">Advanced options</h2>

        <h3>Hot-reload</h3>
        <p>
          Set <code>Reload: true</code> to re-parse changed <code>.vue</code> files
          automatically before each render — no server restart required. Disable in
          production.
        </p>
        <pre><code>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Reload:       true,
})</code></pre>

        <h3>Embedded filesystem</h3>
        <p>
          Set <code>Options.FS</code> to any <code>fs.FS</code> — including
          <code>embed.FS</code> — to load component files from an embedded or virtual
          filesystem instead of the OS filesystem. <code>ComponentDir</code> is then
          interpreted as a path inside the FS.
        </p>
        <pre><code>import "embed"

//go:embed templates
var templateFS embed.FS

engine, err := htmlc.New(htmlc.Options{
    FS:           templateFS,
    ComponentDir: "templates",
})</code></pre>
        <Callout>
          <p>
            <strong>Note:</strong> Hot-reload (<code>Reload: true</code>) only works
            when the FS also implements <code>fs.StatFS</code>. The standard
            <code>embed.FS</code> does <em>not</em> implement <code>fs.StatFS</code>, so
            reload is silently skipped for embedded filesystems.
          </p>
        </Callout>

        <h3>Context-aware rendering</h3>
        <p>
          Use <code>RenderPageContext</code> / <code>RenderFragmentContext</code> to
          propagate cancellation and deadlines through the render pipeline.
          <code>ServeComponent</code> and <code>ServePageComponent</code> forward
          <code>r.Context()</code> automatically.
        </p>
        <pre><code>ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel()

err = engine.RenderPageContext(ctx, w, "Page", data)
err = engine.RenderFragmentContext(ctx, w, "Card", data)</code></pre>

        <h2 id="errors">Error handling</h2>
        <p>
          Parse and render failures carry structured location information.
          Use <code>errors.As</code> to inspect them:
        </p>
        <pre><code>_, err := htmlc.ParseFile("Card.vue", src)
var pe *htmlc.ParseError
if errors.As(err, &pe) {
    fmt.Println(pe.Path)             // "Card.vue"
    if pe.Location != nil {
        fmt.Println(pe.Location.Line)    // 1-based line number
        fmt.Println(pe.Location.Snippet) // 3-line source context
    }
}

err = engine.RenderFragment(w, "Card", data)
var re *htmlc.RenderError
if errors.As(err, &re) {
    fmt.Println(re.Component)
    fmt.Println(re.Expr)             // expression that failed
    if re.Location != nil {
        fmt.Println(re.Location.Line)
        fmt.Println(re.Location.Snippet)
    }
}</code></pre>

        <p>When location is available, <code>err.Error()</code> produces a compiler-style message:</p>
        <pre><code>Card.vue:14:5: render Card.vue: expr "post.Title": cannot access property "Title" of null
  13 |   &lt;div class="card"&gt;
&gt; 14 |     {{ "{{" }} post.Title }}
  15 |   &lt;/div&gt;</code></pre>

        <h2 id="scope-rules">Scope propagation rules</h2>
        <p>
          Each component renders in an <strong>isolated scope</strong> containing only
          its own props. Parent scope variables are not inherited. The one exception
          is functions registered with <code>RegisterFunc</code> — they are injected
          into every component's scope automatically.
        </p>
        <table>
          <thead>
            <tr>
              <th>Mechanism</th>
              <th>Available in top-level page</th>
              <th>Available in child components</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td><code>RenderPage</code> / <code>RenderFragment</code> data map</td>
              <td>Yes</td>
              <td>No — pass as props</td>
            </tr>
            <tr>
              <td><code>WithDataMiddleware</code> values</td>
              <td>Yes</td>
              <td>No — pass as props</td>
            </tr>
            <tr>
              <td><code>RegisterFunc</code> functions</td>
              <td>Yes</td>
              <td>Yes (automatic)</td>
            </tr>
            <tr>
              <td>Explicit <code>:prop="expr"</code></td>
              <td>—</td>
              <td>Yes</td>
            </tr>
          </tbody>
        </table>

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
  .mobile-nav { display: none; }
  .mobile-nav summary { list-style: none; cursor: pointer; font-size: 0.875rem; font-weight: 600; color: var(--muted); padding: 0.75rem 1rem; background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; margin: 1rem 0; user-select: none; transition: color 0.15s; }
  .mobile-nav summary::-webkit-details-marker { display: none; }
  .mobile-nav[open] summary { color: var(--text); border-bottom-left-radius: 0; border-bottom-right-radius: 0; border-bottom-color: transparent; }
  .mobile-nav[open] { background: var(--bg2); border: 1px solid var(--border); border-radius: 8px; margin: 1rem 0; overflow: hidden; }
  .mobile-nav[open] summary { margin: 0; border: none; border-bottom: 1px solid var(--border); border-radius: 0; }
  .mobile-nav .sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--muted); padding: 0.75rem 1rem 0.25rem; }
  .mobile-nav .sidebar-link { display: block; padding: 0.35rem 1rem; font-size: 0.875rem; color: var(--muted); text-decoration: none; transition: color 0.15s, background 0.15s; }
  .mobile-nav .sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.06); }
  @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } .mobile-nav { display: block; } .docs-content { padding: 1.5rem 1rem 3rem; } }
  .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
  .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }
  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.05rem; margin: 1.5rem 0 0.5rem; }
  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }
</style>
