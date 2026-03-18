<template>
  <div class="card">
    <h2 class="card-title">{{ title }}</h2>
    <p class="card-body">{{ description }}</p>
  </div>
</template>
<style scoped>
.card {
  border: 1px solid #ccc;
  border-radius: 6px;
  padding: 1rem;
  margin: 0.5rem 0;
  background: #fff;
}
.card-title { margin: 0 0 0.5rem; font-size: 1.1rem; }
.card-body  { margin: 0; color: #555; }
</style>
