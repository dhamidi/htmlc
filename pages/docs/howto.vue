<template>
  <DocsPage
    pageTitle="How-to Guides — htmlc.sh"
    description="Task-oriented guides for integrating htmlc into a Go web application: HTTP handlers, embed.FS, hot reload, custom directives, and more."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'HTTP'},
      {href: '#serve-http', label: 'Serve via net/http'},
      {label: 'Deployment'},
      {href: '#embed-fs', label: 'Embed into a binary'},
      {href: '#validate-startup', label: 'Validate at startup'},
      {label: 'Development'},
      {href: '#hot-reload', label: 'Hot reload'},
      {href: '#expvars', label: 'Monitor engine metrics'},
      {href: '#slog', label: 'Structured logging (slog)'},
      {label: 'Customization'},
      {href: '#custom-directive', label: 'Custom directive'},
      {href: '#missing-props', label: 'Missing prop handling'},
      {label: 'Static sites'},
      {href: '#static-site', label: 'Static site with layout'},
      {href: '#syntax-highlight', label: 'Syntax highlighting'},
      {href: '#serve-custom-elements', label: 'Serve custom element scripts'},
      {href: '#custom-elements-static-build', label: 'Custom elements in static build'},
      {label: 'Testing'},
      {href: '#testing', label: 'Testing components'}
    ]"
  >
    <h1>How-to Guides</h1>
    <p class="lead">Practical recipes for common tasks. Each guide assumes you have a working htmlc engine — see the <a href="/docs/index.html">overview</a> for initial setup and the <a href="/docs/go-api.html">Go API reference</a> for full API details.</p>

    <!-- ═══════════════════════════════════════════════ Serve via net/http -->
    <h2 id="serve-http">Serve a component via net/http</h2>
    <p class="howto-goal">You want to render htmlc components in response to HTTP requests using the standard library.</p>

    <p>Use <code>ServeComponent</code> for partial HTML responses (HTMX, turbo frames) and <code>ServePageComponent</code> for full HTML pages. Both return an <code>http.HandlerFunc</code> you register on any <code>*http.ServeMux</code>.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;log&#34;
    &#34;net/http&#34;

    &#34;github.com/dhamidi/htmlc&#34;
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
    })
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()

    // Fragment handler — no &lt;html&gt; wrapper, good for HTMX responses.
    // The data function is called once per request.
    mux.HandleFunc(&#34;GET /search&#34;, engine.ServeComponent(
        &#34;SearchResults&#34;,
        func(r *http.Request) map[string]any {
            return map[string]any{&#34;query&#34;: r.URL.Query().Get(&#34;q&#34;)}
        },
    ))

    // Full-page handler — injects &lt;style&gt; into &lt;head&gt; automatically.
    // Return both the data map and the HTTP status code.
    mux.HandleFunc(&#34;GET /post/{id}&#34;, engine.ServePageComponent(
        &#34;PostPage&#34;,
        func(r *http.Request) (map[string]any, int) {
            post, err := db.GetPost(r.PathValue(&#34;id&#34;))
            if err != nil {
                return map[string]any{&#34;error&#34;: err.Error()}, http.StatusNotFound
            }
            return map[string]any{&#34;post&#34;: post}, http.StatusOK
        },
    ))

    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <p>Pass per-request data (current user, CSRF token, feature flags) to every handler at once with <a href="/docs/go-api.html#data-middleware"><code v-pre>WithDataMiddleware</code></a> instead of repeating the logic in each data function.</p>

    <!-- ═══════════════════════════════════════════════ Embed into a binary -->
    <h2 id="embed-fs">Embed components into a Go binary</h2>
    <p class="howto-goal">You want to ship a self-contained binary that has no dependency on files being present at the deployment path.</p>

    <p>Use <code>//go:embed</code> to bundle the <code>components/</code> directory into the binary, then pass the resulting <code>embed.FS</code> as <code>Options.FS</code>. When <code>FS</code> is set, all directory walks and file reads use it instead of the OS filesystem.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;embed&#34;
    &#34;log&#34;
    &#34;net/http&#34;

    &#34;github.com/dhamidi/htmlc&#34;
)

//go:embed components
var componentsFS embed.FS

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;components&#34;, // path inside the embedded FS
        FS:           componentsFS,
    })
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()
    engine.Mount(mux, map[string]string{
        &#34;GET /{$}&#34;:   &#34;HomePage&#34;,
        &#34;GET /about&#34;: &#34;AboutPage&#34;,
    })
    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <p>Expected directory layout:</p>
    <pre v-syntax-highlight="'text'"><code v-pre>myapp/
├── main.go
└── components/
    ├── Layout.vue
    ├── HomePage.vue
    └── AboutPage.vue</code></pre>

    <p>This is recommended for production deployments. Without <code>FS</code>, the engine reads from the OS filesystem and the <code>components/</code> directory must exist at the working directory of the running process.</p>

    <!-- ═══════════════════════════════════════════════ Hot reload -->
    <h2 id="hot-reload">Use hot-reload during development</h2>
    <p class="howto-goal">You want component changes to be reflected in the browser without restarting the server.</p>

    <p>Set <code>Options.Reload = true</code>. The engine will stat every registered file before each render and re-parse any that have changed.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: &#34;./components&#34;,
    Reload:       true,
})</code></pre>

    <p><strong>Tradeoff:</strong> <code>Reload</code> adds a <code>stat</code> syscall per component file on every render. Leave it <code>false</code> in production. A common pattern is to gate it behind a flag:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>import &#34;flag&#34;

var dev = flag.Bool(&#34;dev&#34;, false, &#34;enable hot reload&#34;)

func main() {
    flag.Parse()

    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
        Reload:       *dev,
    })
    // ...
}</code></pre>

    <p>Run with <code>go run . -dev</code> locally and without the flag in production. Alternatively, use a build tag to set the constant at compile time so the production binary has zero overhead.</p>

    <!-- ═══════════════════════════════════════════════ Expvars -->
    <h2 id="expvars">Monitor engine metrics with expvars</h2>
    <p class="howto-goal">You want to expose htmlc engine metrics at the standard Go <code>/debug/vars</code> endpoint for dashboards and health checks.</p>

    <p>Go's built-in <code>expvar</code> package publishes named variables at <code>/debug/vars</code> as a JSON object. Calling <code>engine.PublishExpvars(prefix)</code> registers counters and configuration state under that prefix so any monitoring tool that can read <code>/debug/vars</code> can observe the engine.</p>

    <h3>Step 1 — Create the engine and publish metrics</h3>
    <p>Call <code>PublishExpvars</code> once at startup, before serving any requests. A blank import of <code>expvar</code> registers the <code>/debug/vars</code> handler on <code>http.DefaultServeMux</code> automatically:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;log&#34;
    &#34;net/http&#34;
    _ &#34;expvar&#34; // registers /debug/vars on http.DefaultServeMux

    &#34;github.com/dhamidi/htmlc&#34;
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register all engine metrics under &#34;myapp&#34; in /debug/vars.
    engine.PublishExpvars(&#34;myapp&#34;)

    mux := http.NewServeMux()
    mux.HandleFunc(&#34;GET /{$}&#34;, engine.ServePageComponent(&#34;HomePage&#34;, nil))

    // For a custom mux, add the expvar handler explicitly:
    // mux.Handle(&#34;GET /debug/vars&#34;, expvar.Handler())

    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <h3>Step 2 — Inspect the output</h3>
    <p>With the server running, fetch the metrics and pipe through <code>jq</code> to extract the engine block:</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>curl -s http://localhost:8080/debug/vars | jq '.myapp'</code></pre>

    <p>Example output:</p>

    <pre v-syntax-highlight="'json'"><code v-pre>{
  &#34;reload&#34;: 0,
  &#34;debug&#34;: 0,
  &#34;componentDir&#34;: &#34;./components&#34;,
  &#34;fs&#34;: &#34;&lt;nil&gt;&#34;,
  &#34;renders&#34;: 42,
  &#34;renderErrors&#34;: 0,
  &#34;reloads&#34;: 2,
  &#34;renderNanos&#34;: 125432100,
  &#34;components&#34;: 15,
  &#34;info&#34;: {
    &#34;directives&#34;: [&#34;myCustom&#34;]
  }
}</code></pre>

    <h3>Reading the counters</h3>
    <p>Use the raw counters to compute derived metrics:</p>
    <ul>
      <li><strong>Error rate:</strong> <code>renderErrors / renders</code></li>
      <li><strong>Average render latency:</strong> <code>renderNanos / renders</code> nanoseconds per render</li>
      <li><strong>Reload activity:</strong> <code>reloads</code> should only increment during development; a non-zero value in production means hot-reload is accidentally enabled</li>
    </ul>

    <h3>Live option toggling</h3>
    <p>The setter methods (<code>SetReload</code>, <code>SetDebug</code>, <code>SetComponentDir</code>, <code>SetFS</code>) update both the live engine option and the corresponding expvar immediately. For example, enabling hot-reload at runtime without restarting the process:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine.SetReload(true)
// curl -s .../debug/vars | jq '.myapp.reload'  →  1</code></pre>

    <Callout><strong>Warning:</strong> calling <code>PublishExpvars</code> twice with the same prefix panics. Register metrics exactly once per engine per process, immediately after creating the engine.</Callout>

    <!-- ═══════════════════════════════════════════════ Structured logging -->
    <h2 id="slog">Add structured logging with slog</h2>
    <p class="howto-goal">Produce one structured log record per rendered component so you can identify slow or unexpectedly large components in production.</p>

    <h3>Minimal setup</h3>
    <p>Pass <code>slog.Default()</code> as <code>Options.Logger</code> to start receiving one log record per component on every render:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;log&#34;
    &#34;log/slog&#34;
    &#34;net/http&#34;

    &#34;github.com/dhamidi/htmlc&#34;
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
        Logger:       slog.Default(),
    })
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc(&#34;GET /{$}&#34;, engine.ServePageComponent(&#34;HomePage&#34;, nil))
    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <p>Records are emitted at <code>slog.LevelDebug</code>. With the default text handler you will see one line per component, leaf-first, ending with the root page component.</p>

    <h3>Using a custom handler for machine-readable output</h3>
    <p>For log aggregators (Datadog, Loki, Cloud Logging) create a JSON handler writing to <code>os.Stdout</code>:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;log&#34;
    &#34;log/slog&#34;
    &#34;net/http&#34;
    &#34;os&#34;

    &#34;github.com/dhamidi/htmlc&#34;
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &amp;slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
        Logger:       logger,
    })
    if err != nil {
        log.Fatal(err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc(&#34;GET /{$}&#34;, engine.ServePageComponent(&#34;HomePage&#34;, nil))
    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <p>Each component emits a record like:</p>
    <pre v-syntax-highlight="'json'"><code v-pre>{&#34;time&#34;:&#34;2026-03-16T12:00:00.001Z&#34;,&#34;level&#34;:&#34;DEBUG&#34;,&#34;msg&#34;:&#34;component rendered&#34;,&#34;component&#34;:&#34;NavLink&#34;,&#34;duration&#34;:1200000,&#34;bytes&#34;:142}</code></pre>
    <p>Note: <code>duration</code> is nanoseconds as <code>int64</code> in JSON — this is standard <code>slog</code> behaviour for <code>time.Duration</code> values.</p>

    <h3>Request-scoped logging</h3>
    <p>Attach request metadata (such as a trace or request ID) using <code>logger.With(...)</code> and pass the enriched logger to a per-request renderer via <code>WithLogger</code>:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>func makeHandler(baseLogger *slog.Logger, component *htmlc.Component) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get(&#34;X-Request-ID&#34;)
        logger := baseLogger.With(&#34;request_id&#34;, requestID)

        renderer := htmlc.NewRenderer(component).WithLogger(logger)
        // ... use renderer
    }
}</code></pre>

    <h3>Filtering noise in development</h3>
    <p>All component records are emitted at <code>slog.LevelDebug</code>. To silence them where debug output is unwanted, set <code>HandlerOptions.Level</code> to <code>slog.LevelInfo</code>:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>logger := slog.New(slog.NewTextHandler(os.Stderr, &amp;slog.HandlerOptions{
    Level: slog.LevelInfo, // component records at LevelDebug are suppressed
}))</code></pre>

    <h3>Interpreting the output</h3>
    <p>Each log record contains four attributes:</p>
    <ul>
      <li><code>component</code> — the resolved component name (e.g. <code>NavBar</code>).</li>
      <li><code>duration</code> — wall-clock time for the component subtree. In text format this appears as <code>1.2ms</code>; in JSON it is nanoseconds as <code>int64</code>.</li>
      <li><code>bytes</code> — bytes written by the component subtree.</li>
      <li><code>error</code> — present only on <code>LevelError</code> records for failed renders.</li>
    </ul>
    <p>Records appear <strong>leaf-first</strong> (post-order traversal): child components are logged before their parents. The root page component is always the last record in the batch.</p>

    <h3>Using the constants in tests and alerting</h3>
    <p>Use <code>htmlc.MsgComponentRendered</code> and <code>htmlc.MsgComponentFailed</code> instead of hard-coding strings when writing log-based test assertions or alerting rules:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>// In a test using a log/slog capture handler:
if record.Message != htmlc.MsgComponentRendered {
    t.Errorf(&#34;unexpected log message: %q&#34;, record.Message)
}

// In an alerting rule (pseudo-code):
// alert when msg == htmlc.MsgComponentFailed</code></pre>

    <!-- ═══════════════════════════════════════════════ Custom directive -->
    <h2 id="custom-directive">Write a custom directive</h2>
    <p class="howto-goal">You want to add a reusable HTML attribute behaviour that is not covered by the built-in directives.</p>

    <p>Implement the <code>htmlc.Directive</code> interface and register it via <code>Options.Directives</code> or <code>Engine.RegisterDirective</code>. The interface has two hooks — <code>Created</code> (before rendering) and <code>Mounted</code> (after rendering). Both receive the working node, the binding, and a context.</p>

    <p>Example: a <code>v-uppercase</code> directive that uppercases all direct text children of the element.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;io&#34;
    &#34;strings&#34;

    &#34;golang.org/x/net/html&#34;
    &#34;github.com/dhamidi/htmlc&#34;
)

type UppercaseDirective struct{}

// Created is called before the element is rendered.
// Mutate node.Attr or child text nodes here.
func (d *UppercaseDirective) Created(
    node *html.Node,
    binding htmlc.DirectiveBinding,
    ctx htmlc.DirectiveContext,
) error {
    for c := node.FirstChild; c != nil; c = c.NextSibling {
        if c.Type == html.TextNode {
            c.Data = strings.ToUpper(c.Data)
        }
    }
    return nil
}

// Mounted is called after the element&#39;s closing tag is written to w.
// Bytes written to w appear immediately after the element in the output.
func (d *UppercaseDirective) Mounted(
    w io.Writer,
    node *html.Node,
    binding htmlc.DirectiveBinding,
    ctx htmlc.DirectiveContext,
) error {
    return nil
}

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
        Directives: htmlc.DirectiveRegistry{
            &#34;uppercase&#34;: &amp;UppercaseDirective{},
        },
    })
    // ...
}</code></pre>

    <p>Use in a template:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;p v-uppercase&gt;hello world&lt;/p&gt;
&lt;!-- renders: &lt;p&gt;HELLO WORLD&lt;/p&gt; --&gt;</code></pre>

    <p>See <a href="/docs/go-api.html#directive-types"><code v-pre>DirectiveBinding</code> and <code>DirectiveContext</code></a> in the Go API reference for the full set of fields available to directive implementations.</p>

    <!-- ═══════════════════════════════════════════════ Missing props -->
    <h2 id="missing-props">Handle missing props gracefully</h2>
    <p class="howto-goal">You want to control what happens when a template references a variable that was not passed as a prop.</p>

    <p>By default, a missing prop renders a visible <code>[missing: &lt;name&gt;]</code> placeholder in the HTML. Use <code>WithMissingPropHandler</code> to choose a different behaviour.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>// Abort the render and return an error — recommended for production.
// Any missing prop causes the entire response to fail, making omissions visible
// during development and CI rather than in rendered HTML.
engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)

// Render a visible placeholder string &#34;MISSING PROP: &lt;name&gt;&#34;.
// Useful when gradually migrating templates that have optional props.
engine.WithMissingPropHandler(htmlc.SubstituteMissingProp)</code></pre>

    <p>Both are package-level functions with the <code>MissingPropFunc</code> signature — you can write your own to log, metric-count, or substitute a default value:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine.WithMissingPropHandler(func(name string) (any, error) {
    slog.Warn(&#34;missing prop&#34;, &#34;name&#34;, name)
    return &#34;&#34;, nil // silently substitute empty string
})</code></pre>

    <!-- ═══════════════════════════════════════════════ Validate at startup -->
    <h2 id="validate-startup">Validate all components at startup</h2>
    <p class="howto-goal">You want to catch broken component references before the server starts serving traffic.</p>

    <p>Call <code>ValidateAll</code> after creating the engine. It checks every registered component for child component references that cannot be resolved and returns one <code>ValidationError</code> per problem. An empty slice means all components are valid.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>package main

import (
    &#34;log&#34;
    &#34;net/http&#34;
    &#34;os&#34;

    &#34;github.com/dhamidi/htmlc&#34;
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: &#34;./components&#34;,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Surface missing-component errors before accepting any traffic.
    if errs := engine.ValidateAll(); len(errs) != 0 {
        for _, e := range errs {
            log.Println(e)
        }
        os.Exit(1)
    }

    mux := http.NewServeMux()
    // ... register routes ...
    log.Fatal(http.ListenAndServe(&#34;:8080&#34;, mux))
}</code></pre>

    <p>Run <code>ValidateAll</code> in CI by building a small <code>cmd/validate/main.go</code> that calls it and exits non-zero on any error. This catches typos in component names at review time rather than at runtime.</p>

    <!-- ═══════════════════════════════════════════════ Static site -->
    <h2 id="static-site">Build a static site with layout wrapping</h2>
    <p class="howto-goal">You want to generate static HTML files where every page shares a common layout component.</p>

    <h3>Using the CLI</h3>
    <p>Pass <code>-layout</code> to <code>htmlc build</code>. The named component is used as the outer wrapper for every page in the <code>-pages</code> directory.</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc build \
  -dir   ./components \
  -pages ./pages \
  -out   ./dist \
  -layout Layout</code></pre>

    <p>Each page component receives a <code>slot</code> prop containing the rendered inner page HTML. The layout component must render <code v-pre>{{ slot }}</code> (or use <code>v-html="slot"</code>) where the page content should appear. See the <a href="/docs/cli.html">CLI reference</a> for all flags.</p>

    <h3>Using the Go API</h3>
    <p>Call <code>RenderFragment</code> for the inner page, then pass the result as data to <code>RenderPage</code> on the layout:</p>

    <pre v-syntax-highlight="'go'"><code>// Render the inner page as a fragment (no full &lt;html&gt; document).
inner, err := engine.RenderFragmentString(context.Background(), &#34;BlogPost&#34;, map[string]any{
    &#34;title&#34;:   post.Title,
    &#34;content&#34;: post.Body,
})
if err != nil {
    return err
}

// Wrap the fragment in the layout, which renders a full HTML document.
// The layout template uses {{ &#34;{{&#34; }} slot }} to embed the inner HTML.
html, err := engine.RenderPageString(context.Background(), &#34;Layout&#34;, map[string]any{
    &#34;pageTitle&#34;: post.Title,
    &#34;slot&#34;:      inner,
})
if err != nil {
    return err
}

// Write html to a file or http.ResponseWriter.</code></pre>

    <p>This approach gives you full control over which pages receive which layout and what data is passed to each layer.</p>

    <!-- ═══════════════════════════════════════════════ Syntax highlighting -->
    <h2 id="syntax-highlight">Add syntax highlighting with an external directive</h2>
    <p class="howto-goal">You want source code blocks in your static site to be syntax-highlighted at build time using <code>htmlc build</code>.</p>

    <p><code v-pre>v-syntax-highlight</code> is a ready-made external directive that wraps the <a href="https://github.com/alecthomas/chroma">Chroma</a> library. Place it in your component directory and <code>htmlc build</code> picks it up automatically.</p>

    <p><strong>Prerequisites:</strong> <code>htmlc build</code> is working for your project and Go 1.22+ is installed.</p>

    <h3>Step 1 — Install the directive</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre>go install github.com/dhamidi/htmlc/cmd/v-syntax-highlight@latest</code></pre>
    <p>Then copy the binary into your component directory (the <code>-dir</code> you pass to <code>htmlc build</code>):</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>cp "$(go env GOPATH)/bin/v-syntax-highlight" ./components/</code></pre>

    <h3>Step 2 — Generate a stylesheet</h3>
    <p>The directive uses CSS classes emitted by Chroma. Generate a stylesheet for the <code>monokai</code> theme (or any other <a href="https://xyproto.github.io/splash/docs/">Chroma style</a>) and save it to your public assets directory:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>v-syntax-highlight -print-css -style monokai &gt; public/highlight.css</code></pre>
    <p>Link the stylesheet in your layout component:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;link rel="stylesheet" href="/highlight.css"&gt;</code></pre>

    <h3>Step 3 — Mark code blocks in templates</h3>
    <p>Add <code>v-syntax-highlight="'&lt;language&gt;'"</code> to any <code>&lt;code&gt;</code> or <code>&lt;pre&gt;</code> element. The directive replaces the element's content with highlighted HTML and adds a <code>language-*</code> class:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;pre&gt;&lt;code v-syntax-highlight="'go'"&gt;package main

import "fmt"

func main() {
    fmt.Println("hello, world")
}
&lt;/code&gt;&lt;/pre&gt;</code></pre>

    <h3>Step 4 — Build</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc build -dir ./components -pages ./pages -out ./dist</code></pre>
    <p>The generated HTML will contain highlighted <code>&lt;span&gt;</code> elements styled by the Chroma CSS classes. See the <a href="/docs/cli.html#external-directives">external directives reference</a> for the full protocol and discovery rules.</p>

    <!-- ═══════════════════════════════════════════════ Serve custom elements -->
    <h2 id="serve-custom-elements">Serve custom element scripts from a Go server</h2>
    <p class="howto-goal">You want to write a component with a <code>&lt;script customelement&gt;</code> block and have its JavaScript served automatically from your Go HTTP server.</p>

    <h3>Step 1 — Write a custom element component</h3>
    <p>Create a component with both a <code>&lt;template&gt;</code> block (for server-rendered HTML) and a <code>&lt;script customelement&gt;</code> block (for client-side interactivity). The tag name is derived from the component's directory path and file name.</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/ui/Counter.vue --&gt;
&lt;template&gt;
  &lt;div class="counter"&gt;
    &lt;span&gt;{{ initial }}&lt;/span&gt;
  &lt;/div&gt;
&lt;/template&gt;
&lt;script customelement&gt;
class UiCounter extends HTMLElement {
  connectedCallback() {
    const span = this.querySelector('span')
    let n = parseInt(span.textContent, 10)
    this.addEventListener('click', () =&gt; span.textContent = ++n)
  }
}
customElements.define('ui-counter', UiCounter)
&lt;/script&gt;</code></pre>

    <h3>Step 2 — Mount the script handler</h3>
    <p>Call <code>engine.ScriptHandler()</code> and register it at a path prefix on your mux. Browsers will fetch script files from this prefix.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{ComponentDir: &#34;./components&#34;})
if err != nil {
    log.Fatal(err)
}

http.Handle(&#34;/scripts/&#34;, http.StripPrefix(&#34;/scripts/&#34;, engine.ScriptHandler()))</code></pre>

    <h3>Step 3 — Add <code>importMap()</code> to your page <code>&lt;head&gt;</code></h3>
    <p>The <code>importMap()</code> template function emits an <a href="https://developer.mozilla.org/en-US/docs/Web/HTML/Element/script/type/importmap">import map</a> <code>&lt;script&gt;</code> tag that tells the browser where to find each custom element module. Add it to every layout or page template that may use custom element components:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;head&gt;
  &lt;meta charset="utf-8"&gt;
  {{ importMap() }}
&lt;/head&gt;</code></pre>

    <h3>Step 4 — Use the component in a page</h3>
    <p>Reference the component as usual. It renders as a custom element tag wrapping the server-rendered template output:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;UiCounter :initial="5"&gt;&lt;/UiCounter&gt;
&lt;!-- renders: &lt;ui-counter&gt;&lt;div class="counter"&gt;&lt;span&gt;5&lt;/span&gt;&lt;/div&gt;&lt;/ui-counter&gt; --&gt;</code></pre>

    <p><strong>Note:</strong> <code>importMap()</code> emits nothing when no custom element components are present in the rendered page, so it is safe to include unconditionally in layouts.</p>

    <p>See the <a href="/docs/custom-elements.html">Custom Elements reference</a> for the full API.</p>

    <!-- ═══════════════════════════════════════════════ Custom elements static build -->
    <h2 id="custom-elements-static-build">Include custom element scripts in a static build</h2>
    <p class="howto-goal">You want your <code>htmlc build</code> output to include the JavaScript for custom element components alongside the HTML pages.</p>

    <h3>Step 1 — Write your custom element components</h3>
    <p>Create components with <code>&lt;script customelement&gt;</code> blocks as shown in the <a href="#serve-custom-elements">guide above</a>, or see the <a href="/docs/custom-elements.html">Custom Elements reference</a>.</p>

    <h3>Step 2 — Add <code>importMap()</code> to your page <code>&lt;head&gt;</code></h3>
    <p>The static build uses the same <code>importMap()</code> function as the server — include it in your layout or page template <code>&lt;head&gt;</code>:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;head&gt;
  &lt;meta charset="utf-8"&gt;
  {{ importMap() }}
&lt;/head&gt;</code></pre>

    <h3>Step 3 — Run <code>htmlc build</code></h3>
    <p>No additional flags are needed. The build command detects custom element components automatically:</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc build -dir ./components -pages ./pages -out ./dist</code></pre>

    <h3>Step 4 — Inspect the output</h3>
    <p>The output directory will contain a <code>scripts/</code> subdirectory with one hashed file per unique custom element script and an <code>index.js</code> entry point that imports them all:</p>

    <pre v-syntax-highlight="'text'"><code v-pre>dist/
  index.html
  about.html
  scripts/
    a1b2c3d4e5f6a7b8.js   ← one file per unique custom element script
    index.js               ← ES module entry point that imports all scripts</code></pre>

    <h3>Step 5 — Serve the scripts directory</h3>
    <p>Configure your web server to serve the <code>scripts/</code> directory so browsers can fetch the script files. The import map emitted by <code>importMap()</code> already points to the correct paths.</p>

    <p><strong>Note:</strong> The <code>scripts/</code> directory is only created when at least one custom element component is used. Projects without <code>&lt;script customelement&gt;</code> blocks produce no <code>scripts/</code> directory.</p>

    <p>See the <a href="/docs/custom-elements.html">Custom Elements reference</a> for the full API.</p>

    <!-- ═══════════════════════════════════════════════ Testing -->
    <h2 id="testing">Testing components</h2>
    <p class="howto-goal">Write unit tests for <code>.vue</code> components without touching the filesystem.</p>

    <p>The <code>htmlctest</code> package provides a fluent harness for testing htmlc components using an in-memory filesystem and a DOM-query API. Add it to your module:</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>go get github.com/dhamidi/htmlc/htmlctest</code></pre>

    <h3>Quick start — <code>Build</code> shorthand</h3>
    <p><code>Build</code> wraps a template snippet in <code>&lt;template&gt;…&lt;/template&gt;</code>, registers it as a component named <code>Root</code>, and returns a <code>*Harness</code> ready to render. Chain <code>Fragment</code> → <code>Find</code> → assertion:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>func TestGreeting(t *testing.T) {
    htmlctest.Build(t, `&lt;p class="greeting"&gt;Hello {{<!-- --> name }}!&lt;/p&gt;`).
        Fragment("Root", map[string]any{"name": "World"}).
        Find(htmlctest.ByTag("p").WithClass("greeting")).
        AssertText("Hello World!")
}</code></pre>

    <h3>Multiple components — <code>NewHarness</code></h3>
    <p>When the component under test references child components, register all required <code>.vue</code> files with <code>NewHarness</code>:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>func TestCard(t *testing.T) {
    h := htmlctest.NewHarness(t, map[string]string{
        "Badge.vue": `&lt;template&gt;&lt;span class="badge"&gt;{{<!-- --> label }}&lt;/span&gt;&lt;/template&gt;`,
        "Card.vue": `&lt;template&gt;
            &lt;div class="card"&gt;
                &lt;h2&gt;{{<!-- --> title }}&lt;/h2&gt;
                &lt;Badge :label="status" /&gt;
            &lt;/div&gt;
        &lt;/template&gt;`,
    })

    h.Fragment("Card", map[string]any{
        "title":  "Order #42",
        "status": "shipped",
    }).
        Find(htmlctest.ByTag("h2")).AssertText("Order #42").
        Find(htmlctest.ByClass("badge")).AssertText("shipped")
}</code></pre>

    <h3>Assertion methods</h3>
    <p>All <code>Assert*</code> methods call <code>t.Fatalf</code> on failure and return the receiver for chaining:</p>

    <table>
      <thead><tr><th>Method</th><th>Checks</th></tr></thead>
      <tbody>
        <tr><td><code>r.AssertHTML(want)</code></td><td>Exact HTML after whitespace normalisation; reports tree diff on mismatch</td></tr>
        <tr><td><code>r.Find(query)</code></td><td>Returns a <code>Selection</code> of all matching nodes</td></tr>
        <tr><td><code>s.AssertExists()</code></td><td>At least one node matched</td></tr>
        <tr><td><code>s.AssertNotExists()</code></td><td>No nodes matched</td></tr>
        <tr><td><code>s.AssertCount(n)</code></td><td>Exactly <code>n</code> nodes matched</td></tr>
        <tr><td><code>s.AssertText(text)</code></td><td>Normalised text of the first matched node</td></tr>
        <tr><td><code>s.AssertAttr(attr, value)</code></td><td>Named attribute value of the first matched node</td></tr>
      </tbody>
    </table>

    <h3>Query constructors and combinators</h3>
    <p>Queries are immutable. Build and refine them with constructors and combinators:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>// Match &lt;li class="active" data-id="1"&gt; inside a &lt;ul&gt;
htmlctest.ByTag("li").
    WithClass("active").
    WithAttr("data-id", "1").
    Descendant(htmlctest.ByTag("ul"))</code></pre>

    <table>
      <thead><tr><th>Constructor / combinator</th><th>Matches</th></tr></thead>
      <tbody>
        <tr><td><code>ByTag("div")</code></td><td>Elements by tag name (case-insensitive)</td></tr>
        <tr><td><code>ByClass("active")</code></td><td>Elements that have the given CSS class</td></tr>
        <tr><td><code>ByAttr("data-id", "42")</code></td><td>Elements where <code>data-id="42"</code></td></tr>
        <tr><td><code>q.WithClass(class)</code></td><td>Also requires the given class</td></tr>
        <tr><td><code>q.WithAttr(attr, value)</code></td><td>Also requires the given attribute</td></tr>
        <tr><td><code>q.Descendant(ancestor)</code></td><td>Matched element must be inside <code>ancestor</code></td></tr>
      </tbody>
    </table>

    <h3>Testing v-for output</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func TestList(t *testing.T) {
    htmlctest.Build(t, `
        &lt;ul&gt;
            &lt;li v-for="item in items"&gt;{{<!-- --> item }}&lt;/li&gt;
        &lt;/ul&gt;
    `).
        Fragment("Root", map[string]any{
            "items": []string{"alpha", "beta", "gamma"},
        }).
        Find(htmlctest.ByTag("li")).
        AssertCount(3)
}</code></pre>

    <h3>Testing conditional rendering</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func TestBadge_Hidden(t *testing.T) {
    htmlctest.Build(t, `&lt;span v-if="show" class="badge"&gt;NEW&lt;/span&gt;`).
        Fragment("Root", map[string]any{"show": false}).
        Find(htmlctest.ByClass("badge")).
        AssertNotExists()
}</code></pre>

    <h3>Testing with custom directives</h3>
    <p>Pass an <code>htmlc.Options</code> with a <code>Directives</code> map to <code>NewHarness</code> to test components that use custom directives:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>h := htmlctest.NewHarness(t, map[string]string{
    "Page.vue": `&lt;template&gt;&lt;pre v-upper="text"&gt;&lt;/pre&gt;&lt;/template&gt;`,
}, htmlc.Options{
    Directives: htmlc.DirectiveRegistry{
        "upper": &amp;UpperDirective{},
    },
})</code></pre>

  </DocsPage>
</template>


<style>
  .howto-goal {
    font-style: italic;
    color: #c4c8e2;
    border-left: 3px solid #00ADD8;
    padding-left: 1rem;
    margin: 1rem 0 1.25rem;
  }
</style>
