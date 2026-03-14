<template>
  <Layout pageTitle="htmlc" :siteTitle="siteTitle" :description="description" :fullWidth="false">

    <!-- Hero -->
    <Hero />

    <!-- Features -->
    <div class="features">
      <FeatureCard title="Zero JavaScript runtime">
        <template #icon><IconZap /></template>
        Templates evaluate once per request and produce plain HTML.
        No hydration, no virtual DOM, no client bundles.
      </FeatureCard>

      <FeatureCard title="Vue SFC syntax">
        <template #icon><IconFileCode /></template>
        Author components using the same <code>.vue</code> format you already know —
        <code>v-if</code>, <code>v-for</code>, <code>v-bind</code>, slots, scoped styles.
      </FeatureCard>

      <FeatureCard title="CLI &amp; Go API">
        <template #icon><IconTerminal /></template>
        Use the <code>htmlc</code> CLI for static sites or import the Go package to
        render components inside any HTTP handler.
      </FeatureCard>

      <FeatureCard title="Scoped styles">
        <template #icon><IconPalette /></template>
        <code v-pre>&lt;style scoped&gt;</code> rewrites selectors and injects scope
        attributes automatically — styles never leak between components.
      </FeatureCard>

      <FeatureCard title="Static site generation">
        <template #icon><IconGlobe /></template>
        <code v-pre>htmlc build</code> walks a pages directory and renders every
        <code>.vue</code> file to a matching <code>.html</code> file.
        Props come from sibling JSON files.
      </FeatureCard>

      <FeatureCard title="Debug mode">
        <template #icon><IconBug /></template>
        Pass <code>-debug</code> and the output is annotated with HTML comments
        showing which component rendered each subtree.
      </FeatureCard>
    </div>

    <!-- Quick start (Go API) -->
    <section class="section">
      <div class="section-label">Quick start</div>
      <h2 class="section-title">Embed in any Go application</h2>
      <p class="section-desc">Import the package, create an engine, and render components directly from your HTTP handlers.</p>

      <div class="qs-steps">
        <QuickStartStep label="1. Add the dependency">
          <pre v-syntax-highlight="'bash'"><code v-pre>go get github.com/dhamidi/htmlc</code></pre>
        </QuickStartStep>

        <QuickStartStep label="2. Write a component">
          <pre v-syntax-highlight="'html'"><code v-pre>&lt;!-- templates/Greeting.vue --&gt;
&lt;template&gt;
  &lt;p&gt;Hello, &#123;&#123;<!---><!----> name &#125;&#125;!&lt;/p&gt;
&lt;/template&gt;</code></pre>
        </QuickStartStep>

        <QuickStartStep label="3. Create an engine &amp; render">
          <pre v-syntax-highlight="'go'"><code v-pre>engine, err := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
})

html, err := engine.RenderFragmentString(
    "Greeting",
    map[string]any{"name": "world"},
)
// html == "&lt;p&gt;Hello, world!&lt;/p&gt;"</code></pre>
        </QuickStartStep>

        <QuickStartStep label="4. Serve over HTTP">
          <pre v-syntax-highlight="'go'"><code v-pre>http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    engine.RenderPage(w, "Page", map[string]any{
        "title": "Home",
    })
})</code></pre>
        </QuickStartStep>
      </div>
    </section>

  </Layout>
</template>

<style>
  .features { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1.25rem; margin: 4rem 0; }

  .section { margin: 5rem 0; }
  .section-label { font-size: 0.75rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--accent); margin-bottom: 0.75rem; }
  .section-title { font-size: 1.8rem; font-weight: 800; margin-bottom: 1rem; letter-spacing: -0.03em; }
  .section-desc { color: var(--muted); max-width: 560px; margin-bottom: 2rem; }

  .qs-steps { display: flex; flex-direction: column; gap: 1.25rem; }
</style>
