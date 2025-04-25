// internal/handler/message_handler.go
package handler

import (
	"log"
	"net/http"

	"insider/internal/model"
	"insider/internal/mpostgres"
	"insider/internal/service"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService mpostgres.MessageService
	scheduler      service.SchedulerService
}

func NewMessageHandler(messageService mpostgres.MessageService, scheduler service.SchedulerService) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		scheduler:      scheduler,
	}
}

// StartScheduler godoc
// @Summary Start the message scheduler
// @Description Start the automatic message sending process
// @Tags scheduler
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /scheduler/start [post]
func (h *MessageHandler) StartScheduler(c *gin.Context) {
	if err := h.scheduler.Start(); err != nil {
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

// StopScheduler godoc
// @Summary Stop the message scheduler
// @Description Stop the automatic message sending process
// @Tags scheduler
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /scheduler/stop [post]
func (h *MessageHandler) StopScheduler(c *gin.Context) {
	if err := h.scheduler.Stop(); err != nil {
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

// GetSentMessages godoc
// @Summary Get all sent messages
// @Description Retrieve a list of all sent messages
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {array} model.Message
// @Router /messages/sent [get]
func (h *MessageHandler) GetSentMessages(c *gin.Context) {
	messages, err := h.messageService.GetSentMessages(c.Request.Context())
	if err != nil {
		log.Printf("Error retrieving sent messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sent messages", "details": err.Error()})
		return
	}

	// Return an empty array if no messages are found
	if len(messages) == 0 {
		c.JSON(http.StatusOK, []model.Message{})
		return
	}

	c.JSON(http.StatusOK, messages)
}
