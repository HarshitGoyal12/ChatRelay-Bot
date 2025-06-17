package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"log/slog"

	"github.com/joho/godotenv"
	"chatrelay-bot/pkg/models"
)

func LoadConfig() (*models.AppConfig, error) {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		slog.Warn("Error loading .env file", "error", err)
	}

	cfg := models.NewConfig()

	getEnv := func(key, defaultValue string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return defaultValue
	}

	getRequiredEnv := func(key string) (string, error) {
		if value, exists := os.LookupEnv(key); exists && value != "" {
			return value, nil
		}
		slog.Error("Missing required environment variable", "key", key)
		return "", errors.New("required environment variable " + key + " not set")
	}

	cfg.SlackAppToken, err = getRequiredEnv("SLACK_APP_TOKEN")
	if err != nil {
		return nil, err
	}
	cfg.SlackBotToken, err = getRequiredEnv("SLACK_BOT_TOKEN")
	if err != nil {
		return nil, err
	}
	cfg.ChatBackendURL, err = getRequiredEnv("CHAT_BACKEND_URL")
	if err != nil {
		return nil, err
	}

	cfg.ListenPort = getEnv("LISTEN_PORT", "8080")
	cfg.MockBackendPort = getEnv("MOCK_BACKEND_PORT", "8081")
	cfg.TelemetryExporter = getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	cfg.TelemetryEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	cfg.ServiceName = getEnv("OTEL_SERVICE_NAME", "chatrelay-bot")

	if timeoutStr := getEnv("REQUEST_TIMEOUT", ""); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.RequestTimeout = timeout
		} else {
			slog.Warn("Invalid REQUEST_TIMEOUT format, using default", "value", timeoutStr, "error", err)
			cfg.RequestTimeout = 30 * time.Second
		}
	} else {
		cfg.RequestTimeout = 30 * time.Second
	}

	if retryStr := getEnv("SLACK_API_RETRY_COUNT", ""); retryStr != "" {
		if count, err := strconv.Atoi(retryStr); err == nil {
			cfg.SlackAPIRetryCount = count
		} else {
			slog.Warn("Invalid SLACK_API_RETRY_COUNT, using default", "value", retryStr, "error", err)
			cfg.SlackAPIRetryCount = 3
		}
	} else {
		cfg.SlackAPIRetryCount = 3
	}

	if delayStr := getEnv("SLACK_API_RETRY_DELAY", ""); delayStr != "" {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			cfg.SlackAPIRetryDelay = delay
		} else {
			slog.Warn("Invalid SLACK_API_RETRY_DELAY, using default", "value", delayStr, "error", err)
			cfg.SlackAPIRetryDelay = 1 * time.Second
		}
	} else {
		cfg.SlackAPIRetryDelay = 1 * time.Second
	}

	if retryStr := getEnv("BACKEND_API_RETRY_COUNT", ""); retryStr != "" {
		if count, err := strconv.Atoi(retryStr); err == nil {
			cfg.BackendAPIRetryCount = count
		} else {
			slog.Warn("Invalid BACKEND_API_RETRY_COUNT, using default", "value", retryStr, "error", err)
			cfg.BackendAPIRetryCount = 3
		}
	} else {
		cfg.BackendAPIRetryCount = 3
	}

	if delayStr := getEnv("BACKEND_API_RETRY_DELAY", ""); delayStr != "" {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			cfg.BackendAPIRetryDelay = delay
		} else {
			slog.Warn("Invalid BACKEND_API_RETRY_DELAY, using default", "value", delayStr, "error", err)
			cfg.BackendAPIRetryDelay = 1 * time.Second
		}
	} else {
		cfg.BackendAPIRetryDelay = 1 * time.Second
	}

	return cfg, nil
}
