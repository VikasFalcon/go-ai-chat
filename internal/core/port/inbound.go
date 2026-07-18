package port

import "context"

type ChatService interface {
	Chat(ctx context.Context, prompt string) (string, error)
	Ask(ctx context.Context, question string) (string, error)
	Ingest(ctx context.Context, text string) error
}
