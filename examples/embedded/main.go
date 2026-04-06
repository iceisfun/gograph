// Command embedded demonstrates mounting GoGraph as a sub-handler within
// a larger HTTP application using a route prefix.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/iceisfun/gograph/frontend"
	"github.com/iceisfun/gograph/graph"
	"github.com/iceisfun/gograph/server"
	"github.com/iceisfun/gograph/store"
)

func main() {
	// Register a simple node type.
	reg := graph.NewRegistry()
	reg.Register(graph.NodeType{
		Name:  "process",
		Label: "Process",
		Slots: []graph.Slot{
			{ID: "in", Name: "Input", Direction: graph.Input, DataType: "any"},
			{ID: "out", Name: "Output", Direction: graph.Output, DataType: "any"},
		},
	})

	// Create and persist a sample graph.
	st := store.NewMemoryStore()
	g := graph.NewGraph("workflow")
	g.AddNode(&graph.Node{ID: "a", Type: "process", Label: "Step A", Position: graph.Position{X: 150, Y: 200}})
	g.AddNode(&graph.Node{ID: "b", Type: "process", Label: "Step B", Position: graph.Position{X: 450, Y: 200}})
	g.Connect(graph.NewEventConnection("ab", "a", "out", "b", "in", nil))
	st.Save(context.Background(), g.ID, g)

	// Create the graph server mounted at /graph.
	graphServer := server.New(
		server.WithStaticFS(frontend.FS()),
		server.WithStore(st),
		server.WithRegistry(reg),
		server.WithRoutePrefix("/graph"),
	)

	// Mount alongside other routes in a larger app.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<h1>My Application</h1><p><a href="/graph/">Open Graph Editor</a></p>`)
	})
	mux.Handle("/graph/", graphServer.Handler())

	fmt.Println("Application: http://127.0.0.1:8080")
	fmt.Println("Graph editor: http://127.0.0.1:8080/graph/")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
