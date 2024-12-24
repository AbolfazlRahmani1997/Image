package services

import (
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
	"gopkg.in/resty.v1"
	"image_extracting/internal/initialize"
	"image_extracting/internal/models"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type ImageRequest struct {
	ImageID   uint   `json:"image_id"`
	ImagePath string `json:"image_path"`
}

type ImageResponse struct {
	ImageID  uint     `json:"image_id"`
	Keywords []string `json:"keywords" type:""`
}

func ConnectRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:5672/", os.Getenv("RABBITMQ_USER"), os.Getenv("RABBITMQ_PASSWORD"), os.Getenv("RABBITMQ_HOST")))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open RabbitMQ channel: %v", err)
	}

	return conn, ch, nil
}

func SendToPython(imageID uint, imagePath string) error {
	conn, ch, err := ConnectRabbitMQ()
	if err != nil {
		return err
	}
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {

		}
	}(ch)

	queue, err := ch.QueueDeclare(
		"request_queue", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	request := ImageRequest{ImageID: imageID, ImagePath: imagePath}
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	if err := ch.Publish("", queue.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	log.Printf("Image sent to Python service: %v", request)
	return nil
}

func ConsumeResponses() {
	conn, ch, err := ConnectRabbitMQ()
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	queue, err := ch.QueueDeclare(
		"response_queue", false, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to consume messages: %v", err)
	}

	log.Println("Listening for responses...")

	for msg := range msgs {
		var response ImageResponse
		if err := json.Unmarshal(msg.Body, &response); err != nil {
			log.Printf("Failed to unmarshal response: %v", err)
			continue
		}

		var image models.ImageMetadata
		if err := initialize.DB.First(&image, response.ImageID).Error; err != nil {
			log.Printf("Image not found: %v", err)
			continue
		}
		var query string
		for _, key := range response.Keywords {
			query += key + "+"
		}
		go func() {
			err := searchAndDownloadImages(query, response.ImageID)
			if err != nil {

			}
		}()
		keywordsJSON, _ := json.Marshal(response.Keywords)

		image.Keywords = string(keywordsJSON)
		if err := initialize.DB.Save(&image).Error; err != nil {
			log.Printf("Failed to update image: %v", err)
		} else {
			log.Printf("Image updated: %v", response)
		}
	}

}
func searchAndDownloadImages(query string, imageId uint) error {
	url := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?q=%s&searchType=image&key=%s&cx=%s",
		query, os.Getenv("API_KEY"), os.Getenv("SEID"),
	)

	resp, err := resty.R().Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch search results: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	items, ok := result["items"].([]interface{})
	if !ok {
		return fmt.Errorf("no items found in the response")
	}

	if err := os.MkdirAll("../downloads/", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	for i, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if link, ok := itemMap["link"].(string); ok {
				fileName := fmt.Sprintf("image_%d_%d.jpg", i+1, imageId)
				outputPath := filepath.Join("../downloads/", fileName)
				err := downloadFile(link, outputPath)
				if err != nil {
					fmt.Printf("Failed to download image %d: %v\n", i+1, err)
				} else {
					fmt.Printf("Image %d saved to %s\n", i+1, outputPath)
				}
				var image models.ImageMetadata
				if err := initialize.DB.First(&image, imageId).Error; err != nil {
					log.Printf("Image not found: %v", err)
					continue
				}
				image.OriginalURL = outputPath
				if err := initialize.DB.Save(&image).Error; err != nil {
					log.Printf("Failed to update image: %v", err)
				}
				return nil
			}
		}
	}

	return nil
}

func downloadFile(url, outputPath string) error {

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}
