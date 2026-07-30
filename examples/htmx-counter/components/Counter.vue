<!--
  Counter — the incrementing counter fragment.

  Rendered two different ways by this demo, both proving the same
  component works correctly in either position (RFC 014 §6 Example 4):

    1. As a child of HomePage on the initial full-page GET / (embedded via
       <Counter :count="count">, receiving "count" as a bound prop).
    2. As the root fragment of POST /increment's htmx response, rendered
       directly via engine.RenderFragmentSession(ctx, w, "Counter", data,
       sess) — "count" is then read straight off the top-level data map
       passed to that call, exactly like any other root-rendered component
       (see examples/radix-demo/components/HomePage.vue for the same
       root-render convention). htmlc has no explicit prop-declaration
       syntax: any identifier referenced in the template that isn't bound by
       a parent becomes an implicitly required prop, filled either way.

  The root element carries id="counter" so hx-target="#counter" always
  finds it, and the increment button's hx-swap="outerHTML" response (also
  rooted at id="counter", since RenderFragmentSession re-renders this same
  template) replaces the whole element in place — the standard htmx
  "self-targeting fragment" idiom: the fragment returned by an action
  carries the same id as the element it replaces, so a second click's
  hx-target lookup still succeeds after the swap.
-->
<template>
  <div id="counter" class="counter">
    <p class="count">Count: {{ count }}</p>
    <button
      class="increment-btn"
      hx-post="/increment"
      hx-target="#counter"
      hx-swap="outerHTML"
    >
      Increment
    </button>
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
  margin: 0 0 0.75rem;
}
.increment-btn {
  padding: 0.5rem 1rem;
  font-size: 1rem;
  cursor: pointer;
}
</style>
