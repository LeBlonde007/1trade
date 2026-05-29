package domain

import (
	"testing"
	"time"
)

// TestMoneyParseFormat checks fixed-point parsing/formatting round-trips and rejects over-precision.
func TestMoneyParseFormat(t *testing.T) {
	cases := []struct{ in, out string }{
		{"0", "0.000000"},
		{"100", "100.000000"},
		{"100.5", "100.500000"},
		{"0.000001", "0.000001"},
		{"-12.5", "-12.500000"},
		{"12345678901234.123456", "12345678901234.123456"}, // beyond int64 micros — big.Int handles it
	}
	for _, c := range cases {
		m, err := ParseMoney(c.in)
		if err != nil {
			t.Fatalf("ParseMoney(%q) error: %v", c.in, err)
		}
		if got := m.String(); got != c.out {
			t.Errorf("ParseMoney(%q).String() = %q, want %q", c.in, got, c.out)
		}
	}
	if _, err := ParseMoney("1.1234567"); err == nil {
		t.Error("expected error for 7 decimal places (sub-micro precision)")
	}
}

// TestMoneyArithmetic checks exact add/sub/compare with no float drift.
func TestMoneyArithmetic(t *testing.T) {
	a, _ := ParseMoney("100.000000")
	b, _ := ParseMoney("12.500000")
	if got := a.Sub(b).String(); got != "87.500000" {
		t.Errorf("100 - 12.5 = %s, want 87.500000", got)
	}
	if got := a.Add(b).String(); got != "112.500000" {
		t.Errorf("100 + 12.5 = %s, want 112.500000", got)
	}
	// classic float trap: 0.1 + 0.2 must be exactly 0.3
	x, _ := ParseMoney("0.1")
	y, _ := ParseMoney("0.2")
	if got := x.Add(y).String(); got != "0.300000" {
		t.Errorf("0.1 + 0.2 = %s, want 0.300000", got)
	}
}

const genesis = "GENESIS"

func mk(txid string, op Operation, amount string) Transaction {
	a, _ := ParseMoney(amount)
	return Transaction{
		TxID: txid, TenantID: "t1", CreditType: "text", Operation: op, Amount: a,
		IsPaper: true, CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
	}
}

// TestApplyBalanceMath verifies the signed-delta balance progression.
func TestApplyBalanceMath(t *testing.T) {
	bal := Zero()
	prev := genesis

	buy, err := Apply(bal, prev, mk("tx1", OpPurchase, "100"))
	if err != nil {
		t.Fatal(err)
	}
	if buy.BalanceAfter.String() != "100.000000" {
		t.Fatalf("after purchase = %s, want 100.000000", buy.BalanceAfter)
	}

	use, err := Apply(buy.BalanceAfter, buy.ChainHash, mk("tx2", OpConsumption, "-12.5"))
	if err != nil {
		t.Fatal(err)
	}
	if use.BalanceAfter.String() != "87.500000" {
		t.Fatalf("after consumption = %s, want 87.500000", use.BalanceAfter)
	}
}

// TestApplyInsufficientCredit ensures a debit beyond the balance is rejected.
func TestApplyInsufficientCredit(t *testing.T) {
	bal, _ := ParseMoney("10")
	if _, err := Apply(bal, genesis, mk("tx1", OpConsumption, "-10.000001")); err != ErrInsufficientCredit {
		t.Fatalf("expected ErrInsufficientCredit, got %v", err)
	}
	// exact-to-zero is allowed
	if _, err := Apply(bal, genesis, mk("tx2", OpConsumption, "-10")); err != nil {
		t.Fatalf("debit to exactly zero should be allowed, got %v", err)
	}
}

// TestChainDeterministicAndLinked verifies hashes are deterministic, linked, and tamper-evident.
func TestChainDeterministicAndLinked(t *testing.T) {
	t1, _ := Apply(Zero(), genesis, mk("tx1", OpPurchase, "100"))
	t1b, _ := Apply(Zero(), genesis, mk("tx1", OpPurchase, "100"))
	if t1.ChainHash != t1b.ChainHash {
		t.Fatal("same inputs must produce the same chain hash (determinism)")
	}
	t2, _ := Apply(t1.BalanceAfter, t1.ChainHash, mk("tx2", OpConsumption, "-40"))
	if t2.ChainHash == t1.ChainHash {
		t.Fatal("linked hashes must differ")
	}

	chain := []Transaction{t1, t2}
	if bad := VerifyChain(genesis, chain); bad != -1 {
		t.Fatalf("intact chain reported bad index %d", bad)
	}
	// tamper with tx1's amount → chain must break at index 0
	chain[0].Amount, _ = ParseMoney("999")
	if bad := VerifyChain(genesis, chain); bad != 0 {
		t.Fatalf("tampered chain: expected first-bad index 0, got %d", bad)
	}
}

// TestReconciliation proves replaying signed amounts reproduces the final balance exactly.
func TestReconciliation(t *testing.T) {
	steps := []Transaction{
		mk("a", OpPurchase, "500"),
		mk("b", OpConsumption, "-12.5"),
		mk("c", OpConversion, "-100"),
		mk("d", OpRefund, "12.5"),
	}
	bal := Zero()
	prev := genesis
	sum := Zero()
	for _, s := range steps {
		applied, err := Apply(bal, prev, s)
		if err != nil {
			t.Fatal(err)
		}
		bal = applied.BalanceAfter
		prev = applied.ChainHash
		sum = sum.Add(s.Amount)
	}
	if bal.Cmp(sum) != 0 {
		t.Fatalf("reconciliation mismatch: replayed balance %s != sum of amounts %s", bal, sum)
	}
	if bal.String() != "400.000000" {
		t.Fatalf("final balance = %s, want 400.000000", bal)
	}
}
