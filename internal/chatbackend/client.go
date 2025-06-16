// internal/chatbackend/client.go
// This file defines the client for interacting with the mock chat backend API.

package chatbackend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes" // ✅ Correct OpenTelemetry status codes
	"go.opentelemetry.io/otel/trace"

	"chatrelay-bot/pkg/models" // Our custom models package
)

const tracerName = "chatrelay/internal/chatbackend"

// Client defines the interface for interacting with the chat backend.
type Client interface {
	// SendChatRequest sends a chat query to the backend and returns the response.
	// This implementation focuses on a complete JSON response.
	SendChatRequest(ctx context.Context, req models.ChatRequest) (models.ChatResponse, error)
}

// httpClient implements the Client interface using http.Client.
type httpClient struct {
	baseURL    string
	httpClient *http.Client
	retryCount int
	retryDelay time.Duration
}

// NewClient creates and returns a new Client instance.
func NewClient(baseURL string, timeout time.Duration, retryCount int, retryDelay time.Duration) Client {
	return &httpClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		retryCount: retryCount,
		retryDelay: retryDelay,
	}
}

// SendChatRequest sends a chat query to the backend and returns the response.
func (c *httpClient) SendChatRequest(ctx context.Context, req models.ChatRequest) (res models.ChatResponse, err error) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "SendChatRequest",
		trace.WithAttributes(
			attribute.String("chat.user_id", req.UserID),
			attribute.String("chat.query", req.Query),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error()) // ✅ Correct usage
		} else {
			span.SetStatus(codes.Ok, "Chat request successful")
		}
		span.End()
	}()

	requestBody, err := json.Marshal(req)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal chat request", "error", err)
		return models.ChatResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/stream", c.baseURL)

	for i := 0; i <= c.retryCount; i++ {
		attemptCtx, attemptSpan := tracer.Start(ctx, "HTTP POST to Chat Backend",
			trace.WithAttributes(
				attribute.String("http.url", url),
				attribute.String("http.method", "POST"),
				attribute.Int("attempt", i+1),
			),
		)

		httpReq, err := http.NewRequestWithContext(attemptCtx, "POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			attemptSpan.RecordError(err)
			attemptSpan.SetStatus(codes.Error, "Failed to create HTTP request")
			attemptSpan.End()
			return models.ChatResponse{}, fmt.Errorf("failed to create HTTP request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		slog.InfoContext(attemptCtx, "Sending request to chat backend", "url", url, "attempt", i+1)

		httpRes, err := c.httpClient.Do(httpReq)
		if err != nil {
			slog.ErrorContext(attemptCtx, "HTTP request failed", "error", err, "attempt", i+1)
			attemptSpan.RecordError(err)
			attemptSpan.SetStatus(codes.Error, fmt.Sprintf("HTTP request failed: %v", err))
			attemptSpan.End()
			if i < c.retryCount {
				time.Sleep(c.retryDelay)
				continue
			}
			return models.ChatResponse{}, fmt.Errorf("HTTP request to chat backend failed after %d retries: %w", c.retryCount, err)
		}
		defer httpRes.Body.Close()

		attemptSpan.SetAttributes(attribute.Int("http.status_code", httpRes.StatusCode))

		if httpRes.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(httpRes.Body)
			err = fmt.Errorf("chat backend returned non-OK status: %d, body: %s", httpRes.StatusCode, string(bodyBytes))
			slog.ErrorContext(attemptCtx, "Chat backend returned error status", "status", httpRes.StatusCode, "response_body", string(bodyBytes), "attempt", i+1)
			attemptSpan.RecordError(err)
			attemptSpan.SetStatus(codes.Error, err.Error())
			attemptSpan.End()
			if i < c.retryCount {
				time.Sleep(c.retryDelay)
				continue
			}
			return models.ChatResponse{}, fmt.Errorf("chat backend returned error status after %d retries: %w", c.retryCount, err)
		}

		body, err := io.ReadAll(httpRes.Body)
		if err != nil {
			slog.ErrorContext(attemptCtx, "Failed to read response body", "error", err, "attempt", i+1)
			attemptSpan.RecordError(err)
			attemptSpan.SetStatus(codes.Error, "Failed to read response body")
			attemptSpan.End()
			return models.ChatResponse{}, fmt.Errorf("failed to read response body: %w", err)
		}

		err = json.Unmarshal(body, &res)
		if err != nil {
			slog.ErrorContext(attemptCtx, "Failed to unmarshal chat response", "error", err, "response_body", string(body), "attempt", i+1)
			attemptSpan.RecordError(err)
			attemptSpan.SetStatus(codes.Error, "Failed to unmarshal chat response")
			attemptSpan.End()
			return models.ChatResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		slog.InfoContext(attemptCtx, "Successfully received response from chat backend", "response_text", res.FullResponse)
		attemptSpan.SetStatus(codes.Ok, "Request succeeded") // ✅ Optional: mark attempt span success
		attemptSpan.End()
		return res, nil
	}

	return models.ChatResponse{}, fmt.Errorf("failed to send chat request after %d retries", c.retryCount)
}
