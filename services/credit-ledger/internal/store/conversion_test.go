package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/google/uuid"
)

// TestApplyConversion checks the atomic two-leg conversion: burn from + mint to (value − spread),
// both chained; idempotent replay; insufficient-credit; and the hash chain still verifies.
func TestApplyConversion(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping conversion integration test")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer s.Close()
	tenant := uuid.NewString()
	mk := func(x string) domain.Money { m, _ := domain.ParseMoney(x); return m }

	// Seed 1000 ai_index.
	if _, err := s.ApplyMovement(ctx, Movement{
		TenantID: tenant, CreditType: "ai_index", Operation: domain.OpPurchase, Amount: mk("1000"),
		IdempotencyKey: "seed", IsPaper: true,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Convert 100 ai_index → text (rate 0.830579, 1% spread → 82.227321).
	debit, credit, rate, spread, err := s.ApplyConversion(ctx, Conversion{
		TenantID: tenant, From: "ai_index", To: "text", Amount: mk("100"), IsPaper: true, IdempotencyKey: "conv1",
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if debit.Amount.String() != "-100.000000" || debit.BalanceAfter.String() != "900.000000" {
		t.Fatalf("burn leg wrong: amount=%s after=%s", debit.Amount, debit.BalanceAfter)
	}
	if credit.Amount.String() != "82.227321" || credit.BalanceAfter.String() != "82.227321" {
		t.Fatalf("mint leg wrong: amount=%s after=%s (rate=%s spread=%s)", credit.Amount, credit.BalanceAfter, rate, spread)
	}

	// Idempotent replay: same key → same legs, balances NOT re-applied.
	d2, c2, _, _, err := s.ApplyConversion(ctx, Conversion{
		TenantID: tenant, From: "ai_index", To: "text", Amount: mk("100"), IsPaper: true, IdempotencyKey: "conv1",
	})
	if err != nil || d2.BalanceAfter.String() != "900.000000" || c2.BalanceAfter.String() != "82.227321" {
		t.Fatalf("replay not idempotent: ai=%s text=%s err=%v", d2.BalanceAfter, c2.BalanceAfter, err)
	}

	// Insufficient source → ErrInsufficientCredit (and nothing applied — atomic).
	if _, _, _, _, err := s.ApplyConversion(ctx, Conversion{
		TenantID: tenant, From: "ai_index", To: "text", Amount: mk("100000"), IsPaper: true, IdempotencyKey: "conv2",
	}); !errors.Is(err, domain.ErrInsufficientCredit) {
		t.Fatalf("expected ErrInsufficientCredit, got %v", err)
	}

	// Unknown pair → ErrNoRate.
	if _, _, _, _, err := s.ApplyConversion(ctx, Conversion{
		TenantID: tenant, From: "ai_index", To: "video", Amount: mk("1"), IsPaper: true, IdempotencyKey: "conv3",
	}); !errors.Is(err, ErrNoRate) {
		t.Fatalf("expected ErrNoRate, got %v", err)
	}

	// The hash chain still verifies for the tenant after the conversion.
	ok, _, firstBad, err := s.VerifyChain(ctx, tenant)
	if err != nil || !ok {
		t.Fatalf("chain verify failed: ok=%v firstBad=%s err=%v", ok, firstBad, err)
	}
}
