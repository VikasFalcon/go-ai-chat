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
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		OllamaURL:     getEnv("OLLAMA_URL", "http://localhost:11434"),
		ChatModel:     getEnv("CHAT_MODEL", "qwen3:8b"),
		EmbedModel:    getEnv("EMBED_MODEL", "nomic-embed-text"),
		OllamaTimeout: getEnvDuration("OLLAMA_TIMEOUT", 60*time.Second),
		RAGTopK:       getEnvInt("RAG_TOP_K", 3),
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
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
