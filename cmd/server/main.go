package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpadapter "github.com/VikasFalcon/go-ai-chat/internal/adapter/inbound/http"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/memory"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/ollama"
	"github.com/VikasFalcon/go-ai-chat/internal/adapter/outbound/prompt"
	"github.com/VikasFalcon/go-ai-chat/internal/config"
	"github.com/VikasFalcon/go-ai-chat/internal/core/service"
)

var seedDocuments = []string{
	"Kubernetes manages containers.",
	"Redis is an in-memory database.",
	"Ethereum is a blockchain.",
	"Docker creates containers.",
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.ChatModel, cfg.EmbedModel, cfg.OllamaTimeout)
	store := memory.NewStore()
	promptBuilder, err := prompt.NewBuilder()
	if err != nil {
		logger.Error("failed to load prompt template", "error", err)
		os.Exit(1)
	}

	ragService := service.NewRAGService(ollamaClient, store, ollamaClient, promptBuilder, cfg.RAGTopK)

	seedCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	for _, text := range seedDocuments {
		if err := ragService.Ingest(seedCtx, text); err != nil {
			logger.Warn("failed to seed document", "text", text, "error", err)
		}
	}
	cancel()

	handler := httpadapter.NewHandler(ragService, logger)
	router := httpadapter.NewRouter(handler)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	go func() {
		logger.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")
	ctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(ctx)
}
