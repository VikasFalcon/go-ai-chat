package domain

// Document is a single ingested chunk of text (e.g. one slice of a PDF page)
// along with its embedding vector.
type Document struct {
	ID         int
	Text       string
	Embedding  []float64
	Source     string // e.g. "handbook.pdf" or "seed"
	ChunkIndex int    // position of this chunk within its source document
}

// ScoredDocument pairs a Document with its cosine-similarity score against
// a particular query embedding. Used by search results so callers can decide
// whether a match is actually relevant, not just "the best of what's there".
type ScoredDocument struct {
	Document
	Score float64
}
