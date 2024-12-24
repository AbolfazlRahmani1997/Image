package ImageManager

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"log"
	"math/rand"
)

type Handler struct {
}

type Request struct {
	ImageID   string `json:"image_id"`
	ImagePath string `json:"image_path"`
}

type Response struct {
	ImageID  string   `json:"image_id"`
	Keywords []string `json:"keywords"`
}

type Image struct {
	Id        uint64 `json:"id"`
	ImagePath string `json:"image_path"`
	UserId    string `json:"user_id"`
	Status    string `json:"status"`
}

type Keyword struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type ImageKeyword struct {
	KeywordId string `json:"keyword_id"`
	ImageId   string `json:"image_id"`
}

type SearcherResult struct {
	ImageID string `json:"image_id"`
}

func NewHandler() *Handler {
	return &Handler{}
}

func (receiver Handler) Upload(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(400, gin.H{"error": "No image uploaded"})
		return
	}

	// Save the uploaded file
	imagePath := fmt.Sprintf("./uploads/%s", file.Filename)
	if err := c.SaveUploadedFile(file, imagePath); err != nil {
		c.JSON(500, gin.H{"error": "Failed to save image"})
		return
	}

	// Generate a unique ID for the image
	imageID := fmt.Sprintf("%d", rand.Int())

	// Send the request to RabbitMQ
	keywords, err := receiver.sendToPythonService(imageID, imagePath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"keywords": keywords})
}

func (receiver Handler) sendToPythonService(imageID, imagePath string) ([]string, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare queues
	if _, err := ch.QueueDeclare("request_queue", false, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %v", err)
	}
	if _, err := ch.QueueDeclare("response_queue", false, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %v", err)
	}

	// Publish the request
	request := Request{ImageID: imageID, ImagePath: imagePath}
	body, _ := json.Marshal(request)
	if err := ch.Publish("", "request_queue", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		return nil, fmt.Errorf("failed to publish a message: %v", err)
	}

	// Consume the response
	msgs, err := ch.Consume("response_queue", "", true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to consume from response queue: %v", err)
	}

	for msg := range msgs {
		var response Response
		if err := json.Unmarshal(msg.Body, &response); err != nil {
			log.Printf("failed to unmarshal response: %v", err)
			continue
		}

		// Match response to the image ID
		if response.ImageID == imageID {
			return response.Keywords, nil
		}
	}

	return nil, fmt.Errorf("no response received for image %s", imageID)
}
