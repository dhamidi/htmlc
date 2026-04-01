<template>
  <DocsPage
    pageTitle="Custom Elements — htmlc.sh"
    description="Reference for the &lt;script customelement&gt; block: tag name derivation, Declarative Shadow DOM, rendered output, the importMap template function, Go API, and CLI behavior."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'SFC syntax'},
      {href: '#script-block', label: 'script customelement block'},
      {href: '#shadow-dom', label: 'Declarative Shadow DOM'},
      {label: 'Tag names'},
      {href: '#tag-derivation', label: 'Tag name derivation'},
      {label: 'Rendered output'},
      {href: '#rendered-output', label: 'Light DOM output'},
      {href: '#shadow-dom-output', label: 'Shadow DOM output'},
      {label: 'Template function'},
      {href: '#import-map', label: 'importMap()'},
      {label: 'Go API'},
      {href: '#collect-custom-elements', label: 'CollectCustomElements'},
      {href: '#script-handler', label: 'ScriptHandler'},
      {href: '#render-page-with-collector', label: 'RenderPageWithCollector'},
      {href: '#write-scripts', label: 'WriteScripts'},
      {href: '#new-collector', label: 'NewCustomElementCollector'},
      {href: '#new-script-fs-server', label: 'NewScriptFSServer'},
      {href: '#collector-methods', label: 'Collector methods'},
      {label: 'CLI'},
      {href: '#cli', label: 'CLI behavior'}
    ]"
  >
    <h1>Custom Elements</h1>
    <p class="lead">Reference for Web Component support in htmlc. A <code>.vue</code> component opts in by including a <code>&lt;script customelement&gt;</code> block. The engine derives a kebab-case tag name from the file path, wraps the rendered template in that tag, and collects the JavaScript for serving or writing to disk.</p>

    <!-- ═══════════════════════════════════════════════ SFC syntax -->
    <h2 id="script-block">The <code>&lt;script customelement&gt;</code> block</h2>

    <p>Place a <code>&lt;script customelement&gt;</code> block inside any <code>.vue</code> file alongside the <code>&lt;template&gt;</code> and optional <code>&lt;style&gt;</code> blocks. The block contains a plain Web Component class and a <code>customElements.define()</code> call.</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- components/ui/Counter.vue --&gt;
&lt;template&gt;
  &lt;button class="counter-demo"&gt;Count: &lt;span&gt;{{ initial }}&lt;/span&gt;&lt;/button&gt;
&lt;/template&gt;

&lt;script customelement&gt;
class UiCounter extends HTMLElement {
  connectedCallback() {
    const span = this.querySelector('span')
    let n = parseInt(span.textContent, 10)
    this.addEventListener('click', () =&gt; { span.textContent = ++n })
  }
}
customElements.define('ui-counter', UiCounter)
&lt;/script&gt;</code></pre>

    <p>The rendered HTML output for <code>&lt;UiCounter :initial="0"&gt;&lt;/UiCounter&gt;</code> is:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;ui-counter&gt;&lt;button class="counter-demo"&gt;Count: &lt;span&gt;0&lt;/span&gt;&lt;/button&gt;&lt;/ui-counter&gt;</code></pre>

    <h3>Live demo</h3>
    <p>Click the button to increment the counter:</p>
    <Counter :initial="0"></Counter>

    <Callout>
      <p><strong>Constraint:</strong> <code>&lt;script customelement&gt;</code> cannot coexist with <code>&lt;script&gt;</code> or <code>&lt;script setup&gt;</code> blocks. Combining them causes a parse error.</p>
    </Callout>

    <h2 id="shadow-dom">Declarative Shadow DOM</h2>

    <p>Add the <code>shadowdom</code> attribute to request Declarative Shadow DOM. The renderer wraps the template output in a <code>&lt;template shadowrootmode="..."&gt;</code> element inside the custom element tag.</p>

    <table>
      <thead>
        <tr><th>Attribute</th><th>Shadow root mode</th></tr>
      </thead>
      <tbody>
        <tr><td><code>shadowdom</code></td><td>open</td></tr>
        <tr><td><code>shadowdom="closed"</code></td><td>closed</td></tr>
      </tbody>
    </table>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- open shadow root --&gt;
&lt;script customelement shadowdom&gt;
class MyWidget extends HTMLElement { /* … */ }
customElements.define('my-widget', MyWidget);
&lt;/script&gt;

&lt;!-- closed shadow root --&gt;
&lt;script customelement shadowdom="closed"&gt;
class MyWidget extends HTMLElement { /* … */ }
customElements.define('my-widget', MyWidget);
&lt;/script&gt;</code></pre>

    <!-- ═══════════════════════════════════════════════ Tag name derivation -->
    <h2 id="tag-derivation">Tag name derivation</h2>

    <p>htmlc derives the custom element tag name automatically from the component's file path relative to <code>ComponentDir</code>. No manual registration is required.</p>

    <p><strong>Algorithm:</strong></p>
    <ol>
      <li>Split the path on <code>/</code> to get path segments.</li>
      <li>Strip the <code>.vue</code> extension from the last segment.</li>
      <li>Convert each segment from PascalCase/camelCase to kebab-case.</li>
      <li>Join all segments with <code>-</code> and lowercase the result.</li>
    </ol>

    <table>
      <thead>
        <tr><th>File path (relative to ComponentDir)</th><th>Derived tag</th></tr>
      </thead>
      <tbody>
        <tr><td><code>Button.vue</code></td><td><code>button</code> ⚠ (no hyphen)</td></tr>
        <tr><td><code>ui/Button.vue</code></td><td><code>ui-button</code></td></tr>
        <tr><td><code>widgets/ShapeCanvas.vue</code></td><td><code>widgets-shape-canvas</code></td></tr>
        <tr><td><code>nav/TopBar.vue</code></td><td><code>nav-top-bar</code></td></tr>
      </tbody>
    </table>

    <Callout>
      <p><strong>Warning:</strong> If the derived tag contains no hyphen, htmlc emits a warning at parse time. The Custom Elements specification requires at least one hyphen in the tag name. Place components in a subdirectory (e.g. <code>ui/Button.vue</code>) to ensure a valid tag.</p>
    </Callout>

    <!-- ═══════════════════════════════════════════════ Rendered output -->
    <h2 id="rendered-output">Rendered output — light DOM</h2>

    <p>At render time the component's template output is wrapped in its derived custom element tag. No shadow root is added.</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- widgets/ShapeCanvas.vue template --&gt;
&lt;canvas :width="width" :height="height" :data-src="src"&gt;&lt;/canvas&gt;</code></pre>

    <p>Renders as:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;widgets-shape-canvas&gt;&lt;canvas width="400" height="300" data-src="/api/stream"&gt;&lt;/canvas&gt;&lt;/widgets-shape-canvas&gt;</code></pre>

    <h2 id="shadow-dom-output">Rendered output — Declarative Shadow DOM</h2>

    <p>With the <code>shadowdom</code> attribute the inner HTML is wrapped in a <code>&lt;template shadowrootmode="..."&gt;</code> element, enabling streaming SSR for shadow roots.</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- open shadow root --&gt;
&lt;widgets-shape-canvas&gt;&lt;template shadowrootmode="open"&gt;&lt;canvas width="400" height="300"&gt;&lt;/canvas&gt;&lt;/template&gt;&lt;/widgets-shape-canvas&gt;

&lt;!-- closed shadow root --&gt;
&lt;widgets-shape-canvas&gt;&lt;template shadowrootmode="closed"&gt;&lt;canvas width="400" height="300"&gt;&lt;/canvas&gt;&lt;/template&gt;&lt;/widgets-shape-canvas&gt;</code></pre>

    <!-- ═══════════════════════════════════════════════ importMap template function -->
    <h2 id="import-map">The <code v-pre>importMap()</code> template function</h2>

    <p><code v-pre>importMap(urlPrefix)</code> is a template function automatically available in every component scope. It returns the JSON string produced by <code>collector.ImportMapJSON(urlPrefix)</code>, suitable for embedding inside a <code>&lt;script type="importmap"&gt;</code> element.</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In your page layout &lt;head&gt; --&gt;
&lt;script type="importmap"&gt;{{ importMap("/scripts/") }}&lt;/script&gt;</code></pre>

    <p>The import map JSON maps each custom element tag name to its hashed script URL:</p>

    <pre v-syntax-highlight="'json'"><code v-pre>{
  "imports": {
    "ui-date-picker":       "/scripts/a1b2c3d4e5f6a7b8.js",
    "widgets-shape-canvas": "/scripts/ff00112233445566.js"
  }
}</code></pre>

    <Callout>
      <p><strong>Note:</strong> <code v-pre>importMap()</code> is a no-op when no custom element components are present — it returns an empty import map JSON object. Place it unconditionally in your layout's <code>&lt;head&gt;</code>; pages without custom elements produce no observable overhead.</p>
    </Callout>

    <p>The function is available after <code>RenderPage</code>, <code>RenderFragment</code>, <code>RenderPageWithCollector</code>, or any render path that goes through the engine.</p>

    <!-- ═══════════════════════════════════════════════ Go API -->
    <h2 id="collect-custom-elements">engine.CollectCustomElements</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) CollectCustomElements() (*CustomElementCollector, error)</code></pre>

    <p>Renders all registered pages and collects their custom element scripts without producing any HTML output. Returns a fully-populated <code>*CustomElementCollector</code>.</p>

    <p>Useful for pre-warming an import map or script bundle at startup, and for testing that the expected scripts are registered. Returns an error when the engine has no registered components.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>collector, err := engine.CollectCustomElements()
if err != nil {
    log.Fatal(err)
}
log.Printf("collected %d custom element scripts", collector.Len())</code></pre>

    <h2 id="script-handler">engine.ScriptHandler</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) ScriptHandler() http.Handler</code></pre>

    <p>Returns an <code>http.Handler</code> that serves the engine's collected custom element scripts. Mount it under a path prefix using <code>http.StripPrefix</code>:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>http.Handle("/scripts/", http.StripPrefix("/scripts/", engine.ScriptHandler()))</code></pre>

    <p>The handler serves two kinds of responses:</p>
    <ul>
      <li><strong>Hashed <code>.js</code> files</strong> (e.g. <code>/scripts/a1b2c3d4.js</code>) — served from the in-memory <code>fs.FS</code> with <code>Cache-Control: immutable</code>.</li>
      <li><strong><code>index.js</code></strong> — an ES module entry point that imports all collected scripts using relative paths. Served without a long-lived cache header so it stays fresh after rebuilds.</li>
    </ul>

    <h2 id="render-page-with-collector">engine.RenderPageWithCollector</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) RenderPageWithCollector(ctx context.Context, w io.Writer, name string, data map[string]any, collector *CustomElementCollector) error</code></pre>

    <p>Like <code>RenderPage</code> but populates the given <code>collector</code> with every custom element script encountered during the render. Use this when you manage the collector lifecycle yourself — for example, when rendering multiple pages into a single collector before building an import map.</p>

    <p>Most callers should use <code>RenderPage</code> or <code>RenderFragment</code> instead; those methods manage the collector lifecycle automatically.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>collector := htmlc.NewCustomElementCollector()
var buf bytes.Buffer
if err := engine.RenderPageWithCollector(r.Context(), &amp;buf, "HomePage", data, collector); err != nil {
    http.Error(w, err.Error(), 500)
    return
}</code></pre>

    <h2 id="write-scripts">engine.WriteScripts</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func (e *Engine) WriteScripts(dir string) error</code></pre>

    <p>Writes all collected custom element scripts to <code>dir</code> as content-hashed <code>.js</code> files. Creates <code>dir</code> if it does not exist. This is the static-build equivalent of <code>ScriptHandler</code>. A no-op when no custom element scripts have been collected.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>if err := engine.WriteScripts("dist/scripts/"); err != nil {
    log.Fatal(err)
}</code></pre>

    <h2 id="new-collector">htmlc.NewCustomElementCollector</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func NewCustomElementCollector() *CustomElementCollector</code></pre>

    <p>Creates a new, empty <code>CustomElementCollector</code>. For advanced use cases where you manage the collector lifecycle yourself. Most callers do not need this — the engine creates and manages a collector automatically.</p>

    <h2 id="new-script-fs-server">htmlc.NewScriptFSServer</h2>

    <pre v-syntax-highlight="'go'"><code v-pre>func NewScriptFSServer(collector *CustomElementCollector) http.Handler</code></pre>

    <p>Like <code>engine.ScriptHandler()</code>, but for a manually-created collector. Serves hashed <code>.js</code> files from <code>collector.ScriptsFS()</code> and responds to <code>index.js</code> requests with <code>collector.IndexJS()</code> (relative imports).</p>

    <pre v-syntax-highlight="'go'"><code v-pre>http.Handle("/scripts/", http.StripPrefix("/scripts/",
    htmlc.NewScriptFSServer(collector)))</code></pre>

    <h2 id="collector-methods">Collector methods</h2>

    <h3>ScriptsFS</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (c *CustomElementCollector) ScriptsFS() fs.FS</code></pre>
    <p>Returns an in-memory <code>fs.FS</code> containing one content-hashed <code>.js</code> file per unique script collected. File names are the first 16 hex characters of the SHA-256 hash of the script source.</p>

    <h3>IndexJS</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (c *CustomElementCollector) IndexJS() string</code></pre>
    <p>Returns an ES-module entry point that imports all collected scripts using relative paths (<code v-pre>import "./&lt;hash&gt;.js"</code>). Duplicate hashes (same content registered under different tags) are emitted only once. Returns an empty string when no scripts have been collected.</p>

    <h3>ImportMapJSON</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (c *CustomElementCollector) ImportMapJSON(urlPrefix string) string</code></pre>
    <p>Returns a JSON string suitable for embedding in a <code>&lt;script type="importmap"&gt;</code> element. Each entry maps the custom element tag name to <code>urlPrefix + "&lt;hash&gt;.js"</code>.</p>

    <pre v-syntax-highlight="'go'"><code v-pre>json := collector.ImportMapJSON("/scripts/")
// {"imports":{"ui-date-picker":"/scripts/a1b2c3d4e5f6a7b8.js"}}</code></pre>

    <h3>Len</h3>
    <pre v-syntax-highlight="'go'"><code v-pre>func (c *CustomElementCollector) Len() int</code></pre>
    <p>Returns the number of unique scripts collected (deduplicated by content hash).</p>

    <!-- ═══════════════════════════════════════════════ CLI -->
    <h2 id="cli">CLI behavior</h2>

    <p>The <code>htmlc</code> CLI handles custom element scripts automatically — no extra flags required.</p>

    <table>
      <thead>
        <tr><th>Command</th><th>Custom element behavior</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code>htmlc build</code></td>
          <td>Writes collected scripts to <code>&lt;out&gt;/scripts/</code> after all pages are rendered. The directory is only created when at least one custom element component is present.</td>
        </tr>
        <tr>
          <td><code>htmlc build -dev :addr</code></td>
          <td>Serves scripts from memory at <code>/scripts/</code>, rebuilding automatically when source files change.</td>
        </tr>
      </tbody>
    </table>

    <p>For all CLI flags and options, see the <a href="/docs/cli.html">CLI reference</a>.</p>

    <p>Output directory structure when custom elements are present:</p>

    <pre v-syntax-highlight="'bash'"><code v-pre>out/
  index.html
  about.html
  scripts/
    a1b2c3d4e5f6a7b8.js   # ui-date-picker
    ff00112233445566.js   # widgets-shape-canvas</code></pre>

    <Callout>
      <p><strong>Note:</strong> Script files are deduplicated by content hash. The same <code>&lt;script customelement&gt;</code> source used in multiple components produces exactly one file in <code>scripts/</code>.</p>
    </Callout>
  </DocsPage>
</template>

