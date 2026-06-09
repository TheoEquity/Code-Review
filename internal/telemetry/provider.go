package telemetry

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Global OTel providers. Set by Init, used via otel.GetTracerProvider() / otel.GetMeterProvider().
var (
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	shutdownFuncs  []func(context.Context) error
	initialized    bool
	initOnce       sync.Once
)

// initResult holds the outcome of a one-time telemetry initialization.
type initResult struct {
	ok    bool
	ready bool
}

// serviceName holds the name set during Init.
var serviceName = "open-code-review"

// cachedResult stores the outcome of Init for safe concurrent reads.
var cachedResult initResult

// Init initializes global TracerProvider and MeterProvider based on
// environment variables and optional config file. Returns true when enabled.
// Safe to call multiple times, even concurrently.
func Init(ctx context.Context) bool {
	initOnce.Do(func() {
		cachedResult = doInit(ctx)
		initialized = true
	})
	return cachedResult.ready
}

// doInit performs the actual one-time initialization. Returns ok=true when
// config was loaded and ready=true when providers were set up.
func doInit(ctx context.Context) initResult {
	cfg := ResolveConfig(HomeConfigPath())
	serviceName = cfg.ServiceName
	if !cfg.Enabled {
		return initResult{ok: true}
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ocr] WARNING: failed to create OTel resource: %v\n", err)
		res = resource.Default()
	}

	switch cfg.Exporter {
	case "otlp":
		initOTLPProviders(ctx, res, cfg)
	default:
		initConsoleProviders(res)
	}

	if tracerProvider != nil {
		otel.SetTracerProvider(tracerProvider)
	}
	if meterProvider != nil {
		otel.SetMeterProvider(meterProvider)
	}

	return initResult{ok: true, ready: len(shutdownFuncs) > 0}
}

// IsEnabled returns true when telemetry has been initialized with exporters.
func IsEnabled() bool {
	return initialized && len(shutdownFuncs) > 0
}

// ContentLogging returns true when content logging is enabled via OCR_CONTENT_LOGGING.
func ContentLogging() bool {
	if !IsEnabled() {
		return false
	}
	cfg := ResolveConfig("")
	return cfg.ContentLog
}
