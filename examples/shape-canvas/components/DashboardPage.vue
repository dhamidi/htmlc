<template>
  <html>
    <head>
      <meta charset="utf-8">
      <title>Shape Canvas Demo</title>
      <style>
        body { font-family: sans-serif; background: #f0f2f5; margin: 0; padding: 2rem; }
        h1 { color: #1a1a2e; margin-bottom: 1.5rem; }
        .canvases { display: flex; gap: 2rem; flex-wrap: wrap; }
        .canvas-card { background: white; border-radius: 8px; padding: 1rem;
                       box-shadow: 0 2px 8px rgba(0,0,0,.1); }
        .canvas-card h2 { margin: 0 0 .75rem; font-size: 1rem; color: #555; }
        .badge { display: inline-block; background: #4caf50; color: white;
                 border-radius: 4px; padding: 2px 8px; font-size: .75rem; margin-left: .5rem; }
      </style>
    </head>
    <body>
      <h1>Live Shape Canvas <span class="badge">RFC 006</span></h1>
      <p>Two independent canvases, same SSE stream. Each is a <code>&lt;widgets-shape-canvas&gt;</code> custom element.</p>
      <div class="canvases">
        <div class="canvas-card">
          <h2>Canvas A</h2>
          <ShapeCanvas src="/api/shapes/stream" :width="400" :height="300"></ShapeCanvas>
        </div>
        <div class="canvas-card">
          <h2>Canvas B</h2>
          <ShapeCanvas src="/api/shapes/stream" :width="400" :height="300"></ShapeCanvas>
        </div>
      </div>
    </body>
  </html>
</template>
