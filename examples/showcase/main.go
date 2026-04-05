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
	must(reg.Register(graph.NodeType{
		Name:          "show",
		Label:         "Show",
		Category:      "output",
		ContentHeight: 40,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
		},
		Script: mustReadFile("scripts/show.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:          "randomcase",
		Label:         "RanDOmCaSe",
		Category:      "transform",
		ContentHeight: 40,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/randomcase.lua"),
	}))

	// Logic & utility nodes
	must(reg.Register(graph.NodeType{
		Name: "oscillator", Label: "Oscillator", Category: "source", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/oscillator.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "switch", Label: "Switch", Category: "transform", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "on", Name: "On", Direction: graph.Output, DataType: "string"},
			{ID: "off", Name: "Off", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/switch.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "words", Label: "Words", Category: "source", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/words.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "splitter", Label: "Splitter", Category: "transform", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/splitter.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "ratelimit", Label: "Rate Limit", Category: "transform", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
		Script: mustReadFile("scripts/ratelimit.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "and", Label: "AND", Category: "logic", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "a", Name: "A", Direction: graph.Input, DataType: "string"},
			{ID: "b", Name: "B", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/and.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "or", Label: "OR", Category: "logic", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "a", Name: "A", Direction: graph.Input, DataType: "string"},
			{ID: "b", Name: "B", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/or.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "xor", Label: "XOR", Category: "logic", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "a", Name: "A", Direction: graph.Input, DataType: "string"},
			{ID: "b", Name: "B", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/xor.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "not", Label: "NOT", Category: "logic", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/not.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "counter", Label: "Counter", Category: "source", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/counter.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "shift_register", Label: "Shift Register", Category: "logic", ContentHeight: 30,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
		Script: mustReadFile("scripts/shift_register.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "toggle", Label: "Toggle", Category: "source",
		Interactive: true, ContentHeight: 40,
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/toggle.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name: "gate", Label: "Gate", Category: "transform",
		Interactive: true, ContentHeight: 40,
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
		Script: mustReadFile("scripts/gate.lua"),
	}))

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
	must(g.AddNode(&graph.Node{ID: "src1", Type: "source", Label: "Hello, World!", Position: graph.Position{X: 80, Y: 150}}))
	must(g.AddNode(&graph.Node{ID: "lower", Type: "lowercase", Label: "Lowercase", Position: graph.Position{X: 350, Y: 80}}))
	must(g.AddNode(&graph.Node{ID: "delay1", Type: "delay", Label: "Delay 500ms", Position: graph.Position{X: 620, Y: 80}, Config: map[string]string{"duration": "500"}}))
	must(g.AddNode(&graph.Node{ID: "hex1", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 890, Y: 80}}))

	// Row 2: src1 -> reverse -> print1
	must(g.AddNode(&graph.Node{ID: "reverse", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 350, Y: 280}}))
	must(g.AddNode(&graph.Node{ID: "print1", Type: "show", Label: "Show", Position: graph.Position{X: 620, Y: 280}}))

	// Row 3: src2 -> delay2 -> randomcase -> reverse2 -> hex2
	must(g.AddNode(&graph.Node{ID: "src2", Type: "source2", Label: "GoGraph!", Position: graph.Position{X: 80, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "delay2", Type: "delay", Label: "Delay 1500ms", Position: graph.Position{X: 350, Y: 420}, Config: map[string]string{"duration": "1500"}}))
	must(g.AddNode(&graph.Node{ID: "rcase", Type: "randomcase", Label: "RanDOmCaSe", Position: graph.Position{X: 530, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "reverse2", Type: "reverse", Label: "Reverse", Position: graph.Position{X: 750, Y: 420}}))
	must(g.AddNode(&graph.Node{ID: "hex2", Type: "hexdump", Label: "Hex Dump", Position: graph.Position{X: 1020, Y: 420}}))

	// Wire connections with per-connection traversal durations.
	// Timed connections show animated dots, instant ones just flash/dash.
	must(g.Connect(&graph.Connection{ID: "c1", FromNode: "src1", FromSlot: "out", ToNode: "lower", ToSlot: "in", Config: map[string]string{"duration": "800"}}))
	must(g.Connect(&graph.Connection{ID: "c2", FromNode: "lower", FromSlot: "out", ToNode: "delay1", ToSlot: "in", Config: map[string]string{"duration": "600"}}))
	must(g.Connect(&graph.Connection{ID: "c3", FromNode: "delay1", FromSlot: "out", ToNode: "hex1", ToSlot: "in"}))                                               // instant
	must(g.Connect(&graph.Connection{ID: "c4", FromNode: "src1", FromSlot: "out", ToNode: "reverse", ToSlot: "in", Config: map[string]string{"duration": "1000"}}))
	must(g.Connect(&graph.Connection{ID: "c5", FromNode: "reverse", FromSlot: "out", ToNode: "print1", ToSlot: "in"}))                                              // instant
	must(g.Connect(&graph.Connection{ID: "c6", FromNode: "src2", FromSlot: "out", ToNode: "delay2", ToSlot: "in", Config: map[string]string{"duration": "800"}}))
	must(g.Connect(&graph.Connection{ID: "c7", FromNode: "delay2", FromSlot: "out", ToNode: "rcase", ToSlot: "in", Config: map[string]string{"duration": "1200"}}))
	must(g.Connect(&graph.Connection{ID: "c7b", FromNode: "rcase", FromSlot: "out", ToNode: "reverse2", ToSlot: "in", Config: map[string]string{"duration": "600"}}))
	must(g.Connect(&graph.Connection{ID: "c8", FromNode: "reverse2", FromSlot: "out", ToNode: "hex2", ToSlot: "in"}))                                                 // instant

	// Row 4: oscillator -> switch -> show (on path) / show (off path)
	must(g.AddNode(&graph.Node{ID: "osc1", Type: "oscillator", Label: "Oscillator", Position: graph.Position{X: 80, Y: 580}, Config: map[string]string{"period": "3000"}}))
	must(g.AddNode(&graph.Node{ID: "sw1", Type: "switch", Label: "Switch", Position: graph.Position{X: 400, Y: 580}}))
	must(g.AddNode(&graph.Node{ID: "show_on", Type: "show", Label: "Show (On)", Position: graph.Position{X: 700, Y: 520}}))
	must(g.AddNode(&graph.Node{ID: "show_off", Type: "show", Label: "Show (Off)", Position: graph.Position{X: 700, Y: 650}}))

	must(g.Connect(&graph.Connection{ID: "c9", FromNode: "osc1", FromSlot: "out", ToNode: "sw1", ToSlot: "in", Config: map[string]string{"duration": "600"}}))
	must(g.Connect(&graph.Connection{ID: "c10", FromNode: "sw1", FromSlot: "on", ToNode: "show_on", ToSlot: "in"}))
	must(g.Connect(&graph.Connection{ID: "c11", FromNode: "sw1", FromSlot: "off", ToNode: "show_off", ToSlot: "in"}))

	// Row 5: two oscillators -> AND gate -> show
	must(g.AddNode(&graph.Node{ID: "osc2", Type: "oscillator", Label: "Osc A", Position: graph.Position{X: 80, Y: 780}, Config: map[string]string{"period": "4000"}}))
	must(g.AddNode(&graph.Node{ID: "osc3", Type: "oscillator", Label: "Osc B", Position: graph.Position{X: 80, Y: 900}, Config: map[string]string{"period": "6000"}}))
	must(g.AddNode(&graph.Node{ID: "and1", Type: "and", Label: "AND", Position: graph.Position{X: 400, Y: 840}}))
	must(g.AddNode(&graph.Node{ID: "show_and", Type: "show", Label: "AND Result", Position: graph.Position{X: 700, Y: 840}}))

	must(g.Connect(&graph.Connection{ID: "c12", FromNode: "osc2", FromSlot: "out", ToNode: "and1", ToSlot: "a", Config: map[string]string{"duration": "500"}}))
	must(g.Connect(&graph.Connection{ID: "c13", FromNode: "osc3", FromSlot: "out", ToNode: "and1", ToSlot: "b", Config: map[string]string{"duration": "500"}}))
	must(g.Connect(&graph.Connection{ID: "c14", FromNode: "and1", FromSlot: "out", ToNode: "show_and", ToSlot: "in"}))

	// Row 6: toggle -> gate -> show (interactive demo)
	must(g.AddNode(&graph.Node{ID: "tog1", Type: "toggle", Label: "Toggle", Position: graph.Position{X: 80, Y: 1020}, Config: map[string]string{"state": "on"}}))
	must(g.AddNode(&graph.Node{ID: "gate1", Type: "gate", Label: "Gate", Position: graph.Position{X: 400, Y: 1020}, Config: map[string]string{"state": "on"}}))
	must(g.AddNode(&graph.Node{ID: "show_gate", Type: "show", Label: "Show", Position: graph.Position{X: 700, Y: 1020}}))

	must(g.Connect(&graph.Connection{ID: "c15", FromNode: "tog1", FromSlot: "out", ToNode: "gate1", ToSlot: "in", Config: map[string]string{"duration": "600"}}))
	must(g.Connect(&graph.Connection{ID: "c16", FromNode: "gate1", FromSlot: "out", ToNode: "show_gate", ToSlot: "in"}))

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
		engine.WithWireInterval(200*time.Millisecond),
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

	// Start the engine — executes every 5 seconds in the background.
	eng.Start(context.Background(), 5*time.Second)
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
