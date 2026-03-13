<template>
  <Layout
    pageTitle="htmlc — Server-side Vue component rendering for Go"
    :siteTitle="siteTitle"
    :description="description"
  >

    <!-- Hero -->
    <section class="hero">
      <div class="hero-badge">Go &bull; Zero JS &bull; Vue SFC syntax</div>
      <h1 class="hero-title">Server-side component<br />rendering for Go</h1>
      <p class="hero-sub">
        Write reusable HTML components in <code>.vue</code> files.
        Render them server-side in Go — no Node.js, no JavaScript runtime, no virtual DOM.
      </p>
      <div class="hero-actions">
        <Button href="/docs/tutorial.html" variant="primary">Get started</Button>
        <Button href="/docs/index.html" variant="secondary">Documentation</Button>
      </div>
      <div class="hero-install">
        <code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code>
      </div>
    </section>

    <!-- Features -->
    <div class="features">
      <div class="feature-card">
        <div class="feature-icon">&#127381;</div>
        <div class="feature-title">Zero JavaScript runtime</div>
        <p class="feature-desc">Templates evaluate once per request and produce plain HTML. No hydration, no virtual DOM, no client bundles.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128196;</div>
        <div class="feature-title">Vue SFC syntax</div>
        <p class="feature-desc">Author components using the same <code>.vue</code> format you already know — <code>v-if</code>, <code>v-for</code>, <code>v-bind</code>, slots, scoped styles.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128295;</div>
        <div class="feature-title">CLI &amp; Go API</div>
        <p class="feature-desc">Use the <code>htmlc</code> CLI for static sites or import the Go package to render components inside any HTTP handler.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#127912;</div>
        <div class="feature-title">Scoped styles</div>
        <p class="feature-desc"><code>&lt;style scoped&gt;</code> rewrites selectors and injects scope attributes automatically — styles never leak between components.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128218;</div>
        <div class="feature-title">Static site generation</div>
        <p class="feature-desc"><code>htmlc build</code> walks a pages directory and renders every <code>.vue</code> file to a matching <code>.html</code> file. Props come from sibling JSON files.</p>
      </div>
      <div class="feature-card">
        <div class="feature-icon">&#128270;</div>
        <div class="feature-title">Debug mode</div>
        <p class="feature-desc">Pass <code>-debug</code> and the output is annotated with HTML comments showing which component rendered each subtree.</p>
      </div>
    </div>

    <!-- Quick start -->
    <section class="section">
      <div class="section-label">Quick start</div>
      <h2 class="section-title">Up and running in minutes</h2>
      <p class="section-desc">Install the CLI, write a component, render it.</p>

      <div class="two-col">
        <div>
          <div class="col-label">1. Install</div>
          <pre><code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code></pre>
        </div>
        <div>
          <div class="col-label">2. Write a component</div>
          <pre><code>&lt;!-- templates/Greeting.vue --&gt;
&lt;template&gt;
  &lt;p&gt;Hello, {{ "{{" }} name }}!&lt;/p&gt;
&lt;/template&gt;</code></pre>
        </div>
        <div>
          <div class="col-label">3. Render it</div>
          <pre><code>$ htmlc render -dir ./templates Greeting \
    -props '{"name":"world"}'
&lt;p&gt;Hello, world!&lt;/p&gt;</code></pre>
        </div>
        <div>
          <div class="col-label">4. Build a whole site</div>
          <pre><code>$ htmlc build \
    -dir ./templates \
    -pages ./pages \
    -out ./dist
Build complete: 12 pages, 0 errors.</code></pre>
        </div>
      </div>
    </section>

  </Layout>
</template>

<style>
  .hero { padding: 6rem 0 4rem; text-align: center; max-width: 760px; margin: 0 auto; }
  .hero-badge { display: inline-block; background: var(--accent-alpha12); border: 1px solid var(--accent-alpha35); color: #7dd9ef; font-size: 0.75rem; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; padding: 0.3em 0.9em; border-radius: 100px; margin-bottom: 1.5rem; }
  .hero-title { font-size: clamp(2.2rem, 5vw, 3.6rem); font-weight: 800; line-height: 1.1; margin-bottom: 1.25rem; color: var(--text); letter-spacing: -0.04em; }
  .hero-sub { font-size: 1.1rem; color: var(--muted); margin-bottom: 2.5rem; max-width: 560px; margin-left: auto; margin-right: auto; }
  .hero-actions { display: flex; gap: 1rem; justify-content: center; margin-bottom: 2rem; flex-wrap: wrap; }
  .hero-install { display: inline-block; background: var(--code-bg); border: 1px solid var(--border); border-radius: 8px; padding: 0.6rem 1.2rem; font-size: 0.85rem; color: var(--accent); }
  .hero-install code { background: none; border: none; padding: 0; border-radius: 0; }

  .features { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1.25rem; margin: 4rem 0; }
  .feature-card { background: var(--bg2); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; }
  .feature-icon { font-size: 1.8rem; margin-bottom: 0.75rem; }
  .feature-title { font-size: 1rem; font-weight: 700; margin-bottom: 0.5rem; }
  .feature-desc { font-size: 0.875rem; color: var(--muted); line-height: 1.6; }

  .section { margin: 5rem 0; }
  .section-label { font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--accent); margin-bottom: 0.75rem; }
  .section-title { font-size: 1.8rem; font-weight: 800; margin-bottom: 1rem; letter-spacing: -0.03em; }
  .section-desc { color: var(--muted); max-width: 560px; margin-bottom: 2rem; }

  .two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
  @media (max-width: 700px) { .two-col { grid-template-columns: 1fr; } }

  .col-label { font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); margin-bottom: 0.5rem; }
  .col-title { font-size: 1.1rem; font-weight: 700; margin-bottom: 0.4rem; }
  .col-desc { font-size: 0.875rem; color: var(--muted); margin-bottom: 1rem; }
</style>
