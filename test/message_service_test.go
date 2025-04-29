package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"message-service/internal/handler"
	"message-service/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/useinsider/go-pkg/inslogger"
)

// Mock dependencies
type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) GetMessage(ctx context.Context, id uint) (model.Message, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.Message), args.Error(1)
}

func (m *MockMessageService) CreateMessage(ctx context.Context, message model.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockMessageService) GetSentMessages(ctx context.Context) ([]model.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Message), args.Error(1)
}

func (m *MockMessageService) UpdateMessageSent(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMessageService) GetUnsentMessages(ctx context.Context, limit int) ([]model.Message, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]model.Message), args.Error(1)
}

type MockSchedulerService struct {
	mock.Mock
}

func (m *MockSchedulerService) Start() error {
	return m.Called().Error(0)
}

func (m *MockSchedulerService) Stop() error {
	return m.Called().Error(0)
}

func (m *MockSchedulerService) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockMessageSender struct {
	mock.Mock
}

func (m *MockMessageSender) SendMessage(message model.Message) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockMessageSender) SendMessages(limit int) error {
	args := m.Called(limit)
	return args.Error(0)
}

func (m *MockMessageSender) ClearMessageCache() error {
	args := m.Called()
	return args.Error(0)
}

type MockRedisClient struct {
	mock.Mock
}

// Helper functions for test setup
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.Default()
}

func createMessageRequest(id uint, content, phone string) (*http.Request, error) {
	message := model.SendMessageRequest{
		ID:             id,
		Content:        content,
		RecipientPhone: phone,
	}
	body, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "/api/messages/send", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// Scheduler Tests
func TestStartScheduler(t *testing.T) {
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Start").Return(nil)

	messageHandler := handler.NewMessageHandler(
		nil,                                  // messageService
		mockScheduler,                        // scheduler
		nil,                                  // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)
	router := setupRouter()
	router.POST("/api/scheduler/start", messageHandler.StartScheduler)

	req, _ := http.NewRequest(http.MethodPost, "/api/scheduler/start", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockScheduler.AssertCalled(t, "Start")
}

func TestStopScheduler(t *testing.T) {
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		nil,                                  // messageService
		mockScheduler,                        // scheduler
		nil,                                  // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/scheduler/stop", messageHandler.StopScheduler)

	req, _ := http.NewRequest(http.MethodPost, "/api/scheduler/stop", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockScheduler.AssertCalled(t, "Stop")
}

// SendMessage Tests
func TestSendMessage_NewMessage(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	// Mock GetMessage to return an error, simulating message not found
	mockService.On("GetMessage", mock.Anything, uint(1)).Return(model.Message{}, fmt.Errorf("not found"))
	mockService.On("CreateMessage", mock.Anything, mock.Anything).Return(nil)
	mockSender.On("SendMessage", mock.Anything).Return(nil)
	mockService.On("UpdateMessageSent", mock.Anything, uint(1)).Return(nil)
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		mockScheduler,                        // scheduler
		mockSender,                           // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	req, _ := createMessageRequest(1, "Test Message", "+123456789")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

func TestSendMessage_CreateMessageError(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	// Mock GetMessage to return error (not found)
	mockService.On("GetMessage", mock.Anything, uint(1)).Return(model.Message{}, fmt.Errorf("not found"))
	// Mock CreateMessage to return error
	mockService.On("CreateMessage", mock.Anything, mock.Anything).Return(errors.New("database error"))

	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		mockScheduler,                        // scheduler
		mockSender,                           // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	req, _ := createMessageRequest(1, "Test Message", "+123456789")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	var response map[string]string
	_ = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Failed to create message")
	mockService.AssertExpectations(t)
	mockSender.AssertNotCalled(t, "SendMessage", mock.Anything)
}

func TestSendMessage_SendMessageError(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	// Mock GetMessage to return a message
	existingMessage := model.Message{
		ID:             1,
		Content:        "Test Message",
		RecipientPhone: "+123456789",
	}
	mockService.On("GetMessage", mock.Anything, uint(1)).Return(existingMessage, nil)
	// Mock SendMessage to return error
	mockSender.On("SendMessage", mock.Anything).Return(errors.New("sending error"))

	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		mockScheduler,                        // scheduler
		mockSender,                           // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	req, _ := createMessageRequest(1, "Test Message", "+123456789")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	var response map[string]string
	_ = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Failed to send message")
	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
	mockService.AssertNotCalled(t, "UpdateMessageSent", mock.Anything, mock.Anything)
}

func TestSendMessage_UpdateStatusError(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	// Mock GetMessage to return a message
	existingMessage := model.Message{
		ID:             1,
		Content:        "Test Message",
		RecipientPhone: "+123456789",
	}
	mockService.On("GetMessage", mock.Anything, uint(1)).Return(existingMessage, nil)
	mockSender.On("SendMessage", mock.Anything).Return(nil)
	// Mock UpdateMessageSent to return error
	mockService.On("UpdateMessageSent", mock.Anything, uint(1)).Return(errors.New("update error"))

	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		mockScheduler,                        // scheduler
		mockSender,                           // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	req, _ := createMessageRequest(1, "Test Message", "+123456789")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	var response map[string]string
	_ = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Failed to update message status")
	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

func TestSendMessage_InvalidRequest(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		mockScheduler,                        // scheduler
		mockSender,                           // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	// Invalid JSON
	req, _ := http.NewRequest(http.MethodPost, "/api/messages/send", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	var response map[string]string
	_ = json.Unmarshal(resp.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "Invalid request payload")
}

// Fix TestClearMessageCache - it was incomplete
func TestClearMessageCache(t *testing.T) {
	mockSender := new(MockMessageSender)
	mockSender.On("ClearMessageCache").Return(nil)

	messageHandler := handler.NewMessageHandler(
		nil,                                  // messageService
		nil,                                  // scheduler
		mockSender,                           // messageSender - use the mock
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/cache/clear", messageHandler.ClearMessageCache)

	req, _ := http.NewRequest(http.MethodPost, "/api/messages/cache/clear", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockSender.AssertCalled(t, "ClearMessageCache")
}

// Fix TestGetSentMessages - use the mock service correctly
func TestGetSentMessages(t *testing.T) {
	mockService := new(MockMessageService)
	mockService.On("GetSentMessages", mock.Anything).Return([]model.Message{
		{ID: 1, Content: "Test Message", RecipientPhone: "+123456789", Sent: true},
	}, nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService - use the mock
		nil,                                  // scheduler
		nil,                                  // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.GET("/api/messages/sent", messageHandler.GetSentMessages)

	req, _ := http.NewRequest(http.MethodGet, "/api/messages/sent", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockService.AssertCalled(t, "GetSentMessages", mock.Anything)
}

// Fix TestSendMessage_ExistingMessage - use the mocks correctly
func TestSendMessage_ExistingMessage(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	// Mock GetMessage to return a message, simulating message found
	existingMessage := model.Message{
		ID:             1,
		Content:        "Test Message",
		RecipientPhone: "+123456789",
	}
	mockService.On("GetMessage", mock.Anything, uint(1)).Return(existingMessage, nil)
	mockSender.On("SendMessage", mock.Anything).Return(nil)
	mockService.On("UpdateMessageSent", mock.Anything, uint(1)).Return(nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService - use the mock
		nil,                                  // scheduler
		mockSender,                           // messageSender - use the mock
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.POST("/api/messages/send", messageHandler.SendMessage)

	req, _ := createMessageRequest(1, "Test Message", "+123456789")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	mockService.AssertExpectations(t)
	mockSender.AssertExpectations(t)
}

func TestGetUnsentMessages_Error(t *testing.T) {
	mockService := new(MockMessageService)
	mockService.On("GetUnsentMessages", mock.Anything, 10).Return([]model.Message{}, errors.New("database error"))

	router := setupRouter()
	router.GET("/api/messages/unsent", func(c *gin.Context) {
		limit := 10
		messages, err := mockService.GetUnsentMessages(c.Request.Context(), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, messages)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/messages/unsent", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockService.AssertCalled(t, "GetUnsentMessages", mock.Anything, 10)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "database error")
}

func TestGetSentMessages_Success(t *testing.T) {
	mockService := new(MockMessageService)
	mockService.On("GetSentMessages", mock.Anything).Return([]model.Message{
		{ID: 1, Content: "Test Message", RecipientPhone: "+123456789", Sent: true},
		{ID: 2, Content: "Another Message", RecipientPhone: "+987654321", Sent: true},
	}, nil)

	messageHandler := handler.NewMessageHandler(
		mockService,                          // messageService
		nil,                                  // scheduler
		nil,                                  // messageSender
		inslogger.NewLogger(inslogger.Debug), // logger
		nil,                                  // redisClient
	)

	router := setupRouter()
	router.GET("/api/messages/sent", messageHandler.GetSentMessages)

	req, _ := http.NewRequest(http.MethodGet, "/api/messages/sent", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockService.AssertCalled(t, "GetSentMessages", mock.Anything)

	var responseMessages []model.Message
	err := json.Unmarshal(resp.Body.Bytes(), &responseMessages)
	assert.NoError(t, err)
	assert.Len(t, responseMessages, 2)
}

func TestAll(t *testing.T) {
	t.Run("Scheduler", func(t *testing.T) {
		t.Run("Start", TestStartScheduler)
		t.Run("Stop", TestStopScheduler)
	})

	t.Run("Messages", func(t *testing.T) {
		t.Run("Send", func(t *testing.T) {
			t.Run("NewMessage", TestSendMessage_NewMessage)
			t.Run("CreateError", TestSendMessage_CreateMessageError)
			t.Run("SendError", TestSendMessage_SendMessageError)
			t.Run("UpdateError", TestSendMessage_UpdateStatusError)
			t.Run("InvalidRequest", TestSendMessage_InvalidRequest)
		})

		t.Run("Cache", func(t *testing.T) {
			t.Run("Clear", TestClearMessageCache)
		})
	})
}
