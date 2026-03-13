<template>
  <html lang="en">
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>Directives — {{ siteTitle }}</title>
      <meta name="description" content="Full reference for all htmlc template directives: v-if, v-for, v-bind, v-show, v-html, v-text, v-switch, v-slot." />
      <style>
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
        :root { --bg: #0f1117; --bg2: #1a1d27; --border: #2a2d3e; --text: #e2e4f0; --muted: #8b8fa8; --accent: #7c6af7; --accent2: #5be49b; --code-bg: #161822; --nav-height: 60px; }
        body { background: var(--bg); color: var(--text); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif; font-size: 16px; line-height: 1.65; min-height: 100vh; }
        a { color: var(--accent); text-decoration: none; }
        a:hover { text-decoration: underline; }
        code { font-family: "SF Mono","Fira Code","Cascadia Code",monospace; font-size: 0.875em; }
        pre { background: var(--code-bg); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem 1.5rem; overflow-x: auto; line-height: 1.5; margin: 1.5rem 0; }
        pre code { font-size: 0.85rem; color: #c9d1d9; }
        :not(pre) > code { background: var(--code-bg); border: 1px solid var(--border); border-radius: 4px; padding: 0.15em 0.4em; color: var(--accent2); }
        h1, h2, h3, h4 { font-weight: 700; line-height: 1.2; letter-spacing: -0.02em; }
        p { margin: 1rem 0; }
        ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
        li { margin: 0.25rem 0; }
        .site-nav { position: sticky; top: 0; z-index: 100; background: rgba(15,17,23,0.9); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); height: var(--nav-height); display: flex; align-items: center; padding: 0 2rem; gap: 2rem; }
        .site-nav .logo { font-weight: 800; font-size: 1.1rem; color: var(--text); letter-spacing: -0.03em; }
        .site-nav .logo span { color: var(--accent); }
        .site-nav .nav-links { display: flex; gap: 1.5rem; margin-left: auto; }
        .site-nav .nav-links a { color: var(--muted); font-size: 0.875rem; }
        .site-nav .nav-links a:hover { color: var(--text); text-decoration: none; }
        .docs-layout { display: grid; grid-template-columns: 220px 1fr; gap: 0; max-width: 1200px; margin: 0 auto; }
        @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } }
        .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
        .sidebar-section { margin-bottom: 1.5rem; }
        .sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--muted); margin-bottom: 0.5rem; padding: 0 0.5rem; }
        .sidebar-link { display: block; padding: 0.3rem 0.5rem; font-size: 0.875rem; color: var(--muted); border-radius: 4px; }
        .sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.05); text-decoration: none; }
        .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }
        .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
        .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); }
        .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
        .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }
        .site-footer { border-top: 1px solid var(--border); padding: 2rem; text-align: center; color: var(--muted); font-size: 0.8rem; }
        .site-footer a { color: var(--muted); }
        .callout { background: rgba(124,106,247,0.08); border: 1px solid rgba(124,106,247,0.25); border-radius: 8px; padding: 1rem 1.25rem; margin: 1.5rem 0; }
        .callout p { margin: 0; font-size: 0.9rem; color: #c9ccf5; }
      </style>
    </head>
    <body>
      <nav class="site-nav">
        <a href="/" class="logo">htmlc<span>.sh</span></a>
        <div class="nav-links">
          <a href="/docs/index.html">Docs</a>
          <a href="/docs/cli.html">CLI</a>
          <a href="/docs/directives.html">Directives</a>
          <a href="https://github.com/dhamidi/htmlc" target="_blank" rel="noopener">GitHub</a>
        </div>
      </nav>

      <div class="docs-layout">
        <aside class="docs-sidebar">
          <div class="sidebar-section">
            <div class="sidebar-label">Conditionals</div>
            <a href="#v-if" class="sidebar-link">v-if / v-else-if / v-else</a>
            <a href="#v-show" class="sidebar-link">v-show</a>
            <a href="#v-switch" class="sidebar-link">v-switch / v-case</a>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-label">Lists</div>
            <a href="#v-for" class="sidebar-link">v-for</a>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-label">Binding</div>
            <a href="#v-bind" class="sidebar-link">v-bind / :attr</a>
            <a href="#v-html" class="sidebar-link">v-html</a>
            <a href="#v-text" class="sidebar-link">v-text</a>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-label">Components</div>
            <a href="#v-slot" class="sidebar-link">v-slot / #slot</a>
          </div>
          <div class="sidebar-section">
            <div class="sidebar-label">Not supported</div>
            <a href="#not-supported" class="sidebar-link">Stripped directives</a>
          </div>
        </aside>

        <div class="docs-content">
          <h1>Directives</h1>
          <p class="lead">Full reference for all template directives supported by htmlc.</p>

          <h2 id="v-if">v-if / v-else-if / v-else</h2>
          <p>Renders the element only when the expression is truthy. Whitespace-only text nodes between branches are ignored.</p>
          <pre><code>&lt;p v-if="role === 'admin'"&gt;Admin panel&lt;/p&gt;
&lt;p v-else-if="role === 'editor'"&gt;Editor view&lt;/p&gt;
&lt;p v-else&gt;Read-only view&lt;/p&gt;</code></pre>

          <p>Works on <code>&lt;template&gt;</code> elements too (renders children only, no wrapper element):</p>
          <pre><code>&lt;template v-if="items.length &gt; 0"&gt;
  &lt;ul&gt;
    &lt;li v-for="item in items"&gt;{{ "{{" }} item }}&lt;/li&gt;
  &lt;/ul&gt;
&lt;/template&gt;
&lt;template v-else&gt;
  &lt;p&gt;No items.&lt;/p&gt;
&lt;/template&gt;</code></pre>

          <h2 id="v-show">v-show</h2>
          <p>Adds <code>style="display:none"</code> when the expression is falsy. The element is always rendered (unlike <code>v-if</code>). Merges with any existing <code>style</code> attribute.</p>
          <pre><code>&lt;div v-show="isVisible"&gt;Visible when isVisible is truthy&lt;/div&gt;</code></pre>

          <h2 id="v-switch">v-switch / v-case / v-default</h2>
          <p>Switch/case conditional (implements Vue RFC #482). Must be on a <code>&lt;template&gt;</code> element. Renders the first matching <code>v-case</code> branch.</p>
          <pre><code>&lt;template v-switch="status"&gt;
  &lt;div v-case="'active'"&gt;Active&lt;/div&gt;
  &lt;div v-case="'pending'"&gt;Pending approval&lt;/div&gt;
  &lt;div v-default&gt;Unknown status&lt;/div&gt;
&lt;/template&gt;</code></pre>

          <h2 id="v-for">v-for</h2>
          <p>Repeats the element for each item in the iterable. Supports arrays, maps, and objects.</p>
          <pre><code>&lt;!-- Array --&gt;
&lt;li v-for="item in items"&gt;{{ "{{" }} item }}&lt;/li&gt;

&lt;!-- With index --&gt;
&lt;li v-for="(item, index) in items"&gt;{{ "{{" }} index }}: {{ "{{" }} item }}&lt;/li&gt;

&lt;!-- Object/map --&gt;
&lt;li v-for="(value, key) in obj"&gt;{{ "{{" }} key }}: {{ "{{" }} value }}&lt;/li&gt;

&lt;!-- Range (integer) --&gt;
&lt;li v-for="i in 5"&gt;{{ "{{" }} i }}&lt;/li&gt;</code></pre>

          <div class="callout">
            <p><strong>Note:</strong> Map iteration order follows Go's <code>reflect.MapKeys()</code> — not insertion order. Sort your maps before passing them if order matters.</p>
          </div>

          <h2 id="v-bind">v-bind / :attr</h2>
          <p>Dynamically binds an HTML attribute to an expression. The shorthand is <code>:</code>.</p>
          <pre><code>&lt;!-- Long form --&gt;
&lt;a v-bind:href="url"&gt;Link&lt;/a&gt;

&lt;!-- Shorthand --&gt;
&lt;a :href="url"&gt;Link&lt;/a&gt;
&lt;img :src="imageUrl" :alt="imageAlt" /&gt;

&lt;!-- Boolean attributes: rendered only when truthy --&gt;
&lt;button :disabled="isLoading"&gt;Submit&lt;/button&gt;

&lt;!-- Class binding --&gt;
&lt;div :class="isActive ? 'active' : ''"&gt;...&lt;/div&gt;</code></pre>

          <p>When passing props to a component, <code>:propName</code> evaluates the expression:</p>
          <pre><code>&lt;Card :title="post.title" :author="post.author" /&gt;</code></pre>

          <h2 id="v-html">v-html</h2>
          <p>Sets the element's inner HTML to the expression value. The value is <strong>not</strong> HTML-escaped. Only use with trusted content.</p>
          <pre><code>&lt;div v-html="renderedMarkdown"&gt;&lt;/div&gt;</code></pre>

          <div class="callout">
            <p><strong>Warning:</strong> Never use <code>v-html</code> with user-supplied data — it can introduce XSS vulnerabilities.</p>
          </div>

          <h2 id="v-text">v-text</h2>
          <p>Sets the element's text content to the expression value. HTML-escaped. Replaces all child nodes.</p>
          <pre><code>&lt;span v-text="message"&gt;&lt;/span&gt;
&lt;!-- equivalent to --&gt;
&lt;span&gt;{{ "{{" }} message }}&lt;/span&gt;</code></pre>

          <h2 id="v-slot">v-slot / #slot</h2>
          <p>Passes content into a named slot of a child component.</p>
          <pre><code>&lt;!-- In Layout.vue --&gt;
&lt;header&gt;&lt;slot name="header" /&gt;&lt;/header&gt;
&lt;main&gt;&lt;slot /&gt;&lt;/main&gt;

&lt;!-- Usage --&gt;
&lt;Layout&gt;
  &lt;template #header&gt;
    &lt;nav&gt;&lt;a href="/"&gt;Home&lt;/a&gt;&lt;/nav&gt;
  &lt;/template&gt;
  &lt;p&gt;Page content goes into the default slot.&lt;/p&gt;
&lt;/Layout&gt;</code></pre>

          <p>Scoped slots (slot props):</p>
          <pre><code>&lt;!-- In List.vue --&gt;
&lt;ul&gt;
  &lt;li v-for="item in items"&gt;
    &lt;slot :item="item"&gt;{{ "{{" }} item }}&lt;/slot&gt;
  &lt;/li&gt;
&lt;/ul&gt;

&lt;!-- Usage --&gt;
&lt;List :items="posts"&gt;
  &lt;template #default="{ item }"&gt;
    &lt;a :href="item.url"&gt;{{ "{{" }} item.title }}&lt;/a&gt;
  &lt;/template&gt;
&lt;/List&gt;</code></pre>

          <h2 id="not-supported">Stripped directives</h2>
          <p>These directives are parsed but produce no output — they are client-side only and have no meaning in a server-side renderer:</p>
          <ul>
            <li><code>v-model</code> — two-way binding</li>
            <li><code>@event</code> / <code>v-on:event</code> — event listeners</li>
            <li><code>v-once</code> — one-time render optimisation hint</li>
            <li><code>v-memo</code> — memoisation hint</li>
            <li><code>v-cloak</code> — FOUC prevention</li>
          </ul>
        </div>
      </div>

      <footer class="site-footer">
        <p>htmlc — MIT License &mdash; <a href="https://github.com/dhamidi/htmlc">github.com/dhamidi/htmlc</a></p>
      </footer>
    </body>
  </html>
</template>
