<!--
  TodoCount — a simple count summary, e.g. "3 todos".

  Rendered inside HomePage's initial `<span id="todo-count">` wrapper on
  the full page GET / and, again, directly via
  engine.RenderFragmentString(ctx, "TodoCount", map[string]any{"count": n})
  by POST /todos, whose output is wrapped by
  turbo.WriteStream(w, "update", "todo-count", countHTML) — the RFC's
  second action in the same response (RFC 014 §6 Example 5).

  Deliberately does NOT carry the id "todo-count" on its own root — the
  same reasoning as htmx-counter's StatusBadge
  (examples/htmx-counter/components/StatusBadge.vue): the id lives on the
  wrapping <span id="todo-count"> in HomePage, and Turbo's "update" action
  replaces that wrapper's *children* with this component's rendered
  output. Giving this component's own root the same id would create two
  elements sharing one id after the very first update.
-->
<template>
  <span class="todo-count">{{ count }} todos</span>
</template>
<style scoped>
.todo-count {
  font-weight: bold;
}
</style>
