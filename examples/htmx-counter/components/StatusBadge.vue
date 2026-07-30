<!--
  StatusBadge — small status readout, used purely to demonstrate the
  out-of-band swap + cross-component style-dedup story (RFC 014 §6
  Example 4).

  Deliberately does NOT carry the id "status-badge" on its own root — that
  id lives on the wrapper <div id="status-badge" hx-swap-oob="true"> that
  main.go writes around this component's rendered output. RFC 014 §1.2
  documents exactly this pitfall: giving a component's own root element the
  same id an OOB wrapper also carries produces two DOM elements sharing one
  id after the swap. Keeping the id only on the wrapper avoids that.
-->
<template>
  <div class="status-badge">
    <span class="badge-text">Last updated: {{ count }} increments</span>
  </div>
</template>
<style scoped>
.status-badge {
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: #555;
}
.badge-text {
  font-style: italic;
}
</style>
