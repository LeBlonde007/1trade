package api

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/trade1/inference-gateway/internal/catalog"
)

// clock is the server's time source for lifecycle decisions.
func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// resolveModel looks a model up for serving and applies its lifecycle (F10):
//   - unknown: 404 model_not_found;
//   - retired (past its sunset): 410 model_retired, naming the replacement;
//   - deprecated: served, with Deprecation (RFC 9745), Sunset (RFC 8594) and, when there is one, a
//     successor Link header, so SDKs and the CLI can warn.
//
// ok=false means a response has been written and the handler must return.
func (s *Server) resolveModel(w http.ResponseWriter, id string) (catalog.Model, bool) {
	m, found := catalog.LookupAt(id, s.clock())
	if !found {
		writeErr(w, http.StatusNotFound, "model_not_found", "unknown model: "+id)
		return catalog.Model{}, false
	}
	d := m.Trade1.Deprecation
	switch m.Trade1.Status {
	case catalog.StatusRetired:
		msg := id + " was retired on " + d.SunsetAt.UTC().Format(time.DateOnly)
		if d.Replacement != "" {
			msg += "; use " + d.Replacement
		}
		writeJSON(w, http.StatusGone, map[string]any{
			"code": "model_retired", "message": msg,
			"details": map[string]any{"sunset_at": d.SunsetAt.UTC(), "replacement": d.Replacement},
		})
		return catalog.Model{}, false
	case catalog.StatusDeprecated:
		w.Header().Set("Deprecation", "@"+strconv.FormatInt(d.AnnouncedAt.Unix(), 10))
		w.Header().Set("Sunset", d.SunsetAt.UTC().Format(http.TimeFormat))
		if d.Replacement != "" {
			w.Header().Set("Link", `</v1/models/`+d.Replacement+`>; rel="successor-version"`)
		}
	}
	return m, true
}

// latencyWindow is how many recent requests per model the median is taken over.
const latencyWindow = 200

// latencies keeps a rolling window of served-request latencies per model, for the catalog's
// measured latency_p50_ms. In-process: each gateway replica reports what it has served.
type latencies struct {
	mu sync.Mutex
	by map[string][]int
}

// newLatencies returns an empty tracker.
func newLatencies() *latencies { return &latencies{by: map[string][]int{}} }

// record adds one served request's latency. Non-positive values (unmeasured paths) are ignored.
func (l *latencies) record(model string, ms int) {
	if ms <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.by[model] = append(l.by[model], ms)
	if w := l.by[model]; len(w) > latencyWindow {
		l.by[model] = w[len(w)-latencyWindow:]
	}
}

// p50 is the median of the model's window, or nil if it has served nothing measured.
func (l *latencies) p50(model string) *int {
	l.mu.Lock()
	w := append([]int(nil), l.by[model]...)
	l.mu.Unlock()
	if len(w) == 0 {
		return nil
	}
	sort.Ints(w)
	v := w[(len(w)-1)/2]
	return &v
}

// eventModality maps a catalog modality onto inference.usage.v1's billing enum (text, speech, image,
// video, embeddings). The catalog is finer-grained — vision, docs, code and agent models bill text;
// transcription bills speech — and emitting those raw broke the event contract.
func eventModality(m catalog.Model) string {
	switch m.Trade1.Modality {
	case "text", "speech", "image", "video", "embeddings":
		return m.Trade1.Modality
	case "transcription":
		return "speech"
	}
	return m.Trade1.CreditType // vision, docs, code, agent → their credit (text)
}

// elapsedMS is the time since start in whole milliseconds, rounded up: a sub-millisecond response
// counts as 1 ms. A 0 therefore always means "not measured" (the latency tracker ignores it).
func elapsedMS(start time.Time) int {
	d := time.Since(start)
	return int((d + time.Millisecond - 1) / time.Millisecond)
}
