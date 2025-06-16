// pkg/models/models.go
// This file defines the shared data structures (models) used across the ChatRelay bot and mock backend.

package models

import "time"

// ChatRequest represents the structure of an incoming chat request to the backend.
type ChatRequest struct {
	UserID string `json:"user_id"` // Unique identifier for the user making the request.
	Query  string `json:"query"`   // The actual query text from the user.
}

// ChatResponse represents a complete JSON response from the chat backend.
// This structure is used when the backend sends the full response at once.
type ChatResponse struct {
	FullResponse string `json:"full_response"` // The complete response text from the chat backend.
}

// SSEMessage represents a single Server-Sent Event (SSE) message part.
// This is used if the backend streams responses.
type SSEMessage struct {
	ID    string `json:"id"`    // Unique ID for the event, useful for tracking.
	Event string `json:"event"` // Type of event (e.g., "message_part", "stream_end").
	Data  struct {
		TextChunk string `json:"text_chunk,omitempty"` // A piece of the response text.
		Status    string `json:"status,omitempty"`     // Status (e.g., "done" for stream end).
	} `json:"data"`
}

// SlackEvent represents a generic structure for incoming Slack events.
// This is a simplified version; actual Slack events are more complex.
// We'll primarily focus on 'app_mention' and 'message' events for simplicity.
type SlackEvent struct {
	Type    string `json:"type"`    // Type of the event (e.g., "app_mention", "message").
	Channel string `json:"channel"` // Channel ID where the event occurred.
	User    string `json:"user"`    // User ID who triggered the event.
	Text    string `json:"text"`    // The text content of the message.
	Ts      string `json:"ts"`      // Timestamp of the event.
	// Add more fields as needed for specific event types or detailed parsing
}

// SlackMessageResponse represents the payload for sending a message back to Slack.
type SlackMessageResponse struct {
	Channel   string `json:"channel"`             // The channel ID to send the message to.
	Text      string `json:"text"`                // The text content of the message.
	Timestamp string `json:"ts,omitempty"`        // Optional: Timestamp of the message to update (for streaming updates).
	AsUser    bool   `json:"as_user,omitempty"`   // Optional: Set to true to post as the bot user.
}

// AppConfig holds all the application's configuration parameters.
type AppConfig struct {
	SlackAppToken       string        `env:"SLACK_APP_TOKEN,required"`         // Slack App Level Token (xapp-...)
	SlackBotToken       string        `env:"SLACK_BOT_TOKEN,required"`         // Slack Bot User OAuth Token (xoxb-...)
	ChatBackendURL      string        `env:"CHAT_BACKEND_URL,required"`        // URL of the chat backend (e.g., http://localhost:8081/v1/chat/stream)
	ListenPort          string        `env:"LISTEN_PORT,default=8080"`         // Port for the bot's internal HTTP server (if needed)
	MockBackendPort     string        `env:"MOCK_BACKEND_PORT,default=8081"`   // Port for the mock chat backend
	TelemetryExporter   string        `env:"OTEL_EXPORTER_OTLP_PROTOCOL,default=grpc"` // OpenTelemetry exporter protocol (grpc, http/protobuf, console)
	TelemetryEndpoint   string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=localhost:4317"` // OpenTelemetry collector endpoint
	ServiceName         string        `env:"OTEL_SERVICE_NAME,default=chatrelay-bot"` // Service name for OpenTelemetry
	RequestTimeout      time.Duration `env:"REQUEST_TIMEOUT,default=30s"`      // Timeout for HTTP requests to the chat backend
	SlackAPIRetryCount  int           `env:"SLACK_API_RETRY_COUNT,default=3"`  // Number of retries for Slack API calls
	SlackAPIRetryDelay  time.Duration `env:"SLACK_API_RETRY_DELAY,default=1s"` // Delay between Slack API retries
	BackendAPIRetryCount int          `env:"BACKEND_API_RETRY_COUNT,default=3"` // Number of retries for Backend API calls
	BackendAPIRetryDelay time.Duration `env:"BACKEND_API_RETRY_DELAY,default=1s"`// Delay between Backend API retries
}

// NewConfig returns a new instance of AppConfig with default values.
func NewConfig() *AppConfig {
	return &AppConfig{}
}
