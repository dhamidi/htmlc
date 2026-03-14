<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    :width="size"
    :height="size"
    viewBox="0 0 512 512"
    role="img"
    aria-label="htmlc logo"
    style="border-radius:6px;display:block;flex-shrink:0;"
  >
    <rect fill="#00ADD8" x="0" y="0" width="512" height="512" />
    <g
      transform="translate(468 456)"
      font-family="Inter, SF Pro Display, Segoe UI, Roboto, Helvetica, Arial, sans-serif"
      font-size="128"
      font-weight="800"
      text-anchor="end"
    >
      <text x="0" y="0">
        <tspan fill="#F8FAFC">html</tspan><tspan fill="#1F2937">c</tspan>
      </text>
    </g>
  </svg>
</template>

<script>
export default {
  props: {
    size: { default: 32 }
  }
}
</script>
