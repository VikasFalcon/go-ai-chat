package port

import (
	"context"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type DocumentRepository interface {
	Add(ctx context.Context, doc domain.Document) error
<<<<<<< HEAD
	Search(ctx context.Context, queryEmbedding []float64, topK int) ([]domain.Document, error)
=======
	Search(ctx context.Context, queryEmbedding []float64, topK int) ([]domain.ScoredDocument, error)
	Count() int
>>>>>>> b57e7fc (project updated and accept pdf, convert into chuncks, and answer users questions in sementic manner)
}

type PromptBuilder interface {
	Build(chunks, question string) (string, error)
}
<<<<<<< HEAD
=======

// DocumentLoader extracts raw text from a source file on disk (e.g. a PDF).
type DocumentLoader interface {
	Load(path string) (string, error)
}
>>>>>>> b57e7fc (project updated and accept pdf, convert into chuncks, and answer users questions in sementic manner)
