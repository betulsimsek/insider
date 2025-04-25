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
}

func NewMessageHandler(messageService mpostgres.MessageService, scheduler service.SchedulerService, logger inslogger.Interface) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		scheduler:      scheduler,
		logger:         logger,
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
