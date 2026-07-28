package port

import "context"

type ChatService interface {
	// Chat is a topic-gated RAG query: it embeds the question, searches the
	// vector store, and either answers from the retrieved PDF context or
	// returns domain.OffTopicAnswer when nothing relevant enough was found.
	Chat(ctx context.Context, question string) (string, error)
	// Ask is an alias of Chat kept for backward compatibility with existing callers.
	Ask(ctx context.Context, question string) (string, error)
	// Ingest embeds and stores a single raw text chunk (used for seed data / tests).
	Ingest(ctx context.Context, text string) error
	// IngestPDF extracts, chunks, embeds, and stores an entire PDF file. It
	// returns the number of chunks that were ingested.
	IngestPDF(ctx context.Context, path string) (int, error)
	// DocumentCount reports how many chunks currently live in the vector store.
	DocumentCount() int
}
