<template>
  <DocsPage
    pageTitle="Component System — htmlc.sh"
    description="htmlc component system: SFC format, props, slots, scoped styles, Go API."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Components'},
      {href: '#sfc-format', label: 'SFC format'},
      {href: '#registration', label: 'Registration'},
      {href: '#composition', label: 'Composition'},
      {label: 'Data'},
      {href: '#props', label: 'Props'},
      {href: '#slots', label: 'Slots'},
      {href: '#scoped-styles', label: 'Scoped styles'},
      {label: 'Go API'},
      {href: '#go-api', label: 'Engine'},
      {href: '#rendering', label: 'Rendering'},
      {href: '#http-handlers', label: 'HTTP handlers'},
      {href: '#validate', label: 'ValidateAll'},
      {href: '#missing-props', label: 'Missing props'},
      {href: '#register-func', label: 'RegisterFunc'},
      {href: '#advanced-options', label: 'Hot-reload / FS'},
      {href: '#errors', label: 'Error handling'},
      {href: '#scope-rules', label: 'Scope rules'},
      {href: '#custom-directives', label: 'Custom directives'}
    ]"
  >
    <h1>Component system</h1>
    <p class="lead">htmlc components are Vue Single File Components — <code>.vue</code> files with template, optional script, and optional style sections.</p>

    <h2 id="sfc-format">SFC format</h2>
    <p>A component file has up to three sections:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ title }}&lt;/h2&gt;
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
      <li><code v-pre>&lt;template&gt;</code> — required; contains the HTML template with directives</li>
      <li><code v-pre>&lt;script&gt;</code> — optional; preserved verbatim but never executed by the engine</li>
      <li><code v-pre>&lt;style&gt;</code> — optional; add <code>scoped</code> attribute to scope styles to this component</li>
    </ul>

    <h2 id="registration">Component registration</h2>
    <p>The engine automatically discovers all <code>.vue</code> files in the component directory. Components are referenced by their filename without the extension.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// Go API
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
})

// Register an additional component explicitly
engine.Register("MyCard", "/path/to/MyCard.vue")</code></pre>

    <p>In templates, component names follow PascalCase:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Card.vue in the component dir --&gt;
&lt;Card :title="post.title"&gt;
  &lt;p&gt;{{ post.body }}&lt;/p&gt;
&lt;/Card&gt;</code></pre>

    <h2 id="composition">Component composition</h2>
    <p>Components can nest other components from the same registry. Props are passed as attributes; expressions use <code>:</code> shorthand.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/PostPage.vue --&gt;
&lt;template&gt;
  &lt;Layout :title="title"&gt;
    &lt;Card :title="post.title"&gt;
      &lt;p&gt;{{ post.body }}&lt;/p&gt;
    &lt;/Card&gt;
    &lt;Card v-for="related in relatedPosts" :title="related.title" /&gt;
  &lt;/Layout&gt;
&lt;/template&gt;</code></pre>

    <h2 id="props">Props</h2>
    <p>Props are any data passed to a component. In templates, static props are strings; dynamic props use <code>:</code>.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Static: value is the literal string "Hello" --&gt;
&lt;Card title="Hello" /&gt;

&lt;!-- Dynamic: value is the result of the expression --&gt;
&lt;Card :title="post.title" /&gt;

&lt;!-- Spread all props --&gt;
&lt;Card v-bind="post" /&gt;</code></pre>

    <p>Discover what props a component uses:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc props -dir ./templates Card
title
author
body</code></pre>

    <h2 id="slots">Slots</h2>

    <h3>Default slot</h3>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In Card.vue --&gt;
&lt;div class="card"&gt;
  &lt;slot&gt;Fallback when no content is provided&lt;/slot&gt;
&lt;/div&gt;

&lt;!-- Usage --&gt;
&lt;Card title="Hello"&gt;
  &lt;p&gt;This renders inside the slot.&lt;/p&gt;
&lt;/Card&gt;</code></pre>

    <h3>Named slots</h3>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In Layout.vue --&gt;
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
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In List.vue --&gt;
&lt;ul&gt;
  &lt;li v-for="item in items"&gt;
    &lt;slot :item="item"&gt;{{ item }}&lt;/slot&gt;
  &lt;/li&gt;
&lt;/ul&gt;

&lt;!-- Usage: destructure slot props --&gt;
&lt;List :items="posts"&gt;
  &lt;template #default="{ item }"&gt;
    &lt;a :href="item.url"&gt;{{ item.title }}&lt;/a&gt;
  &lt;/template&gt;
&lt;/List&gt;</code></pre>

    <h2 id="scoped-styles">Scoped styles</h2>
    <p>Add <code>scoped</code> to <code>&lt;style&gt;</code> to confine styles to the component. The engine rewrites selectors and adds a unique scope attribute to matching elements.</p>
    <pre v-syntax-highlight="'css'"><code v-pre>&lt;style scoped&gt;
.card   { background: white; border-radius: 8px; }
h2      { color: #333; }
&lt;/style&gt;</code></pre>
    <p>Output (approximately):</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;style&gt;
.card[data-v-a1b2c3]   { background: white; border-radius: 8px; }
h2[data-v-a1b2c3]      { color: #333; }
&lt;/style&gt;</code></pre>

    <h2 id="go-api">Go API</h2>
    <pre v-syntax-highlight="'go'"><code v-pre>import "github.com/dhamidi/htmlc"

// Create an engine that loads components from a directory
engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
    Debug:        false,
})
if err != nil {
    log.Fatal(err)
}</code></pre>

    <h2 id="rendering">Rendering</h2>
    <pre v-syntax-highlight="'go'"><code v-pre>// Render a fragment (no &lt;!DOCTYPE&gt;)
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
    <pre v-syntax-highlight="'go'"><code v-pre>http.Handle("/widget", engine.ServeComponent("Widget", func(r *http.Request) map[string]any {
    return map[string]any{"id": r.URL.Query().Get("id")}
}))</code></pre>

    <h3>ServePageComponent</h3>
    <p>
      Like <code>ServeComponent</code> but renders a full HTML page (injecting scoped
      styles into <code>&lt;/head&gt;</code>) and lets the data function return an HTTP
      status code alongside the data map. A status code of 0 is treated as 200.
    </p>
    <pre v-syntax-highlight="'go'"><code v-pre>http.Handle("/post", engine.ServePageComponent("PostPage",
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
    <pre v-syntax-highlight="'go'"><code v-pre>engine.Mount(mux, map[string]string{
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
    <pre v-syntax-highlight="'go'"><code v-pre>engine.WithDataMiddleware(func(r *http.Request, data map[string]any) map[string]any {
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
    <pre v-syntax-highlight="'go'"><code v-pre>if errs := engine.ValidateAll(); len(errs) > 0 {
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
    <pre v-syntax-highlight="'go'"><code v-pre>// Abort the render with an error on any missing prop
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
    <pre v-syntax-highlight="'go'"><code v-pre>engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    t, _ := args[0].(time.Time)
    return t.Format("2 Jan 2006"), nil
})

engine.RegisterFunc("url", func(args ...any) (any, error) {
    name, _ := args[0].(string)
    return router.URLFor(name), nil
})</code></pre>

    <p>Use them directly in templates:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;span&gt;{{ formatDate(post.CreatedAt) }}&lt;/span&gt;
&lt;a :href="url('home')"&gt;Home&lt;/a&gt;</code></pre>

    <h2 id="advanced-options">Advanced options</h2>

    <h3>Hot-reload</h3>
    <p>
      Set <code>Reload: true</code> to re-parse changed <code>.vue</code> files
      automatically before each render — no server restart required. Disable in
      production.
    </p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{
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
    <pre v-syntax-highlight="'go'"><code v-pre>import "embed"

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
    <pre v-syntax-highlight="'go'"><code v-pre>ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
defer cancel()

err = engine.RenderPageContext(ctx, w, "Page", data)
err = engine.RenderFragmentContext(ctx, w, "Card", data)</code></pre>

    <h2 id="errors">Error handling</h2>
    <p>
      Parse and render failures carry structured location information.
      Use <code>errors.As</code> to inspect them:
    </p>
    <pre v-syntax-highlight="'go'"><code v-pre>_, err := htmlc.ParseFile("Card.vue", src)
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
    <pre v-syntax-highlight="'text'"><code v-pre>Card.vue:14:5: render Card.vue: expr "post.Title": cannot access property "Title" of null
  13 |   &lt;div class="card"&gt;
&gt; 14 |     {{ post.Title }}
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
          <td><code v-pre>RenderPage</code> / <code>RenderFragment</code> data map</td>
          <td>Yes</td>
          <td>No — pass as props</td>
        </tr>
        <tr>
          <td><code v-pre>WithDataMiddleware</code> values</td>
          <td>Yes</td>
          <td>No — pass as props</td>
        </tr>
        <tr>
          <td><code v-pre>RegisterFunc</code> functions</td>
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
    <pre v-syntax-highlight="'go'"><code v-pre>engine.RegisterDirective("v-highlight", func(ctx *htmlc.DirectiveContext) error {
    // ctx.Node  — the HTML node being rendered
    // ctx.Value — the directive value expression result
    // ctx.Scope — the current render scope
    ctx.Node.Attr = append(ctx.Node.Attr, html.Attribute{
        Key: "class", Val: "highlighted",
    })
    return nil
})</code></pre>
  </DocsPage>
</template>
