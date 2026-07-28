package domain

import "errors"

var (
	ErrEmptyInput          = errors.New("input must not be empty")
	ErrNoRelevantDocuments = errors.New("no relevant documents found")
	ErrEmptyEmbedding      = errors.New("embedding vector is empty")
	ErrUnsupportedFileType = errors.New("unsupported file type, only .pdf is accepted")
	ErrEmptyDocument       = errors.New("no extractable text found in document")
)

// OffTopicAnswer is returned as the literal answer text (not an error) when a
// question's best-matching chunk similarity falls below the configured
// threshold, i.e. the question isn't about the ingested PDF content.
const OffTopicAnswer = "Question is not related to topic"
