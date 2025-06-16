package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"chatrelay-bot/pkg/models"
)

var (
	tp *sdktrace.TracerProvider
	lp *log.LoggerProvider
)

func InitOpenTelemetry(ctx context.Context, cfg *models.AppConfig) error {
	var err error

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Trace Exporter
	switch cfg.TelemetryExporter {
	case "grpc":
		tp, err = initGRPCTracerProvider(ctx, cfg.TelemetryEndpoint, res)
	case "http/protobuf":
		tp, err = initHTTPTracerProvider(ctx, cfg.TelemetryEndpoint, res)
	case "console":
		tp, err = initConsoleTracerProvider(ctx, res)
	default:
		return fmt.Errorf("unsupported telemetry exporter: %s", cfg.TelemetryExporter)
	}
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Log Exporter
	switch cfg.TelemetryExporter {
	case "grpc":
		lp, err = initGRPCLoggerProvider(ctx, cfg.TelemetryEndpoint, res)
	case "http/protobuf":
		lp, err = initHTTPLoggerProvider(ctx, cfg.TelemetryEndpoint, res)
	case "console":
		lp, err = initConsoleLoggerProvider(ctx, res)
	default:
		fmt.Printf("Warning: Unsupported log exporter %s. Using console.\n", cfg.TelemetryExporter)
		lp, err = initConsoleLoggerProvider(ctx, res)
	}
	if err != nil {
		return err
	}

	// Simplified slog setup using default TextHandler
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	// Host and Runtime metrics
	if err := host.Start(); err != nil {
		fmt.Printf("Warning: host instrumentation: %v\n", err)
	}
	if err := runtime.Start(runtime.WithMeterProvider(
		sdkmetric.NewMeterProvider(sdkmetric.WithResource(res)),
	)); err != nil {
		fmt.Printf("Warning: runtime instrumentation: %v\n", err)
	}

	fmt.Println("OpenTelemetry initialized.")
	return nil
}

func initGRPCTracerProvider(ctx context.Context, endpoint string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	), nil
}

func initHTTPTracerProvider(ctx context.Context, endpoint string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithURLPath("/v1/traces"),
	)
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	), nil
}

func initConsoleTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	), nil
}

func initGRPCLoggerProvider(ctx context.Context, endpoint string, res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	return log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	), nil
}

func initHTTPLoggerProvider(ctx context.Context, endpoint string, res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
		otlploghttp.WithURLPath("/v1/logs"),
	)
	if err != nil {
		return nil, err
	}
	return log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	), nil
}

func initConsoleLoggerProvider(ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	return log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	), nil
}

func ShutdownOpenTelemetry(ctx context.Context) {
	if tp != nil {
		fmt.Println("Shutting down tracer provider...")
		if err := tp.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down tracer: %v\n", err)
		}
	}
	if lp != nil {
		fmt.Println("Shutting down logger provider...")
		if err := lp.Shutdown(ctx); err != nil {
			fmt.Printf("Error shutting down logger: %v\n", err)
		}
	}
}
