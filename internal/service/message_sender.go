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
	ClearMessageCache() error
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

	messagesSent := false

	for _, message := range messages {
		messageIdStr := fmt.Sprintf("%d", message.ID)
		s.logger.Logf("Checking cache for message ID: %d", message.ID)
		isCached, err := s.IsMessageCached(messageIdStr)
		if err != nil {
			s.logger.Warnf("Failed to check cache for message ID %d: %v", message.ID, err)
			continue
		}
		if isCached {
			s.logger.Logf("Message ID %d is already cached. Skipping send.", message.ID)
			continue
		}

		s.logger.Logf("Sending message ID: %d", message.ID)
		err = s.SendMessage(message)
		if err != nil {
			s.logger.Log(fmt.Errorf("failed to send message ID %d: %v", message.ID, err))
			continue
		}

		if err := s.messageService.UpdateMessageSent(ctx, message.ID); err != nil {
			s.logger.Log(fmt.Errorf("failed to update message ID %d status: %v", message.ID, err))
			continue
		}

		s.logger.Logf("Message with ID %d updated successfully", message.ID)
		messagesSent = true

		if s.redisClient != nil {
			cacheKey := fmt.Sprintf("message:%s", messageIdStr)
			timestamp := time.Now().Format(time.RFC3339)

			s.logger.Logf("Caching message ID: %s with timestamp: %s", messageIdStr, timestamp)

			if err := s.redisClient.Set(cacheKey, timestamp, 24*time.Hour).Err(); err != nil {
				s.logger.Warnf("Failed to cache message ID: %s, error: %v", messageIdStr, err)
			} else {
				s.logger.Logf("Cached message ID: %s with timestamp: %s", messageIdStr, timestamp)
			}
		} else {
			s.logger.Warn("Redis client is nil. Skipping caching.")
		}
	}

	// Update the messages:sent cache if any messages were sent
	if messagesSent && s.redisClient != nil {
		s.logger.Log("Updating messages:sent cache with latest sent messages")

		// Get all sent messages from the database
		allSentMessages, err := s.messageService.GetSentMessages(ctx)
		if err != nil {
			s.logger.Warnf("Failed to get sent messages for cache update: %v", err)
		} else {
			// Marshal the messages to JSON
			messagesJSON, err := json.Marshal(allSentMessages)
			if err != nil {
				s.logger.Warnf("Failed to marshal sent messages for cache: %v", err)
			} else {
				// Update the messages:sent cache
				if err := s.redisClient.Set("messages:sent", messagesJSON, 10*time.Minute).Err(); err != nil {
					s.logger.Warnf("Failed to update messages:sent cache: %v", err)
				} else {
					s.logger.Logf("Successfully updated messages:sent cache with %d messages", len(allSentMessages))
				}
			}
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
		return fmt.Errorf("rate limited: status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	s.logger.Logf("Message sent successfully: ID=%v", message.ID)
	return nil
}

func (s *messageSender) IsMessageCached(messageId string) (bool, error) {
	if s.redisClient == nil {
		s.logger.Warn("Redis client is nil. Skipping cache check.")
		return false, nil
	}

	cacheKey := fmt.Sprintf("message:%s", messageId)

	exists, err := s.redisClient.Exists(cacheKey).Result()
	if err != nil {
		s.logger.Warnf("Failed to check cache for message ID: %s, error: %v", messageId, err)
		return false, err
	}
	isCached := exists > 0
	s.logger.Logf("Cache check for message ID %s: exists=%v", messageId, isCached)
	return isCached, nil
}

func (s *messageSender) ClearMessageCache() error {
	if s.redisClient == nil {
		s.logger.Warn("Redis client is nil. Cannot clear cache.")
		return nil
	}

	s.logger.Log("Clearing all message caches...")

	keys, err := s.redisClient.Keys("message:*").Result()
	if err != nil {
		s.logger.Errorf("Failed to get message cache keys: %v", err)
		return err
	}

	if len(keys) == 0 {
		s.logger.Log("No message cache keys found.")
		return nil
	}

	if len(keys) > 0 {
		if err := s.redisClient.Del(keys...).Err(); err != nil {
			s.logger.Errorf("Failed to delete message cache keys: %v", err)
			return err
		}
	}

	s.logger.Logf("Successfully cleared %d message cache entries", len(keys))
	return nil
}
