<template>
  <div class="sidebar-section">
    <div class="sidebar-label">{{ label }}</div>
    <slot></slot>
  </div>
</template>

<script>
</script>

<style>
.sidebar-section { margin-bottom: 1.5rem; }
.sidebar-label { font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: rgb(240, 242, 255); margin-bottom: 0.5rem; padding: 0 0.5rem; }
.sidebar-link { display: block; padding: 0.3rem 0.5rem; font-size: 0.875rem; color: var(--muted); border-radius: 4px; }
.sidebar-link:hover { color: var(--text); background: rgba(255,255,255,0.05); text-decoration: none; }
</style>
