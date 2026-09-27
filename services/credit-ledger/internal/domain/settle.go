package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
)

// Settlement validation errors (credit.yaml v1.1 settle-trade 422s). None of them writes anything.
var (
	ErrRealMoneyDisabled = errors.New("real-money settlement is disabled pending counsel (ADR-0004)")
	ErrSettleSelfTrade   = errors.New("buyer and seller are the same tenant")
	ErrInternalOnPaper   = errors.New("internal accounts cannot trade with customer paper accounts")
	ErrBadSettle         = errors.New("invalid settlement request")
)

// SettleParty is one side of a trade to settle.
type SettleParty struct {
	TenantID     string `json:"tenant_id"`
	SubAccountID string `json:"sub_account_id"`
	OrderID      string `json:"order_id"`
	Fee          Money  `json:"fee"`
	IsInternal   bool   `json:"is_internal"`
}

// SettleRequest is a parsed POST /v1/credits/settle-trade body.
type SettleRequest struct {
	TradeID         string      `json:"trade_id"`
	ProductID       string      `json:"product_id"`
	CreditType      CreditType  `json:"credit_type"`
	Price           Money       `json:"price"`
	Quantity        Money       `json:"quantity"`
	Currency        Currency    `json:"currency"`
	Buyer           SettleParty `json:"buyer"`
	Seller          SettleParty `json:"seller"`
	IsPaper         bool        `json:"is_paper"`
	EngineChainHash string      `json:"chain_hash"`
}

// Validate applies every rule that does not need the database. The order puts the policy refusals
// (real money, self-trade, insider rule) before shape errors so the caller learns the real reason.
func (r SettleRequest) Validate() error {
	if !r.IsPaper {
		return ErrRealMoneyDisabled
	}
	if r.Buyer.TenantID == r.Seller.TenantID {
		return ErrSettleSelfTrade
	}
	if r.Buyer.IsInternal || r.Seller.IsInternal {
		return ErrInternalOnPaper // every settlement is paper (above), so no internal party is allowed
	}
	switch {
	case r.TradeID == "", r.ProductID == "", r.CreditType == "", r.EngineChainHash == "",
		r.Buyer.TenantID == "", r.Seller.TenantID == "", r.Buyer.OrderID == "", r.Seller.OrderID == "":
		return ErrBadSettle
	case r.Currency != USD:
		return ErrBadSettle
	case r.Price.Sign() <= 0, r.Quantity.Sign() <= 0, r.Buyer.Fee.IsNegative(), r.Seller.Fee.IsNegative():
		return ErrBadSettle
	}
	return nil
}

// Hash is the idempotency fingerprint of a request: the same trade_id with a different Hash is a
// conflict (409), never a second settlement. Money is hashed in canonical 6-dp form, so "1" and
// "1.000000" are the same request.
func (r SettleRequest) Hash() string {
	b, _ := json.Marshal(r) // fixed struct; Money marshals to its canonical string
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Notional returns floor_6dp(price × quantity) — the matching engine's rule, recomputed here so the
// ledger never trusts a caller-supplied cash amount.
func Notional(price, qty Money) Money {
	p := new(big.Int).Mul(price.i(), qty.i())
	return Money{p.Quo(p, scaleFactor)} // operands are positive, so Quo truncation is floor
}

// CreditLeg is one credit movement of a settlement.
type CreditLeg struct {
	TenantID     string
	SubAccountID string
	CreditType   CreditType
	Amount       Money // signed
	Key          string
	// PayingOrderID is the order whose reservation funds this debit ("" for credits into a balance
	// and for the seller's fee, which is paid out of the proceeds that land just before it).
	PayingOrderID string
}

// CashLeg is one cash movement of a settlement.
type CashLeg struct {
	TenantID     string
	SubAccountID string
	Currency     Currency
	Operation    CashOperation
	Amount       Money // signed
	Key          string
	// PayingOrderID — see CreditLeg.PayingOrderID.
	PayingOrderID string
}

// SettlePlan is every movement a trade causes, in the order they must be applied.
type SettlePlan struct {
	Notional Money
	Credit   []CreditLeg
	Cash     []CashLeg
}

// PlanSettlement computes the legs of a validated trade:
//
//	buyer:  credit +qty · cash −notional (trade) · cash −buyer.fee (fee)
//	seller: credit −qty · cash +notional (trade) · cash −seller.fee (fee)
//
// Zero fees produce no leg. Legs are sorted by (tenant, sub-account) and, within one cash balance,
// trade before fee — so every settlement locks rows in the same global order (no deadlock between
// concurrent settlements) and a seller's proceeds land before their fee is taken.
func PlanSettlement(r SettleRequest) (SettlePlan, error) {
	if err := r.Validate(); err != nil {
		return SettlePlan{}, err
	}
	n := Notional(r.Price, r.Quantity)
	neg := func(m Money) Money { return Zero().Sub(m) }
	p := SettlePlan{Notional: n}
	b, sl := r.Buyer, r.Seller
	p.Credit = []CreditLeg{
		{TenantID: b.TenantID, SubAccountID: b.SubAccountID, CreditType: r.CreditType, Amount: r.Quantity, Key: r.TradeID + ":buy"},
		{TenantID: sl.TenantID, SubAccountID: sl.SubAccountID, CreditType: r.CreditType, Amount: neg(r.Quantity), Key: r.TradeID + ":sell", PayingOrderID: sl.OrderID},
	}
	p.Cash = []CashLeg{
		{TenantID: b.TenantID, SubAccountID: b.SubAccountID, Currency: r.Currency, Operation: CashTrade, Amount: neg(n), Key: r.TradeID + ":buy", PayingOrderID: b.OrderID},
		{TenantID: sl.TenantID, SubAccountID: sl.SubAccountID, Currency: r.Currency, Operation: CashTrade, Amount: n, Key: r.TradeID + ":sell"},
	}
	if b.Fee.Sign() > 0 {
		p.Cash = append(p.Cash, CashLeg{TenantID: b.TenantID, SubAccountID: b.SubAccountID, Currency: r.Currency, Operation: CashFee, Amount: neg(b.Fee), Key: r.TradeID + ":fee", PayingOrderID: b.OrderID})
	}
	if sl.Fee.Sign() > 0 {
		p.Cash = append(p.Cash, CashLeg{TenantID: sl.TenantID, SubAccountID: sl.SubAccountID, Currency: r.Currency, Operation: CashFee, Amount: neg(sl.Fee), Key: r.TradeID + ":fee"})
	}
	sort.SliceStable(p.Credit, func(i, j int) bool {
		a, b := p.Credit[i], p.Credit[j]
		return a.TenantID+"|"+a.SubAccountID < b.TenantID+"|"+b.SubAccountID
	})
	opRank := map[CashOperation]int{CashTrade: 0, CashFee: 1}
	sort.SliceStable(p.Cash, func(i, j int) bool {
		a, b := p.Cash[i], p.Cash[j]
		ka, kb := a.TenantID+"|"+a.SubAccountID, b.TenantID+"|"+b.SubAccountID
		if ka != kb {
			return ka < kb
		}
		return opRank[a.Operation] < opRank[b.Operation]
	})
	return p, nil
}
