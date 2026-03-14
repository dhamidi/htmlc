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
      {label: 'Customization'},
      {href: '#custom-directive', label: 'Custom directive'},
      {href: '#missing-props', label: 'Missing prop handling'},
      {label: 'Static sites'},
      {href: '#static-site', label: 'Static site with layout'},
      {href: '#syntax-highlight', label: 'Syntax highlighting'},
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

    <p>Each page component receives a <code>slot</code> prop containing the rendered inner page HTML. The layout component must render <code>{{ "{{" }} slot }}</code> (or use <code>v-html="slot"</code>) where the page content should appear. See the <a href="/docs/cli.html">CLI reference</a> for all flags.</p>

    <h3>Using the Go API</h3>
    <p>Call <code>RenderFragment</code> for the inner page, then pass the result as data to <code>RenderPage</code> on the layout:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>// Render the inner page as a fragment (no full &lt;html&gt; document).
inner, err := engine.RenderFragmentString(&#34;BlogPost&#34;, map[string]any{
    &#34;title&#34;:   post.Title,
    &#34;content&#34;: post.Body,
})
if err != nil {
    return err
}

// Wrap the fragment in the layout, which renders a full HTML document.
// The layout template uses {{ &#34;{{&#34; }} slot }} to embed the inner HTML.
html, err := engine.RenderPageString(&#34;Layout&#34;, map[string]any{
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

    <!-- ═══════════════════════════════════════════════ Testing -->
    <h2 id="testing">Testing components</h2>
    <p class="howto-goal">Write unit tests for <code>.vue</code> components without touching the filesystem.</p>

    <p>The <code>htmlctest</code> package provides a lightweight test harness for htmlc components. Import it in your <code>_test.go</code> files:</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>go get github.com/dhamidi/htmlc/htmlctest</code></pre>

    <h3>NewEngine</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine</code></pre>
    <p>Creates a test <code>Engine</code> backed by an in-memory filesystem. The <code>files</code> map uses file names as keys (e.g. <code>"Button.vue"</code>) and component source text as values. Pass optional <code>htmlc.Options</code> to configure directives, missing-prop handlers, etc.; the <code>FS</code> and <code>ComponentDir</code> fields are always overridden. The test fails immediately if the engine cannot be created.</p>

    <h3>AssertFragment</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string)</code></pre>
    <p>Renders <code>name</code> as an HTML fragment with <code>data</code> and fails the test if the output does not match <code>want</code> after whitespace normalisation (runs of whitespace collapsed to single spaces).</p>

    <h3>AssertRendersHTML</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string)</code></pre>
    <p>Like <code>AssertFragment</code> but renders a full HTML page (with <code>&lt;!DOCTYPE html&gt;</code> and scoped styles injected into <code>&lt;head&gt;</code>).</p>

    <h3>Example</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>package myapp_test

import (
    &#34;testing&#34;

    &#34;github.com/dhamidi/htmlc/htmlctest&#34;
)

func TestGreeting(t *testing.T) {
    e := htmlctest.NewEngine(t, map[string]string{
        &#34;Greeting.vue&#34;: `&lt;template&gt;&lt;p&gt;Hello {{<!-- -->name }}!&lt;/p&gt;&lt;/template&gt;`,
    })
    htmlctest.AssertFragment(t, e, &#34;Greeting&#34;,
        map[string]any{&#34;name&#34;: &#34;World&#34;},
        &#34;&lt;p&gt;Hello World!&lt;/p&gt;&#34;,
    )
}

func TestCard(t *testing.T) {
    e := htmlctest.NewEngine(t, map[string]string{
        &#34;Card.vue&#34;: `
&lt;template&gt;
  &lt;div class=&#34;card&#34;&gt;
    &lt;h2&gt;{{<!-- -->title }}&lt;/h2&gt;
    &lt;slot&gt;&lt;/slot&gt;
  &lt;/div&gt;
&lt;/template&gt;`,
    })
    htmlctest.AssertFragment(t, e, &#34;Card&#34;,
        map[string]any{&#34;title&#34;: &#34;Hello&#34;},
        `&lt;div class=&#34;card&#34;&gt;&lt;h2&gt;Hello&lt;/h2&gt;&lt;/div&gt;`,
    )
}</code></pre>

    <h3>Testing with custom directives</h3>
    <p>Pass an <code>htmlc.Options</code> with a <code>Directives</code> map to test components that use custom directives:</p>
    <pre v-syntax-highlight="'go'"><code v-pre>e := htmlctest.NewEngine(t, map[string]string{
    &#34;Page.vue&#34;: `&lt;template&gt;&lt;pre v-upper=&#34;text&#34;&gt;&lt;/pre&gt;&lt;/template&gt;`,
}, htmlc.Options{
    Directives: htmlc.DirectiveRegistry{
        &#34;upper&#34;: &amp;UpperDirective{},
    },
})</code></pre>

  </DocsPage>
</template>

<script>
export default {
  props: ['siteTitle']
}
</script>

<style>
  .howto-goal {
    font-style: italic;
    color: #c4c8e2;
    border-left: 3px solid #00ADD8;
    padding-left: 1rem;
    margin: 1rem 0 1.25rem;
  }
</style>
