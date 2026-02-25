package api

import (
	"encoding/json"
	"net/http"

	"mind-zero-five/pkg/task"
)

func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := queryInt(r, "limit", 50)
	tasks, err := s.tasks.List(r.Context(), status, limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, tasks)
}

func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.tasks.Get(r.Context(), id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, t)
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	var t task.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	if t.Subject == "" {
		writeError(w, 400, "subject is required")
		return
	}
	if t.Source == "" {
		t.Source = "ui"
	}
	result, err := s.tasks.Create(r.Context(), &t)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, result)
}

func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	t, err := s.tasks.Update(r.Context(), id, updates)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, t)
}
