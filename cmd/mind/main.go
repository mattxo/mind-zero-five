package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mind-zero-five/internal/db"
	"mind-zero-five/pkg/actor"
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
	actors := actor.NewPgStore(pool)

	// Wait for tables to be ready (web server creates them).
	// Retry for up to 30 seconds on startup.
	var mindActor *actor.Actor
	for i := 0; i < 30; i++ {
		mindActor, err = actors.Register(ctx, "mind", "mind", "")
		if err == nil {
			break
		}
		log.Printf("mind: waiting for tables (attempt %d/30): %v", i+1, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatalf("register mind actor: %v", err)
	}
	log.Printf("mind actor: %s", mindActor.ID)

	repoDir := os.Getenv("MIND_REPO_DIR")
	if repoDir == "" {
		repoDir = "/data/source"
	}

	bus := eventgraph.NewBus(events)
	m := mind.New(bus, tasks, auth, mindActor.ID, repoDir)

	// Signal handling
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		log.Printf("mind: received %s, shutting down", sig)
		cancel()
	}()

	log.Println("mind: process started")
	m.Run(ctx)
}
