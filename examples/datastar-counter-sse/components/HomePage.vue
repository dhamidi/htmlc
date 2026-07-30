<!--
  HomePage — the datastar-counter-sse demo's full-page shell.

  Rendered only by GET / (engine.RenderPage) — the initial page load is
  always a full document, exactly like examples/htmx-counter's HomePage.
  Embeds Counter directly; Counter's own root element already carries
  id="ds-counter" (see components/Counter.vue), which is also the exact
  "#ds-counter" selector RFC 014 §6 Example 6's handleCounterStream passes
  to PatchElementsFragment — so no extra wrapper div is needed here the way
  htmx-counter's #status-badge OOB wrapper was: Datastar's default "outer"
  patch mode replaces the whole matched element (id="ds-counter" and all),
  and the freshly rendered Counter fragment carries that same id again, so
  a second SSE tick's selector lookup still finds it after being replaced.

  The <script> tag loading the Datastar client library is included only so
  this page is genuinely interactive when opened in a real browser: as soon
  as it loads, "data-on-load" below opens an EventSource-backed SSE
  connection to GET /counter-stream and applies the datastar-patch-elements
  events this demo's server pushes at #ds-counter. Neither the <script> tag
  nor the "data-on-load" attribute is exercised by this module's Go tests —
  they drive the HTTP handlers directly (net/http/httptest) with no browser
  or JS runtime involved.
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
      <!-- Datastar client library, CDN reference for real-browser use only;
           not required by (and not reachable from) this module's Go tests. -->
      <script
        type="module"
        src="https://cdn.jsdelivr.net/gh/starfederation/datastar@main/bundles/datastar.js"
      ></script>
    </head>
    <body data-on-load="@get('/counter-stream')">
      <h1>{{ title }}</h1>
      <p>
        This page opens a long-lived SSE connection to /counter-stream on
        load; the server pushes three ticks that patch the counter below
        in place, with no client-initiated request per tick.
      </p>
      <Counter :count="count"></Counter>
    </body>
  </html>
</template>
