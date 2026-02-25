package eventgraph

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComputeHash(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	content, _ := json.Marshal(map[string]any{"key": "value"})

	h1 := computeHash("", "id1", "test.event", "source1", "", now, content)
	h2 := computeHash("", "id1", "test.event", "source1", "", now, content)
	if h1 != h2 {
		t.Fatalf("same inputs should produce same hash: %s != %s", h1, h2)
	}

	h3 := computeHash("", "id2", "test.event", "source1", "", now, content)
	if h1 == h3 {
		t.Fatalf("different ID should produce different hash")
	}

	h4 := computeHash("prevhash", "id1", "test.event", "source1", "", now, content)
	if h1 == h4 {
		t.Fatalf("different prevHash should produce different hash")
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// JSON marshal of map sorts keys deterministically
	content1, _ := json.Marshal(map[string]any{"a": 1, "b": 2})
	content2, _ := json.Marshal(map[string]any{"b": 2, "a": 1})

	h1 := computeHash("", "id", "type", "src", "conv", now, content1)
	h2 := computeHash("", "id", "type", "src", "conv", now, content2)
	if h1 != h2 {
		t.Fatalf("json.Marshal sorts keys, so hashes should match: %s != %s", h1, h2)
	}
}
