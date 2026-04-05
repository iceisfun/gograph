# Basic Example

Minimal GoGraph setup with a source, two transforms, and a sink node.

## Run

```bash
# From the repository root:
make build
go run ./examples/basic

# Development mode (serves frontend from disk):
go run ./examples/basic -dev
```

Open http://127.0.0.1:8080 in a browser. You should see a canvas with four
nodes connected by Bezier curves. In edit mode you can drag nodes, create
new connections, and delete selected items.
