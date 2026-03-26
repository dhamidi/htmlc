<template>
  <canvas :width="width" :height="height" :data-src="src"></canvas>
</template>

<style scoped>
canvas { border: 2px solid #ccc; border-radius: 4px; background: #fafafa; display: block; }
</style>

<script customelement>
class WidgetsShapeCanvas extends HTMLElement {
  #source = null
  #ctx = null

  connectedCallback() {
    const canvas = this.querySelector('canvas')
    this.#ctx = canvas.getContext('2d')
    this.#source = new EventSource(canvas.dataset.src)
    this.#source.onmessage = ({ data }) => this.#draw(JSON.parse(data))
  }

  disconnectedCallback() { this.#source?.close() }

  #draw({ type, color = '#000', x, y, w, h, r }) {
    const ctx = this.#ctx
    ctx.fillStyle = color
    if (type === 'rect')   { ctx.fillRect(x, y, w, h) }
    if (type === 'circle') { ctx.beginPath(); ctx.arc(x, y, r, 0, 2*Math.PI); ctx.fill() }
    if (type === 'clear')  { ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height) }
  }
}
customElements.define('widgets-shape-canvas', WidgetsShapeCanvas)
</script>
