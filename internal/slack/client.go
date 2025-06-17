package slack

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"chatrelay-bot/pkg/models"
)

const tracerName = "chatrelay/internal/slack"

type EventHandler interface {
	HandleAppMention(ctx context.Context, event models.SlackEvent) error
}

type Client struct {
	api          *slack.Client
	socketClient *socketmode.Client
	eventHandler EventHandler
	botUserID    string
	retryCount   int
	retryDelay   time.Duration
}

func NewClient(botToken, appToken string, handler EventHandler, retryCount int, retryDelay time.Duration) *Client {
	slackGoLogger := log.New(log.Writer(), "[slack-go] ", log.LstdFlags)

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionLog(slackGoLogger),
		slack.OptionHTTPClient(NewHTTPClientWithTracing()),
	)

	socketClient := socketmode.New(
		api,
		socketmode.OptionLog(slackGoLogger),
	)

	return &Client{
		api:          api,
		socketClient: socketClient,
		eventHandler: handler,
		botUserID:    "",
		retryCount:   retryCount,
		retryDelay:   retryDelay,
	}
}

func (c *Client) ConnectAndListen(ctx context.Context) error {
	authTest, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate with Slack: %w", err)
	}
	c.botUserID = authTest.UserID
	slog.InfoContext(ctx, "Slack bot connected", "bot_id", c.botUserID, "bot_name", authTest.User)

	go c.listenForEvents(ctx)

	slog.InfoContext(ctx, "Connecting to Slack Socket Mode...")
	return c.socketClient.RunContext(ctx)
}

func (c *Client) listenForEvents(ctx context.Context) {
	tracer := otel.Tracer(tracerName)

	for evt := range c.socketClient.Events {
		eventCtx, span := tracer.Start(ctx, "SlackEventProcessing",
			trace.WithAttributes(attribute.String("slack.event_type", string(evt.Type))),
		)

		switch evt.Type {
		case socketmode.EventTypeConnecting:
			slog.InfoContext(eventCtx, "Connecting to Slack with Socket Mode...", "attempt", evt.Data)
		case socketmode.EventTypeConnectionError:
			slog.ErrorContext(eventCtx, "Connection error to Slack Socket Mode", "error", evt.Data)
		case socketmode.EventTypeConnected:
			slog.InfoContext(eventCtx, "Successfully connected to Slack Socket Mode.")
		case socketmode.EventTypeDisconnect:
			slog.WarnContext(eventCtx, "Disconnected from Slack Socket Mode.")
		case socketmode.EventTypeEventsAPI:
			c.socketClient.Ack(*evt.Request)
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				slog.ErrorContext(eventCtx, "Could not parse EventsAPIEvent", "raw_data", evt.Data)
				span.RecordError(fmt.Errorf("could not parse EventsAPIEvent"))
				span.SetStatus(codes.Error, "Event parsing error")
				span.End()
				continue
			}
			eventCtx, eventSpan := tracer.Start(eventCtx, "EventsAPIEvent",
				trace.WithAttributes(attribute.String("slack.events_api.type", eventsAPIEvent.Type)),
			)
			c.handleEventsAPIEvent(eventCtx, eventsAPIEvent)
			eventSpan.End()
		case socketmode.EventTypeInteractive:
			c.socketClient.Ack(*evt.Request)
		default:
			slog.InfoContext(eventCtx, "Unhandled event type", "event_type", evt.Type, "data", evt.Data)
		}
		span.End()
	}
}

func (c *Client) handleEventsAPIEvent(ctx context.Context, eventsAPIEvent slackevents.EventsAPIEvent) {
	tracer := otel.Tracer(tracerName)

	switch eventsAPIEvent.Type {
	case slackevents.CallbackEvent:
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			slog.InfoContext(ctx, "Received App Mention Event", "text", ev.Text, "user", ev.User, "channel", ev.Channel)
			mentionCtx, mentionSpan := tracer.Start(ctx, "HandleAppMentionEvent",
				trace.WithAttributes(
					attribute.String("slack.event.type", "app_mention"),
					attribute.String("slack.event.channel", ev.Channel),
					attribute.String("slack.event.user", ev.User),
				),
			)
			defer mentionSpan.End()

			botMentionRegex := regexp.MustCompile(fmt.Sprintf("<@%s>", c.botUserID))
			query := strings.TrimSpace(botMentionRegex.ReplaceAllString(ev.Text, ""))

			slackEvent := models.SlackEvent{
				Type:    innerEvent.Type,
				Channel: ev.Channel,
				User:    ev.User,
				Text:    query,
				Ts:      ev.TimeStamp,
			}

			if err := c.eventHandler.HandleAppMention(mentionCtx, slackEvent); err != nil {
				slog.ErrorContext(mentionCtx, "Error handling app mention event", "error", err, "user", ev.User, "channel", ev.Channel)
				mentionSpan.RecordError(err)
				mentionSpan.SetStatus(codes.Error, "Error handling app mention")
				c.SendMessage(ctx, ev.Channel, fmt.Sprintf("Oops! Something went wrong: %v", err))
			} else {
				mentionSpan.SetStatus(codes.Ok, "App mention handled successfully")
			}
		default:
			slog.InfoContext(ctx, "Unhandled inner event type", "type", innerEvent.Type)
		}
	default:
		slog.InfoContext(ctx, "Unhandled Events API event type", "type", eventsAPIEvent.Type)
	}
}

func (c *Client) SendMessage(ctx context.Context, channelID, text string) (string, error) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "SendMessageToSlack",
		trace.WithAttributes(
			attribute.String("slack.channel_id", channelID),
			attribute.Int("slack.message_length", len(text)),
		),
	)
	defer span.End()

	slog.InfoContext(ctx, "Attempting to send message", "channel", channelID, "text", text)

	for i := 0; i <= c.retryCount; i++ {
		_, ts, err := c.api.PostMessageContext(ctx, channelID, slack.MsgOptionText(text, false))
		if err == nil {
			slog.InfoContext(ctx, "Message sent successfully", "channel", channelID, "timestamp", ts)
			span.SetStatus(codes.Ok, "success")
			return ts, nil
		}
		slog.ErrorContext(ctx, "Failed to send message", "error", err, "attempt", i+1)
		span.AddEvent("SlackSendMessageAttemptFailed", trace.WithAttributes(
			attribute.Int("attempt", i+1),
			attribute.String("error", err.Error()),
		))
		time.Sleep(c.retryDelay)
	}

	err := fmt.Errorf("failed to send message after %d retries", c.retryCount)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return "", err
}

func (c *Client) UpdateMessage(ctx context.Context, channelID, timestamp, text string) error {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "UpdateMessageInSlack",
		trace.WithAttributes(
			attribute.String("slack.channel_id", channelID),
			attribute.String("slack.message_ts", timestamp),
		),
	)
	defer span.End()

	slog.InfoContext(ctx, "Attempting to update message", "channel", channelID, "timestamp", timestamp, "text", text)

	for i := 0; i <= c.retryCount; i++ {
		_, _, _, err := c.api.UpdateMessageContext(ctx, channelID, timestamp, slack.MsgOptionText(text, false))
		if err == nil {
			slog.InfoContext(ctx, "Message updated successfully", "channel", channelID, "timestamp", timestamp)
			span.SetStatus(codes.Ok, "success")
			return nil
		}
		slog.ErrorContext(ctx, "Failed to update message", "error", err, "attempt", i+1)
		span.AddEvent("SlackUpdateMessageAttemptFailed", trace.WithAttributes(
			attribute.Int("attempt", i+1),
			attribute.String("error", err.Error()),
		))
		time.Sleep(c.retryDelay)
	}

	err := fmt.Errorf("failed to update message after %d retries", c.retryCount)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func NewHTTPClientWithTracing() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		),
	}
}