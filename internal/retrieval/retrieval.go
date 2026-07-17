package retrieval

import (
	"fmt"
	"math"
)

type Document struct {
	ID        int
	Text      string
	Embedding []float64
}

type Store struct {
	Documents []Document
}

func (s *Store) Add(doc Document) {
	s.Documents = append(s.Documents, doc)
}

func dotProduct(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var sum float64
	for i := range a {
		sum += a[i] * b[i]
	}

	return sum, nil
}

func magnitude(v []float64) float64 {
	var sum float64

	for _, x := range v {
		sum += x * x
	}

	return math.Sqrt(sum)
}

func cosineSimilarity(a, b []float64) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("empty vectors")
	}

	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors length must be same length")
	}

	dot, err := dotProduct(a, b)
	if err != nil {
		return 0, err
	}

	magnitudeA := magnitude(a)
	magnitudeB := magnitude(b)

	if magnitudeA == 0 || magnitudeB == 0 {
		return 0, fmt.Errorf("magnitude value got zero")
	}

	cosineValue := dot / (magnitudeA * magnitudeB)

	return cosineValue, nil
}

func (s *Store) Search(query []float64) (Document, error) {
	bestScore := -1.0
	var best Document

	for _, doc := range s.Documents {
		score, err := cosineSimilarity(query, doc.Embedding)
		if err != nil {
			return Document{}, err
		}

		if score > bestScore {
			bestScore = score
			best = doc
		}
	}

	return best, nil
}
