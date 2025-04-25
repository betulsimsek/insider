package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"insider/internal/config"
	"insider/internal/model"
	"insider/internal/mpostgres"

	"github.com/useinsider/go-pkg/insredis"
)

type MessagePayload struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type MessageResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type MessageSender interface {
	SendMessages(int) error
	SendMessage(message model.Message) (string, error)
}

type messageSender struct {
	messageService mpostgres.MessageService
	redisClient    insredis.RedisInterface
	webhookURL     string
	authKey        string
}

func NewMessageSender(service mpostgres.MessageService, redisClient insredis.RedisInterface, config *config.App) MessageSender {
	return &messageSender{
		messageService: service,
		redisClient:    redisClient,
		webhookURL:     config.WebhookURL,
		authKey:        config.AuthKey,
	}
}

func (s *messageSender) SendMessages(count int) error {
	ctx := context.Background()
	messages, err := s.messageService.GetUnsentMessages(ctx, count)
	if err != nil {
		return fmt.Errorf("failed to get unsent messages: %w", err)
	}

	for _, message := range messages {
		messageID, err := s.SendMessage(message)
		if err != nil {
			log.Printf("Failed to send message %d: %v", message.ID, err)
			continue
		}

		if err := s.messageService.UpdateMessageSent(ctx, message.ID, messageID); err != nil {
			log.Printf("Failed to update message %d status: %v", message.ID, err)
		}
	}
	return nil
}

func (s *messageSender) SendMessage(message model.Message) (string, error) {
	payload := MessagePayload{
		To:      message.RecipientPhone,
		Content: message.Content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ins-auth-key", s.authKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 OK and 202 Accepted as valid responses
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// After successfully sending the message and getting the messageID, cache it
	if s.redisClient != nil {
		// Cache message ID with timestamp as value
		cacheKey := fmt.Sprintf("message:%s", response.MessageID)
		timestamp := time.Now().Format(time.RFC3339)
		if err := s.redisClient.Set(cacheKey, timestamp, 24*time.Hour).Err(); err != nil {
			log.Printf("Failed to cache message ID: %v", err)
		}
	}

	return response.MessageID, nil
}
