<!--
  TodoItem — renders a single todo entry (id, text, and its done state).

  Rendered two different ways, both proving the same component works
  identically in either position (RFC 014 §6 Example 5):

    1. As a child of HomePage's initial <ul id="todo-list"> on the full
       page GET / (embedded via <TodoItem v-for="todo in todos" :todo="todo" />,
       receiving "todo" as a bound prop).
    2. As the root fragment appended by POST /todos, rendered directly via
       engine.RenderFragmentString(ctx, "TodoItem", map[string]any{"todo": todo})
       — exactly the RFC's own handleTodoCreate — and wrapped by
       turbo.WriteStream(w, "append", "todo-list", itemHTML).

  The root <li> carries its own id ("todo-<ID>"), not "todo-list" (that id
  belongs to the wrapping <ul> in HomePage) — so appending a new TodoItem
  never collides with an existing id. Contrast with htmx-counter's
  StatusBadge, which deliberately omits its own id to avoid colliding with
  its hx-swap-oob wrapper (examples/htmx-counter/components/StatusBadge.vue):
  here there is no such collision risk, since "append" only ever adds new
  siblings under target, never replaces target's own id.
-->
<template>
  <li :id="'todo-' + todo.ID" class="todo-item">
    <span class="todo-text" v-if="todo.Done"><s>{{ todo.Text }}</s></span>
    <span class="todo-text" v-else>{{ todo.Text }}</span>
  </li>
</template>
<style scoped>
.todo-item {
  padding: 0.4rem 0;
  border-bottom: 1px solid #eee;
}
.todo-text s {
  color: #999;
}
</style>
