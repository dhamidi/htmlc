<template>
  <div class="layout">
    <header class="layout-header">
      <slot name="header">
        <h1 class="default-title">Default Title</h1>
      </slot>
    </header>
    <main class="layout-main">
      <slot></slot>
    </main>
  </div>
</template>
<style scoped>
.layout { display: flex; flex-direction: column; min-height: 100vh; }
.layout-header { background: #333; color: white; padding: 1rem; }
.layout-main { flex: 1; padding: 1rem; }
</style>
