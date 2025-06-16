// internal/config/config.go
// This file handles loading application configuration from environment variables.

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv" // Library to load .env files
	"chatrelay-bot/pkg/models"     // Our custom models package
)

// LoadConfig loads application configuration from environment variables.
// It prioritizes actual environment variables over those in a .env file.
func LoadConfig() (*models.AppConfig, error) {
	// Load .env file (if present). This doesn't overwrite existing environment variables.
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Error loading .env file: %v. Proceeding without .env file.\n", err)
	}

	cfg := models.NewConfig()

	// Helper function to get an environment variable or use a default value
	getEnv := func(key, defaultValue string) string {
		if value, exists := os.LookupEnv(key); exists {
			return value
		}
		return defaultValue
	}

	// Helper function to get a required environment variable
	getRequiredEnv := func(key string) (string, error) {
		if value, exists := os.LookupEnv(key); exists && value != "" {
			return value, nil
		}
		return "", fmt.Errorf("required environment variable %s not set", key)
	}

	// Load required Slack tokens
	cfg.SlackAppToken, err = getRequiredEnv("SLACK_APP_TOKEN")
	if err != nil {
		return nil, err
	}
	cfg.SlackBotToken, err = getRequiredEnv("SLACK_BOT_TOKEN")
	if err != nil {
		return nil, err
	}

	// Load required Chat Backend URL
	cfg.ChatBackendURL, err = getRequiredEnv("CHAT_BACKEND_URL")
	if err != nil {
		return nil, err
	}

	// Load optional ports with defaults
	cfg.ListenPort = getEnv("LISTEN_PORT", "8080")
	cfg.MockBackendPort = getEnv("MOCK_BACKEND_PORT", "8081")

	// Load OpenTelemetry configuration
	cfg.TelemetryExporter = getEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	cfg.TelemetryEndpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	cfg.ServiceName = getEnv("OTEL_SERVICE_NAME", "chatrelay-bot")

	// Load timeouts and retry counts
	if timeoutStr := getEnv("REQUEST_TIMEOUT", ""); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.RequestTimeout = timeout
		} else {
			fmt.Printf("Warning: Invalid REQUEST_TIMEOUT format '%s'. Using default 30s. Error: %v\n", timeoutStr, err)
			cfg.RequestTimeout = 30 * time.Second // Default in case of parsing error
		}
	} else {
		cfg.RequestTimeout = 30 * time.Second // Default if not set
	}

	if retryStr := getEnv("SLACK_API_RETRY_COUNT", ""); retryStr != "" {
		if count, err := strconv.Atoi(retryStr); err == nil {
			cfg.SlackAPIRetryCount = count
		} else {
			fmt.Printf("Warning: Invalid SLACK_API_RETRY_COUNT format '%s'. Using default 3. Error: %v\n", retryStr, err)
			cfg.SlackAPIRetryCount = 3 // Default
		}
	} else {
		cfg.SlackAPIRetryCount = 3 // Default
	}

	if delayStr := getEnv("SLACK_API_RETRY_DELAY", ""); delayStr != "" {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			cfg.SlackAPIRetryDelay = delay
		} else {
			fmt.Printf("Warning: Invalid SLACK_API_RETRY_DELAY format '%s'. Using default 1s. Error: %v\n", delayStr, err)
			cfg.SlackAPIRetryDelay = 1 * time.Second // Default
		}
	} else {
		cfg.SlackAPIRetryDelay = 1 * time.Second // Default
	}

	if retryStr := getEnv("BACKEND_API_RETRY_COUNT", ""); retryStr != "" {
		if count, err := strconv.Atoi(retryStr); err == nil {
			cfg.BackendAPIRetryCount = count
		} else {
			fmt.Printf("Warning: Invalid BACKEND_API_RETRY_COUNT format '%s'. Using default 3. Error: %v\n", retryStr, err)
			cfg.BackendAPIRetryCount = 3 // Default
		}
	} else {
		cfg.BackendAPIRetryCount = 3 // Default
	}

	if delayStr := getEnv("BACKEND_API_RETRY_DELAY", ""); delayStr != "" {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			cfg.BackendAPIRetryDelay = delay
		} else {
			fmt.Printf("Warning: Invalid BACKEND_API_RETRY_DELAY format '%s'. Using default 1s. Error: %v\n", delayStr, err)
			cfg.BackendAPIRetryDelay = 1 * time.Second // Default
		}
	} else {
		cfg.BackendAPIRetryDelay = 1 * time.Second // Default
	}


	return cfg, nil
}
