// Package vectorstore is a small embeddable vector database: it holds
// document chunks + their embeddings in memory for fast cosine-similarity
// search, and persists them to a single JSON file on disk so ingested PDFs
// survive process restarts. No external DB service is required.
package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
)

type Store struct {
	mu       sync.RWMutex
	docs     []domain.Document
	nextID   int
	filePath string // if non-empty, every mutation is flushed here
}

// NewStore creates an in-memory store. If filePath is non-empty, the store
// loads any existing data from that file on startup and persists to it after
// every Add call. Pass "" to disable persistence (pure in-memory).
func NewStore(filePath string) (*Store, error) {
	s := &Store{filePath: filePath}
	if filePath == "" {
		return s, nil
	}
	if err := s.load(); err != nil {
		return nil, fmt.Errorf("load vector store: %w", err)
	}
	return s, nil
}

func (s *Store) Add(_ context.Context, doc domain.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	doc.ID = s.nextID
	s.docs = append(s.docs, doc)
	return s.persistLocked()
}

// Count returns the number of chunks currently stored.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

// Search returns the topK documents ranked by cosine similarity to query,
// each paired with its similarity score so callers can gate on relevance.
func (s *Store) Search(_ context.Context, query []float64, topK int) ([]domain.ScoredDocument, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("query embedding must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	scored := make([]domain.ScoredDocument, 0, len(s.docs))
	for _, d := range s.docs {
		score, err := cosineSimilarity(query, d.Embedding)
		if err != nil {
			continue // skip a bad row instead of failing the whole search
		}
		scored = append(scored, domain.ScoredDocument{Document: d, Score: score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })

	if topK > len(scored) {
		topK = len(scored)
	}
	return scored[:topK], nil
}

func cosineSimilarity(a, b []float64) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("empty vector")
	}
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector length mismatch: %d vs %d", len(a), len(b))
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	magA, magB = math.Sqrt(magA), math.Sqrt(magB)
	if magA == 0 || magB == 0 {
		return 0, fmt.Errorf("zero-magnitude vector")
	}
	return dot / (magA * magB), nil
}

// --- persistence -----------------------------------------------------------

type onDiskFormat struct {
	NextID int               `json:"next_id"`
	Docs   []domain.Document `json:"docs"`
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil // nothing persisted yet
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	var payload onDiskFormat
	if err := json.Unmarshal(b, &payload); err != nil {
		return fmt.Errorf("parse %s: %w", s.filePath, err)
	}
	s.docs = payload.Docs
	s.nextID = payload.NextID
	return nil
}

// persistLocked writes the store to disk. Caller must hold s.mu.
// Uses write-temp-then-rename so a crash mid-write can't corrupt the file.
func (s *Store) persistLocked() error {
	if s.filePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return fmt.Errorf("create vector store dir: %w", err)
	}
	payload := onDiskFormat{NextID: s.nextID, Docs: s.docs}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal vector store: %w", err)
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return fmt.Errorf("write vector store temp file: %w", err)
	}
	return os.Rename(tmp, s.filePath)
}
