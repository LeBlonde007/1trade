package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/google/uuid"
)

// TestStoreIntegration exercises the real Postgres store: atomic apply, idempotent replay,
// insufficient-credit rejection, and hash-chain integrity. Skips unless DATABASE_URL is set
// (the schema — docs/contracts/schemas/types.sql + migrations/0001_init.sql — must be applied).
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping store integration test")
	}
	ctx := context.Background()
	s, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer s.Close()

	tenant := uuid.NewString() // fresh tenant → isolated from any other data
	money := func(x string) domain.Money {
		m, err := domain.ParseMoney(x)
		if err != nil {
			t.Fatalf("ParseMoney(%q): %v", x, err)
		}
		return m
	}

	// purchase +100 text
	buy, err := s.ApplyMovement(ctx, Movement{
		TenantID: tenant, CreditType: "text", Operation: domain.OpPurchase,
		Amount: money("100"), ReferenceID: "stripe_evt_1", IdempotencyKey: "buy-1", IsPaper: true,
	})
	if err != nil {
		t.Fatalf("purchase: %v", err)
	}
	if buy.BalanceAfter.String() != "100.000000" {
		t.Fatalf("after purchase = %s, want 100.000000", buy.BalanceAfter)
	}

	// debit -12.5
	deb, err := s.ApplyMovement(ctx, Movement{
		TenantID: tenant, CreditType: "text", Operation: domain.OpConsumption,
		Amount: money("-12.5"), IdempotencyKey: "use-1", IsPaper: true,
	})
	if err != nil {
		t.Fatalf("debit: %v", err)
	}
	if deb.BalanceAfter.String() != "87.500000" {
		t.Fatalf("after debit = %s, want 87.500000", deb.BalanceAfter)
	}

	// idempotent replay of the same debit → same tx, balance unchanged
	deb2, err := s.ApplyMovement(ctx, Movement{
		TenantID: tenant, CreditType: "text", Operation: domain.OpConsumption,
		Amount: money("-12.5"), IdempotencyKey: "use-1", IsPaper: true,
	})
	if err != nil {
		t.Fatalf("idempotent debit: %v", err)
	}
	if deb2.TxID != deb.TxID {
		t.Fatalf("idempotent replay made a new tx (%s != %s)", deb2.TxID, deb.TxID)
	}
	bals, err := s.GetBalances(ctx, tenant, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(bals) != 1 || bals[0].Balance.String() != "87.500000" {
		t.Fatalf("balance changed on idempotent replay: %+v", bals)
	}

	// insufficient credit
	if _, err := s.ApplyMovement(ctx, Movement{
		TenantID: tenant, CreditType: "text", Operation: domain.OpConsumption,
		Amount: money("-1000"), IdempotencyKey: "use-2", IsPaper: true,
	}); !errors.Is(err, domain.ErrInsufficientCredit) {
		t.Fatalf("expected ErrInsufficientCredit, got %v", err)
	}

	// hash chain intact across the persisted transactions
	ok, checked, firstBad, err := s.VerifyChain(ctx, tenant)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatalf("chain broken at %s", firstBad)
	}
	if checked < 2 {
		t.Fatalf("expected >=2 transactions checked, got %d", checked)
	}
}
