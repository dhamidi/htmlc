<template>
  <Layout pageTitle="Go API Reference — htmlc.sh" description="Complete reference for every exported type, function, method, and option in the htmlc Go package." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-label">Engine</div>
          <a href="#creating-engine" class="sidebar-link">New / Options</a>
          <a href="#component-management" class="sidebar-link">Register / Has / Components</a>
          <a href="#validate" class="sidebar-link">ValidateAll</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Rendering</div>
          <a href="#render-page" class="sidebar-link">RenderPage</a>
          <a href="#render-fragment" class="sidebar-link">RenderFragment</a>
          <a href="#render-string" class="sidebar-link">String helpers</a>
          <a href="#render-context" class="sidebar-link">Context variants</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">HTTP</div>
          <a href="#serve-component" class="sidebar-link">ServeComponent</a>
          <a href="#serve-page-component" class="sidebar-link">ServePageComponent</a>
          <a href="#mount" class="sidebar-link">Mount</a>
          <a href="#data-middleware" class="sidebar-link">WithDataMiddleware</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Customization</div>
          <a href="#register-func" class="sidebar-link">RegisterFunc</a>
          <a href="#register-directive" class="sidebar-link">RegisterDirective</a>
          <a href="#missing-prop" class="sidebar-link">Missing prop handling</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Low-level API</div>
          <a href="#parse-file" class="sidebar-link">ParseFile / Component</a>
          <a href="#renderer" class="sidebar-link">Renderer</a>
          <a href="#registry" class="sidebar-link">Registry</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Directives</div>
          <a href="#directive-interface" class="sidebar-link">Directive interface</a>
          <a href="#directive-types" class="sidebar-link">DirectiveBinding / Context</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Styles</div>
          <a href="#style-collector" class="sidebar-link">StyleCollector</a>
          <a href="#style-helpers" class="sidebar-link">ScopeID / ScopeCSS</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Errors</div>
          <a href="#error-types" class="sidebar-link">Error types</a>
          <a href="#sentinel-errors" class="sidebar-link">Sentinel errors</a>
        </div>
      </aside>

      <div class="docs-content">
        <h1>Go API Reference</h1>
        <p class="lead">Complete reference for every exported symbol in the <code>htmlc</code> package. Import path: <code>github.com/dhamidi/htmlc</code>.</p>

        <!-- ═══════════════════════════════════════════════ Creating an Engine -->
        <h2 id="creating-engine">Creating an Engine</h2>

        <h3 id="new">New</h3>
        <pre><code>func New(opts Options) (*Engine, error)</code></pre>
        <p>Creates an Engine from <code>opts</code>. If <code>opts.ComponentDir</code> is set the directory is walked recursively and all <code>*.vue</code> files are registered before the engine is returned.</p>
        <pre><code>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
})
if err != nil {
    log.Fatal(err)
}</code></pre>

        <h3 id="options">Options</h3>
        <pre><code>type Options struct {
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
              <td><code>ComponentDir</code></td>
              <td>Directory walked recursively for <code>*.vue</code> files. Each file is registered by its base name without extension (<code>Button.vue</code> → <code>Button</code>). When two files share the same base name the last one in lexical order wins.</td>
            </tr>
            <tr>
              <td><code>Reload</code></td>
              <td>When true the engine checks the modification time of every registered file before each render and re-parses changed files automatically. Use during development; leave false in production.</td>
            </tr>
            <tr>
              <td><code>FS</code></td>
              <td>When set, all file reads and directory walks use this <code>fs.FS</code> instead of the OS filesystem. <code>ComponentDir</code> is interpreted as a path within the FS. Useful with <code>//go:embed</code>. Hot-reload requires the FS to also implement <code>fs.StatFS</code>.</td>
            </tr>
            <tr>
              <td><code>Directives</code></td>
              <td>Custom directives available to all components rendered by this engine. Keys are directive names without the <code>v-</code> prefix. Built-in directives cannot be overridden.</td>
            </tr>
            <tr>
              <td><code>Debug</code></td>
              <td>When true the rendered HTML is annotated with HTML comments describing component boundaries, expression values, and slot contents. Development use only.</td>
            </tr>
          </tbody>
        </table>

        <!-- ═══════════════════════════════════════════════ Rendering -->
        <h2 id="rendering">Rendering</h2>

        <h3 id="render-page">RenderPage</h3>
        <pre><code>func (e *Engine) RenderPage(w io.Writer, name string, data map[string]any) error</code></pre>
        <p>Renders the named component as a full HTML page and writes the result to <code>w</code>. Scoped styles are collected from the entire component tree and injected as a <code>&lt;style&gt;</code> block immediately before the first <code>&lt;/head&gt;</code> tag. If no <code>&lt;/head&gt;</code> is found the style block is prepended to the output.</p>
        <p>Use <code>RenderPage</code> for page components that include <code>&lt;!DOCTYPE html&gt;</code>, <code>&lt;html&gt;</code>, <code>&lt;head&gt;</code>, and <code>&lt;body&gt;</code>.</p>
        <pre><code>var buf bytes.Buffer
err := engine.RenderPage(&amp;buf, "HomePage", map[string]any{
    "title": "Welcome",
})</code></pre>

        <h3 id="render-fragment">RenderFragment</h3>
        <pre><code>func (e *Engine) RenderFragment(w io.Writer, name string, data map[string]any) error</code></pre>
        <p>Renders the named component as an HTML fragment and prepends the collected <code>&lt;style&gt;</code> block to the output. Does not search for a <code>&lt;/head&gt;</code> tag. Use for partial page updates such as HTMX responses or turbo-frame updates.</p>

        <h3 id="render-string">String helpers</h3>
        <pre><code>func (e *Engine) RenderPageString(name string, data map[string]any) (string, error)
func (e *Engine) RenderFragmentString(name string, data map[string]any) (string, error)</code></pre>
        <p>Convenience wrappers around <code>RenderPage</code> and <code>RenderFragment</code> that return the result as a string instead of writing to an <code>io.Writer</code>.</p>

        <h3 id="render-context">Context variants</h3>
        <pre><code>func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error
func (e *Engine) RenderFragmentContext(ctx context.Context, w io.Writer, name string, data map[string]any) error</code></pre>
        <p>Like <code>RenderPage</code> and <code>RenderFragment</code> but accept a <code>context.Context</code>. The render is aborted and <code>ctx.Err()</code> is returned if the context is cancelled or its deadline is exceeded.</p>
        <pre><code>ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
err := engine.RenderPageContext(ctx, w, "HomePage", data)</code></pre>

        <!-- ═══════════════════════════════════════════════ HTTP Integration -->
        <h2 id="http">HTTP Integration</h2>

        <h3 id="serve-component">ServeComponent</h3>
        <pre><code>func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc</code></pre>
        <p>Returns an <code>http.HandlerFunc</code> that renders <code>name</code> as a fragment on every request. The <code>data</code> function is called per-request to build the template scope; it may be <code>nil</code>. Data middleware registered via <code>WithDataMiddleware</code> is applied after the data function. Sets <code>Content-Type: text/html; charset=utf-8</code>.</p>
        <pre><code>mux.HandleFunc("GET /search-results", engine.ServeComponent("SearchResults", func(r *http.Request) map[string]any {
    return map[string]any{"query": r.URL.Query().Get("q")}
}))</code></pre>

        <h3 id="serve-page-component">ServePageComponent</h3>
        <pre><code>func (e *Engine) ServePageComponent(name string, data func(*http.Request) (map[string]any, int)) http.HandlerFunc</code></pre>
        <p>Returns an <code>http.HandlerFunc</code> that renders <code>name</code> as a full HTML page. The data function returns both the template scope and the HTTP status code to send. A status code of <code>0</code> is treated as <code>200</code>. If <code>data</code> is <code>nil</code> a <code>200 OK</code> response with no template data is used.</p>
        <pre><code>mux.HandleFunc("GET /post/{id}", engine.ServePageComponent("PostPage", func(r *http.Request) (map[string]any, int) {
    post, err := db.GetPost(r.PathValue("id"))
    if err != nil {
        return map[string]any{"error": err.Error()}, http.StatusNotFound
    }
    return map[string]any{"post": post}, http.StatusOK
}))</code></pre>

        <h3 id="mount">Mount</h3>
        <pre><code>func (e *Engine) Mount(mux *http.ServeMux, routes map[string]string)</code></pre>
        <p>Registers multiple component routes on <code>mux</code> at once. Keys are <code>http.ServeMux</code> patterns (e.g. <code>"GET /{$}"</code>, <code>"GET /about"</code>); values are component names. Each component is served as a full page via <code>ServePageComponent</code> with no data function.</p>
        <pre><code>engine.Mount(mux, map[string]string{
    "GET /{$}":    "HomePage",
    "GET /about":  "AboutPage",
    "GET /blog":   "BlogPage",
})</code></pre>

        <h3 id="data-middleware">WithDataMiddleware</h3>
        <pre><code>func (e *Engine) WithDataMiddleware(fn func(*http.Request, map[string]any) map[string]any) *Engine</code></pre>
        <p>Adds a function that augments the data map on every HTTP-triggered render. Multiple middleware functions are called in registration order; later middleware can overwrite keys set by earlier ones. Returns the engine for chaining.</p>
        <p>Data middleware applies only to the top-level render scope and is not automatically propagated into child component scopes. Pass shared values via <code>RegisterFunc</code> if child components need them.</p>
        <pre><code>engine.WithDataMiddleware(func(r *http.Request, data map[string]any) map[string]any {
    data["currentUser"] = userFromRequest(r)
    data["csrfToken"]   = csrf.Token(r)
    return data
})</code></pre>

        <!-- ═══════════════════════════════════════════════ Component Management -->
        <h2 id="component-management">Component Management</h2>

        <h3 id="register">Register</h3>
        <pre><code>func (e *Engine) Register(name, path string) error</code></pre>
        <p>Manually adds a component from <code>path</code> to the engine's registry under <code>name</code>, without requiring a directory scan. Useful for programmatically generated components or files outside <code>ComponentDir</code>.</p>

        <h3 id="has">Has</h3>
        <pre><code>func (e *Engine) Has(name string) bool</code></pre>
        <p>Reports whether <code>name</code> is a registered component.</p>

        <h3 id="components">Components</h3>
        <pre><code>func (e *Engine) Components() []string</code></pre>
        <p>Returns the names of all registered components in sorted order. Automatic lowercase aliases added by the engine are excluded.</p>

        <h3 id="validate">ValidateAll</h3>
        <pre><code>func (e *Engine) ValidateAll() []ValidationError</code></pre>
        <p>Checks every registered component for unresolvable child component references. Returns one <code>ValidationError</code> per problem; an empty slice means all components are valid. Call once at application startup to surface missing-component issues early.</p>
        <pre><code>if errs := engine.ValidateAll(); len(errs) != 0 {
    for _, e := range errs {
        log.Println(e)
    }
    os.Exit(1)
}</code></pre>

        <!-- ═══════════════════════════════════════════════ Customization -->
        <h2 id="customization">Customization</h2>

        <h3 id="register-func">RegisterFunc</h3>
        <pre><code>func (e *Engine) RegisterFunc(name string, fn func(...any) (any, error)) *Engine</code></pre>
        <p>Adds a per-engine function available in all template expressions. The function is callable from templates as <code>name()</code>. Engine functions have lower priority than user-provided data keys. They are propagated automatically into every child component's scope.</p>
        <pre><code>engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    t, ok := args[0].(time.Time)
    if !ok {
        return "", fmt.Errorf("formatDate: expected time.Time")
    }
    return t.Format("Jan 2, 2006"), nil
})</code></pre>

        <h3 id="register-directive">RegisterDirective</h3>
        <pre><code>func (e *Engine) RegisterDirective(name string, dir Directive)</code></pre>
        <p>Adds a custom directive under <code>name</code> (without the <code>v-</code> prefix). Replaces any previously registered directive with the same name. Panics if <code>dir</code> is nil.</p>
        <pre><code>engine.RegisterDirective("tooltip", &amp;TooltipDirective{})</code></pre>

        <h3 id="missing-prop">WithMissingPropHandler</h3>
        <pre><code>func (e *Engine) WithMissingPropHandler(fn MissingPropFunc) *Engine</code></pre>
        <p>Sets the function called when any component has a missing prop. The default behaviour renders a visible <code>[missing: &lt;name&gt;]</code> placeholder. Use <code>ErrorOnMissingProp</code> for strict error behaviour or <code>SubstituteMissingProp</code> for legacy placeholder text.</p>
        <pre><code>// Abort rendering on any missing prop
engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)

// Substitute a legacy placeholder string
engine.WithMissingPropHandler(htmlc.SubstituteMissingProp)</code></pre>

        <!-- ═══════════════════════════════════════════════ Low-level API -->
        <h2 id="low-level">Low-level API</h2>

        <h3 id="parse-file">ParseFile</h3>
        <pre><code>func ParseFile(path, src string) (*Component, error)</code></pre>
        <p>Parses a <code>.vue</code> Single File Component from the string <code>src</code>. <code>path</code> is used only for error messages and the scope attribute hash. Returns a <code>*Component</code> whose <code>Template</code> field is a parsed HTML node tree. Only the top-level <code>&lt;template&gt;</code>, <code>&lt;script&gt;</code>, and <code>&lt;style&gt;</code> sections are extracted; <code>&lt;template&gt;</code> is required.</p>

        <h3 id="component-type">Component</h3>
        <pre><code>type Component struct {
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
        <pre><code>func (c *Component) Props() []PropInfo</code></pre>
        <p>Walks the template AST and returns all top-level variable references the template uses. Identifiers starting with <code>$</code> and <code>v-for</code> loop variables are excluded.</p>
        <pre><code>type PropInfo struct {
    Name        string   // identifier name
    Expressions []string // template expressions in which it appears
}</code></pre>

        <h3 id="registry">Registry</h3>
        <pre><code>type Registry map[string]*Component</code></pre>
        <p>Maps component names to their parsed components. Keys may be PascalCase or kebab-case. Most callers use <code>Engine</code>, which builds and maintains a Registry automatically.</p>

        <h3 id="renderer">Renderer</h3>
        <pre><code>func NewRenderer(c *Component) *Renderer</code></pre>
        <p>Creates a Renderer for <code>c</code>. Use the builder methods below to configure it before calling <code>Render</code>.</p>

        <p>Renderer is the low-level rendering primitive. Most callers should use <code>Engine</code> via <code>RenderPage</code> or <code>RenderFragment</code>. Use <code>NewRenderer</code> when you need fine-grained control over style collection or registry attachment.</p>

        <table>
          <thead>
            <tr><th>Builder method</th><th>Description</th></tr>
          </thead>
          <tbody>
            <tr>
              <td><code>WithStyles(sc *StyleCollector) *Renderer</code></td>
              <td>Sets the <code>StyleCollector</code> that receives this component's style contribution.</td>
            </tr>
            <tr>
              <td><code>WithComponents(reg Registry) *Renderer</code></td>
              <td>Attaches a component registry, enabling component composition.</td>
            </tr>
            <tr>
              <td><code>WithMissingPropHandler(fn MissingPropFunc) *Renderer</code></td>
              <td>Sets the handler called when a template prop is absent from the scope.</td>
            </tr>
            <tr>
              <td><code>WithDirectives(dr DirectiveRegistry) *Renderer</code></td>
              <td>Attaches a custom directive registry.</td>
            </tr>
            <tr>
              <td><code>WithContext(ctx context.Context) *Renderer</code></td>
              <td>Attaches a <code>context.Context</code>; the render is aborted on cancellation.</td>
            </tr>
            <tr>
              <td><code>WithFuncs(funcs map[string]any) *Renderer</code></td>
              <td>Attaches engine-registered functions so they are available in expressions and propagated to all child renderers.</td>
            </tr>
          </tbody>
        </table>

        <h3 id="renderer-render">Renderer.Render / RenderString</h3>
        <pre><code>func (r *Renderer) Render(w io.Writer, scope map[string]any) error
func (r *Renderer) RenderString(scope map[string]any) (string, error)</code></pre>
        <p>Evaluates the component's template against <code>scope</code> and writes HTML to <code>w</code> (or returns it as a string). Prop validation and style collection happen here.</p>

        <h3 id="package-helpers">Package-level helpers</h3>
        <pre><code>func Render(w io.Writer, c *Component, scope map[string]any) error
func RenderString(c *Component, scope map[string]any) (string, error)</code></pre>
        <p>Convenience wrappers that create a temporary <code>Renderer</code> for <code>c</code>. They do not collect styles or support component composition. Use <code>NewRenderer</code> with <code>WithStyles</code> and <code>WithComponents</code> when those features are needed.</p>

        <!-- ═══════════════════════════════════════════════ Directives -->
        <h2 id="directives">Directives</h2>

        <h3 id="directive-interface">Directive interface</h3>
        <pre><code>type Directive interface {
    Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
    Mounted(w io.Writer, node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
}</code></pre>
        <p>Implemented by custom directive types. Register with <code>Engine.RegisterDirective</code> or pass in <code>Options.Directives</code>. Only the <code>Created</code> and <code>Mounted</code> hooks are called because htmlc renders server-side.</p>
        <ul>
          <li><code>Created</code> — called before the element is rendered. Receives a shallow-cloned working node whose <code>Attr</code> slice and <code>Data</code> field may be freely mutated. Return a non-nil error to abort rendering.</li>
          <li><code>Mounted</code> — called after the element's closing tag has been written to <code>w</code>. Bytes written to <code>w</code> appear immediately after the element. Return a non-nil error to abort rendering.</li>
        </ul>

        <h3 id="directive-types">DirectiveBinding / DirectiveContext</h3>
        <pre><code>type DirectiveBinding struct {
    Value     any             // result of evaluating the directive expression
    RawExpr   string          // un-evaluated expression string from the template
    Arg       string          // directive argument after the colon, e.g. "href" in v-bind:href
    Modifiers map[string]bool // dot-separated modifiers, e.g. {"prevent": true}
}</code></pre>
        <pre><code>type DirectiveContext struct {
    Registry Registry // component registry the renderer is using
}</code></pre>

        <h3 id="directive-registry">DirectiveRegistry</h3>
        <pre><code>type DirectiveRegistry map[string]Directive</code></pre>
        <p>Maps directive names (without the <code>v-</code> prefix) to their implementations. Keys are lower-kebab-case; the renderer normalises names before lookup.</p>

        <!-- ═══════════════════════════════════════════════ Style Collection -->
        <h2 id="styles">Style Collection</h2>

        <h3 id="style-collector">StyleCollector</h3>
        <pre><code>type StyleCollector struct { /* unexported fields */ }</code></pre>
        <p>Accumulates <code>StyleContribution</code> values from one or more component renders into a single ordered list, deduplicating contributions from the same scoped component. <code>Engine</code> creates and manages a <code>StyleCollector</code> automatically on each render call.</p>

        <pre><code>func (sc *StyleCollector) Add(c StyleContribution)
func (sc *StyleCollector) All() []StyleContribution</code></pre>
        <ul>
          <li><code>Add</code> — appends <code>c</code>, skipping duplicates. Two contributions are duplicates when they share the same composite key (<code>ScopeID + CSS</code>).</li>
          <li><code>All</code> — returns all contributions in the order they were added.</li>
        </ul>

        <h3 id="style-contribution">StyleContribution</h3>
        <pre><code>type StyleContribution struct {
    ScopeID string // scope attribute name (e.g. "data-v-a1b2c3d4"), empty for global styles
    CSS     string // stylesheet text, already rewritten by ScopeCSS for scoped components
}</code></pre>

        <h3 id="style-helpers">ScopeID / ScopeCSS</h3>
        <pre><code>func ScopeID(path string) string
func ScopeCSS(css, scopeAttr string) string</code></pre>
        <ul>
          <li><code>ScopeID</code> — returns <code>"data-v-"</code> followed by 8 hex digits derived from the FNV-1a 32-bit hash of <code>path</code>.</li>
          <li><code>ScopeCSS</code> — rewrites <code>css</code> so that every selector in every non-<code>@</code>-rule has <code>scopeAttr</code> appended to its last compound selector. <code>@</code>-rules are passed through verbatim.</li>
        </ul>

        <!-- ═══════════════════════════════════════════════ Error Handling -->
        <h2 id="errors">Error Handling</h2>

        <h3 id="error-types">Error types</h3>

        <h4>ParseError</h4>
        <pre><code>type ParseError struct {
    Path     string          // source file path
    Msg      string          // human-readable description
    Location *SourceLocation // source position, or nil if unknown
}</code></pre>
        <p>Returned by <code>ParseFile</code> when a <code>.vue</code> file cannot be parsed.</p>

        <h4>RenderError</h4>
        <pre><code>type RenderError struct {
    Component string          // component name being rendered
    Expr      string          // expression that triggered the error (may be empty)
    Wrapped   error           // underlying error
    Location  *SourceLocation // source position, or nil if unknown
}</code></pre>
        <p>Returned by render methods when template evaluation fails. Implements <code>Unwrap</code>.</p>

        <h4>ValidationError</h4>
        <pre><code>type ValidationError struct {
    Component string // name of the component with the problem
    Message   string // description of the problem
}</code></pre>
        <p>One entry per problem in the slice returned by <code>ValidateAll</code>.</p>

        <h4>SourceLocation</h4>
        <pre><code>type SourceLocation struct {
    File    string // source file path
    Line    int    // 1-based line number (0 = unknown)
    Column  int    // 1-based column (0 = unknown)
    Snippet string // ~3-line context around the error (may be empty)
}</code></pre>

        <h3 id="missing-prop-types">MissingPropFunc / handlers</h3>
        <pre><code>type MissingPropFunc func(name string) (any, error)

func ErrorOnMissingProp(name string) (any, error)
func SubstituteMissingProp(name string) (any, error)</code></pre>
        <ul>
          <li><code>MissingPropFunc</code> — signature for missing-prop handlers; receive the prop name, return a substitute value or an error.</li>
          <li><code>ErrorOnMissingProp</code> — aborts rendering with an error whenever a prop is absent. Use for strict validation.</li>
          <li><code>SubstituteMissingProp</code> — returns <code>"MISSING PROP: &lt;name&gt;"</code> as a placeholder string.</li>
        </ul>

        <h3 id="sentinel-errors">Sentinel errors</h3>
        <pre><code>var ErrComponentNotFound = errors.New("htmlc: component not found")
var ErrMissingProp       = errors.New("htmlc: missing required prop")</code></pre>
        <ul>
          <li><code>ErrComponentNotFound</code> — wrapped inside the error returned by render methods when the requested component name is not registered.</li>
          <li><code>ErrMissingProp</code> — returned (wrapped) when a required prop is absent and no <code>MissingPropFunc</code> has been set.</li>
        </ul>
        <pre><code>if errors.Is(err, htmlc.ErrComponentNotFound) {
    http.NotFound(w, r)
    return
}</code></pre>

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
  color: var(--text);
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

.docs-content h3 {
  margin-top: 2rem;
  margin-bottom: 0.4rem;
  font-size: 1.05rem;
  color: #00ADD8;
}

.docs-content h4 {
  margin-top: 1.25rem;
  margin-bottom: 0.3rem;
  font-size: 0.95rem;
  color: #e2e4f0;
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
