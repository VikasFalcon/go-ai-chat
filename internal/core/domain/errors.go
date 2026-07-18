package domain

import "errors"

var (
	ErrEmptyInput          = errors.New("inout must not be empty")
	ErrNoRelevantDocuments = errors.New("no relevant documents found")
	ErrEmptyEmbedding      = errors.New("embedding vector is empty")
)
