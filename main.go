package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/insredis"

	_ "insider/docs"
	"insider/internal/config"
	"insider/internal/handler"
	"insider/internal/mpostgres"
	"insider/internal/pkg/gpostgresql"
	"insider/internal/service"
)

// @title Insider API
// @version 1.0
// @description This is the API documentation for the Insider project.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @schemes http
func main() {
	logger := inslogger.NewLogger(inslogger.Debug)
	logger.Log("Starting the application...")

	ctx := context.Background()

	logger.Log("Reading configuration...")
	appConfig := config.ReadEnvironment(ctx, &config.AppEnv, logger)

	logger.Log("Connecting to the database...")
	dbPool, err := gpostgresql.NewDBConnection(ctx, &appConfig.Database, logger)
	if err != nil {
		logger.Fatal(fmt.Errorf("database connection failed: %w", err))
	}
	defer gpostgresql.Close(ctx, dbPool, logger)
	logger.Log("Connected to the database.")

	logger.Log("Initializing services...")
	messageService := mpostgres.NewMessageService(dbPool, logger)

	redisCfg := insredis.Config{
		RedisHost:     fmt.Sprintf("%s:%d", appConfig.Redis.Host, appConfig.Redis.Port),
		RedisPoolSize: 10,
		DialTimeout:   500 * time.Millisecond,
		ReadTimeout:   500 * time.Millisecond,
		MaxRetries:    3,
	}

	redisClient := insredis.Init(redisCfg)
	if err := redisClient.Ping().Err(); err != nil {
		logger.Fatal(fmt.Errorf("failed to connect to Redis: %w", err))
	}
	logger.Log("Connected to Redis.")

	messageSender := service.NewMessageSender(messageService, redisClient, appConfig, logger)
	schedulerService := service.NewSchedulerService(messageSender, 2*time.Minute, 2, logger)

	logger.Log("Creating message handler...")
	messageHandler := handler.NewMessageHandler(messageService, schedulerService, messageSender, logger)
	logger.Log("Setting up the router...")
	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Log("Registering routes...")

	router.POST("/api/messages/send", messageHandler.SendMessage)
	router.POST("/api/scheduler/start", messageHandler.StartScheduler)
	router.POST("/api/scheduler/stop", messageHandler.StopScheduler)
	router.GET("/api/messages/sent", messageHandler.GetSentMessages)

	logger.Log("Starting the server...")
	err = router.Run(fmt.Sprintf(":%d", appConfig.Server.Port))
	if err != nil {
		logger.Fatal(fmt.Errorf("failed to start server: %w", err))
	}
}
