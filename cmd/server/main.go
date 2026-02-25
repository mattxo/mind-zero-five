package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"mind-zero-five/internal/api"
	"mind-zero-five/internal/db"
	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

func main() {
	ctx := context.Background()

	pool, err := db.Connect(ctx)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	events := eventgraph.NewPgStore(pool)
	tasks := task.NewPgStore(pool)
	auth := authority.NewPgStore(pool)

	// Ensure tables exist
	if err := events.EnsureTable(ctx); err != nil {
		log.Fatalf("ensure events table: %v", err)
	}
	if err := tasks.EnsureTable(ctx); err != nil {
		log.Fatalf("ensure tasks table: %v", err)
	}
	if err := auth.EnsureTable(ctx); err != nil {
		log.Fatalf("ensure authority table: %v", err)
	}

	server := api.New(events, tasks, auth)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("mind-zero-five listening on :%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
