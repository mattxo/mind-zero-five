package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

// Server is the HTTP API server.
type Server struct {
	events eventgraph.EventStore
	tasks  task.Store
	auth   authority.Store
	mux    *http.ServeMux
}

// New creates a new Server.
func New(events eventgraph.EventStore, tasks task.Store, auth authority.Store) *Server {
	s := &Server{
		events: events,
		tasks:  tasks,
		auth:   auth,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Events
	s.mux.HandleFunc("GET /api/events", s.handleEventList)
	s.mux.HandleFunc("POST /api/events", s.handleEventCreate)
	s.mux.HandleFunc("GET /api/events/types", s.handleEventTypes)
	s.mux.HandleFunc("GET /api/events/sources", s.handleEventSources)
	s.mux.HandleFunc("GET /api/events/stream", s.handleEventStream)
	s.mux.HandleFunc("GET /api/events/{id}", s.handleEventGet)
	s.mux.HandleFunc("GET /api/events/{id}/ancestors", s.handleEventAncestors)
	s.mux.HandleFunc("GET /api/events/{id}/descendants", s.handleEventDescendants)

	// Tasks
	s.mux.HandleFunc("GET /api/tasks", s.handleTaskList)
	s.mux.HandleFunc("POST /api/tasks", s.handleTaskCreate)
	s.mux.HandleFunc("GET /api/tasks/{id}", s.handleTaskGet)
	s.mux.HandleFunc("PATCH /api/tasks/{id}", s.handleTaskUpdate)

	// Authority
	s.mux.HandleFunc("GET /api/authority", s.handleAuthorityList)
	s.mux.HandleFunc("GET /api/authority/{id}", s.handleAuthorityGet)
	s.mux.HandleFunc("POST /api/authority/{id}/resolve", s.handleAuthorityResolve)

	// System
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)

	// Static files (Gio WASM UI)
	wasmDir := os.Getenv("WASM_DIR")
	if wasmDir == "" {
		wasmDir = filepath.Join(".", "web")
	}
	s.mux.Handle("GET /", http.FileServer(http.Dir(wasmDir)))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
