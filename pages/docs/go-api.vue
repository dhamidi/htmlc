<template>
  <DocsPage
    pageTitle="Go API Reference — htmlc.sh"
    description="Complete reference for every exported type, function, method, and option in the htmlc Go package."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Engine'},
      {href: '#creating-engine', label: 'New / Options'},
      {href: '#expvars', label: 'Runtime Metrics (expvars)'},
      {href: '#slog', label: 'Structured Logging (slog)'},
      {href: '#component-management', label: 'Register / Has / Components'},
      {href: '#validate', label: 'ValidateAll'},
      {label: 'Rendering'},
      {href: '#render-page', label: 'RenderPage'},
      {href: '#render-fragment', label: 'RenderFragment'},
      {href: '#render-string', label: 'String helpers'},
      {href: '#render-context', label: 'Context variants'},
      {label: 'HTTP'},
      {href: '#serve-component', label: 'ServeComponent'},
      {href: '#serve-page-component', label: 'ServePageComponent'},
      {href: '#mount', label: 'Mount'},
      {href: '#data-middleware', label: 'WithDataMiddleware'},
      {label: 'Customization'},
      {href: '#register-func', label: 'RegisterFunc'},
      {href: '#register-directive', label: 'RegisterDirective'},
      {href: '#missing-prop', label: 'Missing prop handling'},
      {label: 'Low-level API'},
      {href: '#parse-file', label: 'ParseFile / Component'},
      {href: '#renderer', label: 'Renderer'},
      {href: '#registry', label: 'Registry'},
      {label: 'Directives'},
      {href: '#directive-interface', label: 'Directive interface'},
      {href: '#directive-types', label: 'DirectiveBinding / Context'},
      {href: '#directive-registry', label: 'DirectiveRegistry'},
      {href: '#directive-with-content', label: 'DirectiveWithContent'},
      {label: 'Styles'},
      {href: '#style-collector', label: 'StyleCollector'},
      {href: '#style-helpers', label: 'ScopeID / ScopeCSS'},
      {label: 'Errors'},
      {href: '#error-types', label: 'Error types'},
      {href: '#sentinel-errors', label: 'Sentinel errors'}
    ]"
  >
    <h1>Go API Reference</h1>
    <p class="lead">Complete reference for every exported symbol in the <code>htmlc</code> package. Import path: <code>github.com/dhamidi/htmlc</code>.</p>

    <!-- ═══════════════════════════════════════════════ Creating an Engine -->
    <h2 id="creating-engine">Creating an Engine</h2>

    <h3 id="new">New</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func New(opts Options) (*Engine, error)</code></pre>
    <p>Creates an Engine from <code>opts</code>. If <code>opts.ComponentDir</code> is set the directory is walked recursively and all <code>*.vue</code> files are registered before the engine is returned.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
})
if err != nil {
    log.Fatal(err)
}</code></pre>

    <h3 id="options">Options</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type Options struct {
    ComponentDir string
    Reload       bool
    FS           fs.FS
    Directives   DirectiveRegistry
    Debug        bool
}</code></pre>
    <table>
      <thead>
        <tr><th>Field</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code v-pre>ComponentDir</code></td>
          <td>Directory walked recursively for <code>*.vue</code> files. Each file is registered by its base name without extension (<code>Button.vue</code> → <code>Button</code>). When two files share the same base name the last one in lexical order wins.</td>
        </tr>
        <tr>
          <td><code v-pre>Reload</code></td>
          <td>When true the engine checks the modification time of every registered file before each render and re-parses changed files automatically. Use during development; leave false in production.</td>
        </tr>
        <tr>
          <td><code v-pre>FS</code></td>
          <td>When set, all file reads and directory walks use this <code>fs.FS</code> instead of the OS filesystem. <code>ComponentDir</code> is interpreted as a path within the FS. Useful with <code>//go:embed</code>. Hot-reload requires the FS to also implement <code>fs.StatFS</code>.</td>
        </tr>
        <tr>
          <td><code v-pre>Directives</code></td>
          <td>Custom directives available to all components rendered by this engine. Keys are directive names without the <code>v-</code> prefix. Built-in directives cannot be overridden.</td>
        </tr>
        <tr>
          <td><code v-pre>Debug</code></td>
          <td>When true the rendered HTML is annotated with HTML comments describing component boundaries, expression values, and slot contents. Development use only.</td>
        </tr>
      </tbody>
    </table>

    <!-- ═══════════════════════════════════════════════ Runtime Metrics -->
    <h2 id="expvars">Runtime Metrics (expvars)</h2>

    <h3 id="publish-expvars">PublishExpvars</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) PublishExpvars(prefix string) *Engine</code></pre>
    <p>Registers the engine's metrics in the global <code>expvar</code> registry under <code>prefix</code>. After calling this method all counters and configuration state are visible at the <code>/debug/vars</code> HTTP endpoint as a JSON object keyed by <code>prefix</code>. Returns the engine for method chaining.</p>
    <p>Internally this creates a top-level <code>*expvar.Map</code> via <code>expvar.NewMap(prefix)</code>. It panics if called twice with the same prefix — the same semantics as <code>expvar.NewMap</code>. Call it exactly once per engine per process, immediately after creating the engine.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{ComponentDir: &#34;./components&#34;})
if err != nil {
    log.Fatal(err)
}
engine.PublishExpvars(&#34;myapp&#34;) // registers metrics under &#34;myapp&#34; key</code></pre>

    <h3 id="expvars-table">Exposed variables</h3>
    <p>All variables appear as children of the <code>prefix</code> map in the <code>/debug/vars</code> JSON output:</p>
    <table>
      <thead>
        <tr><th>Key</th><th>Type</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code>reload</code></td>
          <td><code>expvar.Int</code> (0/1)</td>
          <td>Whether hot-reload is enabled.</td>
        </tr>
        <tr>
          <td><code>debug</code></td>
          <td><code>expvar.Int</code> (0/1)</td>
          <td>Whether debug mode is enabled.</td>
        </tr>
        <tr>
          <td><code>componentDir</code></td>
          <td><code>expvar.String</code></td>
          <td>Current component directory path.</td>
        </tr>
        <tr>
          <td><code>fs</code></td>
          <td><code>expvar.String</code></td>
          <td>Type name of the current <code>fs.FS</code>, or <code>"&lt;nil&gt;"</code> when no custom FS is set.</td>
        </tr>
        <tr>
          <td><code>renders</code></td>
          <td><code>expvar.Int</code></td>
          <td>Cumulative number of successful render calls.</td>
        </tr>
        <tr>
          <td><code>renderErrors</code></td>
          <td><code>expvar.Int</code></td>
          <td>Cumulative number of failed render calls.</td>
        </tr>
        <tr>
          <td><code>reloads</code></td>
          <td><code>expvar.Int</code></td>
          <td>Cumulative number of hot-reload scans performed.</td>
        </tr>
        <tr>
          <td><code>renderNanos</code></td>
          <td><code>expvar.Int</code></td>
          <td>Cumulative render time in nanoseconds.</td>
        </tr>
        <tr>
          <td><code>components</code></td>
          <td><code>expvar.Func</code> → Int</td>
          <td>Number of currently registered components (computed on each read).</td>
        </tr>
        <tr>
          <td><code>info.directives</code></td>
          <td><code>expvar.Func</code> → Array</td>
          <td>Sorted list of registered custom directive names (computed on each read).</td>
        </tr>
      </tbody>
    </table>

    <h3 id="setter-methods">Setter methods</h3>
    <p>These methods update both the live engine option and the corresponding expvar immediately. They are usable whether or not <code>PublishExpvars</code> has been called.</p>

    <h4>SetDebug</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) SetDebug(enabled bool)</code></pre>
    <p>Enables or disables debug mode. When <code>enabled</code> is true, rendered HTML is annotated with comments describing component boundaries, expression values, and slot contents. Updates the <code>debug</code> expvar to <code>1</code> or <code>0</code> immediately.</p>

    <h4>SetReload</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) SetReload(enabled bool)</code></pre>
    <p>Enables or disables hot-reload. When <code>enabled</code> is true, the engine stats every registered component file before each render and re-parses any that have changed. Updates the <code>reload</code> expvar to <code>1</code> or <code>0</code> immediately.</p>

    <h4>SetComponentDir</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) SetComponentDir(dir string) error</code></pre>
    <p>Changes the component directory to <code>dir</code>, walks the new directory recursively, and rebuilds the component registry. Returns an error if the directory cannot be walked. Updates the <code>componentDir</code> expvar to the new path immediately.</p>

    <h4>SetFS</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) SetFS(fsys fs.FS) error</code></pre>
    <p>Replaces the engine's filesystem with <code>fsys</code>, then rebuilds the component registry by walking the current <code>ComponentDir</code> inside the new FS. Returns an error if the directory walk fails. Updates the <code>fs</code> expvar to the type name of <code>fsys</code> (or <code>"&lt;nil&gt;"</code>) immediately.</p>

    <p><strong>Integration note:</strong> The <code>/debug/vars</code> handler is registered on <code>http.DefaultServeMux</code> automatically when the <code>expvar</code> package is imported. For a custom <code>*http.ServeMux</code>, register it explicitly:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>import &#34;expvar&#34;

mux.Handle(&#34;GET /debug/vars&#34;, expvar.Handler())</code></pre>

    <!-- ═══════════════════════════════════════════════ Structured Logging -->
    <h2 id="slog">Structured Logging (slog)</h2>

    <h3 id="options-logger">Options.Logger</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type Options struct {
    // ...
    Logger *slog.Logger
}</code></pre>
    <table>
      <thead>
        <tr><th>Detail</th><th>Value</th></tr>
      </thead>
      <tbody>
        <tr>
          <td>Type</td>
          <td><code>*slog.Logger</code></td>
        </tr>
        <tr>
          <td>Default</td>
          <td><code>nil</code> (no logging)</td>
        </tr>
        <tr>
          <td>Effect when nil</td>
          <td>A single pointer-nil check is performed per render; no allocations, no timing, behaviour identical to before structured logging was added.</td>
        </tr>
        <tr>
          <td>Effect when non-nil</td>
          <td>One structured log record is emitted for every component in the render tree. Records include component name, render duration, bytes written, and (on failure) the error.</td>
        </tr>
      </tbody>
    </table>
    <Callout><strong>Minimum Go version:</strong> <code>log/slog</code> was added in Go 1.21. Passing a non-nil <code>Logger</code> requires Go 1.21 or later.</Callout>

    <h3 id="slog-record">Log record specification</h3>
    <p>Every rendered component emits one record. The attributes are:</p>
    <table>
      <thead>
        <tr><th>Attribute key</th><th>slog type</th><th>Value</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code>component</code></td>
          <td><code>slog.String</code></td>
          <td>Resolved component name (e.g. <code>NavBar</code>)</td>
        </tr>
        <tr>
          <td><code>duration</code></td>
          <td><code>slog.Duration</code></td>
          <td>Wall-clock time for the component subtree. Text handlers format this as <code>1.2ms</code>; JSON handlers emit it as nanoseconds (<code>int64</code>).</td>
        </tr>
        <tr>
          <td><code>bytes</code></td>
          <td><code>slog.Int64</code></td>
          <td>Bytes written by the component subtree.</td>
        </tr>
        <tr>
          <td><code>error</code></td>
          <td><code>slog.Any</code></td>
          <td>Non-nil only on <code>LevelError</code> records for failed renders.</td>
        </tr>
      </tbody>
    </table>
    <p>Example text-handler output:</p>
    <pre v-syntax-highlight="'text'"><code v-pre>time=2026-03-16T12:00:00.001Z level=DEBUG msg="component rendered" component=NavLink duration=1.2ms bytes=142
time=2026-03-16T12:00:00.002Z level=DEBUG msg="component rendered" component=NavBar duration=4.5ms bytes=612
time=2026-03-16T12:00:00.149Z level=DEBUG msg="component rendered" component=HomePage duration=148.6ms bytes=24576</code></pre>
    <p>Example JSON-handler output (note: <code>duration</code> is nanoseconds as <code>int64</code>):</p>
    <pre v-syntax-highlight="'json'"><code v-pre>{&#34;time&#34;:&#34;2026-03-16T12:00:00.001Z&#34;,&#34;level&#34;:&#34;DEBUG&#34;,&#34;msg&#34;:&#34;component rendered&#34;,&#34;component&#34;:&#34;NavLink&#34;,&#34;duration&#34;:1200000,&#34;bytes&#34;:142}
{&#34;time&#34;:&#34;2026-03-16T12:00:00.149Z&#34;,&#34;level&#34;:&#34;DEBUG&#34;,&#34;msg&#34;:&#34;component rendered&#34;,&#34;component&#34;:&#34;HomePage&#34;,&#34;duration&#34;:148600000,&#34;bytes&#34;:24576}</code></pre>

    <h3 id="slog-levels">Log levels</h3>
    <ul>
      <li><strong>Successful render</strong> — emitted at <code>slog.LevelDebug</code> with message <code>htmlc.MsgComponentRendered</code>.</li>
      <li><strong>Failed render</strong> — emitted at <code>slog.LevelError</code> with message <code>htmlc.MsgComponentFailed</code>, plus a non-nil <code>error</code> attribute.</li>
    </ul>

    <h3 id="slog-constants">Constants</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>const MsgComponentRendered = &#34;component rendered&#34;
const MsgComponentFailed   = &#34;component render failed&#34;</code></pre>
    <p>These constants are the stable message strings used in every log record. Reference them in log-based test assertions or alerting rules so that a future rename is caught at compile time rather than by a broken alert:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// Log assertion in a test:
if record.Message != htmlc.MsgComponentRendered {
    t.Errorf(&#34;unexpected message: %q&#34;, record.Message)
}</code></pre>

    <h3 id="with-logger">Renderer.WithLogger</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (r *Renderer) WithLogger(l *slog.Logger) *Renderer</code></pre>
    <p>Attaches <code>l</code> to the renderer. Use this when working with the low-level <code>Renderer</code> API directly — for example, to attach a request-scoped logger enriched with <code>logger.With("request_id", id)</code> to a per-request renderer. Returns <code>r</code> for method chaining.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>renderer := htmlc.NewRenderer(component).
    WithComponents(reg).
    WithLogger(logger.With(&#34;request_id&#34;, requestID))</code></pre>

    <h3 id="slog-ordering">Record ordering</h3>
    <p>Records are emitted in <strong>leaf-first</strong> (post-order) order because the log call fires after the entire component subtree finishes rendering. The root page component is always the last record. This is useful for identifying which child component is the bottleneck: the first record with a high <code>duration</code> is where time is actually spent.</p>

    <h3 id="slog-performance">Performance</h3>
    <p>When <code>Options.Logger</code> is <code>nil</code> (the default), a single pointer-nil check is the only overhead per component. No timing, no allocations — behaviour is identical to before structured logging was added. Logging costs are incurred only when a non-nil logger is supplied.</p>

    <!-- ═══════════════════════════════════════════════ Rendering -->
    <h2 id="rendering">Rendering</h2>

    <h3 id="render-page">RenderPage</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RenderPage(w io.Writer, name string, data map[string]any) error</code></pre>
    <p>Renders the named component as a full HTML page and writes the result to <code>w</code>. Scoped styles are collected from the entire component tree and injected as a <code>&lt;style&gt;</code> block immediately before the first <code>&lt;/head&gt;</code> tag. If no <code>&lt;/head&gt;</code> is found the style block is prepended to the output.</p>
    <p>Use <code>RenderPage</code> for page components that include <code>&lt;!DOCTYPE html&gt;</code>, <code>&lt;html&gt;</code>, <code>&lt;head&gt;</code>, and <code>&lt;body&gt;</code>.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>var buf bytes.Buffer
err := engine.RenderPage(&amp;buf, "HomePage", map[string]any{
    "title": "Welcome",
})</code></pre>

    <h3 id="render-fragment">RenderFragment</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RenderFragment(w io.Writer, name string, data map[string]any) error</code></pre>
    <p>Renders the named component as an HTML fragment and prepends the collected <code>&lt;style&gt;</code> block to the output. Does not search for a <code>&lt;/head&gt;</code> tag. Use for partial page updates such as HTMX responses or turbo-frame updates.</p>

    <h3 id="render-string">String helpers</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RenderPageString(name string, data map[string]any) (string, error)
func (e *Engine) RenderFragmentString(name string, data map[string]any) (string, error)</code></pre>
    <p>Convenience wrappers around <code>RenderPage</code> and <code>RenderFragment</code> that return the result as a string instead of writing to an <code>io.Writer</code>.</p>

    <h3 id="render-context">Context variants</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderFragmentContext(ctx context.Context, w io.Writer, name string, data map[string]any) error</code></pre>
    <p>Like <code>RenderPage</code> and <code>RenderFragment</code> but accept a <code>context.Context</code>. The render is aborted and <code>ctx.Err()</code> is returned if the context is cancelled or its deadline is exceeded.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
err := engine.RenderPageContext(ctx, w, "HomePage", data)</code></pre>

    <!-- ═══════════════════════════════════════════════ HTTP Integration -->
    <h2 id="http">HTTP Integration</h2>

    <h3 id="serve-component">ServeComponent</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc</code></pre>
    <p>Returns an <code>http.HandlerFunc</code> that renders <code>name</code> as a fragment on every request. The <code>data</code> function is called per-request to build the template scope; it may be <code>nil</code>. Data middleware registered via <code>WithDataMiddleware</code> is applied after the data function. Sets <code>Content-Type: text/html; charset=utf-8</code>.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>mux.HandleFunc("GET /search-results", engine.ServeComponent("SearchResults", func(r *http.Request) map[string]any {
    return map[string]any{"query": r.URL.Query().Get("q")}
}))</code></pre>

    <h3 id="serve-page-component">ServePageComponent</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) ServePageComponent(name string, data func(*http.Request) (map[string]any, int)) http.HandlerFunc</code></pre>
    <p>Returns an <code>http.HandlerFunc</code> that renders <code>name</code> as a full HTML page. The data function returns both the template scope and the HTTP status code to send. A status code of <code>0</code> is treated as <code>200</code>. If <code>data</code> is <code>nil</code> a <code>200 OK</code> response with no template data is used.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>mux.HandleFunc("GET /post/{id}", engine.ServePageComponent("PostPage", func(r *http.Request) (map[string]any, int) {
    post, err := db.GetPost(r.PathValue("id"))
    if err != nil {
        return map[string]any{"error": err.Error()}, http.StatusNotFound
    }
    return map[string]any{"post": post}, http.StatusOK
}))</code></pre>

    <h3 id="mount">Mount</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) Mount(mux *http.ServeMux, routes map[string]string)</code></pre>
    <p>Registers multiple component routes on <code>mux</code> at once. Keys are <code>http.ServeMux</code> patterns (e.g. <code>"GET /{$}"</code>, <code>"GET /about"</code>); values are component names. Each component is served as a full page via <code>ServePageComponent</code> with no data function.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine.Mount(mux, map[string]string{
    "GET /{$}":    "HomePage",
    "GET /about":  "AboutPage",
    "GET /blog":   "BlogPage",
})</code></pre>

    <h3 id="data-middleware">WithDataMiddleware</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) WithDataMiddleware(fn func(*http.Request, map[string]any) map[string]any) *Engine</code></pre>
    <p>Adds a function that augments the data map on every HTTP-triggered render. Multiple middleware functions are called in registration order; later middleware can overwrite keys set by earlier ones. Returns the engine for chaining.</p>
    <p>Data middleware applies only to the top-level render scope and is not automatically propagated into child component scopes. Pass shared values via <code>RegisterFunc</code> if child components need them.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine.WithDataMiddleware(func(r *http.Request, data map[string]any) map[string]any {
    data["currentUser"] = userFromRequest(r)
    data["csrfToken"]   = csrf.Token(r)
    return data
})</code></pre>

    <!-- ═══════════════════════════════════════════════ Component Management -->
    <h2 id="component-management">Component Management</h2>

    <h3 id="register">Register</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) Register(name, path string) error</code></pre>
    <p>Manually adds a component from <code>path</code> to the engine's registry under <code>name</code>, without requiring a directory scan. Useful for programmatically generated components or files outside <code>ComponentDir</code>.</p>

    <h3 id="has">Has</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) Has(name string) bool</code></pre>
    <p>Reports whether <code>name</code> is a registered component.</p>

    <h3 id="components">Components</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) Components() []string</code></pre>
    <p>Returns the names of all registered components in sorted order. Automatic lowercase aliases added by the engine are excluded.</p>

    <h3 id="validate">ValidateAll</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) ValidateAll() []ValidationError</code></pre>
    <p>Checks every registered component for unresolvable child component references. Returns one <code>ValidationError</code> per problem; an empty slice means all components are valid. Call once at application startup to surface missing-component issues early.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>if errs := engine.ValidateAll(); len(errs) != 0 {
    for _, e := range errs {
        log.Println(e)
    }
    os.Exit(1)
}</code></pre>

    <!-- ═══════════════════════════════════════════════ Customization -->
    <h2 id="customization">Customization</h2>

    <h3 id="register-func">RegisterFunc</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RegisterFunc(name string, fn func(...any) (any, error)) *Engine</code></pre>
    <p>Adds a per-engine function available in all template expressions. The function is callable from templates as <code>name()</code>. Engine functions have lower priority than user-provided data keys. They are propagated automatically into every child component's scope.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    t, ok := args[0].(time.Time)
    if !ok {
        return "", fmt.Errorf("formatDate: expected time.Time")
    }
    return t.Format("Jan 2, 2006"), nil
})</code></pre>

    <h3 id="register-directive">RegisterDirective</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RegisterDirective(name string, dir Directive)</code></pre>
    <p>Adds a custom directive under <code>name</code> (without the <code>v-</code> prefix). Replaces any previously registered directive with the same name. Panics if <code>dir</code> is nil.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>engine.RegisterDirective("tooltip", &amp;TooltipDirective{})</code></pre>

    <h3 id="missing-prop">WithMissingPropHandler</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) WithMissingPropHandler(fn MissingPropFunc) *Engine</code></pre>
    <p>Sets the function called when any component has a missing prop. The default behaviour renders a visible <code>[missing: &lt;name&gt;]</code> placeholder. Use <code>ErrorOnMissingProp</code> for strict error behaviour or <code>SubstituteMissingProp</code> for legacy placeholder text.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// Abort rendering on any missing prop
engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)

// Substitute a legacy placeholder string
engine.WithMissingPropHandler(htmlc.SubstituteMissingProp)</code></pre>

    <!-- ═══════════════════════════════════════════════ Low-level API -->
    <h2 id="low-level">Low-level API</h2>

    <h3 id="parse-file">ParseFile</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func ParseFile(path, src string) (*Component, error)</code></pre>
    <p>Parses a <code>.vue</code> Single File Component from the string <code>src</code>. <code>path</code> is used only for error messages and the scope attribute hash. Returns a <code>*Component</code> whose <code>Template</code> field is a parsed HTML node tree. Only the top-level <code>&lt;template&gt;</code>, <code>&lt;script&gt;</code>, and <code>&lt;style&gt;</code> sections are extracted; <code>&lt;template&gt;</code> is required.</p>

    <h3 id="component-type">Component</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type Component struct {
    Template *html.Node // root of the parsed template node tree
    Script   string     // raw &lt;script&gt; content (empty if absent)
    Style    string     // raw &lt;style&gt; content (empty if absent)
    Scoped   bool       // true when &lt;style scoped&gt; was present
    Path     string     // source file path passed to ParseFile
    Source   string     // raw source text (for error location reporting)
    Warnings []string   // non-fatal issues found during parsing
}</code></pre>
    <p>Holds the parsed representation of a <code>.vue</code> Single File Component. The <code>Warnings</code> field may contain notices about auto-corrected self-closing component tags.</p>

    <h3 id="props-method">Component.Props</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (c *Component) Props() []PropInfo</code></pre>
    <p>Walks the template AST and returns all top-level variable references the template uses. Identifiers starting with <code>$</code> and <code>v-for</code> loop variables are excluded.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>type PropInfo struct {
    Name        string   // identifier name
    Expressions []string // template expressions in which it appears
}</code></pre>

    <h3 id="registry">Registry</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type Registry map[string]*Component</code></pre>
    <p>Maps component names to their parsed components. Keys may be PascalCase or kebab-case. Most callers use <code>Engine</code>, which builds and maintains a Registry automatically.</p>

    <h3 id="renderer">Renderer</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func NewRenderer(c *Component) *Renderer</code></pre>
    <p>Creates a Renderer for <code>c</code>. Use the builder methods below to configure it before calling <code>Render</code>.</p>

    <p>Renderer is the low-level rendering primitive. Most callers should use <code>Engine</code> via <code>RenderPage</code> or <code>RenderFragment</code>. Use <code>NewRenderer</code> when you need fine-grained control over style collection or registry attachment.</p>

    <table>
      <thead>
        <tr><th>Builder method</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code v-pre>WithStyles(sc *StyleCollector) *Renderer</code></td>
          <td>Sets the <code>StyleCollector</code> that receives this component's style contribution.</td>
        </tr>
        <tr>
          <td><code v-pre>WithComponents(reg Registry) *Renderer</code></td>
          <td>Attaches a component registry, enabling component composition.</td>
        </tr>
        <tr>
          <td><code v-pre>WithMissingPropHandler(fn MissingPropFunc) *Renderer</code></td>
          <td>Sets the handler called when a template prop is absent from the scope.</td>
        </tr>
        <tr>
          <td><code v-pre>WithDirectives(dr DirectiveRegistry) *Renderer</code></td>
          <td>Attaches a custom directive registry.</td>
        </tr>
        <tr>
          <td><code v-pre>WithContext(ctx context.Context) *Renderer</code></td>
          <td>Attaches a <code>context.Context</code>; the render is aborted on cancellation.</td>
        </tr>
        <tr>
          <td><code v-pre>WithFuncs(funcs map[string]any) *Renderer</code></td>
          <td>Attaches engine-registered functions so they are available in expressions and propagated to all child renderers.</td>
        </tr>
      </tbody>
    </table>

    <h3 id="renderer-render">Renderer.Render / RenderString</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (r *Renderer) Render(w io.Writer, scope map[string]any) error
func (r *Renderer) RenderString(scope map[string]any) (string, error)</code></pre>
    <p>Evaluates the component's template against <code>scope</code> and writes HTML to <code>w</code> (or returns it as a string). Prop validation and style collection happen here.</p>

    <h3 id="package-helpers">Package-level helpers</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func Render(w io.Writer, c *Component, scope map[string]any) error
func RenderString(c *Component, scope map[string]any) (string, error)</code></pre>
    <p>Convenience wrappers that create a temporary <code>Renderer</code> for <code>c</code>. They do not collect styles or support component composition. Use <code>NewRenderer</code> with <code>WithStyles</code> and <code>WithComponents</code> when those features are needed.</p>

    <!-- ═══════════════════════════════════════════════ Directives -->
    <h2 id="directives">Directives</h2>

    <h3 id="directive-interface">Directive interface</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type Directive interface {
    Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
    Mounted(w io.Writer, node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
}</code></pre>
    <p>Implemented by custom directive types. Register with <code>Engine.RegisterDirective</code> or pass in <code>Options.Directives</code>. Only the <code>Created</code> and <code>Mounted</code> hooks are called because htmlc renders server-side.</p>
    <ul>
      <li><code v-pre>Created</code> — called before the element is rendered. Receives a shallow-cloned working node whose <code>Attr</code> slice and <code>Data</code> field may be freely mutated. Return a non-nil error to abort rendering.</li>
      <li><code v-pre>Mounted</code> — called after the element's closing tag has been written to <code>w</code>. Bytes written to <code>w</code> appear immediately after the element. Return a non-nil error to abort rendering.</li>
    </ul>

    <h3 id="directive-types">DirectiveBinding / DirectiveContext</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type DirectiveBinding struct {
    Value     any             // result of evaluating the directive expression
    RawExpr   string          // un-evaluated expression string from the template
    Arg       string          // directive argument after the colon, e.g. "href" in v-bind:href
    Modifiers map[string]bool // dot-separated modifiers, e.g. {"prevent": true}
}</code></pre>
    <pre v-syntax-highlight="'go'"><code v-pre>type DirectiveContext struct {
    Registry          Registry // component registry the renderer is using
    RenderedChildHTML string   // fully rendered inner HTML of the host element; empty for void elements
}</code></pre>
    <p><code v-pre>RenderedChildHTML</code> contains the fully rendered children of the directive's host element — all template expressions evaluated, child components expanded — before either hook runs. It is available in both <code>Created</code> and <code>Mounted</code>. Use it to inspect or transform the rendered subtree, for example in a syntax-highlighting directive that needs the source text after expression evaluation.</p>

    <h3 id="directive-registry">DirectiveRegistry</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type DirectiveRegistry map[string]Directive</code></pre>
    <p>Maps directive names (without the <code>v-</code> prefix) to their implementations. Keys are lower-kebab-case; the renderer normalises names before lookup.</p>

    <h3 id="directive-with-content">DirectiveWithContent</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type DirectiveWithContent interface {
    Directive
    InnerHTML() (html string, ok bool)
}</code></pre>
    <p>Optional extension of <code>Directive</code>. When a directive's <code>Created</code> hook wants to replace the element's children with custom HTML, implement this interface. The renderer calls <code>InnerHTML</code> after <code>Created</code> returns; if it returns a non-empty string, the element's template children are skipped and the returned HTML is written verbatim between the opening and closing tags (equivalent to <code>v-html</code> on the element itself). Return <code>("", false)</code> to fall back to normal child rendering.</p>
    <pre v-syntax-highlight="'go'"><code v-pre>// Example: a directive that wraps children in a callout box
type CalloutDirective struct {
    renderedHTML string
}

func (d *CalloutDirective) Created(node *html.Node, b htmlc.DirectiveBinding, ctx htmlc.DirectiveContext) error {
    d.renderedHTML = ctx.RenderedChildHTML
    return nil
}

func (d *CalloutDirective) Mounted(w io.Writer, node *html.Node, b htmlc.DirectiveBinding, ctx htmlc.DirectiveContext) error {
    return nil
}

func (d *CalloutDirective) InnerHTML() (string, bool) {
    h := d.renderedHTML
    d.renderedHTML = ""
    if h == "" {
        return "", false
    }
    return `&lt;div class="callout"&gt;` + h + `&lt;/div&gt;`, true
}</code></pre>

    <!-- ═══════════════════════════════════════════════ Style Collection -->
    <h2 id="styles">Style Collection</h2>

    <h3 id="style-collector">StyleCollector</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type StyleCollector struct { /* unexported fields */ }</code></pre>
    <p>Accumulates <code>StyleContribution</code> values from one or more component renders into a single ordered list, deduplicating contributions from the same scoped component. <code>Engine</code> creates and manages a <code>StyleCollector</code> automatically on each render call.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>func (sc *StyleCollector) Add(c StyleContribution)
func (sc *StyleCollector) All() []StyleContribution</code></pre>
    <ul>
      <li><code v-pre>Add</code> — appends <code>c</code>, skipping duplicates. Two contributions are duplicates when they share the same composite key (<code>ScopeID + CSS</code>).</li>
      <li><code v-pre>All</code> — returns all contributions in the order they were added.</li>
    </ul>

    <h3 id="style-contribution">StyleContribution</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type StyleContribution struct {
    ScopeID string // scope attribute name (e.g. "data-v-a1b2c3d4"), empty for global styles
    CSS     string // stylesheet text, already rewritten by ScopeCSS for scoped components
}</code></pre>

    <h3 id="style-helpers">ScopeID / ScopeCSS</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func ScopeID(path string) string
func ScopeCSS(css, scopeAttr string) string</code></pre>
    <ul>
      <li><code v-pre>ScopeID</code> — returns <code>"data-v-"</code> followed by 8 hex digits derived from the FNV-1a 32-bit hash of <code>path</code>.</li>
      <li><code v-pre>ScopeCSS</code> — rewrites <code>css</code> so that every selector in every non-<code>@</code>-rule has <code>scopeAttr</code> appended to its last compound selector. <code>@</code>-rules are passed through verbatim.</li>
    </ul>

    <!-- ═══════════════════════════════════════════════ Error Handling -->
    <h2 id="errors">Error Handling</h2>

    <h3 id="error-types">Error types</h3>

    <h4>ParseError</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>type ParseError struct {
    Path     string          // source file path
    Msg      string          // human-readable description
    Location *SourceLocation // source position, or nil if unknown
}</code></pre>
    <p>Returned by <code>ParseFile</code> when a <code>.vue</code> file cannot be parsed.</p>

    <h4>RenderError</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>type RenderError struct {
    Component string          // component name being rendered
    Expr      string          // expression that triggered the error (may be empty)
    Wrapped   error           // underlying error
    Location  *SourceLocation // source position, or nil if unknown
}</code></pre>
    <p>Returned by render methods when template evaluation fails. Implements <code>Unwrap</code>.</p>

    <h4>ValidationError</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>type ValidationError struct {
    Component string // name of the component with the problem
    Message   string // description of the problem
}</code></pre>
    <p>One entry per problem in the slice returned by <code>ValidateAll</code>.</p>

    <h4>SourceLocation</h4>
    <pre v-syntax-highlight="'go'"><code v-pre>type SourceLocation struct {
    File    string // source file path
    Line    int    // 1-based line number (0 = unknown)
    Column  int    // 1-based column (0 = unknown)
    Snippet string // ~3-line context around the error (may be empty)
}</code></pre>

    <h3 id="missing-prop-types">MissingPropFunc / handlers</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>type MissingPropFunc func(name string) (any, error)

func ErrorOnMissingProp(name string) (any, error)
func SubstituteMissingProp(name string) (any, error)</code></pre>
    <ul>
      <li><code v-pre>MissingPropFunc</code> — signature for missing-prop handlers; receive the prop name, return a substitute value or an error.</li>
      <li><code v-pre>ErrorOnMissingProp</code> — aborts rendering with an error whenever a prop is absent. Use for strict validation.</li>
      <li><code v-pre>SubstituteMissingProp</code> — returns <code>"MISSING PROP: &lt;name&gt;"</code> as a placeholder string.</li>
    </ul>

    <h3 id="sentinel-errors">Sentinel errors</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>var ErrComponentNotFound = errors.New("htmlc: component not found")
var ErrMissingProp       = errors.New("htmlc: missing required prop")</code></pre>
    <ul>
      <li><code v-pre>ErrComponentNotFound</code> — wrapped inside the error returned by render methods when the requested component name is not registered.</li>
      <li><code v-pre>ErrMissingProp</code> — returned (wrapped) when a required prop is absent and no <code>MissingPropFunc</code> has been set.</li>
    </ul>
    <pre v-syntax-highlight="'go'"><code v-pre>if errors.Is(err, htmlc.ErrComponentNotFound) {
    http.NotFound(w, r)
    return
}</code></pre>

  </DocsPage>
</template>

