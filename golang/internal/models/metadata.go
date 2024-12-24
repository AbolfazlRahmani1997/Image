package models

import "time"

type ImageMetadata struct {
	ID          uint      `gorm:"primaryKey"`
	ImagePath   string    `gorm:"not null"`
	Keywords    string    `gorm:"type:jsonb"`
	OriginalURL string    `gorm:"type:text"`
	Width       int       `gorm:"not null"`
	Height      int       `gorm:"not null"`
	UploadedAt  time.Time `gorm:"autoCreateTime"`
}
