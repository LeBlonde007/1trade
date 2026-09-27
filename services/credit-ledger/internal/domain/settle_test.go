package domain

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// m parses a money literal for tests.
func m(s string) Money {
	v, err := ParseMoney(s)
	if err != nil {
		panic(err) //nolint:forbidigo // test helper on constant input
	}
	return v
}

// trade returns a valid paper settlement request.
func trade() SettleRequest {
	return SettleRequest{
		TradeID: "t-1", ProductID: "H100-SPOT", CreditType: "gpu_h100", Price: m("2.99"), Quantity: m("10"),
		Currency: USD, IsPaper: true, EngineChainHash: "abc",
		Buyer:  SettleParty{TenantID: "buyer", OrderID: "o-b", Fee: m("0.299")},
		Seller: SettleParty{TenantID: "seller", OrderID: "o-s", Fee: m("0.1495")},
	}
}

// TestPlanSettlement checks the legs of a standard trade.
func TestPlanSettlement(t *testing.T) {
	p, err := PlanSettlement(trade())
	if err != nil {
		t.Fatal(err)
	}
	if p.Notional.String() != "29.900000" {
		t.Fatalf("notional = %s, want 29.900000", p.Notional)
	}
	credit := map[string]string{}
	for _, l := range p.Credit {
		credit[l.TenantID] = l.Amount.String()
	}
	if credit["buyer"] != "10.000000" || credit["seller"] != "-10.000000" {
		t.Errorf("credit legs = %v", credit)
	}
	cash := map[string]string{}
	for _, l := range p.Cash {
		cash[l.TenantID+"/"+string(l.Operation)] = l.Amount.String()
	}
	want := map[string]string{
		"buyer/trade": "-29.900000", "buyer/fee": "-0.299000",
		"seller/trade": "29.900000", "seller/fee": "-0.149500",
	}
	for k, v := range want {
		if cash[k] != v {
			t.Errorf("cash %s = %s, want %s", k, cash[k], v)
		}
	}
	// Within the seller's balance, proceeds land before the fee is taken.
	for i, l := range p.Cash {
		if l.TenantID == "seller" && l.Operation == CashFee && (i == 0 || p.Cash[i-1].Operation != CashTrade) {
			t.Errorf("seller fee applied before proceeds: %+v", p.Cash)
		}
	}
}

// TestZeroFeesWriteNoLeg checks a zero fee produces no fee row.
func TestZeroFeesWriteNoLeg(t *testing.T) {
	r := trade()
	r.Buyer.Fee, r.Seller.Fee = Zero(), Zero()
	p, err := PlanSettlement(r)
	if err != nil || len(p.Cash) != 2 {
		t.Fatalf("plan = %+v, %v; want 2 cash legs", p.Cash, err)
	}
}

// TestSettleValidation checks every refusal, and that policy reasons win over shape errors.
func TestSettleValidation(t *testing.T) {
	cases := []struct {
		name string
		mod  func(r *SettleRequest)
		want error
	}{
		{"real money", func(r *SettleRequest) { r.IsPaper = false }, ErrRealMoneyDisabled},
		{"self trade", func(r *SettleRequest) { r.Seller.TenantID = "buyer" }, ErrSettleSelfTrade},
		{"internal buyer", func(r *SettleRequest) { r.Buyer.IsInternal = true }, ErrInternalOnPaper},
		{"internal seller", func(r *SettleRequest) { r.Seller.IsInternal = true }, ErrInternalOnPaper},
		{"real money beats bad price", func(r *SettleRequest) { r.IsPaper = false; r.Price = Zero() }, ErrRealMoneyDisabled},
		{"zero price", func(r *SettleRequest) { r.Price = Zero() }, ErrBadSettle},
		{"zero quantity", func(r *SettleRequest) { r.Quantity = Zero() }, ErrBadSettle},
		{"negative fee", func(r *SettleRequest) { r.Buyer.Fee = m("-0.01") }, ErrBadSettle},
		{"wrong currency", func(r *SettleRequest) { r.Currency = "JPY" }, ErrBadSettle},
		{"missing trade id", func(r *SettleRequest) { r.TradeID = "" }, ErrBadSettle},
		{"missing chain hash", func(r *SettleRequest) { r.EngineChainHash = "" }, ErrBadSettle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := trade()
			tc.mod(&r)
			if _, err := PlanSettlement(r); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestSettleHashIsCanonical checks the idempotency fingerprint ignores decimal formatting but not
// content.
func TestSettleHashIsCanonical(t *testing.T) {
	a, b := trade(), trade()
	b.Quantity = m("10.000000")
	if a.Hash() != b.Hash() {
		t.Error("equal requests with different decimal spelling hash differently")
	}
	b.Quantity = m("10.000001")
	if a.Hash() == b.Hash() {
		t.Error("different quantities hash the same")
	}
}

// TestSettlementConserves is the conservation property over random trades: across buyer and seller,
// credits net to exactly zero and cash nets to exactly −(buyer fee + seller fee). Nothing is created or
// lost except the fees.
func TestSettlementConserves(t *testing.T) {
	rng := rand.New(rand.NewSource(1)) // #nosec G404 -- deterministic test input
	for i := range 5000 {
		r := trade()
		r.Price = FromMicros(1 + rng.Int63n(5_000_000_000))
		r.Quantity = FromMicros(1 + rng.Int63n(1_000_000_000_000))
		r.Buyer.Fee = FromMicros(rng.Int63n(1_000_000))
		r.Seller.Fee = FromMicros(rng.Int63n(1_000_000))
		p, err := PlanSettlement(r)
		if err != nil {
			t.Fatal(err)
		}
		creditSum, cashSum := Zero(), Zero()
		for _, l := range p.Credit {
			creditSum = creditSum.Add(l.Amount)
		}
		for _, l := range p.Cash {
			cashSum = cashSum.Add(l.Amount)
		}
		wantCash := Zero().Sub(r.Buyer.Fee).Sub(r.Seller.Fee)
		if creditSum.Sign() != 0 || cashSum.Cmp(wantCash) != 0 {
			t.Fatalf("case %d: credit net %s, cash net %s (want 0, %s)", i, creditSum, cashSum, wantCash)
		}
		if p.Notional.Cmp(Notional(r.Price, r.Quantity)) != 0 || p.Notional.IsNegative() {
			t.Fatalf("case %d: bad notional %s", i, p.Notional)
		}
	}
}

// TestCashChain checks cash rows chain, verify, and detect tampering; and that a cash row never hashes
// like a credit row with the same fields.
func TestCashChain(t *testing.T) {
	at := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	rows := make([]CashTx, 0, 3)
	prev, bal := "", Zero()
	for i, amt := range []string{"10000", "-29.9", "-0.299"} {
		tx, err := ApplyCash(bal, prev, CashTx{TxID: fmt.Sprint(i), TenantID: "t", Currency: USD, Operation: CashTrade, Amount: m(amt), IsPaper: true, CreatedAt: at})
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, tx)
		prev, bal = tx.ChainHash, tx.BalanceAfter
	}
	if bal.String() != "9969.801000" || VerifyCashChain("", rows) != -1 {
		t.Fatalf("balance %s, verify %d", bal, VerifyCashChain("", rows))
	}
	rows[1].Amount = m("-1")
	if VerifyCashChain("", rows) != 1 {
		t.Error("tampered cash row not detected")
	}
	if _, err := ApplyCash(m("1"), "", CashTx{Amount: m("-2")}); !errors.Is(err, ErrInsufficientCash) {
		t.Errorf("overdraw err = %v", err)
	}
	credit, _ := Apply(Zero(), "", Transaction{TxID: "x", TenantID: "t", CreditType: "USD", Operation: "trade", Amount: m("1"), IsPaper: true, CreatedAt: at})
	cash, _ := ApplyCash(Zero(), "", CashTx{TxID: "x", TenantID: "t", Currency: "USD", Operation: "trade", Amount: m("1"), IsPaper: true, CreatedAt: at})
	if credit.ChainHash == cash.ChainHash {
		t.Error("a cash row hashes identically to a credit row")
	}
}
