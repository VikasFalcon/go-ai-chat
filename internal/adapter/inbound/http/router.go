package http

import "github.com/gin-gonic/gin"

func NewRouter(h *Handler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	v1 := r.Group("/api")
	{
		v1.GET("/health", h.Health)
		v1.POST("/chat", h.Chat)
		v1.POST("/ask", h.Ask)
		v1.POST("/ingest", h.Ingest)
		v1.GET("/stats", h.Stats)
	}
	return r
}
