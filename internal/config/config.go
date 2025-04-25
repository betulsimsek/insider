package config

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/useinsider/go-pkg/inslogger"
)

var AppEnv App

type App struct {
	Config
	WebhookConfig
}

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Port int `env:"SERVER_PORT,required"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST,required"`
	Port     int    `env:"DB_PORT,required"`
	User     string `env:"DB_USER,required"`
	Password string `env:"DB_PASSWORD,required"`
	Name     string `env:"DB_NAME,required"`
}

type RedisConfig struct {
	Host string `env:"REDIS_HOST,required"`
	Port int    `env:"REDIS_PORT,required"`
}

type WebhookConfig struct {
	WebhookURL string `env:"WEBHOOK_URL,required"`
	AuthKey    string `env:"AUTH_KEY,required"`
}

func ReadEnvironment(ctx context.Context, envParam any, logger inslogger.Interface) *App {
	_ = godotenv.Load()
	var config App
	err := envconfig.Process(ctx, &config)
	if err != nil {
		logger.Fatal(fmt.Errorf("error processing environment variables: %v", err))
	}

	return &config
}
