package handler

import (
	"net/http"

	"insider/internal/model"
	"insider/internal/mpostgres"
	"insider/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/useinsider/go-pkg/inslogger"
)

type MessageHandler struct {
	messageService mpostgres.MessageService
	scheduler      service.SchedulerService
	logger         inslogger.Interface
	messageSender  service.MessageSender
}

func NewMessageHandler(
	messageService mpostgres.MessageService,
	scheduler service.SchedulerService,
	messageSender service.MessageSender,
	logger inslogger.Interface,
) *MessageHandler {

	return &MessageHandler{
		messageService: messageService,
		scheduler:      scheduler,
		messageSender:  messageSender,
		logger:         logger,
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
func (h *MessageHandler) GetSentMessages(c *gin.Context) {
	messages, err := h.messageService.GetSentMessages(c.Request.Context())
	if err != nil {
		h.logger.Errorf("error retrieving sent messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sent messages", "details": err.Error()})
		return
	}

	// Return an empty array if no messages are found
	if len(messages) == 0 {
		h.logger.Log("No sent messages found")
		c.JSON(http.StatusOK, []model.Message{})
		return
	}
	h.logger.Logf("Retrieved %d sent messages", len(messages))
	c.JSON(http.StatusOK, messages)
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
	message := model.Message{
		ID:             req.ID,
		Content:        req.Content,
		RecipientPhone: req.RecipientPhone,
	}

	// Bind the JSON payload to the message struct
	if err := c.ShouldBindJSON(&message); err != nil {
		h.logger.Errorf("Invalid request payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	err := h.messageSender.SendMessage(message)
	if err != nil {
		h.logger.Errorf("Failed to send message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	if err := h.messageService.UpdateMessageSent(c.Request.Context(), message.ID); err != nil {
		h.logger.Logf("Failed to update message status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update message status"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "Accepted",
		"messageId": message.ID,
	})
}
