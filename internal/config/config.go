package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	OllamaURL     string
	ChatModel     string
	EmbedModel    string
	OllamaTimeout time.Duration
	RAGTopK       int

	// SimilarityThreshold: the top search result's cosine similarity must be
	// >= this value for a question to be treated as "about the PDF". Lower it
	// if relevant questions are wrongly rejected; raise it if off-topic
	// questions are wrongly answered. Range is roughly 0.0-1.0.
	SimilarityThreshold float64

	ChunkSize    int // approx. characters per chunk before embedding
	ChunkOverlap int // characters of overlap between consecutive chunks

	PDFDir          string // directory auto-scanned for *.pdf on startup
	VectorStorePath string // JSON file the vector store persists to; "" disables persistence
	MaxUploadSizeMB int64  // cap for /api/ingest multipart uploads
}

func Load() Config {
	return Config{
		Port:                getEnv("PORT", "8080"),
		OllamaURL:           getEnv("OLLAMA_URL", "http://localhost:11434"),
		ChatModel:           getEnv("CHAT_MODEL", "qwen3:8b"),
		EmbedModel:          getEnv("EMBED_MODEL", "nomic-embed-text"),
		OllamaTimeout:       getEnvDuration("OLLAMA_TIMEOUT", 60*time.Second),
		RAGTopK:             getEnvInt("RAG_TOP_K", 3),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.5),
		ChunkSize:           getEnvInt("CHUNK_SIZE", 800),
		ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 150),
		PDFDir:              getEnv("PDF_DIR", "./data/pdfs"),
		VectorStorePath:     getEnv("VECTOR_STORE_PATH", "./data/vectorstore.json"),
		MaxUploadSizeMB:     int64(getEnvInt("MAX_UPLOAD_SIZE_MB", 20)),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
