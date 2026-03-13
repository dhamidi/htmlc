<template>
  <article class="docs">
    <h1>htmlc Documentation</h1>

    <section class="docs-section">
      <h2>Template Syntax</h2>
      <p>
        htmlc renders Vue-style Single File Components to static HTML.
        Templates support mustache interpolation, directives, and component
        composition.
      </p>

      <h3>Mustache Interpolation</h3>
      <p>Use double-curly-brace syntax to output values from the render scope:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;span&gt;{{ items.length }}&lt;/span&gt;
&lt;p&gt;Hello, {{ user.name }}!&lt;/p&gt;</code></pre>

      <h3>Directives</h3>
      <p>Directives are special attributes prefixed with <code>v-</code>.</p>

      <h4>v-if / v-else-if / v-else</h4>
      <p>Conditionally render elements:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;p v-if="count &gt; 0"&gt;{{ count }} items&lt;/p&gt;
&lt;p v-else&gt;No items.&lt;/p&gt;</code></pre>

      <h4>v-for</h4>
      <p>Render a list by iterating over a collection:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;ul&gt;
  &lt;li v-for="item in items"&gt;{{ item.name }}&lt;/li&gt;
&lt;/ul&gt;</code></pre>

      <h4>v-text and v-html</h4>
      <p>Set element text or raw HTML content:</p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;span v-text="message"&gt;&lt;/span&gt;
&lt;div v-html="richContent"&gt;&lt;/div&gt;</code></pre>

      <h4>v-pre</h4>
      <p>
        Skip mustache evaluation for an element and its children. Useful when
        displaying template syntax as literal text:
      </p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;code v-pre&gt;{{ this.is.not.evaluated }}&lt;/code&gt;</code></pre>
    </section>

    <section class="docs-section">
      <h2>Components</h2>
      <p>
        Components are <code>.vue</code> files placed in the component directory
        passed to <code>htmlc build -dir</code>. They can define a
        <code>&lt;template&gt;</code>, <code>&lt;script&gt;</code> (for prop
        declarations), and <code>&lt;style&gt;</code> blocks.
      </p>

      <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- Card.vue --&gt;
&lt;template&gt;
  &lt;div class="card"&gt;
    &lt;h2&gt;{{ title }}&lt;/h2&gt;
    &lt;slot&gt;Default content.&lt;/slot&gt;
  &lt;/div&gt;
&lt;/template&gt;

&lt;script&gt;
export default {
  props: {
    title: { type: String, default: "" }
  }
}
&lt;/script&gt;</code></pre>
    </section>

    <section class="docs-section">
      <h2>External Directives</h2>
      <p>
        Custom directives can be implemented as external executables. Place an
        executable named <code>v-&lt;directive-name&gt;</code> in the component
        directory and htmlc will discover and use it automatically.
      </p>

      <pre v-syntax-highlight="'go'"><code v-pre>// Request received by the executable (as NDJSON on stdin)
{
  "hook":    "created",
  "id":      "1",
  "tag":     "pre",
  "text":    "func main() {}",
  "binding": { "value": "go" }
}</code></pre>

      <h3>Syntax Highlighting with v-syntax-highlight</h3>
      <p>
        The bundled <code>v-syntax-highlight</code> directive highlights code
        blocks using the <a href="https://github.com/alecthomas/chroma">chroma</a>
        library:
      </p>
      <pre v-syntax-highlight="'html'"><code v-pre>&lt;pre v-syntax-highlight="'go'"&gt;func main() {
    fmt.Println("hello, world")
}&lt;/pre&gt;</code></pre>

      <p>Generate the required CSS stylesheet:</p>
      <pre v-syntax-highlight="'sh'"><code v-pre>v-syntax-highlight -print-css -style monokai &gt; assets/highlight.css</code></pre>
    </section>

    <section class="docs-section">
      <h2>Building a Site</h2>
      <p>
        Use <code>htmlc build</code> to render every <code>.vue</code> page to
        static HTML:
      </p>
      <pre v-syntax-highlight="'sh'"><code v-pre>htmlc build -dir ./components -pages ./pages -out ./dist -layout Layout</code></pre>
    </section>
  </article>
</template>

<script>
export default {
  props: {
    pageTitle: { type: String, default: "Documentation" }
  }
}
</script>

<style>
.docs h1 {
  font-size: 2.2rem;
  margin-bottom: 2rem;
  border-bottom: 2px solid #1a1a1a;
  padding-bottom: 0.5rem;
}

.docs-section {
  margin-bottom: 3rem;
}

.docs-section h2 {
  font-size: 1.6rem;
  margin: 2rem 0 1rem;
}

.docs-section h3 {
  font-size: 1.2rem;
  margin: 1.5rem 0 0.5rem;
}

.docs-section h4 {
  font-size: 1rem;
  margin: 1rem 0 0.25rem;
  font-family: "SF Mono", "Fira Code", monospace;
  color: #444;
}

.docs-section p {
  margin-bottom: 0.75rem;
}

.docs-section pre {
  background: #272822;
  border-radius: 4px;
  padding: 1rem 1.25rem;
  overflow-x: auto;
  margin: 0.75rem 0 1.25rem;
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.875rem;
  line-height: 1.5;
}

.docs-section code {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.875em;
  background: #e8e6e1;
  padding: 0.1em 0.3em;
  border-radius: 2px;
}

.docs-section pre code {
  background: none;
  padding: 0;
  font-size: inherit;
  color: #f8f8f2;
}
</style>
