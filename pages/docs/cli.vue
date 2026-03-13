<template>
  <Layout pageTitle="CLI Reference — htmlc.sh" description="htmlc CLI reference: render, page, build, props, ast subcommands." :siteTitle="siteTitle">

    <div class="docs-layout">
      <aside class="docs-sidebar">
        <div class="sidebar-section">
          <div class="sidebar-label">Overview</div>
          <a href="#synopsis" class="sidebar-link">Synopsis</a>
          <a href="#installation" class="sidebar-link">Installation</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Subcommands</div>
          <a href="#render" class="sidebar-link">render</a>
          <a href="#page" class="sidebar-link">page</a>
          <a href="#build" class="sidebar-link">build</a>
          <a href="#props" class="sidebar-link">props</a>
          <a href="#ast" class="sidebar-link">ast</a>
          <a href="#help" class="sidebar-link">help</a>
        </div>
        <div class="sidebar-section">
          <div class="sidebar-label">Guides</div>
          <a href="#static-site" class="sidebar-link">Static sites</a>
          <a href="#layouts" class="sidebar-link">Layouts</a>
          <a href="#data-files" class="sidebar-link">Data files</a>
        </div>
      </aside>

      <div class="docs-content">
        <h1 id="synopsis">CLI Reference</h1>
        <p class="lead"><code>htmlc</code> renders Vue Single File Components (<code>.vue</code>) to HTML entirely in Go — no Node.js, no browser, no JavaScript runtime.</p>

        <h2 id="installation">Installation</h2>
        <pre><code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>

        <h2 id="render">render</h2>
        <p>Renders a <code>.vue</code> component as an HTML fragment (no <code>&lt;!DOCTYPE&gt;</code>). Scoped styles are prepended as a <code>&lt;style&gt;</code> block.</p>
        <pre><code>htmlc render [-strict] [-dir &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-props &lt;json&gt;] &lt;ComponentName&gt;</code></pre>

        <h3>Flags</h3>
        <div class="flag-row"><span class="flag-name">-dir string</span><span class="flag-desc">Directory containing <code>.vue</code> components. Default: <code>.</code></span></div>
        <div class="flag-row"><span class="flag-name">-props string</span><span class="flag-desc">JSON object of props to pass to the component.</span></div>
        <div class="flag-row"><span class="flag-name">-layout string</span><span class="flag-desc">Wrap the fragment in a layout component.</span></div>
        <div class="flag-row"><span class="flag-name">-debug</span><span class="flag-desc">Annotate output with HTML comments showing render trace.</span></div>
        <div class="flag-row"><span class="flag-name">-strict</span><span class="flag-desc">Abort on missing props.</span></div>

        <h3>Examples</h3>
        <pre><code># Render a greeting fragment
htmlc render -dir ./templates Greeting -props '{"name":"world"}'

# Render with layout
htmlc render -dir ./templates Article -layout AppLayout -props '{"title":"Hello"}'

# Pipe props from stdin
echo '{"name":"world"}' | htmlc render -dir ./templates Greeting</code></pre>

        <h2 id="page">page</h2>
        <p>Like <code>render</code>, but outputs a full HTML page (adds <code>&lt;!DOCTYPE html&gt;</code> and injects scoped styles into <code>&lt;head&gt;</code>).</p>
        <pre><code>htmlc page [-strict] [-dir &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-props &lt;json&gt;] &lt;ComponentName&gt;</code></pre>

        <pre><code>$ htmlc page -dir ./templates HomePage -props '{"title":"My site"}'
&lt;!DOCTYPE html&gt;
&lt;html&gt;
  &lt;head&gt;&lt;title&gt;My site&lt;/title&gt;&lt;/head&gt;
  &lt;body&gt;&lt;h1&gt;My site&lt;/h1&gt;&lt;/body&gt;
&lt;/html&gt;</code></pre>

        <h2 id="build">build</h2>
        <p>Walks the pages directory recursively, renders every <code>.vue</code> file as a full HTML page, and writes results to the output directory. The directory hierarchy is preserved.</p>
        <pre><code>htmlc build [-strict] [-dir &lt;path&gt;] [-pages &lt;path&gt;] [-out &lt;path&gt;] [-layout &lt;name&gt;] [-debug] [-dev &lt;addr&gt;]</code></pre>

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
          <li><code>pages/_data.json</code> — root defaults (all pages)</li>
          <li><code>pages/subdir/_data.json</code> — subdirectory defaults</li>
          <li><code>pages/subdir/hello.json</code> — page-level props (highest priority)</li>
        </ol>

        <h3 id="static-site">Examples</h3>
        <pre><code># Build with defaults (components in ., pages in ./pages, output to ./out)
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
        <pre><code>&lt;!-- templates/PostPage.vue --&gt;
&lt;template&gt;
  &lt;AppLayout :title="title"&gt;
    &lt;article&gt;{{ "{{" }} body }}&lt;/article&gt;
  &lt;/AppLayout&gt;
&lt;/template&gt;</code></pre>

        <p><strong>Pattern 2 — <code>-layout</code> flag:</strong> The page renders as a fragment; htmlc passes the HTML as <code>content</code> prop to the layout. The page needs no knowledge of the layout.</p>
        <pre><code>&lt;!-- templates/AppLayout.vue --&gt;
&lt;template&gt;
  &lt;html&gt;
    &lt;body&gt;
      &lt;main v-html="content"&gt;&lt;/main&gt;
    &lt;/body&gt;
  &lt;/html&gt;
&lt;/template&gt;</code></pre>
        <pre><code>htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout</code></pre>

        <h2 id="props">props</h2>
        <p>Lists the props referenced by a component — useful for discovering what data a component expects.</p>
        <pre><code>htmlc props [-dir &lt;path&gt;] &lt;ComponentName&gt;</code></pre>
        <pre><code>$ htmlc props -dir ./templates Card
title
body
author</code></pre>

        <p>Export as shell variables:</p>
        <pre><code>$ htmlc props -dir ./templates Card -export
export title=""
export body=""
export author=""</code></pre>

        <h2 id="ast">ast</h2>
        <p>Prints the parsed template as a JSON AST. Useful for debugging parsing problems or understanding how htmlc sees a template.</p>
        <pre><code>htmlc ast [-dir &lt;path&gt;] &lt;ComponentName&gt;</code></pre>

        <h2 id="help">help</h2>
        <pre><code>htmlc help [&lt;subcommand&gt;]</code></pre>
        <pre><code># Show general help
htmlc help

# Show help for a specific subcommand
htmlc help build</code></pre>
      </div>
    </div>

  </Layout>
</template>

<style>
  p { margin: 1rem 0; }
  ul, ol { padding-left: 1.5rem; margin: 1rem 0; }
  li { margin: 0.25rem 0; }

  .docs-layout { display: grid; grid-template-columns: 220px 1fr; gap: 0; max-width: 1200px; margin: 0 auto; }
  @media (max-width: 800px) { .docs-layout { grid-template-columns: 1fr; } .docs-sidebar { display: none; } }
  .docs-sidebar { border-right: 1px solid var(--border); padding: 2rem 1.5rem; position: sticky; top: var(--nav-height); height: calc(100vh - var(--nav-height)); overflow-y: auto; }
  .sidebar-section { margin-bottom: 1.5rem; }
  .sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--text); margin-bottom: 0.5rem; padding: 0 0.5rem; }
  .sidebar-link { display: block; padding: 0.3rem 0.5rem; font-size: 0.875rem; color: var(--muted); border-radius: 4px; }
  .sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.05); text-decoration: none; }

  .docs-content { padding: 3rem 3rem 5rem; max-width: 800px; }
  .docs-content h1 { font-size: 2.2rem; margin-bottom: 0.75rem; color: #f0f2ff; }
  .docs-content h2 { font-size: 1.4rem; margin: 2.5rem 0 0.75rem; padding-top: 2.5rem; border-top: 1px solid var(--border); }
  .docs-content h2:first-of-type { border-top: none; padding-top: 0; }
  .docs-content h3 { font-size: 1.05rem; margin: 1.5rem 0 0.5rem; color: var(--accent); font-family: "SF Mono","Fira Code",monospace; }
  .lead { font-size: 1.1rem; color: var(--muted); margin-bottom: 2rem; }
  .flag-row { display: flex; gap: 0.75rem; align-items: baseline; margin: 0.5rem 0; }
  .flag-name { font-family: "SF Mono","Fira Code",monospace; font-size: 0.85rem; color: var(--accent); white-space: nowrap; }
  .flag-desc { font-size: 0.875rem; color: var(--muted); }
</style>
