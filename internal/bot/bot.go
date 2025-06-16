package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"   // ✅ Correct package
	"go.opentelemetry.io/otel/trace"

	"chatrelay-bot/internal/chatbackend"
	"chatrelay-bot/internal/slack"
	"chatrelay-bot/pkg/models"
)

const tracerName = "chatrelay/internal/bot"

type ChatRelayBot struct {
	slackClient         *slack.Client
	backendClient       chatbackend.Client
	ongoingConversations map[string]string
	mu                   sync.Mutex
}

func NewChatRelayBot(sc *slack.Client, bc chatbackend.Client) *ChatRelayBot {
	return &ChatRelayBot{
		slackClient:         sc,
		backendClient:       bc,
		ongoingConversations: make(map[string]string),
	}
}

func (b *ChatRelayBot) SetSlackClient(sc *slack.Client) {
	b.slackClient = sc
}

func (b *ChatRelayBot) StartBot(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting ChatRelay Bot...")
	if b.slackClient == nil {
		return fmt.Errorf("slack client is not set for ChatRelayBot")
	}
	return b.slackClient.ConnectAndListen(ctx)
}

func (b *ChatRelayBot) HandleAppMention(ctx context.Context, event models.SlackEvent) error {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "HandleAppMention",
		trace.WithAttributes(
			attribute.String("slack.event.channel", event.Channel),
			attribute.String("slack.event.user", event.User),
			attribute.String("slack.event.query", event.Text),
		),
	)
	defer span.End()

	slog.InfoContext(ctx, "Processing app mention", "user", event.User, "channel", event.Channel, "query", event.Text)

	if b.slackClient == nil {
		slog.ErrorContext(ctx, "Slack client is nil, cannot send messages.")
		span.RecordError(fmt.Errorf("slack client not initialized"))
		span.SetStatus(codes.Error, "Slack client not initialized") // ✅ Correct usage
		return fmt.Errorf("slack client not initialized")
	}

	initialMessage := "Thinking..."
	ts, err := b.slackClient.SendMessage(ctx, event.Channel, initialMessage)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send initial message to Slack", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send initial message")
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	conversationKey := fmt.Sprintf("%s-%s", event.Channel, event.User)
	b.mu.Lock()
	b.ongoingConversations[conversationKey] = ts
	b.mu.Unlock()

	chatReq := models.ChatRequest{
		UserID: event.User,
		Query:  event.Text,
	}

	backendRes, err := b.backendClient.SendChatRequest(ctx, chatReq)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get response from chat backend", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Backend request failed")

		updateErr := b.slackClient.UpdateMessage(ctx, event.Channel, ts, fmt.Sprintf("Apologies, I encountered an error: %v", err))
		if updateErr != nil {
			slog.ErrorContext(ctx, "Failed to update message with error", "error", updateErr)
		}
		return fmt.Errorf("backend communication failed: %w", err)
	}

	slog.InfoContext(ctx, "Received response from chat backend", "response_length", len(backendRes.FullResponse))

	fullResponse := backendRes.FullResponse
	var currentResponse strings.Builder
	sentences := splitIntoSentences(fullResponse)

	for i, sentence := range sentences {
		currentResponse.WriteString(sentence)
		if i < len(sentences)-1 {
			currentResponse.WriteString("...")
		}

		err := b.slackClient.UpdateMessage(ctx, event.Channel, ts, currentResponse.String())
		if err != nil {
			slog.ErrorContext(ctx, "Failed to update Slack message during streaming", "error", err)
			span.RecordError(err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	finalMessage := fullResponse + "\n\n_Powered by ChatRelay_"
	err = b.slackClient.UpdateMessage(ctx, event.Channel, ts, finalMessage)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send final Slack message", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send final message")
		return fmt.Errorf("failed to send final message: %w", err)
	}

	slog.InfoContext(ctx, "Successfully relayed response to Slack", "user", event.User)
	span.SetStatus(codes.Ok, "Response relayed successfully") // ✅ Correct usage

	b.mu.Lock()
	delete(b.ongoingConversations, conversationKey)
	b.mu.Unlock()

	return nil
}

func splitIntoSentences(text string) []string {
	sentences := regexp.MustCompile(`([.!?])\s+`).Split(text, -1)
	cleanedSentences := make([]string, 0, len(sentences))
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s != "" {
			cleanedSentences = append(cleanedSentences, s)
		}
	}
	return cleanedSentences
}
