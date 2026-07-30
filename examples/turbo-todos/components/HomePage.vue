<!--
  HomePage — the turbo-todos demo's full-page shell.

  Rendered only by GET / (engine.RenderPage) — the initial page load is
  always a full document, listing whatever todos already exist in the
  in-memory store, per RFC 014 §6 Example 5.

  The <form> posts to /todos. Turbo Drive (loaded from a CDN purely so the
  page is genuinely interactive in a real browser — this demo's Go tests
  drive the handlers directly over HTTP, with no browser or JS runtime
  involved) intercepts the submission and sends it with
  "Accept: text/vnd.turbo-stream.html" among its accepted types, which
  main.go's turbo.WantsStream check recognizes to choose the two-action
  Turbo Stream response over the plain-POST fallback (a 303 redirect back
  to "/", for a client that isn't Turbo-aware — e.g. JS disabled).

  <ul id="todo-list"> and <span id="todo-count"> are the two targets
  POST /todos' response updates in place: turbo.WriteStream(w, "append",
  "todo-list", ...) appends a new TodoItem inside the <ul>;
  turbo.WriteStream(w, "update", "todo-count", ...) replaces the <span>'s
  children with a freshly rendered TodoCount — both actions land in one
  response, with no special framing required between the two
  <turbo-stream> elements (RFC 014 §6 Example 5's central, already-confirmed
  claim).
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
      <!-- Turbo itself is not required to run this demo's Go tests; it is
           included here only so the page is genuinely interactive when
           opened in a real browser. -->
      <script type="module" src="https://unpkg.com/@hotwired/turbo@8.0.4/dist/turbo.es2017-umd.js"></script>
    </head>
    <body>
      <h1>{{ title }}</h1>
      <p>Add a todo below; a Turbo-aware client swaps in the new item and count without a full page reload.</p>
      <p>Total: <span id="todo-count"><TodoCount :count="count"></TodoCount></span></p>
      <form method="post" action="/todos">
        <input type="text" name="text" placeholder="What needs doing?" required />
        <button type="submit">Add</button>
      </form>
      <ul id="todo-list">
        <TodoItem v-for="todo in todos" :todo="todo"></TodoItem>
      </ul>
    </body>
  </html>
</template>
