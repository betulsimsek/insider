package model

import (
	"time"
)

type Message struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	RecipientPhone string    `gorm:"type:varchar(20);not null" json:"recipient_phone"`
	Sent           bool      `gorm:"default:false" json:"sent"`
	SentAt         time.Time `json:"sent_at"`
	MessageID      string    `gorm:"type:varchar(100)" json:"message_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
