package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"message-service/internal/config"
	"message-service/internal/model"
	"message-service/internal/mpostgres"

	"github.com/useinsider/go-pkg/inslogger"
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
	SendMessage(message model.Message) error
}

type messageSender struct {
	logger         inslogger.Interface
	messageService mpostgres.MessageService
	redisClient    insredis.RedisInterface
	webhookURL     string
	authKey        string
}

func NewMessageSender(service mpostgres.MessageService, redisClient insredis.RedisInterface, config *config.App, logger inslogger.Interface) MessageSender {
	return &messageSender{
		logger:         logger,
		messageService: service,
		redisClient:    redisClient,
		webhookURL:     config.WebhookURL,
		authKey:        config.AuthKey,
	}
}

func (s *messageSender) SendMessages(count int) error {
	s.logger.Log("Fetching unsent messages...")
	ctx := context.Background()
	s.logger.Log("Fetching unsent messages...")
	messages, err := s.messageService.GetUnsentMessages(ctx, count)
	if err != nil {
		s.logger.Log(fmt.Errorf("failed to get unsent messages: %v", err))
		return err
	}
	s.logger.Logf("Fetched %d unsent messages", len(messages))

	if len(messages) == 0 {
		s.logger.Log("No unsent messages found.")
		return nil
	}

	for _, message := range messages {
		s.logger.Log(fmt.Sprintf("Sending message ID: %d", message.ID))
		err := s.SendMessage(message)
		if err != nil {
			s.logger.Log(fmt.Errorf("failed to send message ID %d: %v", message.ID, err))
			continue
		}

		if err := s.messageService.UpdateMessageSent(ctx, message.ID); err != nil {
			s.logger.Log(fmt.Errorf("failed to update message ID %d status: %v", message.ID, err))
		}
	}

	return nil
}
func (s *messageSender) SendMessage(message model.Message) error {
	payload := MessagePayload{
		To:      message.RecipientPhone,
		Content: message.Content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ins-auth-key", s.authKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		s.logger.Warnf("Rate limit hit. Retrying... Headers: %v", resp.Header)
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Check for valid response status codes (202 Accepted or 200 OK)
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	s.logger.Logf("Message sent successfully: %v", message.ID)

	// Cache the message ID in Redis (if Redis is enabled)
	if s.redisClient != nil {
		messageId := fmt.Sprintf("%v", message.ID)
		cacheKey := fmt.Sprintf("message:%s", messageId)
		timestamp := time.Now().Format(time.RFC3339)

		s.logger.Logf("Caching message ID: %s with timestamp: %s", messageId, timestamp)

		if err := s.redisClient.Set(cacheKey, timestamp, 24*time.Hour).Err(); err != nil {
			s.logger.Warnf("Failed to cache message ID: %s, error: %v", messageId, err)
		} else {
			s.logger.Logf("Cached message ID: %s with timestamp: %s", messageId, timestamp)
		}
	} else {
		s.logger.Warn("Redis client is nil. Skipping caching.")
	}

	return nil
}
