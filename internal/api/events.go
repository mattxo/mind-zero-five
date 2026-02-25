package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleEventList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := queryInt(r, "limit", 50)

	if t := r.URL.Query().Get("type"); t != "" {
		events, err := s.events.ByType(ctx, t, limit)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, events)
		return
	}
	if src := r.URL.Query().Get("source"); src != "" {
		events, err := s.events.BySource(ctx, src, limit)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, events)
		return
	}
	if conv := r.URL.Query().Get("conversation"); conv != "" {
		events, err := s.events.ByConversation(ctx, conv, limit)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, events)
		return
	}

	events, err := s.events.Recent(ctx, limit)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleEventCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type           string         `json:"type"`
		Source         string         `json:"source"`
		Content        map[string]any `json:"content"`
		Causes         []string       `json:"causes"`
		ConversationID string         `json:"conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return
	}
	if req.Type == "" {
		writeError(w, 400, "type is required")
		return
	}
	if req.Source == "" {
		req.Source = "api"
	}
	e, err := s.events.Append(r.Context(), req.Type, req.Source, req.Content, req.Causes, req.ConversationID)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, e)
}

func (s *Server) handleEventGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := s.events.Get(r.Context(), id)
	if err != nil {
		writeError(w, 404, err.Error())
		return
	}
	writeJSON(w, 200, e)
}

func (s *Server) handleEventAncestors(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	depth := queryInt(r, "depth", 10)
	events, err := s.events.Ancestors(r.Context(), id, depth)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleEventDescendants(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	depth := queryInt(r, "depth", 10)
	events, err := s.events.Descendants(r.Context(), id, depth)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) handleEventTypes(w http.ResponseWriter, r *http.Request) {
	types, err := s.events.DistinctTypes(r.Context())
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, types)
}

func (s *Server) handleEventSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.events.DistinctSources(r.Context())
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, sources)
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher.Flush()

	ctx := r.Context()
	lastID := r.URL.Query().Get("after")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var events []any
			if lastID != "" {
				evts, err := s.events.Since(ctx, lastID, 50)
				if err != nil {
					log.Printf("SSE poll: %v", err)
					continue
				}
				for i := range evts {
					events = append(events, evts[i])
					lastID = evts[i].ID
				}
			} else {
				evts, err := s.events.Recent(ctx, 1)
				if err != nil {
					log.Printf("SSE poll: %v", err)
					continue
				}
				if len(evts) > 0 {
					lastID = evts[0].ID
				}
			}
			for _, e := range events {
				fmt.Fprintf(w, "data: ")
				writeJSONRaw(w, e)
				fmt.Fprintf(w, "\n\n")
				flusher.Flush()
			}
		}
	}
}

func writeJSONRaw(w http.ResponseWriter, v any) {
	enc := json.NewEncoder(w)
	enc.Encode(v)
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
