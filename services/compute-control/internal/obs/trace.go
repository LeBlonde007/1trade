package obs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Tracing (OpenTelemetry). Like the metrics half, this file is copied verbatim into each Go service.
//
// Every service always accepts and forwards W3C trace context (traceparent/tracestate), so a trace that
// crosses it is never broken. Spans are exported only when OTEL_EXPORTER_OTLP_ENDPOINT (or
// OTEL_EXPORTER_OTLP_TRACES_ENDPOINT) is set; everything else — headers, sampler
// (OTEL_TRACES_SAMPLER), resource attributes — comes from the standard OTEL_* variables.

// InitTracing installs the global propagator and, when an OTLP endpoint is configured, a batching
// tracer provider named service. The returned func flushes and stops it; call it on shutdown.
func InitTracing(ctx context.Context, service string) (shutdown func(context.Context) error, err error) {
	// Trace context only. Baggage is deliberately not propagated: nothing uses it, and at the public
	// gateway it would carry caller-chosen data into every internal call and message.
	otel.SetTextMapPropagator(propagation.TraceContext{})
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("obs: otlp exporter: %w", err)
	}
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(attribute.String("service.name", service)))
	if err != nil {
		return nil, fmt.Errorf("obs: trace resource: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// untraced are probe and scrape paths: high-frequency and never interesting in a trace.
var untraced = map[string]bool{"/healthz": true, "/readyz": true, "/metrics": true}

// Trace wraps an inbound handler with a server span that continues the caller's trace. Span names
// are "HTTP <method>" — never the path, which carries ids.
func Trace(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "http.server",
		otelhttp.WithFilter(func(r *http.Request) bool { return !untraced[r.URL.Path] }),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return "HTTP " + r.Method }))
}

// Transport wraps an outbound transport (nil means http.DefaultTransport) so each call gets a client
// span and carries the current trace context. Use it only for calls to 1Trade's own services: sending
// traceparent to a third party leaks internal trace ids.
func Transport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return "HTTP " + r.Method }))
}

// InjectHeader writes the current trace context into message headers h (e.g. a NATS msg.Header;
// http.Header works too). Outbound HTTP is already covered by Transport.
func InjectHeader(ctx context.Context, h map[string][]string) {
	otel.GetTextMapPropagator().Inject(ctx, headerCarrier(h))
}

// ExtractHeader returns ctx carrying the trace context found in message headers h (unchanged if there
// is none).
func ExtractHeader(ctx context.Context, h map[string][]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, headerCarrier(h))
}

// headerCarrier adapts a string-slice header map to OTel. Keys are written exactly as the propagator
// gives them (lowercase W3C names) and read case-insensitively: HTTP canonicalises keys
// ("Traceparent") while NATS keeps them as sent ("traceparent"), and propagation.HeaderCarrier, which
// canonicalises the lookup, misses the NATS form — the trace would silently break at every message.
type headerCarrier map[string][]string

// Get returns the first value for key, matching the key case-insensitively.
func (h headerCarrier) Get(key string) string {
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	for k, v := range h {
		if len(v) > 0 && strings.EqualFold(k, key) {
			return v[0]
		}
	}
	return ""
}

// Set replaces key's values with value.
func (h headerCarrier) Set(key, value string) { h[key] = []string{value} }

// Keys lists the header keys.
func (h headerCarrier) Keys() []string {
	out := make([]string, 0, len(h))
	for k := range h {
		out = append(out, k)
	}
	return out
}

// StartSpan starts a span from the global tracer (e.g. a consumer span for one message). Its name
// must be low-cardinality: never an id.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("1trade").Start(ctx, name, opts...)
}
