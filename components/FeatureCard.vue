<template>
  <div class="feature-card">
    <div class="feature-icon"><slot name="icon" /></div>
    <h3 class="feature-title">{{ title }}</h3>
    <div class="feature-desc"><slot /></div>
  </div>
</template>

<style scoped>
.feature-card {
  background: var(--bg2);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1.5rem;
}

.feature-icon {
  margin-bottom: 0.75rem;
  color: var(--accent);
}

.feature-icon :deep(.icon) {
  width: 1.5rem;
  height: 1.5rem;
}

.feature-title {
  font-size: 1rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
}

.feature-desc {
  font-size: 0.875rem;
  color: var(--muted);
  line-height: 1.6;
}
</style>
