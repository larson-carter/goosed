package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// Init configures OpenTelemetry tracing, propagation, and structured logging for a service.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, func(http.Handler) http.Handler, *log.Logger, error) {
	if serviceName == "" {
		return nil, nil, nil, errors.New("telemetry: service name is required")
	}

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil, nil, nil, errors.New("telemetry: OTEL_EXPORTER_OTLP_ENDPOINT is not set")
	}

	exporter, err := newTraceExporter(ctx, endpoint)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("telemetry: create exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("telemetry: create resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logWriter := newJSONLogWriter(serviceName, os.Stdout)
	logger := log.New(logWriter, "", 0)

	middleware := func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(recorder, r)

			spanCtx := trace.SpanFromContext(r.Context()).SpanContext()
			traceID := ""
			if spanCtx.IsValid() {
				traceID = spanCtx.TraceID().String()
			}

			duration := time.Since(start)
			msg := fmt.Sprintf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, duration)
			if err := logWriter.Log("INFO", msg, traceID); err != nil {
				fmt.Fprintf(os.Stderr, "telemetry: failed to write request log: %v\n", err)
			}
		})

		return otelhttp.NewHandler(handler, serviceName)
	}

	shutdown := func(ctx context.Context) error {
		return tracerProvider.Shutdown(ctx)
	}

	return shutdown, middleware, logger, nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func newTraceExporter(ctx context.Context, endpoint string) (*otlptrace.Exporter, error) {
	var opts []otlptracehttp.Option

	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Scheme != "" {
		if parsed.Host == "" {
			return nil, fmt.Errorf("invalid OTLP endpoint: %s", endpoint)
		}
		opts = append(opts, otlptracehttp.WithEndpoint(parsed.Host))
		if parsed.Path != "" && parsed.Path != "/" {
			opts = append(opts, otlptracehttp.WithURLPath(parsed.Path))
		}
		if parsed.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
	} else {
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

type jsonLogWriter struct {
	mu      sync.Mutex
	service string
	out     io.Writer
}

func newJSONLogWriter(service string, out io.Writer) *jsonLogWriter {
	if out == nil {
		out = os.Stdout
	}
	return &jsonLogWriter{service: service, out: out}
}

func (w *jsonLogWriter) Write(p []byte) (int, error) {
	level, message := parseLevel(strings.TrimSpace(string(p)))
	if err := w.Log(level, message, ""); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *jsonLogWriter) Log(level, message, traceID string) error {
	entry := map[string]string{
		"ts":       time.Now().UTC().Format(time.RFC3339Nano),
		"level":    level,
		"service":  w.service,
		"msg":      message,
		"trace_id": traceID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.out.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func parseLevel(message string) (string, string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "INFO", ""
	}

	if strings.HasPrefix(trimmed, "[") {
		if idx := strings.Index(trimmed, "]"); idx > 1 {
			level := strings.ToUpper(trimmed[1:idx])
			rest := strings.TrimSpace(trimmed[idx+1:])
			if isLevel(level) {
				return level, rest
			}
		}
	}

	if idx := strings.Index(trimmed, ":"); idx > 0 {
		level := strings.ToUpper(strings.TrimSpace(trimmed[:idx]))
		rest := strings.TrimSpace(trimmed[idx+1:])
		if isLevel(level) {
			return level, rest
		}
	}

	fields := strings.Fields(trimmed)
	if len(fields) > 1 {
		level := strings.ToUpper(fields[0])
		if isLevel(level) {
			return level, strings.TrimSpace(trimmed[len(fields[0]):])
		}
	}

	return "INFO", trimmed
}

func isLevel(level string) bool {
	switch level {
	case "INFO", "ERROR", "WARN", "WARNING", "DEBUG":
		return true
	default:
		return false
	}
}
