package obs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// incoming is a caller's W3C trace context: trace id 4bf9…4736, sampled.
const (
	incomingTrace = "4bf92f3577b34da6a3ce929d0e0e4736"
	incoming      = "00-" + incomingTrace + "-00f067aa0ba902b7-01"
)

// recordSpans installs an in-memory tracer provider for one test.
func recordSpans(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	if _, err := InitTracing(context.Background(), "test"); err != nil { // no endpoint: propagator only
		t.Fatal(err)
	}
	exp := tracetest.NewInMemoryExporter()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })
	return exp
}

// TestTraceCrossesServices: A (instrumented) calls B through Transport. The caller's trace continues
// through A's server span, A's client span and B's server span, and B receives that trace id.
func TestTraceCrossesServices(t *testing.T) {
	exp := recordSpans(t)
	var gotAtB atomic.Value
	b := httptest.NewServer(Instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAtB.Store(r.Header.Get("traceparent"))
	})))
	defer b.Close()
	client := &http.Client{Transport: Transport(nil)}
	a := httptest.NewServer(Instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, b.URL+"/v1/credits/balances", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		resp.Body.Close()
	})))
	defer a.Close()

	req, _ := http.NewRequest(http.MethodPost, a.URL+"/v1/chat/completions", nil)
	req.Header.Set("traceparent", incoming)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if tp, _ := gotAtB.Load().(string); !strings.Contains(tp, incomingTrace) {
		t.Fatalf("B received traceparent %q, want trace %s", tp, incomingTrace)
	}
	spans := exp.GetSpans()
	names := map[string]int{}
	for _, s := range spans {
		if s.SpanContext.TraceID().String() != incomingTrace {
			t.Fatalf("span %q is in trace %s, not the caller's", s.Name, s.SpanContext.TraceID())
		}
		if strings.Contains(s.Name, "/") {
			t.Fatalf("span name %q carries a path (ids would explode cardinality)", s.Name)
		}
		names[s.Name]++
	}
	if len(spans) != 3 || names["HTTP POST"] != 1 || names["HTTP GET"] != 2 {
		t.Fatalf("spans = %v, want A's server span, A's client span, B's server span", names)
	}
}

// TestProbesAreNotTraced keeps health checks and scrapes out of traces.
func TestProbesAreNotTraced(t *testing.T) {
	exp := recordSpans(t)
	srv := httptest.NewServer(Instrument(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))
	defer srv.Close()
	for _, p := range []string{"/healthz", "/readyz", "/metrics"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	if n := len(exp.GetSpans()); n != 0 {
		t.Fatalf("%d probe spans recorded", n)
	}
}

// TestExportsOverOTLP: with an endpoint set, spans reach an OTLP/HTTP receiver on shutdown.
func TestExportsOverOTLP(t *testing.T) {
	var posts atomic.Int32
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path == "/v1/traces" && r.Method == http.MethodPost && len(body) > 0 {
			posts.Add(1)
		}
	}))
	defer recv.Close()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", recv.URL)
	prev := otel.GetTracerProvider()
	defer otel.SetTracerProvider(prev)
	shutdown, err := InitTracing(context.Background(), "inference-gateway")
	if err != nil {
		t.Fatal(err)
	}
	_, span := otel.Tracer("test").Start(context.Background(), "work")
	span.End()
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if posts.Load() == 0 {
		t.Fatal("no spans reached the OTLP receiver")
	}
}

// TestContextCrossesAMessage: what InjectHeader writes (e.g. into NATS headers), ExtractHeader reads
// back, so a consumer span joins the producer's trace.
func TestContextCrossesAMessage(t *testing.T) {
	exp := recordSpans(t)
	ctx, producer := StartSpan(context.Background(), "publish")
	h := map[string][]string{} // message headers keep keys as written
	InjectHeader(ctx, h)
	producer.End()
	_, consumer := StartSpan(ExtractHeader(context.Background(), h), "consume")
	consumer.End()
	spans := exp.GetSpans()
	if len(spans) != 2 || spans[0].SpanContext.TraceID() != spans[1].SpanContext.TraceID() ||
		spans[1].Parent.SpanID() != spans[0].SpanContext.SpanID() {
		t.Fatalf("consumer span is not the producer's child: %+v", spans)
	}
	// The same context read back through either key form: NATS keeps "traceparent", HTTP
	// canonicalises to "Traceparent".
	for _, key := range []string{"traceparent", "Traceparent"} {
		got := ExtractHeader(context.Background(), map[string][]string{key: h["traceparent"]})
		if sc := trace.SpanContextFromContext(got); sc.TraceID() != spans[0].SpanContext.TraceID() {
			t.Fatalf("key %q: extracted trace %s", key, sc.TraceID())
		}
	}
	if got := ExtractHeader(context.Background(), http.Header{}); got != context.Background() {
		t.Fatal("extracting from empty headers changed the context")
	}
}
