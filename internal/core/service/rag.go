package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
	"github.com/VikasFalcon/go-ai-chat/internal/core/port"
)

const (
	defaultTopK             = 3
	defaultSimilarityThresh = 0.5
	defaultChunkSize        = 800
	defaultChunkOverlap     = 150
)

type RAGService struct {
	embedder  port.Embedder
	repo      port.DocumentRepository
	generator port.Generator
	prompts   port.PromptBuilder
	loader    port.DocumentLoader

	topK         int
	simThreshold float64
	chunkSize    int
	chunkOverlap int
	ingestMu     sync.Mutex
}

// NewRAGService wires up the RAG pipeline. loader may be nil if PDF ingestion
// is not needed (e.g. in tests that only exercise plain-text Ingest).
func NewRAGService(
	embedder port.Embedder,
	repo port.DocumentRepository,
	generator port.Generator,
	prompts port.PromptBuilder,
	loader port.DocumentLoader,
	topK int,
	similarityThreshold float64,
	chunkSize int,
	chunkOverlap int,
) *RAGService {
	if topK <= 0 {
		topK = defaultTopK
	}
	if similarityThreshold <= 0 {
		similarityThreshold = defaultSimilarityThresh
	}
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	if chunkOverlap < 0 {
		chunkOverlap = defaultChunkOverlap
	}
	return &RAGService{
		embedder:     embedder,
		repo:         repo,
		generator:    generator,
		prompts:      prompts,
		loader:       loader,
		topK:         topK,
		simThreshold: similarityThreshold,
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

// Chat performs a topic-gated RAG query: it embeds the question, retrieves
// the closest chunks from the vector store, and only calls the LLM if the
// best match clears simThreshold. Otherwise it returns domain.OffTopicAnswer
// as a normal (nil-error) answer, per the "not related to topic" requirement.
func (s *RAGService) Chat(ctx context.Context, question string) (string, error) {
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

	scored, err := s.repo.Search(ctx, queryEmbedding, s.topK)
	if err != nil {
		return "", fmt.Errorf("search documents: %w", err)
	}
	if len(scored) == 0 || scored[0].Score < s.simThreshold {
		return domain.OffTopicAnswer, nil
	}

	prompt, err := s.prompts.Build(joinChunks(scored), question)
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

	answer, err := s.generator.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generate answer: %w", err)
	}
	return answer, nil
}

// Ask is kept as an alias so any existing caller of the previous Ask method
// keeps working unchanged; it's the same topic-gated RAG flow as Chat.
func (s *RAGService) Ask(ctx context.Context, question string) (string, error) {
	return s.Chat(ctx, question)
}

// Ingest embeds and stores a single raw text chunk, tagging it as "manual".
func (s *RAGService) Ingest(ctx context.Context, text string) error {
	return s.ingestChunk(ctx, text, "manual", 0)
}

// IngestPDF extracts text from the PDF at path, splits it into overlapping
// chunks, embeds each chunk, and stores it in the vector store. It returns
// how many chunks were ingested.
func (s *RAGService) IngestPDF(ctx context.Context, path string) (int, error) {
	if s.loader == nil {
		return 0, fmt.Errorf("no document loader configured")
	}
	if !strings.EqualFold(filepath.Ext(path), ".pdf") {
		return 0, domain.ErrUnsupportedFileType
	}

	// Serialize ingestion so simultaneous requests for the same source cannot
	// both pass the duplicate check and append duplicate chunks.
	s.ingestMu.Lock()
	defer s.ingestMu.Unlock()

	source := filepath.Base(path)
	alreadyIndexed, err := s.repo.HasSource(ctx, source)
	if err != nil {
		return 0, fmt.Errorf("check indexed source %s: %w", source, err)
	}
	if alreadyIndexed {
		return 0, nil
	}

	text, err := s.loader.Load(path)
	if err != nil {
		return 0, fmt.Errorf("load pdf %s: %w", path, err)
	}
	if strings.TrimSpace(text) == "" {
		return 0, domain.ErrEmptyDocument
	}

	chunks := chunkText(text, s.chunkSize, s.chunkOverlap)
	if len(chunks) == 0 {
		return 0, domain.ErrEmptyDocument
	}

	for i, chunk := range chunks {
		if err := s.ingestChunk(ctx, chunk, source, i); err != nil {
			return i, fmt.Errorf("ingest chunk %d of %s: %w", i, source, err)
		}
	}
	return len(chunks), nil
}

// DocumentCount reports how many chunks currently live in the vector store.
func (s *RAGService) DocumentCount() int {
	return s.repo.Count()
}

func (s *RAGService) ingestChunk(ctx context.Context, text, source string, index int) error {
	if strings.TrimSpace(text) == "" {
		return domain.ErrEmptyInput
	}
	embedding, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed document: %w", err)
	}
	return s.repo.Add(ctx, domain.Document{
		Text:       text,
		Embedding:  embedding,
		Source:     source,
		ChunkIndex: index,
	})
}

func joinChunks(scored []domain.ScoredDocument) string {
	texts := make([]string, len(scored))
	for i, d := range scored {
		texts[i] = d.Text
	}
	return strings.Join(texts, "\n---\n")
}
