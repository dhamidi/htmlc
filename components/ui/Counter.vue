<template>
  <button class="counter-demo">Count: <span>{{ initial }}</span></button>
</template>

<script customelement>
class UiCounter extends HTMLElement {
  connectedCallback() {
    const span = this.querySelector('span')
    let n = parseInt(span.textContent, 10)
    this.addEventListener('click', () => { span.textContent = ++n })
  }
}
customElements.define('ui-counter', UiCounter)
</script>

<style scoped>
.counter-demo {
  padding: 0.5em 1.25em;
  font-size: 1rem;
  border: 2px solid #4f46e5;
  border-radius: 6px;
  background: #ede9fe;
  color: #3730a3;
  cursor: pointer;
  font-weight: 600;
}
.counter-demo:hover {
  background: #c7d2fe;
}
</style>
