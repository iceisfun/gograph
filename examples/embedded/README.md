# Embedded Example

Demonstrates mounting GoGraph within a larger HTTP application using
`gograph.Mount()`. The graph editor is served at `/graph/` while the
parent application handles other routes.

## Go Setup

```go
mux := http.NewServeMux()
srv := gograph.Mount(mux, "/graph", gograph.MountOptions{
    Store:    store.NewMemoryStore(),
    Registry: reg,
})
srv.SetEngine(eng)
```

The `/config` endpoint automatically returns
`{"apiBase": "/graph/api", "mode": "edit"}`, so the frontend discovers
its API base without extra configuration.

## Frontend

The embedded frontend can be mounted via data attributes:

```html
<div data-gograph data-graph-id="demo" data-api="/graph/api"
     style="width:100%;height:600px"></div>
```

Or programmatically:

```js
const g = await GoGraph.create(document.getElementById('editor'), {
    apiBase: '/graph/api',
});
```

## Run

```bash
# From the repository root:
make build
go run ./examples/embedded
```

Open http://127.0.0.1:8080 to see the parent application, then follow the
link to `/graph/` for the graph editor.
