<template>
  <html lang="en">
    <head>
      <meta charset="utf-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <meta name="description" :content="description ?? 'htmlc — server-side Vue component rendering in Go'" />
      <title>{{ pageTitle ?? siteTitle }}</title>
      <link rel="stylesheet" href="/fonts.css" />
      <link rel="icon" href="/images/logo.svg" type="image/svg+xml" />
      <link rel="icon" href="/images/logo.png" type="image/png" />
      <style>
        *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

        :root {
          --bg: #0f1117;
          --bg2: #1a1d27;
          --border: #2a2d3e;
          --text: #e2e4f0;
          --muted: #8b8fa8;
          --accent: #00ADD8;
          --accent-hover: #0099bf;
          --accent-alpha12: rgba(0, 173, 216, 0.12);
          --accent-alpha35: rgba(0, 173, 216, 0.35);
          --accent2: #5be49b;
          --code-bg: #161822;
          --nav-height: 60px;
          --max-width: 1100px;
        }

        body {
          background: var(--bg);
          color: var(--text);
          font-family: 'Outfit', system-ui, sans-serif;
          font-size: 16px;
          line-height: 1.6;
          min-height: 100vh;
        }

        a { color: var(--accent); text-decoration: none; }
        a:hover { text-decoration: underline; }

        code, kbd {
          font-family: "SF Mono", "Fira Code", "Cascadia Code", monospace;
          font-size: 0.875em;
        }

        pre {
          background: var(--code-bg);
          border: 1px solid var(--border);
          border-radius: 8px;
          padding: 1.25rem 1.5rem;
          overflow-x: auto;
          line-height: 1.5;
        }

        pre code { font-size: 0.85rem; color: #c9d1d9; }

        :not(pre) > code {
          background: var(--code-bg);
          border: 1px solid var(--border);
          border-radius: 4px;
          padding: 0.15em 0.4em;
          color: var(--accent);
        }

        h1, h2, h3, h4 {
          font-family: 'Alegreya', Georgia, serif;
          line-height: 1.2;
          font-weight: 700;
          letter-spacing: -0.02em;
        }

        table {
          border-collapse: collapse;
          width: 100%;
          margin: 1.5rem 0;
        }
        th, td {
          text-align: left;
          padding: 0.6rem 1rem;
          border: 1px solid var(--border);
        }
        th { background: var(--bg2); font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--muted); }

        .site-nav {
          position: sticky;
          top: 0;
          z-index: 100;
          background: rgba(15, 17, 23, 0.9);
          backdrop-filter: blur(12px);
          border-bottom: 1px solid var(--border);
          height: var(--nav-height);
          display: flex;
          align-items: center;
          padding: 0 2rem;
          gap: 2rem;
        }

        .site-nav .logo {
          font-family: 'Outfit', system-ui, sans-serif;
          font-weight: 700;
          font-size: 1.1rem;
          color: var(--text);
          letter-spacing: -0.03em;
          display: flex;
          align-items: center;
          gap: 0.4rem;
        }

        .site-nav .nav-links {
          display: flex;
          gap: 1.5rem;
          margin-left: auto;
        }

        .site-nav .nav-links a {
          color: var(--muted);
          font-size: 0.875rem;
          transition: color 0.15s;
        }

        .site-nav .nav-links a:hover { color: var(--text); text-decoration: none; }

        .site-main {
          max-width: var(--max-width);
          margin: 0 auto;
          padding: 0 2rem;
        }

        .site-footer {
          border-top: 1px solid var(--border);
          margin-top: 6rem;
          padding: 2rem;
          text-align: center;
          color: var(--muted);
          font-size: 0.8rem;
        }

        .site-footer a { color: var(--muted); }
        .site-footer a:hover { color: var(--text); }
      </style>
    </head>
    <body>
      <nav class="site-nav">
        <a href="/" class="logo">
          <img src="/images/logo.webp" alt="htmlc logo" width="32" height="32" style="border-radius:6px;">htmlc</a>
        <div class="nav-links">
          <a href="/docs/index.html">Docs</a>
          <a href="/docs/cli.html">CLI</a>
          <a href="/docs/directives.html">Directives</a>
          <a href="https://github.com/dhamidi/htmlc" target="_blank" rel="noopener">GitHub</a>
        </div>
      </nav>
      <main class="site-main">
        <slot><!-- page content --></slot>
      </main>
      <footer class="site-footer">
        <p>
          htmlc — MIT License &mdash;
          <a href="https://github.com/dhamidi/htmlc">github.com/dhamidi/htmlc</a>
        </p>
      </footer>
    </body>
  </html>
</template>
