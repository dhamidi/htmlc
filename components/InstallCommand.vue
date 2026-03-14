<template>
  <div class="install-command">
    <code>go install github.com/dhamidi/htmlc/cmd/htmlc@latest</code>
  </div>
</template>

<style scoped>
.install-command {
  display: inline-block;
  background: var(--code-bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.6rem 1.2rem;
  font-size: 0.85rem;
}

/* Reset the global :not(pre) > code border so there is only one border */
.install-command code {
  background: none;
  border: none;
  border-radius: 0;
  padding: 0;
  font-size: inherit;
  color: var(--accent);
}
</style>
