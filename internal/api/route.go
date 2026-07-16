package api

import "github.com/gin-gonic/gin"

func SetupRoutes(router *gin.Engine) {
	v1 := router.Group("/api")
	{
		v1.GET("/health", HealthHandler)
		v1.POST("/chat", ChatHandler)
		v1.POST("/embed", EmbedHandler)
	}
}
