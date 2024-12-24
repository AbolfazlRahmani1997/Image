package handlers

import (
	"encoding/json"
	"fmt"
	"image_extracting/internal/initialize"
	"image_extracting/internal/models"
	"image_extracting/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func UploadHandler(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image uploaded"})
		return
	}

	// Save the uploaded file
	imagePath := fmt.Sprintf("./uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, imagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}
	keywords := []string{}
	keywordsJSON, err := json.Marshal(keywords)
	image := models.ImageMetadata{
		ImagePath: imagePath,
		Keywords:  string(keywordsJSON),
	}
	if err := initialize.DB.Create(&image).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save metadata"})
		return
	}

	if err := services.SendToPython(image.ID, imagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send image to service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image uploaded successfully", "image_id": image.ID})
}
