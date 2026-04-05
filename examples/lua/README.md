# Lua Example

Demonstrates GoGraph with Lua-scripted node execution. A source node
produces "hello world", an uppercase node transforms it, and a sink
node prints the result.

The graph executes every 5 seconds, producing animated events that
traverse the connections in the frontend.

## Run

```bash
# From the repository root:
make build
cd examples/lua
go run .
```

Open http://127.0.0.1:8080 to see the graph. Watch for animated dots
moving along the connections every 5 seconds as the graph executes.

## Scripts

- `scripts/example.lua` - The uppercase transformation script. It reads
  `inputs["in"]` and returns `{ out = string.upper(data) }`.
