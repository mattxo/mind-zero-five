package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"mind-zero-five/internal/api"
	"mind-zero-five/internal/db"
	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/mind"
	"mind-zero-five/pkg/task"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Wrap EventStore in Bus for in-process event subscription
	bus := eventgraph.NewBus(events)

	// API server uses Bus (satisfies EventStore interface) so events flow through it
	server := api.New(bus, tasks, auth)

	// Start mind if enabled
	if os.Getenv("MIND_ENABLED") == "true" {
		repoDir := os.Getenv("MIND_REPO_DIR")
		if repoDir == "" {
			repoDir = "/usr/local/share/mz5-source"
		}
		m := mind.New(bus, tasks, auth, repoDir)
		go m.Run(ctx)
		log.Println("mind: enabled")
	}

	// Signal handling for graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		log.Printf("received %s, shutting down", sig)
		cancel()
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("mind-zero-five listening on :%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
