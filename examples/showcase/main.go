// Command showcase demonstrates all GoGraph node types with rich animations.
// It builds a multi-path graph with source, transform, delay, and output
// nodes to exercise different categories, per-node duration overrides, and
// animated connection events.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/iceisfun/gograph/engine"
	"github.com/iceisfun/gograph/frontend"
	"github.com/iceisfun/gograph/graph"
	golua "github.com/iceisfun/gograph/lua"
	"github.com/iceisfun/gograph/server"
	"github.com/iceisfun/gograph/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dev := flag.Bool("dev", false, "serve frontend from disk")
	flag.Parse()

	// Register node types — Lua scripts define their own shape.
	reg := graph.NewRegistry()
	scripts := []string{
		"source", "lowercase", "reverse", "hexdump", "print",
		"delay", "show", "randomcase", "oscillator", "switch",
		"words", "splitter", "ratelimit",
		"and", "or", "xor", "not",
		"counter", "shift_register", "toggle", "gate",
		"dashboard",
	}
	for _, name := range scripts {
		must(golua.Register(reg, name, mustReadFile("scripts/"+name+".lua")))
	}

	// Build the showcase graph with multiple paths.
	//
	//   src1 ("Hello, World!") --> lower --> delay1 (500ms) --> hex1
	//        |
	//        +--> reverse --> print1
	//
	//   src2 ("GoGraph!") --> delay2 (1500ms) --> randomcase --> reverse2 --> hex2
	//
	g := graph.NewGraph("showcase")

	// Row 1: src1 -> lower -> delay1 -> hex1
	must(g.AddNode(&graph.Node{ID: "src1", Type: "source", Label: "Source", Position: graph.Position{X: 80, Y: 150}, Config: map[string]string{"message": "Hello, World!", "interval": "5000"}}))
	must(g.AddNode(&graph.Node{ID: "lower", Type: "lowercase", Label: "Lowercase", Position: graph.Position{X: 350, Y: 80}}))
	must(g.AddNode(&graph.Node{ID: "delay1", Type: "delay", Label: "Delay", Position: graph.Position{X: 620, Y: 80}, Config: map[string]string{"duration": "500"}}))
	must(g.AddNode(&graph.Node{ID: "hex1", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 890, Y: 80}}))

	// Row 2: src1 -> reverse -> print1
	must(g.AddNode(&graph.Node{ID: "reverse", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 350, Y: 280}}))
	must(g.AddNode(&graph.Node{ID: "print1", Type: "show", Label: "Show", Position: graph.Position{X: 620, Y: 280}}))

	// Row 3: src2 -> delay2 -> randomcase -> reverse2 -> hex2
	must(g.AddNode(&graph.Node{ID: "src2", Type: "source", Label: "Source", Position: graph.Position{X: 80, Y: 420}, Config: map[string]string{"message": "GoGraph!", "interval": "5000"}}))
	must(g.AddNode(&graph.Node{ID: "delay2", Type: "delay", Label: "Delay", Position: graph.Position{X: 350, Y: 420}, Config: map[string]string{"duration": "1500"}}))
	must(g.AddNode(&graph.Node{ID: "rcase", Type: "randomcase", Label: "RanDOmCaSe", Position: graph.Position{X: 530, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "reverse2", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 750, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "hex2", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 1020, Y: 420}}))

	// Wire connections with per-connection traversal durations.
	// Timed connections show animated dots, instant ones just flash/dash.
	dur := func(ms string) map[string]string { return map[string]string{"duration": ms} }
	must(g.Connect(graph.NewEventConnection("c1", "src1", "out", "lower", "in", dur("800"))))
	must(g.Connect(graph.NewEventConnection("c2", "lower", "out", "delay1", "in", dur("600"))))
	must(g.Connect(graph.NewEventConnection("c3", "delay1", "out", "hex1", "in", nil)))
	must(g.Connect(graph.NewEventConnection("c4", "src1", "out", "reverse", "in", dur("1000"))))
	must(g.Connect(graph.NewEventConnection("c5", "reverse", "out", "print1", "in", nil)))
	must(g.Connect(graph.NewEventConnection("c6", "src2", "out", "delay2", "in", dur("800"))))
	must(g.Connect(graph.NewEventConnection("c7", "delay2", "out", "rcase", "in", dur("1200"))))
	must(g.Connect(graph.NewEventConnection("c7b", "rcase", "out", "reverse2", "in", dur("600"))))
	must(g.Connect(graph.NewEventConnection("c8", "reverse2", "out", "hex2", "in", nil)))

	// Row 4: oscillator enables switch, src1 feeds data through it
	must(g.AddNode(&graph.Node{ID: "osc1", Type: "oscillator", Label: "Oscillator", Position: graph.Position{X: 80, Y: 580}, Config: map[string]string{"period": "3000"}}))
	must(g.AddNode(&graph.Node{ID: "sw1", Type: "switch", Label: "Switch", Position: graph.Position{X: 400, Y: 580}}))
	must(g.AddNode(&graph.Node{ID: "show_on", Type: "show", Label: "Show (Pass)", Position: graph.Position{X: 700, Y: 520}}))
	must(g.AddNode(&graph.Node{ID: "show_off", Type: "show", Label: "Show (Discard)", Position: graph.Position{X: 700, Y: 650}}))

	must(g.Connect(graph.NewStateConnection("c9", "osc1", "out", "sw1", "en", "bool", nil)))
	must(g.Connect(graph.NewEventConnection("c9b", "src1", "out", "sw1", "in", dur("600"))))
	must(g.Connect(graph.NewEventConnection("c10", "sw1", "out", "show_on", "in", nil)))
	must(g.Connect(graph.NewEventConnection("c11", "sw1", "discard", "show_off", "in", nil)))

	// Row 5: two oscillators -> AND gate -> show
	must(g.AddNode(&graph.Node{ID: "osc2", Type: "oscillator", Label: "Osc A", Position: graph.Position{X: 80, Y: 780}, Config: map[string]string{"period": "4000"}}))
	must(g.AddNode(&graph.Node{ID: "osc3", Type: "oscillator", Label: "Osc B", Position: graph.Position{X: 80, Y: 900}, Config: map[string]string{"period": "6000"}}))
	must(g.AddNode(&graph.Node{ID: "and1", Type: "and", Label: "AND", Position: graph.Position{X: 400, Y: 840}}))
	must(g.AddNode(&graph.Node{ID: "show_and", Type: "show", Label: "AND Result", Position: graph.Position{X: 700, Y: 840}}))

	must(g.Connect(graph.NewStateConnection("c12", "osc2", "out", "and1", "a", "bool", nil)))
	must(g.Connect(graph.NewStateConnection("c13", "osc3", "out", "and1", "b", "bool", nil)))
	must(g.Connect(graph.NewStateConnection("c14", "and1", "out", "show_and", "in", "bool", nil)))

	// Row 6: toggle -> gate -> show (interactive demo)
	must(g.AddNode(&graph.Node{ID: "tog1", Type: "toggle", Label: "Toggle", Position: graph.Position{X: 80, Y: 1020}, Config: map[string]string{"state": "on"}}))
	must(g.AddNode(&graph.Node{ID: "gate1", Type: "gate", Label: "Gate", Position: graph.Position{X: 400, Y: 1020}, Config: map[string]string{"state": "on"}}))
	must(g.AddNode(&graph.Node{ID: "show_gate", Type: "show", Label: "Show", Position: graph.Position{X: 700, Y: 1020}}))

	must(g.Connect(graph.NewStateConnection("c15", "tog1", "out", "gate1", "in", "bool", nil)))
	must(g.Connect(graph.NewEventConnection("c16", "gate1", "out", "show_gate", "in", nil)))

	// Row 7: dashboard (all slot types demo)
	must(g.AddNode(&graph.Node{ID: "dash1", Type: "dashboard", Label: "Dashboard", Position: graph.Position{X: 80, Y: 1180}, Config: map[string]string{"interval": "1500"}}))

	// Persist.
	st := store.NewMemoryStore()
	must(st.Save(context.Background(), g.ID, g))

	// Server.
	opts := []server.Option{
		server.WithStore(st),
		server.WithRegistry(reg),
	}
	if *dev {
		opts = append(opts, server.WithStaticFS(os.DirFS("frontend/dist")))
	} else {
		opts = append(opts, server.WithStaticFS(frontend.FS()))
	}
	srv := server.New(opts...)

	// Engine — supervisor, not orchestrator. Server IS the event broker.
	eng := engine.New(
		engine.WithRegistry(reg),
		engine.WithStore(st, g.ID),
		engine.WithBroker(srv),
	)
	srv.SetEngine(eng)

	// Start — creates goroutines for all nodes, wires for all connections.
	must(eng.Start(context.Background()))
	defer eng.Stop()

	fmt.Printf("GoGraph Showcase: http://127.0.0.1%s\n", *addr)
	log.Fatal(srv.ListenAndServe(*addr))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustReadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
