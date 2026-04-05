# Embedded Example

Demonstrates mounting GoGraph within a larger HTTP application using a
route prefix. The graph editor is served at `/graph/` while the parent
application handles other routes.

## Run

```bash
# From the repository root:
make build
go run ./examples/embedded
```

Open http://127.0.0.1:8080 to see the parent application, then follow the
link to `/graph/` for the graph editor.
