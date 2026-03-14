<template>
  <div class="qs-step">
    <div class="col-label">{{ label }}</div>
    <slot />
  </div>
</template>

<script>
export default {
  props: {
    label: { type: String, required: true },
  },
}
</script>

<style scoped>
.qs-step pre {
  margin-top: 0.4rem;
}

.col-label {
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: rgb(240, 242, 255);
  margin-bottom: 0.5rem;
}
</style>
