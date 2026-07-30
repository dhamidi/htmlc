<!--
  Counter — the counter fragment driven entirely by server-pushed Datastar
  SSE patches (RFC 014 §6 Example 6).

  Unlike examples/htmx-counter's Counter.vue, this component carries no
  hx-* (or any client-initiated-request) attributes at all: nothing on this
  page ever issues a request to update the count. GET /counter-stream opens
  one long-lived text/event-stream response and pushes a bounded sequence of
  "datastar-patch-elements" events at #ds-counter (HomePage.vue's target
  div) from the server side; the client's only job is to have the Datastar
  runtime attached (via data-on-load, see HomePage.vue) so it applies those
  patches as they arrive.

  Rendered two different ways by this demo, both proving the same component
  works correctly in either position:

    1. As a child of HomePage on the initial full-page GET / (embedded via
       <Counter :count="count">, receiving "count" as a bound prop) — the
       count starts at 0 before the stream has sent anything.
    2. As the root fragment of each GET /counter-stream tick, rendered
       directly via hypermedia/datastar's PatchElementsFragment, which in
       turn calls engine.RenderFragmentSession(ctx, w, "Counter", data,
       sess) — "count" is then read straight off the top-level data map
       passed to that call, exactly like examples/htmx-counter's Counter.
-->
<template>
  <div id="ds-counter" class="counter">
    <p class="count">Count: {{ count }}</p>
  </div>
</template>
<style scoped>
.counter {
  border: 1px solid #ccc;
  border-radius: 6px;
  padding: 1rem;
  display: inline-block;
}
.count {
  font-size: 1.5rem;
  margin: 0;
}
</style>
