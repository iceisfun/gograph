# Showcase Example

A complex example demonstrating all GoGraph node types with rich animations.
The graph uses multiple paths with source, transform, delay, and output nodes
across different categories. It also demonstrates state connections with
oscillators, toggles, and logic gates.

## Graph Layout

```
src1 ("Hello, World!") --> Lowercase --> Delay (500ms) --> Hex Dump
       |
       +--> Reverse --> Hex Dump

src2 ("GoGraph!") --> Delay (1500ms) --> Reverse --> Hex Dump

oscillator --[state]--> switch --[event]--> sink
toggle -----[state]--/  (state enable + event data passthrough)

toggle --[state]--> AND gate --[state]--> output
oscillator ------/
```

## Running

From the `examples/showcase` directory:

```sh
go run . -addr :8080
```

Then open http://127.0.0.1:8080 in a browser.

Use `-dev` to serve the frontend from disk during development:

```sh
go run . -dev -addr :8080
```

## What to Look For

- **Category colors** -- each node category (source, transform, delay, output) renders with a distinct color theme.
- **Animated dashes** -- active event connections show marching-dash animations while events travel along them.
- **State connection steady glows** -- state wires glow steadily when their value is truthy, dim when falsy. No dot animation.
- **Mixed event+state nodes** -- the switch node uses a state input for enable/disable and passes event data through when enabled.
- **Node border animations** -- nodes pulse when they are executing.
- **Per-node duration** -- the delay nodes have different durations (500ms vs 1500ms), so their connection animations move at different speeds.
- **Multiple paths** -- src1 fans out to two independent pipelines; src2 feeds a separate chain. All paths animate concurrently.
