package api

import (
	"net/http"
)

func (s *Server) handleAuthorityList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.URL.Query().Get("status") == "pending" {
		reqs, err := s.auth.Pending(ctx)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, reqs)
		return
	}
	limit := queryInt(r, "limit", 50)
	reqs, err := s.auth.Recent(ctx, limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, reqs)
}

func (s *Server) handleAuthorityGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	req, err := s.auth.Get(r.Context(), id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, req)
}

func (s *Server) handleAuthorityResolve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	approved := r.URL.Query().Get("approved") == "true"
	req, err := s.auth.Resolve(r.Context(), id, approved)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	// Emit authority.resolved event so the mind can react
	s.events.Append(r.Context(), "authority.resolved", "api", map[string]any{
		"authority_id": req.ID,
		"action":       req.Action,
		"approved":     approved,
	}, nil, "")
	writeJSON(w, 200, req)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventCount, _ := s.events.Count(ctx)
	taskCount, _ := s.tasks.Count(ctx)
	pendingTasks, _ := s.tasks.PendingCount(ctx)
	pendingAuth, _ := s.auth.PendingCount(ctx)

	writeJSON(w, 200, map[string]any{
		"events":            eventCount,
		"tasks":             taskCount,
		"pending_tasks":     pendingTasks,
		"pending_approvals": pendingAuth,
	})
}
