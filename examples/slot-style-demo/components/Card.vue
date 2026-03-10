<template>
  <div class="card">
    <div class="card-body">
      <slot></slot>
    </div>
  </div>
</template>
<style scoped>
.card { border: 1px solid #ddd; border-radius: 4px; margin: 1rem 0; }
.card-body { padding: 1rem; }
</style>
