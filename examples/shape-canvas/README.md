# shape-canvas

A live demo of server-driven custom elements built with htmlc.

Two independent canvas elements each receive a real-time stream of randomly
generated rectangles and circles from the server.  The canvases are implemented
as a `<script customelement>` block in `components/widgets/ShapeCanvas.vue`.

## Running with `go run`

The `main.go` server handles three routes:

| Route | Description |
|-------|-------------|
| `/` | Renders `DashboardPage` as a full HTML page |
| `/scripts/` | Serves collected custom element scripts |
| `/api/shapes/stream` | SSE endpoint that streams random shape data |

```sh
cd examples/shape-canvas
go run .
```

Open <http://localhost:8081>.  Both canvases will start receiving shapes
immediately.

## Running with the `htmlc` CLI

The same page can be built and served with the `htmlc` CLI.  Because `htmlc`
is a static renderer it does not provide the `/api/shapes/stream` SSE endpoint;
the canvases will be present in the page but will not receive shape data unless
a server providing that endpoint is also running.

### Build the HTML once

```sh
cd examples/shape-canvas
go run ../../cmd/htmlc build \
  -dir ./components \
  -pages ./components \
  -out ./out
```

Output:

```
out/
  DashboardPage.html
  widgets/
    ShapeCanvas.html
  scripts/
    components-widgets-shape-canvas.<hash>.js
    index.js
```

### Serve with live rebuild

The `-dev` flag starts a development server that watches for component changes
and rebuilds automatically.

```sh
cd examples/shape-canvas
go run ../../cmd/htmlc build \
  -dir ./components \
  -pages ./components \
  -out ./out \
  -dev :8080
```

Open <http://localhost:8080/DashboardPage.html>.

Custom element scripts are served from `/scripts/` automatically by the dev
server, so the `<script customelement>` code in `ShapeCanvas.vue` loads
correctly in the browser.

### Render the page to stdout

To inspect the rendered HTML without writing files:

```sh
cd examples/shape-canvas
go run ../../cmd/htmlc page -dir ./components DashboardPage
```
