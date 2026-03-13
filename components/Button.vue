<template>
  <component
    :is="href ? 'a' : 'button'"
    :href="href"
    :class="['btn', 'btn-' + variant]"
  ><slot /></component>
</template>

<script>
export default {
  props: {
    href:    { type: String,  default: null },
    variant: { type: String,  default: 'primary' },
  },
}
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  padding: 0.65rem 1.5rem;
  border-radius: 8px;
  font-weight: 600;
  font-size: 0.9rem;
  transition: all 0.15s;
  text-decoration: none;
  cursor: pointer;
  border: none;
}

.btn-primary {
  background: var(--accent);
  color: #fff;
}
.btn-primary:hover { background: var(--accent-hover); text-decoration: none; }

.btn-secondary {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.12);
  color: var(--text);
}
.btn-secondary:hover { background: rgba(255,255,255,0.1); text-decoration: none; }
</style>
