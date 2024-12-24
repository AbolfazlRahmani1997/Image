package main

import (
	"image_extracting/internal/services"
	"log"

	"github.com/gin-gonic/gin"
	"image_extracting/internal/handelers"
	"image_extracting/internal/initialize"
)

func main() {

	initialize.LoadEnv()

	go services.ConsumeResponses()
	initialize.InitDB()
	initialize.Migrate()

	r := gin.Default()

	r.POST("/upload", handlers.UploadHandler)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
