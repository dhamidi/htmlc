<template>
  <nav class="sidebar">
    <div class="sidebar-section">
      <div class="sidebar-label">Getting Started</div>
      <a href="/docs/index.html" class="sidebar-link">Overview</a>
      <a href="/docs/index.html#installation" class="sidebar-link">Installation</a>
      <a href="/docs/index.html#quick-start" class="sidebar-link">Quick start</a>
      <a href="/docs/tutorial.html" class="sidebar-link">Tutorial</a>
    </div>
    <div class="sidebar-section">
      <div class="sidebar-label">Reference</div>
      <a href="/docs/directives.html" class="sidebar-link">Directives</a>
      <a href="/docs/components.html" class="sidebar-link">Component system</a>
      <a href="/docs/cli.html" class="sidebar-link">CLI</a>
      <a href="/docs/go-api.html" class="sidebar-link">Go API</a>
    </div>
    <div class="sidebar-section">
      <div class="sidebar-label">Guides</div>
      <a href="/docs/concepts.html" class="sidebar-link">Concepts</a>
      <a href="/docs/howto.html" class="sidebar-link">How-to guides</a>
    </div>
    <div class="sidebar-section">
      <div class="sidebar-label">Project</div>
      <a href="https://github.com/dhamidi/htmlc" class="sidebar-link" target="_blank" rel="noopener">GitHub</a>
    </div>
  </nav>
</template>

<style scoped>
.sidebar {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding-top: 2rem;
}

.sidebar-section {
  margin-bottom: 1.5rem;
}

.sidebar-label {
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: #8b8fa8;
  margin-bottom: 0.5rem;
  padding: 0 0.75rem;
}

.sidebar-link {
  display: block;
  padding: 0.35rem 0.75rem;
  font-size: 0.875rem;
  color: #8b8fa8;
  border-radius: 6px;
  transition: color 0.15s, background 0.15s;
  text-decoration: none;
}

.sidebar-link:hover {
  color: #e2e4f0;
  background: rgba(255,255,255,0.06);
  text-decoration: none;
}
</style>
