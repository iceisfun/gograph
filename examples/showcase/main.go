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
	"time"

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

	// Register node types across several categories.
	reg := graph.NewRegistry()

	must(reg.Register(graph.NodeType{
		Name:     "source",
		Label:    "Source",
		Category: "source",
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: `return { out = "Hello, World!" }`,
	}))
	must(reg.Register(graph.NodeType{
		Name:     "source2",
		Label:    "Source 2",
		Category: "source",
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: `return { out = "GoGraph!" }`,
	}))
	must(reg.Register(graph.NodeType{
		Name:     "lowercase",
		Label:    "Lowercase",
		Category: "transform",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/lowercase.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:     "reverse",
		Label:    "Reverse",
		Category: "transform",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/reverse.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:     "hexdump",
		Label:    "Hex Dump",
		Category: "output",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
		},
		Script: mustReadFile("scripts/hexdump.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:     "print",
		Label:    "Print",
		Category: "output",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
		},
		Script: mustReadFile("scripts/print.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:     "delay",
		Label:    "Delay",
		Category: "delay",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
		Script: mustReadFile("scripts/delay.lua"),
	}))

	// Build the showcase graph with multiple paths.
	//
	//   src1 ("Hello, World!") --> lower --> delay1 (500ms) --> hex1
	//        |
	//        +--> reverse --> print1
	//
	//   src2 ("GoGraph!") --> delay2 (1500ms) --> reverse2 --> hex2
	//
	g := graph.NewGraph("showcase")

	// Row 1: src1 -> lower -> delay1 -> hex1
	must(g.AddNode(&graph.Node{ID: "src1", Type: "source", Label: "Hello, World!", Position: graph.Position{X: 80, Y: 150}}))
	must(g.AddNode(&graph.Node{ID: "lower", Type: "lowercase", Label: "Lowercase", Position: graph.Position{X: 350, Y: 80}}))
	must(g.AddNode(&graph.Node{ID: "delay1", Type: "delay", Label: "Delay 500ms", Position: graph.Position{X: 620, Y: 80}, Config: map[string]string{"duration": "500"}}))
	must(g.AddNode(&graph.Node{ID: "hex1", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 890, Y: 80}}))

	// Row 2: src1 -> reverse -> print1
	must(g.AddNode(&graph.Node{ID: "reverse", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 350, Y: 280}}))
	must(g.AddNode(&graph.Node{ID: "print1", Type: "print", Label: "Print", Position: graph.Position{X: 620, Y: 280}}))

	// Row 3: src2 -> delay2 -> reverse2 -> hex2
	must(g.AddNode(&graph.Node{ID: "src2", Type: "source2", Label: "GoGraph!", Position: graph.Position{X: 80, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "delay2", Type: "delay", Label: "Delay 1500ms", Position: graph.Position{X: 350, Y: 420}, Config: map[string]string{"duration": "1500"}}))
	must(g.AddNode(&graph.Node{ID: "reverse2", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 620, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "hex2", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 890, Y: 420}}))

	// Wire connections with per-connection traversal durations.
	// Timed connections show animated dots, instant ones just flash/dash.
	must(g.Connect(&graph.Connection{ID: "c1", FromNode: "src1", FromSlot: "out", ToNode: "lower", ToSlot: "in", Config: map[string]string{"duration": "800"}}))
	must(g.Connect(&graph.Connection{ID: "c2", FromNode: "lower", FromSlot: "out", ToNode: "delay1", ToSlot: "in", Config: map[string]string{"duration": "600"}}))
	must(g.Connect(&graph.Connection{ID: "c3", FromNode: "delay1", FromSlot: "out", ToNode: "hex1", ToSlot: "in"}))                                               // instant
	must(g.Connect(&graph.Connection{ID: "c4", FromNode: "src1", FromSlot: "out", ToNode: "reverse", ToSlot: "in", Config: map[string]string{"duration": "1000"}}))
	must(g.Connect(&graph.Connection{ID: "c5", FromNode: "reverse", FromSlot: "out", ToNode: "print1", ToSlot: "in"}))                                              // instant
	must(g.Connect(&graph.Connection{ID: "c6", FromNode: "src2", FromSlot: "out", ToNode: "delay2", ToSlot: "in", Config: map[string]string{"duration": "800"}}))
	must(g.Connect(&graph.Connection{ID: "c7", FromNode: "delay2", FromSlot: "out", ToNode: "reverse2", ToSlot: "in", Config: map[string]string{"duration": "1200"}}))
	must(g.Connect(&graph.Connection{ID: "c8", FromNode: "reverse2", FromSlot: "out", ToNode: "hex2", ToSlot: "in"}))                                                // instant

	// Persist.
	st := store.NewMemoryStore()
	must(st.Save(context.Background(), g.ID, g))

	// Engine with Lua executor.
	luaExec := golua.New(reg)
	eng := engine.New(g,
		engine.WithRegistry(reg),
		engine.WithExecutor(luaExec),
		engine.WithEventDuration(1000),
		engine.WithStore(st, g.ID),
		engine.WithNodeLogger(engine.DebugNodeLogger{}),
		engine.WithEventLogger(engine.DebugEventLogger{}),
	)

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

	// Wire engine events to SSE.
	sub := eng.Subscribe(64)
	go func() {
		for evt := range sub.Events() {
			srv.Publish(g.ID, evt.Type, evt.Payload)
		}
	}()

	// Execute every 5 seconds after a 2s startup delay.
	go func() {
		time.Sleep(2 * time.Second)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := eng.Execute(ctx); err != nil {
				log.Printf("execution error: %v", err)
			}
			cancel()
			time.Sleep(5 * time.Second)
		}
	}()

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
