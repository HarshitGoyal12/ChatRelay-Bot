package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"chatrelay-bot/internal/config"
	"chatrelay-bot/internal/telemetry"
	"chatrelay-bot/pkg/models"
)

const tracerName = "mockbackend/main"

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := telemetry.InitOpenTelemetry(ctx, cfg); err != nil {
		slog.Error("Failed to initialize OpenTelemetry", "error", err)
	}
	defer telemetry.ShutdownOpenTelemetry(ctx)

	tracer := otel.Tracer(tracerName)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		requestCtx := r.Context()
		ctx, span := tracer.Start(requestCtx, "MockBackendChatStreamHandler",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			),
		)
		defer span.End()

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			slog.WarnContext(ctx, "Received non-POST request", "method", r.Method)
			span.SetStatus(codes.Error, "Method not allowed")
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			slog.ErrorContext(ctx, "Failed to read request body", "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		var chatReq models.ChatRequest
		err = json.Unmarshal(body, &chatReq)
		if err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			slog.ErrorContext(ctx, "Failed to unmarshal request body", "error", err, "body", string(body))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Invalid JSON payload")
			return
		}

		slog.InfoContext(ctx, "Received chat request", "user_id", chatReq.UserID, "query", chatReq.Query)
		span.SetAttributes(
			attribute.String("chat.user_id", chatReq.UserID),
			attribute.String("chat.query", chatReq.Query),
		)

		processingDelay := time.Millisecond * time.Duration(500+time.Now().UnixNano()%1000)
		slog.InfoContext(ctx, "Simulating processing delay", "duration", processingDelay)
		time.Sleep(processingDelay)
		span.AddEvent("SimulatedProcessingDelay", trace.WithAttributes(attribute.String("duration", processingDelay.String())))

		mockResponse := models.ChatResponse{
			FullResponse: fmt.Sprintf("Hello %s! Your query about '%s' has been processed by the mock backend. This is a detailed and insightful response demonstrating efficient handling of concurrent requests and robust error management. We believe in providing scalable solutions with comprehensive observability features.", chatReq.UserID, chatReq.Query),
		}

		responseBody, err := json.Marshal(mockResponse)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			slog.ErrorContext(ctx, "Failed to marshal mock response", "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to marshal response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(responseBody)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to write response", "error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to write response")
			return
		}

		slog.InfoContext(ctx, "Sent mock response", "user_id", chatReq.UserID, "response_length", len(responseBody))
		span.SetStatus(codes.Ok, "Response sent")
	})

	server := &http.Server{
		Addr:         ":" + cfg.MockBackendPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		slog.Info(fmt.Sprintf("Mock Chat Backend listening on http://localhost:%s", cfg.MockBackendPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Mock backend server failed to start", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down mock backend server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Mock backend server forced to shutdown", "error", err)
	} else {
		slog.Info("Mock backend server gracefully stopped.")
	}
}
