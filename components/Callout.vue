<template>
  <div v-if="type === 'info'" class="callout callout-info">
    <slot />
  </div>
  <div v-else class="callout">
    <slot />
  </div>
</template>

<style>
.callout {
  background: rgba(124,106,247,0.08);
  border: 1px solid rgba(124,106,247,0.25);
  border-radius: 8px;
  padding: 1rem 1.25rem;
  margin: 1.5rem 0;
}

.callout p {
  margin: 0;
  font-size: 0.9rem;
  color: #c9ccf5;
}

.callout pre {
  margin: 0.5rem 0 0;
}

.callout-info {
  background: rgba(0, 173, 216, 0.08);
  border: none;
  border-left: 3px solid #00ADD8;
  border-radius: 0 6px 6px 0;
  color: #c4c8e2;
  font-size: 0.95rem;
}
</style>
