package api

import (
	"log"
	"net/http"

	"github.com/VikasFalcon/go-ai-chat/internal/embed"
	"github.com/VikasFalcon/go-ai-chat/internal/llm"
	"github.com/VikasFalcon/go-ai-chat/internal/model"
	"github.com/gin-gonic/gin"
)

func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func ChatHandler(c *gin.Context) {
	var req model.ChatReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	log.Printf("Request prompt: %s", req.Prompt)
	resp, err := llm.Generate(req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func EmbedHandler(c *gin.Context) {
	var req model.EmbeddingReq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	log.Printf("Request prompt: %s", req.Text)
	resp, err := embed.Generate(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)

}
