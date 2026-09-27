package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trade1/surveillance/internal/detect"
	"github.com/trade1/surveillance/internal/store"
)

// fakeLister records the filter it was asked for.
type fakeLister struct{ got store.Filter }

// ListAlerts records f and returns one alert.
func (f *fakeLister) ListAlerts(_ context.Context, flt store.Filter) ([]detect.Alert, error) {
	f.got = flt
	return []detect.Alert{{AlertID: "sva_1", Rule: detect.RuleWash}}, nil
}

// Ping always succeeds.
func (f *fakeLister) Ping(context.Context) error { return nil }

// TestAlertsAPI: service token required (constant-time), filters and limit parsed, bad limit refused.
func TestAlertsAPI(t *testing.T) {
	fl := &fakeLister{}
	h := New(fl, "svc")
	call := func(path, tok string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if w := call("/v1/surveillance/alerts", ""); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := call("/v1/surveillance/alerts", "wrong"); w.Code != 401 {
		t.Errorf("wrong token: %d", w.Code)
	}
	unset := New(fl, "")
	req := httptest.NewRequest(http.MethodGet, "/v1/surveillance/alerts", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	unset.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("unset token must refuse everyone: %d", rec.Code)
	}
	w := call("/v1/surveillance/alerts?rule=wash_trade&tenant_id=T1&is_paper=true&limit=7", "svc")
	var out struct{ Alerts []detect.Alert }
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != 200 || len(out.Alerts) != 1 || fl.got.Rule != "wash_trade" || fl.got.TenantID != "T1" || fl.got.IsPaper == nil || !*fl.got.IsPaper || fl.got.Limit != 7 {
		t.Errorf("list: %d %+v filter %+v", w.Code, out, fl.got)
	}
	if w := call("/v1/surveillance/alerts?limit=9999", "svc"); w.Code != 422 {
		t.Errorf("bad limit: %d", w.Code)
	}
}
