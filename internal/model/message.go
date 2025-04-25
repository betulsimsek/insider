package model

import (
	"time"
)

// Message represents a message entity.
// @Description Message entity
type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	RecipientPhone string    `gorm:"type:varchar(20);not null" json:"recipient_phone"`
	Sent           bool      `gorm:"default:false" json:"sent"`
	SentAt         time.Time `json:"sent_at"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type SendMessageRequest struct {
	ID             uint   `json:"id" example:"5"`
	Content        string `json:"content" example:"message-service - Project"`
	RecipientPhone string `json:"recipient_phone" example:"+905551111111"`
}
