<template>
  <div class="feature-card">
    <div class="feature-icon">{{ icon }}</div>
    <h3 class="feature-title">{{ title }}</h3>
    <p class="feature-desc">{{ description }}</p>
  </div>
</template>

<style scoped>
.feature-card {
  background: #1a1d27;
  border: 1px solid #2a2d3e;
  border-radius: 12px;
  padding: 1.5rem;
}

.feature-icon {
  font-size: 1.8rem;
  margin-bottom: 0.75rem;
}

.feature-title {
  font-size: 1rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  color: #e2e4f0;
}

.feature-desc {
  font-size: 0.875rem;
  color: #8b8fa8;
  line-height: 1.6;
}
</style>
