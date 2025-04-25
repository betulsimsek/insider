package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"insider/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/useinsider/go-pkg/inslogger"
)

// Mock dependencies
type MockMessageService struct {
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
func TestStartScheduler(t *testing.T) {
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Start").Return(nil)

	handler := &MessageHandler{
		scheduler: mockScheduler,
		logger:    inslogger.NewLogger(inslogger.Debug),
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/scheduler/start", handler.StartScheduler)

	req, _ := http.NewRequest(http.MethodPost, "/api/scheduler/start", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockScheduler.AssertCalled(t, "Start")
}

func TestStopScheduler(t *testing.T) {
	mockScheduler := new(MockSchedulerService)
	mockScheduler.On("Stop").Return(nil)

	handler := &MessageHandler{
		scheduler: mockScheduler,
		logger:    inslogger.NewLogger(inslogger.Debug),
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/scheduler/stop", handler.StopScheduler)

	req, _ := http.NewRequest(http.MethodPost, "/api/scheduler/stop", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockScheduler.AssertCalled(t, "Stop")
}

func TestGetSentMessages(t *testing.T) {
	mockService := new(MockMessageService)
	mockService.On("GetSentMessages", mock.Anything).Return([]model.Message{
		{ID: 1, Content: "Test Message", RecipientPhone: "+123456789", Sent: true},
	}, nil)

	handler := &MessageHandler{
		messageService: mockService,
		logger:         inslogger.NewLogger(inslogger.Debug),
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/api/messages/sent", handler.GetSentMessages)

	req, _ := http.NewRequest(http.MethodGet, "/api/messages/sent", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockService.AssertCalled(t, "GetSentMessages", mock.Anything)
}

func TestSendMessage(t *testing.T) {
	mockService := new(MockMessageService)
	mockSender := new(MockMessageSender)

	mockSender.On("SendMessage", mock.Anything).Return(nil)
	mockService.On("UpdateMessageSent", mock.Anything, mock.Anything).Return(nil)

	handler := &MessageHandler{
		messageService: mockService,
		messageSender:  mockSender,
		logger:         inslogger.NewLogger(inslogger.Debug),
	}

	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/api/messages/send", handler.SendMessage)

	message := model.SendMessageRequest{
		ID:             1,
		Content:        "Test Message",
		RecipientPhone: "+123456789",
	}
	body, _ := json.Marshal(message)

	req, _ := http.NewRequest(http.MethodPost, "/api/messages/send", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusAccepted, resp.Code)
	mockSender.AssertCalled(t, "SendMessage", mock.Anything)
	mockService.AssertCalled(t, "UpdateMessageSent", mock.Anything, uint(1))
}
