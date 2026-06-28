package telemetry

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/attribute"
)

// MultiHandler forwards slog Records to multiple handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m.handlers {
		if err := h.Handle(ctx, r); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

// Telemetry holds reference to resources to be cleaned up.
type Telemetry struct {
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
}

// Shutdown flushes and closes all telemetry exporters.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if t.loggerProvider != nil {
		if err := t.loggerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// InitTelemetry sets up OTLP HTTP metrics and logs exporters, and redirects slog/standard log.
func InitTelemetry(ctx context.Context, otlpEndpoint string) (*Telemetry, error) {
	// 1. Create standard resource attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("abel"),
			attribute.String("app.role", "server"),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	var metricOpts []otlpmetrichttp.Option
	var logOpts []otlploghttp.Option

	if otlpEndpoint != "" {
		endpointHost := otlpEndpoint
		isInsecure := true
		if strings.HasPrefix(otlpEndpoint, "http://") {
			endpointHost = strings.TrimPrefix(otlpEndpoint, "http://")
			isInsecure = true
		} else if strings.HasPrefix(otlpEndpoint, "https://") {
			endpointHost = strings.TrimPrefix(otlpEndpoint, "https://")
			isInsecure = false
		}

		metricOpts = append(metricOpts, otlpmetrichttp.WithEndpoint(endpointHost))
		logOpts = append(logOpts, otlploghttp.WithEndpoint(endpointHost))

		if isInsecure {
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
			logOpts = append(logOpts, otlploghttp.WithInsecure())
		}
	}

	// 2. Set up metric provider (OTLP/HTTP)
	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		// Fallback or warning instead of crashing
		slog.Warn("Failed to create OTLP metric exporter, metrics will be disabled", "error", err)
	}

	var mp *sdkmetric.MeterProvider
	if metricExporter != nil {
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
		)
		otel.SetMeterProvider(mp)
	}

	// 3. Set up logging provider (OTLP/HTTP)
	logExporter, err := otlploghttp.New(ctx, logOpts...)
	if err != nil {
		slog.Warn("Failed to create OTLP log exporter, OTel logging will be disabled", "error", err)
	}

	var lp *sdklog.LoggerProvider
	var otelHandler slog.Handler
	if logExporter != nil {
		lp = sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		)
		global.SetLoggerProvider(lp)
		otelHandler = otelslog.NewHandler("abel")
	}

	// 4. Set up MultiHandler (Stdout + OTel)
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	var activeHandler slog.Handler
	if otelHandler != nil {
		activeHandler = NewMultiHandler(stdoutHandler, otelHandler)
	} else {
		activeHandler = stdoutHandler
	}

	logger := slog.New(activeHandler)
	slog.SetDefault(logger)

	// 5. Redirect standard logger
	log.SetFlags(0)
	log.SetOutput(slog.NewLogLogger(activeHandler, slog.LevelInfo).Writer())

	// Initialize metrics definitions
	if err := InitMetrics(); err != nil {
		slog.Error("Failed to initialize OTel metrics", "error", err)
	}

	return &Telemetry{
		meterProvider:  mp,
		loggerProvider: lp,
	}, nil
}
