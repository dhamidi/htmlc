<template>
  <DocsPage
    pageTitle="CLI Reference — htmlc.sh"
    description="htmlc CLI reference: render, page, build, props, ast subcommands."
    :siteTitle="siteTitle"
    :navItems="[
      {label: 'Overview'},
      {href: '#synopsis', label: 'Synopsis'},
      {href: '#installation', label: 'Installation'},
      {label: 'Subcommands'},
      {href: '#render', label: 'render'},
      {href: '#page', label: 'page'},
      {href: '#build', label: 'build'},
      {href: '#props', label: 'props'},
      {href: '#ast', label: 'ast'},
      {href: '#help', label: 'help'},
      {label: 'Guides'},
      {href: '#static-site', label: 'Static sites'},
      {href: '#layouts', label: 'Layouts'},
      {href: '#data-files', label: 'Data files'},
      {href: '#page-partials', label: 'Page partials'},
      {label: 'External directives'},
      {href: '#external-directives', label: 'Overview'},
      {href: '#directive-discovery', label: 'Discovery'},
      {href: '#directive-protocol', label: 'Protocol'},
      {href: '#directive-created', label: 'created hook'},
      {href: '#directive-mounted', label: 'mounted hook'}
    ]"
  >
    <h1 id="synopsis">CLI Reference</h1>
    <p class="lead"><code v-pre>htmlc</code> renders Vue Single File Components (<code>.vue</code>) to HTML entirely in Go — no Node.js, no browser, no JavaScript runtime.</p>

    <h2 id="installation">Installation</h2>
    <pre v-syntax-highlight="'bash'"><code v-pre>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

    <h2 id="render">render</h2>
    <p>Renders a <code>.vue</code> component as an HTML fragment (no <code>&lt;!DOCTYPE&gt;</code>). Scoped styles are prepended as a <code>&lt;style&gt;</code> block.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc render [-strict] [-dir &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-props &lt;json&gt;] &lt;ComponentName&gt;</code></pre>

    <h3>Flags</h3>
    <div class="flag-row"><span class="flag-name">-dir string</span><span class="flag-desc">Directory containing <code>.vue</code> components. Default: <code>.</code></span></div>
    <div class="flag-row"><span class="flag-name">-props string</span><span class="flag-desc">JSON object of props to pass to the component.</span></div>
    <div class="flag-row"><span class="flag-name">-layout string</span><span class="flag-desc">Wrap the fragment in a layout component.</span></div>
    <div class="flag-row"><span class="flag-name">-debug</span><span class="flag-desc">Annotate output with HTML comments showing render trace.</span></div>
    <div class="flag-row"><span class="flag-name">-strict</span><span class="flag-desc">Abort on missing props.</span></div>

    <h3>Examples</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre># Render a greeting fragment
htmlc render -dir ./templates Greeting -props '{"name":"world"}'

# Render with layout
htmlc render -dir ./templates Article -layout AppLayout -props '{"title":"Hello"}'

# Pipe props from stdin
echo '{"name":"world"}' | htmlc render -dir ./templates Greeting</code></pre>

    <h2 id="page">page</h2>
    <p>Like <code>render</code>, but outputs a full HTML page (adds <code>&lt;!DOCTYPE html&gt;</code> and injects scoped styles into <code>&lt;head&gt;</code>).</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc page [-strict] [-dir &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-props &lt;json&gt;] &lt;ComponentName&gt;</code></pre>

    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc page -dir ./templates HomePage -props '{"title":"My site"}'
&lt;!DOCTYPE html&gt;
&lt;html&gt;
  &lt;head&gt;&lt;title&gt;My site&lt;/title&gt;&lt;/head&gt;
  &lt;body&gt;&lt;h1&gt;My site&lt;/h1&gt;&lt;/body&gt;
&lt;/html&gt;</code></pre>

    <h2 id="build">build</h2>
    <p>Walks the pages directory recursively, renders every <code>.vue</code> file as a full HTML page, and writes results to the output directory. The directory hierarchy is preserved. Supports <a href="#external-directives">external directives</a> for custom element transformations.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc build [-strict] [-dir &lt;path&gt;] [-pages &lt;path&gt;] [-out &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-dev &lt;addr&gt;]</code></pre>

    <h3>Flags</h3>
    <div class="flag-row"><span class="flag-name">-dir string</span><span class="flag-desc">Directory containing shared <code>.vue</code> components. Default: <code>.</code></span></div>
    <div class="flag-row"><span class="flag-name">-pages string</span><span class="flag-desc">Root of the page tree. Default: <code>./pages</code></span></div>
    <div class="flag-row"><span class="flag-name">-out string</span><span class="flag-desc">Output directory. Created if missing. Default: <code>./out</code></span></div>
    <div class="flag-row"><span class="flag-name">-layout string</span><span class="flag-desc">Layout component (from <code>-dir</code>) to wrap every page.</span></div>
    <div class="flag-row"><span class="flag-name">-dev string</span><span class="flag-desc">Start a dev server at <code>addr</code> with live rebuild (e.g. <code>:8080</code>).</span></div>
    <div class="flag-row"><span class="flag-name">-strict</span><span class="flag-desc">Abort on missing props; validate all components before rendering.</span></div>
    <div class="flag-row"><span class="flag-name">-debug</span><span class="flag-desc">Annotate output with diagnostic HTML comments.</span></div>

    <h3 id="data-files">Data files</h3>
    <p>Props for each page are loaded by merging JSON data files in order (later wins):</p>
    <ol>
      <li><code v-pre>pages/_data.json</code> — root defaults (all pages)</li>
      <li><code v-pre>pages/subdir/_data.json</code> — subdirectory defaults</li>
      <li><code v-pre>pages/subdir/hello.json</code> — page-level props (highest priority)</li>
    </ol>

    <h3 id="page-partials">Shared page partials</h3>
    <p>Any <code>.vue</code> file whose base name starts with <code>_</code> is treated as a <strong>shared partial</strong> and is skipped during page discovery. Partials are not rendered as standalone HTML pages; they exist solely to be referenced as child components by other pages.</p>

    <pre v-syntax-highlight="'text'"><code v-pre>pages/
  _Header.vue          ← skipped (partial, not a page)
  _Footer.vue          ← skipped (partial, not a page)
  index.vue            → dist/index.html
  about.vue            → dist/about.html
  blog/
    _PostCard.vue      ← skipped (partial)
    index.vue          → dist/blog/index.html
    hello-world.vue    → dist/blog/hello-world.html</code></pre>

    <p>The partial is still registered as a component and can be used inside other pages:</p>

    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- pages/index.vue --&gt;
&lt;template&gt;
  &lt;_Header :title="siteTitle" /&gt;
  &lt;main&gt;…&lt;/main&gt;
  &lt;_Footer /&gt;
&lt;/template&gt;</code></pre>

    <p>The leading <code>_</code> is part of the component name when referenced in templates. Use it to signal to readers that the file is a layout aid rather than a user-facing page.</p>

    <h3 id="static-site">Examples</h3>
    <pre v-syntax-highlight="'bash'"><code v-pre># Build with defaults (components in ., pages in ./pages, output to ./out)
htmlc build

# Explicit paths
htmlc build -dir ./templates -pages ./pages -out ./dist

# With a shared layout
htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout

# Development server with live rebuild
htmlc build -dir ./templates -pages ./pages -out ./dist -dev :8080</code></pre>

    <h2 id="layouts">Layouts</h2>
    <p>Two patterns for layouts:</p>
    <p><strong>Pattern 1 — Component-embedded layout:</strong> The page component references the layout directly using slots. No CLI flag needed.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/PostPage.vue --&gt;
&lt;template&gt;
  &lt;AppLayout :title="title"&gt;
    &lt;article&gt;&#123;&#123;<!---><!----> body }}&lt;/article&gt;
  &lt;/AppLayout&gt;
&lt;/template&gt;</code></pre>

    <p><strong>Pattern 2 — <code>-layout</code> flag:</strong> The page renders as a fragment; htmlc passes the HTML as <code>content</code> prop to the layout. The page needs no knowledge of the layout.</p>
    <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/AppLayout.vue --&gt;
&lt;template&gt;
  &lt;html&gt;
    &lt;body&gt;
      &lt;main v-html="content"&gt;&lt;/main&gt;
    &lt;/body&gt;
  &lt;/html&gt;
&lt;/template&gt;</code></pre>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout</code></pre>

    <h2 id="props">props</h2>
    <p>Lists the props referenced by a component — useful for discovering what data a component expects.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc props [-dir &lt;path&gt;] &lt;ComponentName&gt;</code></pre>
    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc props -dir ./templates Card
title
body
author</code></pre>

    <p>Export as shell variables:</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>$ htmlc props -dir ./templates Card -export
export title=""
export body=""
export author=""</code></pre>

    <h2 id="ast">ast</h2>
    <p>Prints the parsed template as a JSON AST. Useful for debugging parsing problems or understanding how htmlc sees a template.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc ast [-dir &lt;path&gt;] &lt;ComponentName&gt;</code></pre>

    <h2 id="help">help</h2>
    <pre v-syntax-highlight="'bash'"><code v-pre>htmlc help [&lt;subcommand&gt;]</code></pre>
    <pre v-syntax-highlight="'bash'"><code v-pre># Show general help
htmlc help

# Show help for a specific subcommand
htmlc help build</code></pre>

    <h2 id="external-directives">External directives</h2>
    <p>External directives extend <code>htmlc build</code> with custom element transformations. They are standalone executables that communicate with the build via newline-delimited JSON (NDJSON) over stdin/stdout.</p>

    <h3 id="directive-discovery">Discovery</h3>
    <p>During <code>build</code>, <code>htmlc</code> walks the component directory (<code>-dir</code>) and registers every file that satisfies all three conditions:</p>
    <table>
      <thead><tr><th>Condition</th><th>Rule</th></tr></thead>
      <tbody>
        <tr><td>Name</td><td>Base name without extension matches <code>v-&lt;directive-name&gt;</code></td></tr>
        <tr><td>Directive name format</td><td>Lower-kebab-case: <code>[a-z][a-z0-9-]*</code></td></tr>
        <tr><td>Executable</td><td>File mode has at least one executable bit set (<code>mode &amp; 0111 != 0</code>)</td></tr>
      </tbody>
    </table>
    <p>Hidden directories (names starting with <code>.</code>) are skipped. Extensions are ignored, so <code>v-foo</code>, <code>v-foo.sh</code>, and <code>v-foo.py</code> all register as directive <code>foo</code>.</p>
    <pre v-syntax-highlight="'bash'"><code v-pre>v-syntax-highlight      → directive name: syntax-highlight
v-upper.sh              → directive name: upper
v-toc-builder.py        → directive name: toc-builder</code></pre>

    <p>Each directive is started once at the beginning of the build and stopped when the build finishes. A non-zero exit code is treated as a warning; the build continues.</p>

    <h3 id="directive-protocol">Protocol</h3>
    <p>Communication is NDJSON: one JSON object per line, no pretty-printing. Requests flow from <code>htmlc</code> to the directive on stdin; responses flow back on stdout. Requests are sent sequentially. The directive's stderr is forwarded verbatim to <code>htmlc</code>'s stderr.</p>
    <p><strong>Request envelope</strong> (sent for every element carrying the directive's attribute). Both <code>text</code> and <code>inner_html</code> are populated from the element's fully pre-rendered children — all template expressions are already evaluated before the directive hooks run.</p>
    <pre v-syntax-highlight="'json'"><code v-pre>{
  "hook":       "created" | "mounted",
  "id":         "&lt;opaque string&gt;",
  "tag":        "&lt;element tag name&gt;",
  "attrs":      { "&lt;name&gt;": "&lt;value&gt;", ... },
  "text":       "&lt;plain text extracted from rendered children&gt;",
  "inner_html": "&lt;fully rendered inner HTML of the element's children&gt;",
  "binding": {
    "value":     "&lt;evaluated expression&gt;",
    "raw_expr":  "&lt;unevaluated expression string&gt;",
    "arg":       "&lt;directive argument, or empty string&gt;",
    "modifiers": { "&lt;modifier&gt;": true, ... }
  }
}</code></pre>

    <h3 id="directive-created">created hook</h3>
    <p>Called <strong>before</strong> the element is rendered. The response may mutate the element's tag, attributes, or inner content.</p>
    <pre v-syntax-highlight="'json'"><code v-pre>{
  "id":         "&lt;same id as request&gt;",
  "tag":        "&lt;optional: replacement tag name&gt;",
  "attrs":      { "&lt;name&gt;": "&lt;value&gt;", ... },
  "inner_html": "&lt;optional: verbatim HTML to use as element content&gt;",
  "error":      "&lt;optional: non-empty string aborts rendering of this element&gt;"
}</code></pre>
    <table>
      <thead><tr><th>Field</th><th>Effect</th></tr></thead>
      <tbody>
        <tr><td><code v-pre>id</code></td><td>Required. Must match the request <code>id</code>.</td></tr>
        <tr><td><code v-pre>tag</code></td><td>If non-empty, replaces the element's tag name.</td></tr>
        <tr><td><code v-pre>attrs</code></td><td>If present, replaces all element attributes.</td></tr>
        <tr><td><code v-pre>inner_html</code></td><td>If non-empty, replaces the element's children with this HTML verbatim. Template children are discarded.</td></tr>
        <tr><td><code v-pre>error</code></td><td>If non-empty, aborts rendering of this element.</td></tr>
      </tbody>
    </table>

    <h3 id="directive-mounted">mounted hook</h3>
    <p>Called <strong>after</strong> the element's closing tag has been written.</p>
    <pre v-syntax-highlight="'json'"><code v-pre>{
  "id":    "&lt;same id as request&gt;",
  "html":  "&lt;optional: HTML injected immediately after the closing tag&gt;",
  "error": "&lt;optional: non-empty string aborts rendering&gt;"
}</code></pre>
    <table>
      <thead><tr><th>Field</th><th>Effect</th></tr></thead>
      <tbody>
        <tr><td><code v-pre>id</code></td><td>Required. Must match the request <code>id</code>.</td></tr>
        <tr><td><code v-pre>html</code></td><td>If non-empty, written verbatim after the element's closing tag.</td></tr>
        <tr><td><code v-pre>error</code></td><td>If non-empty, aborts rendering and logs the message.</td></tr>
      </tbody>
    </table>
  </DocsPage>
</template>

<style>
  .flag-row { display: flex; gap: 0.75rem; align-items: baseline; margin: 0.5rem 0; }
  .flag-name { font-family: "SF Mono","Fira Code",monospace; font-size: 0.85rem; color: var(--accent); white-space: nowrap; }
  .flag-desc { font-size: 0.875rem; color: var(--muted); }
</style>
