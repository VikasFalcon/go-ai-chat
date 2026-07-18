package memory

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
)

type Store struct {
	mu     sync.RWMutex
	docs   []domain.Document
	nextID int
}

func NewStore() *Store { return &Store{} }

func (s *Store) Add(_ context.Context, doc domain.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	doc.ID = s.nextID
	s.docs = append(s.docs, doc)
	return nil
}

func (s *Store) Search(_ context.Context, query []float64, topK int) ([]domain.Document, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("query embedding must not be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scoredDoc struct {
		doc   domain.Document
		score float64
	}
	scored := make([]scoredDoc, 0, len(s.docs))
	for _, d := range s.docs {
		score, err := cosineSimilarity(query, d.Embedding)
		if err != nil {
			continue // skip a bad row instead of failing the whole search
		}
		scored = append(scored, scoredDoc{doc: d, score: score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })

	if topK > len(scored) {
		topK = len(scored)
	}
	result := make([]domain.Document, topK)
	for i := 0; i < topK; i++ {
		result[i] = scored[i].doc
	}
	return result, nil
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
