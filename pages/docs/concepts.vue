<template>
  <DocsPage
    pageTitle="Concepts — htmlc.sh"
    description="Understanding how htmlc works: the rendering model, expression language, scoped styles, the Engine vs Renderer API, and how it differs from client-side Vue."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Rendering'},
      {href: '#rendering-model', label: 'The rendering model'},
      {href: '#components-as-templates', label: 'Components as templates'},
      {label: 'Expressions'},
      {href: '#expression-language', label: 'Expression language'},
      {label: 'Styles'},
      {href: '#scoped-styles', label: 'Scoped styles'},
      {label: 'API'},
      {href: '#engine-vs-renderer', label: 'Engine vs Renderer'},
      {label: 'Design'},
      {href: '#ssr-vs-csr', label: 'Server-side vs client-side'}
    ]"
  >
    <h1>Concepts</h1>
    <p class="lead">This page explains how htmlc works internally — the mental models and design decisions behind it. It is aimed at developers who want to reason about performance, debug unexpected output, understand limitations, or integrate htmlc into complex Go applications.</p>

    <!-- ═══════════════════════════════════════════════ Rendering Model -->
    <h2 id="rendering-model">The Rendering Model</h2>

    <p>At its core, htmlc is a template engine that turns a <code>.vue</code> file and a Go data map into a string of HTML. The data flow is straightforward:</p>

    <div class="data-flow">
      <div class="flow-box"><code v-pre>map[string]any</code><span class="flow-label">scope / data</span></div>
      <div class="flow-arrow">+</div>
      <div class="flow-box"><code v-pre>*.vue</code><span class="flow-label">parsed AST</span></div>
      <div class="flow-arrow">→</div>
      <div class="flow-box flow-box--accent"><code v-pre>Renderer</code><span class="flow-label">walks the AST</span></div>
      <div class="flow-arrow">→</div>
      <div class="flow-box"><code v-pre>HTML string</code><span class="flow-label">output bytes</span></div>
    </div>

    <p>When a <code>.vue</code> file is first loaded, htmlc parses it into an HTML abstract syntax tree (AST) using <code>golang.org/x/net/html</code>. The <code>&lt;template&gt;</code>, <code>&lt;style&gt;</code>, and <code>&lt;script&gt;</code> sections are separated and stored on the <code>Component</code> struct. The AST is kept in memory so that repeated renders incur no parsing cost.</p>

    <p>At render time, the <code>Renderer</code> walks the AST node by node. For each node it:</p>
    <ol>
      <li>Evaluates any directive attributes (<code>v-if</code>, <code>v-for</code>, <code>v-bind</code>, etc.) against the current scope.</li>
      <li>Interpolates <code v-pre>{{ expr }}</code> text nodes by evaluating the embedded expression.</li>
      <li>Recursively descends into child nodes, potentially with a modified scope (e.g., inside a <code>v-for</code> loop).</li>
      <li>Writes the resulting bytes to the output <code>io.Writer</code>.</li>
    </ol>

    <p>No JavaScript engine is involved at any point. The expression evaluator is a purpose-built Go library (the <code>expr</code> package) that understands a subset of JavaScript-like expression syntax and evaluates it directly against a <code>map[string]any</code>.</p>

    <Callout type="info">
      <strong>Key insight:</strong> Because the AST is built once and reused, and because evaluation is a pure in-memory traversal with no I/O, rendering a component is fast even under concurrent load. The only shared state is read-only (the parsed AST); the scope map is private to each render call.
    </Callout>

    <!-- ═══════════════════════════════════════════════ Components as Templates -->
    <h2 id="components-as-templates">Components as Templates, Not Objects</h2>

    <p>In client-side Vue, every component instance is a JavaScript object with its own reactive state, lifecycle hooks, and event listeners. htmlc takes a fundamentally different approach: a component is a <em>stateless template</em>. It has no instance, no lifecycle, and no reactivity.</p>

    <p>Each call to <code>RenderPage</code> or <code>RenderFragment</code> is a pure function call:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>f(name string, data map[string]any) → (HTML string, error)</code></pre>

    <p>The same component can be rendered a thousand times concurrently with different data maps and it will produce independent, deterministic output each time. There is no shared mutable state between renders.</p>

    <p>This design makes htmlc easy to reason about and easy to test: if you know the input data, you know the output HTML. It also means that things that are natural in Vue — computed properties, watchers, <code>$emit</code>, two-way binding — simply do not exist in htmlc. They are client-side concerns that belong in JavaScript, not in a Go server renderer.</p>

    <p>When a component includes another component (a child tag in the template), the renderer looks up the child by name in the component registry, creates a new scope from the parent's attribute expressions, and renders the child inline. The child has no reference to the parent; prop passing is one-directional and happens at the point of use.</p>

    <!-- ═══════════════════════════════════════════════ Expression Language -->
    <h2 id="expression-language">Expression Language</h2>

    <p>The <code v-pre>{{ expr }}</code> interpolation syntax and directive value expressions (e.g., <code>v-if="user.isAdmin"</code>, <code>:class="active ? 'on' : 'off'"</code>) are all evaluated by a custom Go library: the <code>htmlc/expr</code> package. It is not JavaScript — it is a declarative, side-effect-free subset of JavaScript expression syntax evaluated against a Go <code>map[string]any</code> scope.</p>

    <h3>What the expression language supports</h3>

    <table class="support-table">
      <thead>
        <tr><th>Feature</th><th>Example</th></tr>
      </thead>
      <tbody>
        <tr><td>Arithmetic</td><td><code v-pre>price * qty + shipping</code></td></tr>
        <tr><td>Comparison &amp; equality</td><td><code v-pre>count &gt; 0</code>, <code>status === 'active'</code></td></tr>
        <tr><td>Logical operators</td><td><code v-pre>isAdmin &amp;&amp; !isBanned</code></td></tr>
        <tr><td>Nullish coalescing</td><td><code v-pre>user.name ?? 'Anonymous'</code></td></tr>
        <tr><td>Ternary</td><td><code v-pre>age &gt;= 18 ? 'adult' : 'minor'</code></td></tr>
        <tr><td>Member access (dot)</td><td><code v-pre>post.author.name</code></td></tr>
        <tr><td>Member access (bracket)</td><td><code v-pre>items[0]</code>, <code>obj["key"]</code></td></tr>
        <tr><td>Optional chaining</td><td><code v-pre>user?.address?.city</code></td></tr>
        <tr><td>Array / object literals</td><td><code v-pre>[1, 2, 3]</code>, <code>{ "k": v }</code></td></tr>
        <tr><td>Function calls</td><td><code v-pre>formatDate(post.createdAt)</code></td></tr>
        <tr><td>String concatenation</td><td><code v-pre>'Hello, ' + name + '!'</code></td></tr>
        <tr><td><code v-pre>in</code> operator</td><td><code v-pre>"key" in obj</code></td></tr>
        <tr><td>Typeof</td><td><code v-pre>typeof value === 'string'</code></td></tr>
      </tbody>
    </table>

    <h3>What the expression language does NOT support</h3>

    <table class="support-table">
      <thead>
        <tr><th>Unsupported construct</th><th>Reason</th></tr>
      </thead>
      <tbody>
        <tr><td>Assignment (<code>x = y</code>, <code>x++</code>)</td><td>The evaluator is side-effect-free by design</td></tr>
        <tr><td>Arrow functions / closures</td><td>No JavaScript runtime; function values must come from Go</td></tr>
        <tr><td><code v-pre>this</code></td><td>No component instance concept</td></tr>
        <tr><td>JS builtins (<code>JSON.parse</code>, <code>Math.max</code>, <code>parseInt</code>)</td><td>Not registered by default; add via <code>RegisterFunc</code></td></tr>
        <tr><td>Template literals (<code>`${x}`</code>)</td><td>Not supported; use string concatenation instead</td></tr>
        <tr><td><code v-pre>new</code>, <code>class</code>, <code>delete</code></td><td>Object-oriented constructs are not applicable in this context</td></tr>
        <tr><td>Spread operator (<code>...arr</code>)</td><td>Not implemented</td></tr>
        <tr><td>Regular expressions</td><td>Not implemented</td></tr>
      </tbody>
    </table>

    <h3>Exposing Go functions to templates with RegisterFunc</h3>

    <p>The <code>Engine.RegisterFunc</code> method is the bridge between Go and the expression language. Any function registered this way becomes available by name in every expression evaluated by that engine:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine.RegisterFunc("formatDate", func(args ...any) (any, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("formatDate: want 1 arg")
    }
    t, ok := args[0].(time.Time)
    if !ok {
        return "", nil
    }
    return t.Format("2 Jan 2006"), nil
})</code></pre>

    <p>Once registered, templates can call it like any other expression:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;span&gt;&#123;&#123;<!---><!----> formatDate(post.publishedAt) }}&lt;/span&gt;</code></pre>

    <p>Functions registered via <code>RegisterFunc</code> are scoped to a single engine instance. For truly global functions (available to all engines in a process), use <code>expr.RegisterBuiltin</code> from the <code>htmlc/expr</code> package directly — but note that it modifies global state and must be called before any concurrent evaluation begins.</p>

    <Callout type="info">
      <strong>Identifiers and scope resolution:</strong> When the evaluator encounters an identifier, it checks the scope map first, then the engine's registered functions, then the global built-in table. If the name is absent from all three, it evaluates to <code>undefined</code> (a Go sentinel value, <code>expr.UndefinedValue</code>), not to an error. Missing props surface as <code>undefined</code> values and are handled by the configured <code>MissingPropHandler</code>.
    </Callout>

    <!-- ═══════════════════════════════════════════════ Scoped Styles -->
    <h2 id="scoped-styles">Scoped Styles</h2>

    <p>When a <code>.vue</code> file contains a <code>&lt;style scoped&gt;</code> block, htmlc transforms its CSS so that the rules apply only to elements produced by that component. This is done without a build step — the transformation happens at parse time when the component is first loaded.</p>

    <h3>Scope ID generation</h3>

    <p>Each component gets a stable, unique scope identifier derived from its file path. The <code>ScopeID</code> function computes an FNV-1a 32-bit hash of the path and formats it as an 8-character lowercase hex string:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>// Result: "data-v-a1b2c3d4" (the exact value depends on the file path)
id := htmlc.ScopeID("./components/Button.vue")</code></pre>

    <p>The scope ID is stable across restarts as long as the component file path does not change. Because it is derived purely from the path, no state is needed — any process that loads the same file will generate the same ID.</p>

    <h3>CSS selector rewriting</h3>

    <p>The <code>ScopeCSS</code> function rewrites every CSS selector in a scoped style block by appending an attribute selector to the last compound selector in each rule:</p>

    <pre v-syntax-highlight="'css'"><code v-pre>/* Before scoping */
p { color: red; }
.title h2 { font-size: 1.5rem; }

/* After scoping (scope ID: data-v-a1b2c3d4) */
p[data-v-a1b2c3d4] { color: red; }
.title h2[data-v-a1b2c3d4] { font-size: 1.5rem; }</code></pre>

    <p>This mirrors the same approach used by the Vue CLI / Vite for client-side SFCs. At-rules such as <code>@media</code> and <code>@keyframes</code> are passed through verbatim; the rewriting only targets regular selector blocks.</p>

    <h3>Attribute injection</h3>

    <p>At render time, every HTML element produced by a scoped component receives the scope attribute as an additional HTML attribute. For a component with scope ID <code>data-v-a1b2c3d4</code>:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Template --&gt;
&lt;p class="intro"&gt;Hello&lt;/p&gt;

&lt;!-- Rendered output --&gt;
&lt;p class="intro" data-v-a1b2c3d4&gt;Hello&lt;/p&gt;</code></pre>

    <p>Because the CSS rules now target <code>p[data-v-a1b2c3d4]</code> and the rendered element carries that attribute, the styles are effectively scoped to that component's output without affecting other <code>&lt;p&gt;</code> elements on the page.</p>

    <h3>StyleCollector and style delivery</h3>

    <p>htmlc does not inline <code>&lt;style&gt;</code> tags inside component output as it renders. Instead, a <code>StyleCollector</code> accumulates the transformed CSS from every component that participates in a render (the root component and all its children, transitively). The collected styles are returned alongside the rendered HTML and injected into the page in a single location.</p>

    <p>The injection strategy differs depending on which render method you use:</p>

    <table class="support-table">
      <thead>
        <tr><th>Method</th><th>Style injection</th><th>Use when</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code v-pre>RenderPage</code></td>
          <td>Finds the <code>&lt;/head&gt;</code> tag in the output and inserts a <code>&lt;style&gt;</code> block immediately before it</td>
          <td>Rendering a full HTML document that contains a <code>&lt;head&gt;</code> section</td>
        </tr>
        <tr>
          <td><code v-pre>RenderFragment</code></td>
          <td>Prepends a <code>&lt;style&gt;</code> block to the beginning of the output</td>
          <td>Rendering a partial HTML snippet (HTMX, turbo frames, layout slots)</td>
        </tr>
      </tbody>
    </table>

    <p>This separation of rendering and style collection means you can render the same component into different contexts without changing the component itself. It also avoids duplicate <code>&lt;style&gt;</code> blocks: styles from a component that is used multiple times in the same tree are collected and deduplicated before injection.</p>

    <!-- ═══════════════════════════════════════════════ Engine vs Renderer -->
    <h2 id="engine-vs-renderer">The Engine vs. The Renderer</h2>

    <p>htmlc exposes two levels of API. Understanding when each applies helps you choose the right tool and makes the codebase easier to navigate.</p>

    <h3>Engine (high-level)</h3>

    <p>The <code>Engine</code> type is what most applications use. It manages the full lifecycle of component rendering:</p>

    <ul>
      <li><strong>Discovery:</strong> On creation, it walks <code>ComponentDir</code> recursively and parses every <code>.vue</code> file it finds, registering each by its base name without extension.</li>
      <li><strong>Caching:</strong> Parsed ASTs are held in memory and reused across render calls. With <code>Options.Reload = true</code>, the engine stat-checks each file before rendering and re-parses any that have changed.</li>
      <li><strong>Concurrency:</strong> All render methods on <code>Engine</code> are safe for concurrent use from multiple goroutines.</li>
      <li><strong>HTTP integration:</strong> <code>ServeComponent</code> and <code>ServePageComponent</code> wrap components as <code>net/http</code> handler functions. <code>Mount</code> registers a map of routes at once.</li>
      <li><strong>Data middleware:</strong> <code>WithDataMiddleware</code> injects per-request data (current user, CSRF token, feature flags) into every render without modifying individual handler functions.</li>
    </ul>

    <p>Create one <code>Engine</code> at application startup and share it across all handlers:</p>

    <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "./components",
})
if err != nil {
    log.Fatal(err)
}</code></pre>

    <h3>Renderer (low-level)</h3>

    <p>The <code>Renderer</code> type renders a single component given an explicit component registry. It does not perform file discovery, caching, or hot-reload — the caller is responsible for providing all the parsed <code>Component</code> values. This makes it useful for:</p>

    <ul>
      <li><strong>Testing:</strong> Construct a registry with only the components under test; no filesystem access needed.</li>
      <li><strong>Advanced scenarios:</strong> Generate or transform <code>Component</code> values programmatically before rendering.</li>
      <li><strong>Embedding htmlc into a larger framework</strong> that manages its own component lifecycle.</li>
    </ul>

    <p>Most application code should use <code>Engine</code>. <code>Renderer</code> is an implementation detail and an escape hatch — it is what <code>Engine</code> uses internally for each render call, but you rarely need to instantiate one directly.</p>

    <Callout type="info">
      <strong>Summary:</strong> <code>Engine</code> = file discovery + caching + HTTP helpers + concurrency management. <code>Renderer</code> = one component + one registry + one render call. Start with <code>Engine</code>; reach for <code>Renderer</code> only in tests or when you need to control the registry yourself.
    </Callout>

    <!-- ═══════════════════════════════════════════════ SSR vs CSR Vue -->
    <h2 id="ssr-vs-csr">Server-Side vs. Client-Side Vue</h2>

    <p>htmlc deliberately uses Vue SFC syntax — the same <code>&lt;template&gt;</code>, <code>&lt;style scoped&gt;</code>, and <code>v-*</code> directives that Vue developers already know. This familiarity is intentional: it lowers the learning curve for teams that use Vue on the frontend while still writing Go on the backend.</p>

    <p>However, htmlc is not a port of Vue to Go. It is a strict subset of Vue SFC syntax adapted for pure server-side rendering. The differences are fundamental, not incidental:</p>

    <table class="support-table">
      <thead>
        <tr><th>Aspect</th><th>htmlc (server-side)</th><th>Vue (client-side)</th></tr>
      </thead>
      <tbody>
        <tr>
          <td>Runtime</td>
          <td>Go process; no JavaScript engine</td>
          <td>Browser JavaScript engine</td>
        </tr>
        <tr>
          <td>Component lifecycle</td>
          <td>None — render is a pure function</td>
          <td><code v-pre>onMounted</code>, <code>onUpdated</code>, <code>onUnmounted</code>, etc.</td>
        </tr>
        <tr>
          <td>Reactivity</td>
          <td>None — data is static for the duration of a render</td>
          <td>Reactive proxies via <code>ref</code> / <code>reactive</code></td>
        </tr>
        <tr>
          <td>Event handling</td>
          <td>Stripped from output (<code>@click</code> attributes are removed)</td>
          <td>DOM event listeners attached by the runtime</td>
        </tr>
        <tr>
          <td>Two-way binding</td>
          <td>Not supported (<code>v-model</code> is stripped)</td>
          <td>Core feature</td>
        </tr>
        <tr>
          <td>Output</td>
          <td>A string of HTML bytes, ready to serve</td>
          <td>A virtual DOM tree that the runtime reconciles with the real DOM</td>
        </tr>
      </tbody>
    </table>

    <h3>What happens to unsupported directives</h3>

    <p>htmlc does not error on directives it cannot meaningfully execute in a server context. Instead, it <em>strips</em> them from the output. This means:</p>

    <ul>
      <li><code v-pre>@click="handler"</code>, <code>v-on:submit="..."</code> — the event binding attribute is removed; the element itself is kept.</li>
      <li><code v-pre>v-model="value"</code> — removed from the element.</li>
      <li><code v-pre>v-once</code>, <code>v-memo</code> — removed; htmlc always renders each node fresh.</li>
    </ul>

    <p>This stripping behaviour is intentional. It allows you to write SFC templates that are shared in concept with a client-side Vue component — the server renders the initial HTML, and a separately bundled Vue application may hydrate and take over. The server's job is to produce ready-to-serve HTML; wiring up interactivity is the client's job.</p>

    <h3>The design philosophy</h3>

    <p>htmlc is designed around a single constraint: produce correct HTML as quickly as possible from a Go data map. Everything that would require a JavaScript runtime, mutable state, or asynchronous execution is out of scope. This constraint is what makes htmlc small, fast, and easy to embed — and it is why the library can confidently claim that rendering is a pure, deterministic function from data to HTML.</p>

    <p>See the <a href="/docs/directives.html">directives reference</a> for the complete list of supported directives, and the <a href="/docs/go-api.html">Go API reference</a> for the full type signatures discussed on this page.</p>
  </DocsPage>
</template>


<style>
  .data-flow {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
    background: #1a1d27;
    border: 1px solid rgba(255,255,255,0.06);
    border-radius: 8px;
    padding: 1.25rem 1.5rem;
    margin: 1.5rem 0;
  }

  .flow-box {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    background: #0f1117;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 6px;
    padding: 0.6rem 1rem;
  }

  .flow-box--accent {
    border-color: rgba(0, 173, 216, 0.4);
    background: rgba(0, 173, 216, 0.06);
  }

  .flow-label {
    font-size: 0.7rem;
    color: #8b8fa8;
    white-space: nowrap;
  }

  .flow-arrow {
    color: #8b8fa8;
    font-size: 1.1rem;
    font-weight: 300;
  }

  .support-table {
    width: 100%;
    border-collapse: collapse;
    margin: 1rem 0 1.5rem;
    font-size: 0.875rem;
  }

  .support-table th {
    text-align: left;
    padding: 0.5rem 0.75rem;
    background: #1a1d27;
    color: #8b8fa8;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid rgba(255,255,255,0.06);
  }

  .support-table td {
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid rgba(255,255,255,0.04);
    color: #c4c8e2;
    vertical-align: top;
  }

  .support-table tr:last-child td {
    border-bottom: none;
  }

  @media (max-width: 800px) {
    .data-flow {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
