<template>
  <DocsPage
    pageTitle="Documentation — htmlc.sh"
    description="htmlc documentation: template syntax, directives, component system, Go API, and expression language."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Getting started'},
      {href: '#overview', label: 'Overview'},
      {href: '#installation', label: 'Installation'},
      {href: '#quick-start', label: 'Quick start'},
      {href: '/docs/tutorial.html', label: 'Tutorial'},
      {label: 'Template syntax'},
      {href: '#interpolation', label: 'Interpolation'},
      {href: '#expressions', label: 'Expressions'},
      {href: '#directives', label: 'Directives'},
      {label: 'Components'},
      {href: '#component-system', label: 'Component system'},
      {href: '#props', label: 'Props'},
      {href: '#slots', label: 'Slots'},
      {href: '#scoped-styles', label: 'Scoped styles'},
      {label: 'Reference'},
      {href: '/docs/directives.html', label: 'All directives'},
      {href: '/docs/cli.html', label: 'CLI reference'},
      {href: '/docs/components.html', label: 'Component API'}
    ]"
  >
    <h1 id="overview">htmlc</h1>
    <p class="lead">A server-side Go template engine that uses Vue.js Single File Component (<code>.vue</code>) syntax for authoring but renders entirely in Go with no JavaScript runtime.</p>

    <Callout>
      <p><strong>This is a static rendering engine.</strong> There is no reactivity, virtual DOM, or client-side hydration. Templates are evaluated once per request and produce plain HTML.</p>
    </Callout>

    <h2 id="installation">Installation</h2>

    <h3>CLI</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

    <h3>Go package</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre>go get github.com/dhamidi/htmlc</code></pre>

    <h2 id="quick-start">Quick start</h2>

    <p>Create a component file:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/Greeting.vue --&gt;
&lt;template&gt;
  &lt;p&gt;Hello, &#123;&#123;<!---><!----> name }}!&lt;/p&gt;
&lt;/template&gt;</code></pre>

    <p>Render it:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc render -dir ./templates Greeting -props '{"name":"world"}'
&lt;p&gt;Hello, world!&lt;/p&gt;</code></pre>

    <p>Render as a full HTML page:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc page -dir ./templates Greeting -props '{"name":"world"}'
&lt;!DOCTYPE html&gt;
&lt;p&gt;Hello, world!&lt;/p&gt;</code></pre>

    <h2 id="interpolation">Text interpolation</h2>

    <p><code>{{ "{{" }} expr }}</code> evaluates the expression against the current render scope and HTML-escapes the result.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;p&gt;Hello, &#123;&#123;<!---><!----> name }}!&lt;/p&gt;
&lt;p&gt;&#123;&#123;<!---><!----> a }} + &#123;&#123;<!---> b }} = &#123;&#123;<!---> a + b }}&lt;/p&gt;</code></pre>

    <h2 id="expressions">Expression language</h2>

    <table>
      <thead>
        <tr><th>Category</th><th>Operators / Syntax</th></tr>
      </thead>
      <tbody>
        <tr><td>Arithmetic</td><td><code v-pre>+  -  *  /  %  **</code></td></tr>
        <tr><td>Comparison</td><td><code v-pre>===  !==  &gt;  &lt;  &gt;=  &lt;=  ==  !=</code></td></tr>
        <tr><td>Logical</td><td><code v-pre>&amp;&amp;  ||  !</code></td></tr>
        <tr><td>Nullish coalescing</td><td><code v-pre>??</code></td></tr>
        <tr><td>Optional chaining</td><td><code v-pre>obj?.key  arr?.[i]</code></td></tr>
        <tr><td>Ternary</td><td><code v-pre>condition ? then : else</code></td></tr>
        <tr><td>Member access</td><td><code v-pre>obj.key  arr[i]  arr.length</code></td></tr>
        <tr><td>Function calls</td><td><code v-pre>fn(args)</code> via <code>engine.RegisterFunc</code></td></tr>
        <tr><td>Array literals</td><td><code v-pre>[a, b, c]</code></td></tr>
        <tr><td>Object literals</td><td><code v-pre>&#123;&#123;<!---><!----> key: value }</code></td></tr>
      </tbody>
    </table>

    <p>Use <code>.length</code> to measure collections — it works on strings, slices, arrays, and maps:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;span&gt;&#123;&#123;<!---><!----> items.length }}&lt;/span&gt;</code></pre>

    <h2 id="directives">Directives overview</h2>

    <table>
      <thead>
        <tr><th>Directive</th><th>Supported</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr><td><code v-pre>v-if</code></td><td><span class="tag-supported">Yes</span></td><td>Renders element only when expression is truthy</td></tr>
        <tr><td><code v-pre>v-else-if</code></td><td><span class="tag-supported">Yes</span></td><td>Must follow <code>v-if</code> or <code>v-else-if</code></td></tr>
        <tr><td><code v-pre>v-else</code></td><td><span class="tag-supported">Yes</span></td><td>Must follow <code>v-if</code> or <code>v-else-if</code></td></tr>
        <tr><td><code v-pre>v-for</code></td><td><span class="tag-supported">Yes</span></td><td>Repeats element for each item</td></tr>
        <tr><td><code v-pre>v-show</code></td><td><span class="tag-supported">Yes</span></td><td>Toggles <code>display:none</code></td></tr>
        <tr><td><code v-pre>v-bind</code></td><td><span class="tag-supported">Yes</span></td><td>Dynamically binds attribute or prop</td></tr>
        <tr><td><code v-pre>v-html</code></td><td><span class="tag-supported">Yes</span></td><td>Sets inner HTML (unescaped)</td></tr>
        <tr><td><code v-pre>v-text</code></td><td><span class="tag-supported">Yes</span></td><td>Sets text content (HTML-escaped)</td></tr>
        <tr><td><code v-pre>v-pre</code></td><td><span class="tag-supported">Yes</span></td><td>Skips interpolation and directives for element and descendants</td></tr>
        <tr><td><code v-pre>v-switch / v-case</code></td><td><span class="tag-supported">Yes</span></td><td>Switch/case conditional; use with <code>v-case</code> and <code>v-default</code> on child elements</td></tr>
        <tr><td><code v-pre>v-slot</code></td><td><span class="tag-supported">Yes</span></td><td>Named and scoped slots</td></tr>
        <tr><td><code v-pre>v-model</code></td><td><span class="tag-no">Stripped</span></td><td>Client-side only; removed from output</td></tr>
        <tr><td><code v-pre>@event</code></td><td><span class="tag-no">Stripped</span></td><td>Client-side only; removed from output</td></tr>
      </tbody>
    </table>

    <p>See the <a href="/docs/directives.html">full directives reference</a> for detailed examples.</p>

    <h2 id="component-system">Component system</h2>

    <p>Components are <code>.vue</code> Single File Components with up to three sections:</p>
    <ul>
      <li><code v-pre>&lt;template&gt;</code> — required; the HTML template with directives</li>
      <li><code v-pre>&lt;script&gt;</code> — optional; preserved verbatim in output but never executed</li>
      <li><code v-pre>&lt;style&gt;</code> — optional; global or scoped CSS</li>
    </ul>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;&#123;&#123;<!---><!----> title }}&lt;/h2&gt;
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

    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc render -dir ./templates Card -props '{"title":"Hello"}'</code></pre>

    <pre v-syntax-highlight="'go'"><code v-pre>// In Go
html, err := engine.RenderFragmentString("Card", map[string]any{
    "title": "Hello",
})</code></pre>

    <h2 id="slots">Slots</h2>

    <p>Default slot:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In Card.vue --&gt;
&lt;slot&gt;Fallback content&lt;/slot&gt;

&lt;!-- Usage --&gt;
&lt;Card title="Hello"&gt;
  &lt;p&gt;This goes into the slot.&lt;/p&gt;
&lt;/Card&gt;</code></pre>

    <p>Named slots:</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- In Layout.vue --&gt;
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

    <pre v-syntax-highlight="'css'"><code v-pre>&lt;style scoped&gt;
.card { background: white; }
p    { color: gray; }
&lt;/style&gt;</code></pre>

    <p>Becomes (approximately):</p>
    <pre v-syntax-highlight="'css'"><code v-pre>&lt;style&gt;
.card[data-v-3a2b1c] { background: white; }
p[data-v-3a2b1c]    { color: gray; }
&lt;/style&gt;</code></pre>
  </DocsPage>
</template>

<style>
  .tag-supported { display: inline-block; background: var(--accent-alpha12); color: var(--accent); font-size: 0.75rem; font-weight: 600; padding: 0.1em 0.5em; border-radius: 4px; }
  .tag-no { display: inline-block; background: rgba(255,100,100,0.1); color: #ff8080; font-size: 0.75rem; font-weight: 600; padding: 0.1em 0.5em; border-radius: 4px; }
</style>
