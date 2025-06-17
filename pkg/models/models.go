package models

import (
	"log"
	"time"
)

type ChatRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

type ChatResponse struct {
	FullResponse string `json:"full_response"`
}

type SSEMessage struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  struct {
		TextChunk string `json:"text_chunk,omitempty"`
		Status    string `json:"status,omitempty"`
	} `json:"data"`
}

type SlackEvent struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
	Ts      string `json:"ts"`
}

type SlackMessageResponse struct {
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Timestamp string `json:"ts,omitempty"`
	AsUser    bool   `json:"as_user,omitempty"`
}

type AppConfig struct {
	SlackAppToken        string        `env:"SLACK_APP_TOKEN,required"`
	SlackBotToken        string        `env:"SLACK_BOT_TOKEN,required"`
	ChatBackendURL       string        `env:"CHAT_BACKEND_URL,required"`
	ListenPort           string        `env:"LISTEN_PORT,default=8080"`
	MockBackendPort      string        `env:"MOCK_BACKEND_PORT,default=8081"`
	TelemetryExporter    string        `env:"OTEL_EXPORTER_OTLP_PROTOCOL,default=grpc"`
	TelemetryEndpoint    string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=localhost:4317"`
	ServiceName          string        `env:"OTEL_SERVICE_NAME,default=chatrelay-bot"`
	RequestTimeout       time.Duration `env:"REQUEST_TIMEOUT,default=30s"`
	SlackAPIRetryCount   int           `env:"SLACK_API_RETRY_COUNT,default=3"`
	SlackAPIRetryDelay   time.Duration `env:"SLACK_API_RETRY_DELAY,default=1s"`
	BackendAPIRetryCount int           `env:"BACKEND_API_RETRY_COUNT,default=3"`
	BackendAPIRetryDelay time.Duration `env:"BACKEND_API_RETRY_DELAY,default=1s"`
}

func NewConfig() *AppConfig {
	log.Println("[INFO] Initializing new AppConfig with default values")
	return &AppConfig{}
}
