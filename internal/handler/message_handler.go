package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"message-service/internal/model"
	"message-service/internal/mpostgres"
	"message-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/insredis"
)

type MessageHandler struct {
	messageService mpostgres.MessageService
	scheduler      service.SchedulerService
	logger         inslogger.Interface
	messageSender  service.MessageSender
	redisClient    insredis.RedisInterface
}

func NewMessageHandler(
	messageService mpostgres.MessageService,
	scheduler service.SchedulerService,
	messageSender service.MessageSender,
	logger inslogger.Interface,
	redisClient insredis.RedisInterface,
) *MessageHandler {

	return &MessageHandler{
		messageService: messageService,
		scheduler:      scheduler,
		messageSender:  messageSender,
		logger:         logger,
		redisClient:    redisClient,
	}
}

// StartScheduler starts the message scheduler.
// @Summary Start the message scheduler
// @Description Start the automatic message sending process
// @Tags scheduler
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/scheduler/start [post]
func (h *MessageHandler) StartScheduler(c *gin.Context) {
	if err := h.scheduler.Start(); err != nil {
		h.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start scheduler",
		})
		return
	}
	if h.redisClient != nil {
		if err := h.redisClient.Set("scheduler:state", "running", 0).Err(); err != nil {
			h.logger.Warnf("Failed to cache scheduler state: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduler started successfully",
		"status":  "running",
	})
}

// StopScheduler stops the message scheduler.
// @Summary Stop the message scheduler
// @Description Stop the automatic message sending process
// @Tags scheduler
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/scheduler/stop [post]
func (h *MessageHandler) StopScheduler(c *gin.Context) {
	if err := h.scheduler.Stop(); err != nil {
		h.logger.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to stop scheduler",
		})
		return
	}

	if h.redisClient != nil {
		if err := h.redisClient.Del("scheduler:state").Err(); err != nil {
			h.logger.Warnf("Failed to remove scheduler state from cache: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduler stopped successfully",
		"status":  "stopped",
	})
}

// GetSentMessages retrieves all sent messages.
// @Summary Get all sent messages
// @Description Retrieve a list of all sent messages
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {array} model.Message
// @Router /api/messages/sent [get]
// GetSentMessages retrieves all sent messages.
// @Summary Get all sent messages
// @Description Retrieve a list of all sent messages
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {array} model.Message
// @Router /api/messages/sent [get]
// GetSentMessages retrieves all sent messages with proper Redis caching.
func (h *MessageHandler) GetSentMessages(c *gin.Context) {
	cacheKey := "messages:sent"
	h.logger.Logf("Attempting to retrieve sent messages, cache key: %s", cacheKey)

	// Check if the sent messages are cached in Redis
	if h.redisClient != nil {
		cachedMessages, err := h.redisClient.Get(cacheKey).Result()
		if err == nil && cachedMessages != "" {
			h.logger.Log("Cache hit! Returning cached sent messages.")
			c.Data(http.StatusOK, "application/json", []byte(cachedMessages))
			return
		} else if err != nil && err.Error() != "redis: nil" {
			h.logger.Warnf("Redis error while reading cache for sent messages: %v", err)
			h.logger.Log("Falling back to database due to Redis error")
		} else {
			h.logger.Log("Cache miss for sent messages. Querying database.")
		}
	} else {
		h.logger.Warn("Redis client is nil. Skipping cache check.")
	}

	// Fetch sent messages from the database
	messages, err := h.messageService.GetSentMessages(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Error retrieving sent messages from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve sent messages",
			"details": err.Error(),
		})
		return
	}

	// Return an empty array if no messages are found
	if len(messages) == 0 {
		h.logger.Log("No sent messages found in database")
		emptyResult := "[]"

		// Cache the empty result too to prevent repeated database queries
		if h.redisClient != nil {
			if err := h.redisClient.Set(cacheKey, emptyResult, 5*time.Minute).Err(); err != nil {
				h.logger.Warnf("Failed to cache empty sent messages result: %v", err)
			} else {
				h.logger.Log("Cached empty sent messages result for 5 minutes")
			}
		}

		c.Data(http.StatusOK, "application/json", []byte(emptyResult))
		return
	}

	// Cache the sent messages with a TTL
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		h.logger.Warnf("Failed to marshal messages to JSON: %v", err)
		c.JSON(http.StatusOK, messages)
		return
	}

	if h.redisClient != nil {
		cacheTTL := 10 * time.Minute
		if err := h.redisClient.Set(cacheKey, messagesJSON, cacheTTL).Err(); err != nil {
			h.logger.Warnf("Failed to cache sent messages: %v", err)
		} else {
			h.logger.Logf("Successfully cached %d sent messages with TTL of %v", len(messages), cacheTTL)
		}
	}

	h.logger.Logf("Retrieved %d sent messages from database", len(messages))
	c.Data(http.StatusOK, "application/json", messagesJSON)
}

// SendMessage handles sending a message.
// @Summary Send a message
// @Description Send a message to a recipient
// @Tags messages
// @Accept json
// @Produce json
// @Param message body model.SendMessageRequest true "Message payload"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/messages/send [post]
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req model.SendMessageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Invalid request payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()
	h.logger.Logf("Processing request to send message ID: %d", req.ID)

	// First check if message exists in database
	_, err := h.messageService.GetMessage(ctx, req.ID)
	if err != nil {
		h.logger.Logf("Message with ID %d not found, creating it", req.ID)

		newMessage := model.Message{
			ID:             req.ID,
			Content:        req.Content,
			RecipientPhone: req.RecipientPhone,
			Sent:           false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Save message to database
		if err := h.messageService.CreateMessage(ctx, newMessage); err != nil {
			h.logger.Errorf("Failed to create message: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
			return
		}
		h.logger.Logf("Created new message with ID: %d", req.ID)
	} else {
		h.logger.Logf("Found existing message with ID: %d", req.ID)
	}

	// Prepare message for sending
	message := model.Message{
		ID:             req.ID,
		Content:        req.Content,
		RecipientPhone: req.RecipientPhone,
	}

	// Send the message
	h.logger.Logf("Sending message with ID: %d", req.ID)
	err = h.messageSender.SendMessage(message)
	if err != nil {
		h.logger.Errorf("Failed to send message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}
	h.logger.Logf("Successfully sent message with ID: %d", req.ID)

	// Update message status in database
	h.logger.Logf("Updating message status in database for ID: %d", req.ID)
	if err := h.messageService.UpdateMessageSent(ctx, message.ID); err != nil {
		h.logger.Errorf("Failed to update message status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message status"})
		return
	}
	h.logger.Logf("Successfully updated message status in database for ID: %d", req.ID)

	// Update the sent messages cache after sending a new message
	if h.redisClient != nil {
		cacheKey := "messages:sent"
		h.logger.Logf("Updating cache for key: %s after sending new message", cacheKey)

		// Get all sent messages
		messages, err := h.messageService.GetSentMessages(ctx)
		if err != nil {
			h.logger.Warnf("Failed to fetch sent messages for cache update: %v", err)
		} else if len(messages) == 0 {
			h.logger.Warn("No sent messages found in database - this is unexpected!")
			// Cache empty array instead of null
			if err := h.redisClient.Set(cacheKey, "[]", 10*time.Minute).Err(); err != nil {
				h.logger.Warnf("Failed to update sent messages cache: %v", err)
			}
		} else {
			messagesJSON, err := json.Marshal(messages)
			if err != nil {
				h.logger.Warnf("Failed to marshal messages: %v", err)
			} else {
				if err := h.redisClient.Set(cacheKey, messagesJSON, 10*time.Minute).Err(); err != nil {
					h.logger.Warnf("Failed to update sent messages cache: %v", err)
				} else {
					h.logger.Logf("Successfully updated sent messages cache with %d messages", len(messages))
				}
			}
		}

		// Also clear any outdated message cache entries
		messageCacheKey := fmt.Sprintf("message:%d", req.ID)
		if err := h.redisClient.Del(messageCacheKey).Err(); err != nil {
			h.logger.Warnf("Failed to clear message cache for ID %d: %v", req.ID, err)
		}
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Accepted",
		"messageId": message.ID,
	})
}

// ClearMessageCache clears all message caches
// @Summary Clear message cache
// @Description Clear all message cache entries
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/messages/cache/clear [post]
func (h *MessageHandler) ClearMessageCache(c *gin.Context) {
	h.logger.Log("Request to clear message cache received")

	if err := h.messageSender.ClearMessageCache(); err != nil {
		h.logger.Errorf("Failed to clear message cache: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear message cache",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Message cache cleared successfully",
	})
}
