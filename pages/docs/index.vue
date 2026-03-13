<template>
  <Layout pageTitle="Documentation — htmlc.sh" description="htmlc documentation: template syntax, directives, component system, Go API, and expression language." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <SidebarSection label="Getting started">
          <a href="#overview" class="sidebar-link">Overview</a>
          <a href="#installation" class="sidebar-link">Installation</a>
          <a href="#quick-start" class="sidebar-link">Quick start</a>
          <a href="/docs/tutorial.html" class="sidebar-link">Tutorial</a>
        </SidebarSection>
        <SidebarSection label="Template syntax">
          <a href="#interpolation" class="sidebar-link">Interpolation</a>
          <a href="#expressions" class="sidebar-link">Expressions</a>
          <a href="#directives" class="sidebar-link">Directives</a>
        </SidebarSection>
        <SidebarSection label="Components">
          <a href="#component-system" class="sidebar-link">Component system</a>
          <a href="#props" class="sidebar-link">Props</a>
          <a href="#slots" class="sidebar-link">Slots</a>
          <a href="#scoped-styles" class="sidebar-link">Scoped styles</a>
        </SidebarSection>
        <SidebarSection label="Reference">
          <a href="/docs/directives.html" class="sidebar-link">All directives</a>
          <a href="/docs/cli.html" class="sidebar-link">CLI reference</a>
          <a href="/docs/components.html" class="sidebar-link">Component API</a>
        </SidebarSection>
      </aside>

      <div class="docs-content">
        <h1 id="overview">htmlc</h1>
        <p class="lead">A server-side Go template engine that uses Vue.js Single File Component (<code>.vue</code>) syntax for authoring but renders entirely in Go with no JavaScript runtime.</p>

        <Callout>
          <p><strong>This is a static rendering engine.</strong> There is no reactivity, virtual DOM, or client-side hydration. Templates are evaluated once per request and produce plain HTML.</p>
        </Callout>

        <h2 id="installation">Installation</h2>

        <h3>CLI</h3>
        <pre><code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

        <h3>Go package</h3>
        <pre><code>go get github.com/dhamidi/htmlc</code></pre>

        <h2 id="quick-start">Quick start</h2>

        <p>Create a component file:</p>
        <pre><code>&lt;!-- templates/Greeting.vue --&gt;
&lt;template&gt;
  &lt;p&gt;Hello, {{ "{{" }} name }}!&lt;/p&gt;
&lt;/template&gt;</code></pre>

        <p>Render it:</p>
        <pre><code>$ htmlc render -dir ./templates Greeting -props '{"name":"world"}'
&lt;p&gt;Hello, world!&lt;/p&gt;</code></pre>

        <p>Render as a full HTML page:</p>
        <pre><code>$ htmlc page -dir ./templates Greeting -props '{"name":"world"}'
&lt;!DOCTYPE html&gt;
&lt;p&gt;Hello, world!&lt;/p&gt;</code></pre>

        <h2 id="interpolation">Text interpolation</h2>

        <p><code>{{ "{{" }} expr }}</code> evaluates the expression against the current render scope and HTML-escapes the result.</p>
        <pre><code>&lt;p&gt;Hello, {{ "{{" }} name }}!&lt;/p&gt;
&lt;p&gt;{{ "{{" }} a }} + {{ "{{" }} b }} = {{ "{{" }} a + b }}&lt;/p&gt;</code></pre>

        <h2 id="expressions">Expression language</h2>

        <table>
          <thead>
            <tr><th>Category</th><th>Operators / Syntax</th></tr>
          </thead>
          <tbody>
            <tr><td>Arithmetic</td><td><code>+  -  *  /  %  **</code></td></tr>
            <tr><td>Comparison</td><td><code>===  !==  &gt;  &lt;  &gt;=  &lt;=  ==  !=</code></td></tr>
            <tr><td>Logical</td><td><code>&amp;&amp;  ||  !</code></td></tr>
            <tr><td>Nullish coalescing</td><td><code>??</code></td></tr>
            <tr><td>Optional chaining</td><td><code>obj?.key  arr?.[i]</code></td></tr>
            <tr><td>Ternary</td><td><code>condition ? then : else</code></td></tr>
            <tr><td>Member access</td><td><code>obj.key  arr[i]  arr.length</code></td></tr>
            <tr><td>Function calls</td><td><code>fn(args)</code> via <code>engine.RegisterFunc</code></td></tr>
            <tr><td>Array literals</td><td><code>[a, b, c]</code></td></tr>
            <tr><td>Object literals</td><td><code>&#123;&#123; key: value }</code></td></tr>
          </tbody>
        </table>

        <p>Use <code>.length</code> to measure collections — it works on strings, slices, arrays, and maps:</p>
        <pre><code>&lt;span&gt;{{ "{{" }} items.length }}&lt;/span&gt;</code></pre>

        <h2 id="directives">Directives overview</h2>

        <table>
          <thead>
            <tr><th>Directive</th><th>Supported</th><th>Description</th></tr>
          </thead>
          <tbody>
            <tr><td><code>v-if</code></td><td><span class="tag-supported">Yes</span></td><td>Renders element only when expression is truthy</td></tr>
            <tr><td><code>v-else-if</code></td><td><span class="tag-supported">Yes</span></td><td>Must follow <code>v-if</code> or <code>v-else-if</code></td></tr>
            <tr><td><code>v-else</code></td><td><span class="tag-supported">Yes</span></td><td>Must follow <code>v-if</code> or <code>v-else-if</code></td></tr>
            <tr><td><code>v-for</code></td><td><span class="tag-supported">Yes</span></td><td>Repeats element for each item</td></tr>
            <tr><td><code>v-show</code></td><td><span class="tag-supported">Yes</span></td><td>Toggles <code>display:none</code></td></tr>
            <tr><td><code>v-bind</code></td><td><span class="tag-supported">Yes</span></td><td>Dynamically binds attribute or prop</td></tr>
            <tr><td><code>v-html</code></td><td><span class="tag-supported">Yes</span></td><td>Sets inner HTML (unescaped)</td></tr>
            <tr><td><code>v-text</code></td><td><span class="tag-supported">Yes</span></td><td>Sets text content (HTML-escaped)</td></tr>
            <tr><td><code>v-pre</code></td><td><span class="tag-supported">Yes</span></td><td>Skips interpolation and directives for element and descendants</td></tr>
            <tr><td><code>v-switch / v-case</code></td><td><span class="tag-supported">Yes</span></td><td>Switch/case conditional; use with <code>v-case</code> and <code>v-default</code> on child elements</td></tr>
            <tr><td><code>v-slot</code></td><td><span class="tag-supported">Yes</span></td><td>Named and scoped slots</td></tr>
            <tr><td><code>v-model</code></td><td><span class="tag-no">Stripped</span></td><td>Client-side only; removed from output</td></tr>
            <tr><td><code>@event</code></td><td><span class="tag-no">Stripped</span></td><td>Client-side only; removed from output</td></tr>
          </tbody>
        </table>

        <p>See the <a href="/docs/directives.html">full directives reference</a> for detailed examples.</p>

        <h2 id="component-system">Component system</h2>

        <p>Components are <code>.vue</code> Single File Components with up to three sections:</p>
        <ul>
          <li><code>&lt;template&gt;</code> — required; the HTML template with directives</li>
          <li><code>&lt;script&gt;</code> — optional; preserved verbatim in output but never executed</li>
          <li><code>&lt;style&gt;</code> — optional; global or scoped CSS</li>
        </ul>

        <pre><code>&lt;!-- templates/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ "{{" }} title }}&lt;/h2&gt;
    &lt;slot&gt;Default content&lt;/slot&gt;
  &lt;/div&gt;
&lt;/template&gt;

&lt;style scoped&gt;
.card {
  border: 1px solid #ccc;
  border-radius: 8px;
  padding: 1rem;
}
&lt;/style&gt;</code></pre>

        <h2 id="props">Props</h2>

        <p>Props are passed as a JSON map at render time. In <code>htmlc build</code>, props come from sibling <code>.json</code> files and <code>_data.json</code> files in parent directories.</p>

        <pre><code>$ htmlc render -dir ./templates Card -props '{"title":"Hello"}'</code></pre>

        <pre><code>// In Go
html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
})</code></pre>

        <h2 id="slots">Slots</h2>

        <p>Default slot:</p>
        <pre><code>&lt;!-- In Card.vue --&gt;
&lt;slot&gt;Fallback content&lt;/slot&gt;

&lt;!-- Usage --&gt;
&lt;Card title="Hello"&gt;
  &lt;p&gt;This goes into the slot.&lt;/p&gt;
&lt;/Card&gt;</code></pre>

        <p>Named slots:</p>
        <pre><code>&lt;!-- In Layout.vue --&gt;
&lt;header&gt;&lt;slot name="header" /&gt;&lt;/header&gt;
&lt;main&gt;&lt;slot /&gt;&lt;/main&gt;
&lt;footer&gt;&lt;slot name="footer" /&gt;&lt;/footer&gt;

&lt;!-- Usage --&gt;
&lt;Layout&gt;
  &lt;template #header&gt;&lt;nav&gt;...&lt;/nav&gt;&lt;/template&gt;
  &lt;p&gt;Main content&lt;/p&gt;
  &lt;template #footer&gt;&lt;p&gt;&amp;copy; 2024&lt;/p&gt;&lt;/template&gt;
&lt;/Layout&gt;</code></pre>

        <h2 id="scoped-styles">Scoped styles</h2>

        <p>Add <code>scoped</code> to <code>&lt;style&gt;</code> to keep styles contained to the component. The engine rewrites CSS selectors and adds a scope attribute to matching elements automatically.</p>

        <pre><code>&lt;style scoped&gt;
.card { background: white; }
p    { color: gray; }
&lt;/style&gt;</code></pre>

        <p>Becomes (approximately):</p>
        <pre><code>&lt;style&gt;
.card[data-v-3a2b1c] { background: white; }
p[data-v-3a2b1c]    { color: gray; }
&lt;/style&gt;</code></pre>
      </div>
    </div>

  </Layout>
</template>

<style>
  p { margin: 1rem 0; }
  ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
  li { margin: 0.25rem 0; }

  .docs-layout { display: grid; grid-template-columns: 220px 1fr; gap: 0; max-width: 1200px; margin: 0 auto; min-height: calc(100vh - var(--nav-height)); }
  @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } }

  .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
  .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }

  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); color: #e2e4f0; }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.1rem; margin: 2rem 0 0.5rem; color: #e2e4f0; }

  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }

  .tag-supported { display: inline-block; background: var(--accent-alpha12); color: var(--accent); font-size: 0.75rem; font-weight: 600; padding: 0.1em 0.5em; border-radius: 4px; }
  .tag-no { display: inline-block; background: rgba(255,100,100,0.1); color: #ff8080; font-size: 0.75rem; font-weight: 600; padding: 0.1em 0.5em; border-radius: 4px; }
</style>
