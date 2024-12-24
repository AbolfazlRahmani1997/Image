package initialize

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"image_extracting/internal/models"
	"log"
	"os"
)

var DB *gorm.DB

func InitDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Tehran",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	fmt.Println("Connected to database!")
}

func Migrate() {
	err := DB.AutoMigrate(models.ImageMetadata{})
	if err != nil {
		return
	}
}
