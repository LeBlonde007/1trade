package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/trade1/surveillance/internal/detect"
)

// TestStoreIntegration: idempotent save, filtered listing, append-only. Skips without DATABASE_URL.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Microsecond)
	id := "sva_" + strings.Repeat("0", 8) + now.Format("150405.000000")[:6] + "abcdefabcd"
	a := detect.Alert{AlertID: id, Rule: detect.RuleExcessCancel, Severity: "medium", Action: detect.ActionRateLimit,
		TenantID: "tenant-" + id, ProductID: "EAI-IDX", IsPaper: true, WindowStart: now, WindowEnd: now, DetectedAt: now,
		Evidence: map[string]any{"orders": 41}, TradeIDs: []string{"t1"}, OrderIDs: []string{}}
	if isNew, err := s.SaveAlert(ctx, a); err != nil || !isNew {
		t.Fatalf("first save: %v %v", isNew, err)
	}
	if isNew, err := s.SaveAlert(ctx, a); err != nil || isNew {
		t.Fatalf("second save: new=%v err=%v (want not new)", isNew, err)
	}
	paper := true
	got, err := s.ListAlerts(ctx, Filter{TenantID: a.TenantID, IsPaper: &paper})
	if err != nil || len(got) != 1 || got[0].Evidence["orders"] != float64(41) || !got[0].DetectedAt.Equal(now) {
		t.Fatalf("list = %+v, %v", got, err)
	}
	if got, _ := s.ListAlerts(ctx, Filter{TenantID: a.TenantID, Rule: detect.RuleWash}); len(got) != 0 {
		t.Errorf("rule filter ignored: %d rows", len(got))
	}
	if _, err := s.pool.Exec(ctx, `UPDATE surveillance_alerts SET severity='low' WHERE alert_id=$1`, id); err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Errorf("update not refused: %v", err)
	}
}
