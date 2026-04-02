<template>
  <html>
    <head>
      <meta charset="utf-8">
      <meta name="viewport" content="width=device-width, initial-scale=1">
      <title>htmlc Live Demo — Server-Driven Custom Elements</title>
      <style>
        *, *::before, *::after { box-sizing: border-box; }

        body {
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
          background: #0f172a;
          color: #e2e8f0;
          margin: 0;
          min-height: 100vh;
          display: flex;
          flex-direction: column;
        }

        /* ── Header ── */
        .site-header {
          padding: 2.5rem 1.5rem 2rem;
          text-align: center;
          background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
          border-bottom: 1px solid #1e3a5f;
        }

        .logo-label {
          display: inline-block;
          font-size: .75rem;
          font-weight: 700;
          letter-spacing: .12em;
          text-transform: uppercase;
          color: #38bdf8;
          margin-bottom: .75rem;
        }

        .site-header h1 {
          margin: 0 0 .75rem;
          font-size: clamp(1.75rem, 5vw, 2.75rem);
          font-weight: 800;
          line-height: 1.15;
          background: linear-gradient(90deg, #38bdf8, #818cf8);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .site-header p {
          margin: 0 auto;
          max-width: 52ch;
          font-size: 1rem;
          line-height: 1.6;
          color: #94a3b8;
        }

        .site-header p code {
          font-family: "SFMono-Regular", Consolas, monospace;
          font-size: .875em;
          color: #7dd3fc;
          background: #1e3a5f;
          border-radius: 4px;
          padding: 1px 5px;
        }

        /* ── Main content ── */
        main {
          flex: 1;
          padding: 2.5rem 1.5rem;
        }

        .canvases {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(min(100%, 480px), 1fr));
          gap: 1.5rem;
          max-width: 1100px;
          margin: 0 auto;
        }

        .canvas-card {
          background: #1e293b;
          border: 1px solid #334155;
          border-radius: 12px;
          padding: 1.25rem;
          box-shadow: 0 4px 24px rgba(0, 0, 0, .4);
        }

        .canvas-card-header {
          display: flex;
          align-items: center;
          gap: .5rem;
          margin-bottom: 1rem;
        }

        .canvas-dot {
          width: 10px;
          height: 10px;
          border-radius: 50%;
          background: #22d3ee;
          box-shadow: 0 0 6px #22d3ee;
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: .4; }
        }

        .canvas-card h2 {
          margin: 0;
          font-size: .9rem;
          font-weight: 600;
          color: #cbd5e1;
          letter-spacing: .04em;
          text-transform: uppercase;
        }

        .canvas-card canvas,
        .canvas-card widgets-shape-canvas {
          display: block;
          width: 100%;
          max-width: 100%;
          border-radius: 6px;
          overflow: hidden;
        }

        /* ── Footer ── */
        footer {
          padding: 1.25rem 1.5rem;
          text-align: center;
          font-size: .85rem;
          color: #475569;
          border-top: 1px solid #1e293b;
        }

        footer a {
          color: #38bdf8;
          text-decoration: none;
        }

        footer a:hover {
          text-decoration: underline;
        }
      </style>
    <script type="importmap">{{ importMap("/scripts/") }}</script>
    <script type="module" src="/scripts/index.js"></script>
  </head>
    <body>
      <header class="site-header">
        <span class="logo-label">htmlc</span>
        <h1>Server-Driven Custom Elements</h1>
        <p>
          <code>htmlc</code> compiles Go templates into HTML pages with live server-side components.
          Each canvas below is an independent custom element receiving a real-time shape stream from the server — no client-side framework required.
        </p>
      </header>

      <main>
        <div class="canvases">
          <div class="canvas-card">
            <div class="canvas-card-header">
              <span class="canvas-dot"></span>
              <h2>Live Canvas A</h2>
            </div>
            <ShapeCanvas src="/api/shapes/stream" :width="480" :height="360"></ShapeCanvas>
          </div>
          <div class="canvas-card">
            <div class="canvas-card-header">
              <span class="canvas-dot"></span>
              <h2>Live Canvas B</h2>
            </div>
            <ShapeCanvas src="/api/shapes/stream" :width="480" :height="360"></ShapeCanvas>
          </div>
        </div>
      </main>

      <footer>
        <p>
          Built with <a href="https://github.com/dhamidi/htmlc" target="_blank" rel="noopener">htmlc</a> —
          Go-powered HTML components with server-side rendering and live custom elements.
        </p>
      </footer>
    </body>
  </html>
</template>
