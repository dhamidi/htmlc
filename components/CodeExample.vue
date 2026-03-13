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
  border: 1px solid #2a2d3e;
  border-radius: 8px;
  overflow: hidden;
  margin: 1.5rem 0;
}

.code-header {
  background: #1a1d27;
  border-bottom: 1px solid #2a2d3e;
  padding: 0.5rem 1rem;
}

.code-filename {
  font-family: "SF Mono", "Fira Code", monospace;
  font-size: 0.75rem;
  color: #8b8fa8;
}

.code-example pre {
  border: none;
  border-radius: 0;
  margin: 0;
}
</style>
