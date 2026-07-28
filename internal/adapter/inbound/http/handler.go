package http

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
	"github.com/VikasFalcon/go-ai-chat/internal/core/port"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	chat           port.ChatService
	log            *slog.Logger
	maxUploadBytes int64
	uploadTmpDir   string
}

func NewHandler(chat port.ChatService, log *slog.Logger, maxUploadSizeMB int64, uploadTmpDir string) *Handler {
	if maxUploadSizeMB <= 0 {
		maxUploadSizeMB = 20
	}
	if uploadTmpDir == "" {
		uploadTmpDir = os.TempDir()
	}
	return &Handler{chat: chat, log: log, maxUploadBytes: maxUploadSizeMB << 20, uploadTmpDir: uploadTmpDir}
}

func (h *Handler) Health(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	answer, err := h.chat.Chat(c.Request.Context(), req.Prompt)
	if err != nil {
		h.respondError(c, "chat", err)
		return
	}
	c.JSON(http.StatusOK, ChatResponse{Answer: answer})
}

func (h *Handler) Ask(c *gin.Context) {
	var req AskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	answer, err := h.chat.Ask(c.Request.Context(), req.Question)
	if err != nil {
		h.respondError(c, "ask", err)
		return
	}
	c.JSON(http.StatusOK, AskResponse{Answer: answer})
}

// Ingest handles POST /api/ingest — a multipart/form-data upload with the
// PDF under the "file" field. It extracts, chunks, embeds, and stores the
// document, then reports how many chunks were added.
func (h *Handler) Ingest(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "missing \"file\" field with a .pdf upload"})
		return
	}
	if !strings.EqualFold(filepath.Ext(fileHeader.Filename), ".pdf") {
		c.JSON(http.StatusBadRequest, errorResponse{Error: domain.ErrUnsupportedFileType.Error()})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		h.respondError(c, "ingest", fmt.Errorf("open upload: %w", err))
		return
	}
	defer src.Close()

	// PDF allows its header to appear within the first 1,024 bytes. Read that
	// prefix before saving so an arbitrary file renamed to .pdf is rejected.
	prefix, err := io.ReadAll(io.LimitReader(src, 1024))
	if err != nil {
		h.respondError(c, "ingest", fmt.Errorf("read upload header: %w", err))
		return
	}
	if !bytes.Contains(prefix, []byte("%PDF-")) {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "uploaded file is not a valid PDF"})
		return
	}

	if err := os.MkdirAll(h.uploadTmpDir, 0o700); err != nil {
		h.respondError(c, "ingest", fmt.Errorf("prepare upload dir: %w", err))
		return
	}
	requestDir, err := os.MkdirTemp(h.uploadTmpDir, "upload-")
	if err != nil {
		h.respondError(c, "ingest", fmt.Errorf("create upload dir: %w", err))
		return
	}
	defer os.RemoveAll(requestDir)

	// Never use an untrusted filename as a directory path. The unique temporary
	// directory prevents concurrent uploads with the same filename from colliding.
	tmpPath := filepath.Join(requestDir, filepath.Base(fileHeader.Filename))
	dst, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		h.respondError(c, "ingest", fmt.Errorf("create temp file: %w", err))
		return
	}
	if _, err := io.Copy(dst, io.MultiReader(bytes.NewReader(prefix), src)); err != nil {
		_ = dst.Close()
		h.respondError(c, "ingest", fmt.Errorf("save upload: %w", err))
		return
	}
	if err := dst.Close(); err != nil {
		h.respondError(c, "ingest", fmt.Errorf("finalize upload: %w", err))
		return
	}

	n, err := h.chat.IngestPDF(c.Request.Context(), tmpPath)
	if err != nil {
		h.respondError(c, "ingest", err)
		return
	}
	c.JSON(http.StatusOK, IngestResponse{Source: fileHeader.Filename, ChunksIngested: n})
}

// Stats handles GET /api/stats — a quick sanity check of how many chunks are
// currently searchable in the vector store.
func (h *Handler) Stats(c *gin.Context) {
	c.JSON(http.StatusOK, StatsResponse{DocumentChunks: h.chat.DocumentCount()})
}

func (h *Handler) respondError(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, domain.ErrEmptyInput):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrUnsupportedFileType):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrEmptyDocument):
		c.JSON(http.StatusUnprocessableEntity, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrNoRelevantDocuments):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		h.log.Error("request failed", "op", op, "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error, please try again"})
	}
}
