package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
)

func TestStore_PersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "vectorstore.json")

	s1, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	ctx := context.Background()
	if err := s1.Add(ctx, domain.Document{Text: "alpha", Embedding: []float64{1, 0, 0}, Source: "a.pdf"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s1.Add(ctx, domain.Document{Text: "beta", Embedding: []float64{0, 1, 0}, Source: "a.pdf"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if s1.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", s1.Count())
	}
	if found, err := s1.HasSource(ctx, "a.pdf"); err != nil || !found {
		t.Fatalf("HasSource(a.pdf) = %v, %v; want true, nil", found, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted file to exist: %v", err)
	}

	// Simulate a restart: new Store instance pointed at the same file.
	s2, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore (reload): %v", err)
	}
	if s2.Count() != 2 {
		t.Fatalf("after reload Count() = %d, want 2", s2.Count())
	}
	if found, err := s2.HasSource(ctx, "missing.pdf"); err != nil || found {
		t.Fatalf("HasSource(missing.pdf) = %v, %v; want false, nil", found, err)
	}

	results, err := s2.Search(ctx, []float64{1, 0, 0}, 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Text != "alpha" {
		t.Fatalf("expected top match 'alpha', got %+v", results)
	}
	if results[0].Score < 0.99 {
		t.Fatalf("expected near-perfect cosine match, got score=%.3f", results[0].Score)
	}
}

func TestStore_NoPersistenceWhenPathEmpty(t *testing.T) {
	s, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Add(context.Background(), domain.Document{Text: "x", Embedding: []float64{1}}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if s.Count() != 1 {
		t.Fatalf("Count() = %d, want 1", s.Count())
	}
}
