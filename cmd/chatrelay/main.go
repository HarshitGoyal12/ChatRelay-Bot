package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"chatrelay-bot/internal/bot"
	"chatrelay-bot/internal/chatbackend"
	"chatrelay-bot/internal/config"
	"chatrelay-bot/internal/slack"
	"chatrelay-bot/internal/telemetry"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load application configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := telemetry.InitOpenTelemetry(ctx, cfg); err != nil {
		slog.Error("Failed to initialize OpenTelemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		telemetry.ShutdownOpenTelemetry(shutdownCtx)
	}()

	slog.Info("ChatRelay Bot is starting...")
	backendClient := chatbackend.NewClient(cfg.ChatBackendURL, cfg.RequestTimeout, cfg.BackendAPIRetryCount, cfg.BackendAPIRetryDelay)
	slog.Info("Chat backend client initialized", "url", cfg.ChatBackendURL)

	chatRelayBot := bot.NewChatRelayBot(nil, backendClient)


	slackClient := slack.NewClient(cfg.SlackBotToken, cfg.SlackAppToken, chatRelayBot, cfg.SlackAPIRetryCount, cfg.SlackAPIRetryDelay)
	chatRelayBot.SetSlackClient(slackClient)

	slog.Info("Slack client initialized",
		"bot_token_prefix", cfg.SlackBotToken[:5]+"...",
		"app_token_prefix", cfg.SlackAppToken[:5]+"...",
	)


	slog.Info("Connecting to Slack and starting event listener...")
	err = chatRelayBot.StartBot(ctx)
	if err != nil {
		slog.Error("ChatRelay Bot failed to start or stopped with an error", "error", err)
		os.Exit(1)
	}

	slog.Info("ChatRelay Bot stopped gracefully.")
}
