package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
	"github.com/VikasFalcon/go-ai-chat/internal/core/port"
)

const defaultTopK = 3

type RAGService struct {
	embedder  port.Embedder
	repo      port.DocumentRepository
	generator port.Generator
	prompts   port.PromptBuilder
	topK      int
}

func NewRAGService(embedder port.Embedder, repo port.DocumentRepository, generator port.Generator, prompts port.PromptBuilder, topK int) *RAGService {
	if topK <= 0 {
		topK = defaultTopK
	}
	return &RAGService{embedder: embedder, repo: repo, generator: generator, prompts: prompts, topK: topK}
}

func (s *RAGService) Chat(ctx context.Context, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", domain.ErrEmptyInput
	}
	return s.generator.Generate(ctx, prompt)
}

func (s *RAGService) Ask(ctx context.Context, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", domain.ErrEmptyInput
	}

	queryEmbedding, err := s.embedder.Embed(ctx, question)
	if err != nil {
		return "", fmt.Errorf("embed question: %w", err)
	}
	if len(queryEmbedding) == 0 {
		return "", domain.ErrEmptyEmbedding
	}

	docs, err := s.repo.Search(ctx, queryEmbedding, s.topK) // <- uses the injected repo, not a new Store{}
	if err != nil {
		return "", fmt.Errorf("search documents: %w", err)
	}
	if len(docs) == 0 {
		return "", domain.ErrNoRelevantDocuments
	}

	prompt, err := s.prompts.Build(joinChunks(docs), question)
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

	return s.generator.Generate(ctx, prompt) // err checked by caller pattern below if you prefer
}

func (s *RAGService) Ingest(ctx context.Context, text string) error {
	if strings.TrimSpace(text) == "" {
		return domain.ErrEmptyInput
	}
	embedding, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed document: %w", err)
	}
	return s.repo.Add(ctx, domain.Document{Text: text, Embedding: embedding})
}

func joinChunks(docs []domain.Document) string {
	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.Text
	}
	return strings.Join(texts, "\n---\n")
}
