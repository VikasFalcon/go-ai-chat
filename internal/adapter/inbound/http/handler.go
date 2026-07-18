package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/VikasFalcon/go-ai-chat/internal/core/domain"
	"github.com/VikasFalcon/go-ai-chat/internal/core/port"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	chat port.ChatService
	log  *slog.Logger
}

func NewHandler(chat port.ChatService, log *slog.Logger) *Handler {
	return &Handler{chat: chat, log: log}
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

func (h *Handler) respondError(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, domain.ErrEmptyInput):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrNoRelevantDocuments):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		h.log.Error("request failed", "op", op, "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal error, please try again"})
	}
}
