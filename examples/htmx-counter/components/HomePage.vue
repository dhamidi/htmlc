<!--
  HomePage — the htmx-counter demo's full-page shell.

  Rendered only by GET / (engine.RenderPage), regardless of any request
  header — the initial page load is always a full document, per RFC 014 §6
  Example 4.

  Embeds Counter for the initial render and a pre-existing #status-badge
  wrapper (empty of htmx attributes on first load) so that POST /increment's
  htmx response — a `<div id="status-badge" hx-swap-oob="true">…</div>`
  fragment — has a real element with a matching id already present in the
  DOM to swap into. Without this element existing up front, htmx's
  hx-swap-oob="true" mechanism has nothing to target on the very first
  increment.
-->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ title }}</title>
      <!-- htmx itself is not required to run this demo's Go tests (which
           drive the handlers directly over HTTP without a browser or a JS
           runtime); it is included here only so the page is genuinely
           interactive when opened in a real browser. -->
      <script src="https://unpkg.com/htmx.org@2.0.4"></script>
    </head>
    <body>
      <h1>{{ title }}</h1>
      <p>Click the button; htmx posts to /increment and swaps the counter in place.</p>
      <Counter :count="count"></Counter>
      <div id="status-badge">
        <StatusBadge :count="count"></StatusBadge>
      </div>
    </body>
  </html>
</template>
