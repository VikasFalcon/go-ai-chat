package main

import (
	"log"

	"github.com/VikasFalcon/go-ai-chat/internal/api"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	api.SetupRoutes(r)

	log.Println("Server running on: 8080")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
