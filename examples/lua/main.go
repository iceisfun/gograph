// Command lua demonstrates GoGraph with Lua-scripted node execution.
// Nodes have attached Lua scripts that process inputs and produce outputs.
// When executed, events animate along connections in the frontend.
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

	// Register node types with Lua scripts.
	reg := graph.NewRegistry()
	must(reg.Register(graph.NodeType{
		Name:  "source",
		Label: "Source",
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: `return { out = "hello world" }`,
	}))
	must(reg.Register(graph.NodeType{
		Name:  "upper",
		Label: "Uppercase",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "string"},
		},
		Script: mustReadFile("scripts/example.lua"),
	}))
	must(reg.Register(graph.NodeType{
		Name:  "sink",
		Label: "Sink",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "string"},
		},
		Script: `print("received: " .. tostring(inputs["in"])) return {}`,
	}))

	// Create a graph.
	g := graph.NewGraph("lua-demo")
	must(g.AddNode(&graph.Node{ID: "src", Type: "source", Label: "Hello", Position: graph.Position{X: 100, Y: 200}}))
	must(g.AddNode(&graph.Node{ID: "up", Type: "upper", Label: "Uppercase", Position: graph.Position{X: 400, Y: 200}}))
	must(g.AddNode(&graph.Node{ID: "dst", Type: "sink", Label: "Print", Position: graph.Position{X: 700, Y: 200}}))
	must(g.Connect(&graph.Connection{ID: "c1", FromNode: "src", FromSlot: "out", ToNode: "up", ToSlot: "in"}))
	must(g.Connect(&graph.Connection{ID: "c2", FromNode: "up", FromSlot: "out", ToNode: "dst", ToSlot: "in"}))

	// Persist the graph.
	st := store.NewMemoryStore()
	must(st.Save(context.Background(), g.ID, g))

	// Set up the engine with Lua executor.
	luaExec := golua.New(reg)
	eng := engine.New(g,
		engine.WithRegistry(reg),
		engine.WithExecutor(luaExec),
		engine.WithEventDuration(1500),
	)

	// Configure the server.
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

	// Execute the graph periodically to demonstrate animations.
	go func() {
		time.Sleep(2 * time.Second) // Wait for server startup.
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := eng.Execute(ctx); err != nil {
				log.Printf("execution error: %v", err)
			}
			cancel()
			time.Sleep(5 * time.Second)
		}
	}()

	fmt.Printf("GoGraph (Lua): http://127.0.0.1%s\n", *addr)
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
