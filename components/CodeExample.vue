<template>
  <div class="code-example">
    <div class="code-header" v-if="filename">
      <span class="code-filename">{{ filename }}</span>
    </div>
    <pre><code v-html="code"></code></pre>
  </div>
</template>

<style scoped>
.code-example {
  border: 1px solid var(--border);
  border-radius: 8px;
  margin: 1.5rem 0;
  max-width: 100%;
}

.code-header {
  background: #1a1d27;
  border-bottom: 1px solid var(--border);
  padding: 0.5rem 1rem;
  border-radius: 8px 8px 0 0;
}

.code-filename {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.75rem;
  color: #8b8fa8;
}

.code-example pre {
  border: none;
  border-radius: 0 0 8px 8px;
  margin: 0;
  overflow-x: auto;
}
</style>
