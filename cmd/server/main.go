package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	httpadapter "github.com/VikasFalcon/go-ai-chat/internal/adapter/inbound/http"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/ollama"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/pdfloader"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/prompt"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/vectorstore"
	"github.com/VikasFalcon/go-ai-chat/internal/config"
	"github.com/VikasFalcon/go-ai-chat/internal/core/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.ChatModel, cfg.EmbedModel, cfg.OllamaTimeout)

	store, err := vectorstore.NewStore(cfg.VectorStorePath)
	if err != nil {
		logger.Error("failed to load vector store", "error", err)
		os.Exit(1)
	}
	logger.Info("vector store loaded", "path", cfg.VectorStorePath, "existing_chunks", store.Count())

	pdfLoader := pdfloader.NewLoader()

	promptBuilder, err := prompt.NewBuilder()
	if err != nil {
		logger.Error("failed to load prompt template", "error", err)
		os.Exit(1)
	}

	ragService := service.NewRAGService(
		ollamaClient, store, ollamaClient, promptBuilder, pdfLoader,
		cfg.RAGTopK, cfg.SimilarityThreshold, cfg.ChunkSize, cfg.ChunkOverlap,
	)

	// Auto-ingest every *.pdf found in cfg.PDFDir on startup. This is the
	// "load PDF into the server" step: drop files in that folder before
	// starting the server (or use POST /api/ingest afterwards).
	ingestSeedPDFs(ragService, cfg.PDFDir, logger)

	handler := httpadapter.NewHandler(ragService, logger, cfg.MaxUploadSizeMB, filepath.Join(os.TempDir(), "go-ai-chat-uploads"))
	router := httpadapter.NewRouter(handler)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		logger.Info("server starting", "port", cfg.Port, "document_chunks", ragService.DocumentCount())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func ingestSeedPDFs(rag *service.RAGService, dir string, logger *slog.Logger) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("could not read PDF_DIR", "dir", dir, "error", err)
		} else {
			logger.Info("PDF_DIR does not exist, skipping startup ingestion", "dir", dir)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		n, err := rag.IngestPDF(ctx, path)
		if err != nil {
			logger.Warn("failed to ingest seed PDF", "file", e.Name(), "error", err)
			continue
		}
		logger.Info("ingested seed PDF", "file", e.Name(), "chunks", n)
	}
}
