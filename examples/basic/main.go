// Command basic demonstrates a minimal GoGraph setup with a few nodes and
// connections. It serves the embedded frontend and provides a REST API for
// graph manipulation.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/iceisfun/gograph/frontend"
	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/server"
	"github.com/iceisfun/gograph/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dev := flag.Bool("dev", false, "serve frontend from disk (development mode)")
	flag.Parse()

	// Register node types.
	reg := graph.NewRegistry()
	must(reg.Register(graph.NodeType{
		Name:  "source",
		Label: "Source",
		Slots: []graph.Slot{
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
	}))
	must(reg.Register(graph.NodeType{
		Name:  "transform",
		Label: "Transform",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
	}))
	must(reg.Register(graph.NodeType{
		Name:  "sink",
		Label: "Sink",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
		},
	}))

	// Create a sample graph.
	g := graph.NewGraph("demo")
	must(g.AddNode(&graph.Node{ID: "n1", Type: "source", Label: "Data Source", Position: graph.Position{X: 100, Y: 200}}))
	must(g.AddNode(&graph.Node{ID: "n2", Type: "transform", Label: "Filter", Position: graph.Position{X: 400, Y: 150}}))
	must(g.AddNode(&graph.Node{ID: "n3", Type: "transform", Label: "Map", Position: graph.Position{X: 400, Y: 350}}))
	must(g.AddNode(&graph.Node{ID: "n4", Type: "sink", Label: "Output", Position: graph.Position{X: 700, Y: 250}}))
	must(g.Connect(graph.NewEventConnection("c1", "n1", "out", "n2", "in", nil)))
	must(g.Connect(graph.NewEventConnection("c2", "n1", "out", "n3", "in", nil)))
	must(g.Connect(graph.NewEventConnection("c3", "n2", "out", "n4", "in", nil)))

	// Persist the graph.
	st := store.NewMemoryStore()
	must(st.Save(context.Background(), g.ID, g))

	// Configure the server.
	opts := []server.Option{
		server.WithStore(st),
		server.WithRegistry(reg),
	}

	if *dev {
		opts = append(opts, server.WithStaticFS(os.DirFS("frontend/dist")))
		fmt.Printf("GoGraph (dev mode): http://127.0.0.1%s\n", *addr)
	} else {
		opts = append(opts, server.WithStaticFS(frontend.FS()))
		fmt.Printf("GoGraph: http://127.0.0.1%s\n", *addr)
	}

	srv := server.New(opts...)
	log.Fatal(srv.ListenAndServe(*addr))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
