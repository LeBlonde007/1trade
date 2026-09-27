package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/trade1/credit-ledger/internal/domain"
	"github.com/trade1/credit-ledger/internal/store"
)

// cashTxDTO is the contract shape of a CashTransaction (credit.yaml v1.1).
type cashTxDTO struct {
	TxID         string  `json:"tx_id"`
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	Currency     string  `json:"currency"`
	Operation    string  `json:"operation"`
	Amount       string  `json:"amount"`
	ReferenceID  *string `json:"reference_id"`
	BalanceAfter string  `json:"balance_after"`
	IsPaper      bool    `json:"is_paper"`
	CreatedAt    string  `json:"created_at"`
	ChainHash    string  `json:"chain_hash"`
}

// strOrNil returns nil for "" so nullable fields serialise as JSON null.
func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// toCashTxDTO maps a cash row to its contract shape.
func toCashTxDTO(t domain.CashTx) cashTxDTO {
	return cashTxDTO{
		TxID: t.TxID, TenantID: t.TenantID, SubAccountID: strOrNil(t.SubAccountID), Currency: string(t.Currency),
		Operation: string(t.Operation), Amount: t.Amount.String(), ReferenceID: strOrNil(t.ReferenceID),
		BalanceAfter: t.BalanceAfter.String(), IsPaper: t.IsPaper,
		CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), ChainHash: t.ChainHash,
	}
}

// cashBalances lists the tenant's paper cash balances (customer).
func (s *Server) cashBalances(w http.ResponseWriter, r *http.Request) {
	p, err := tenantPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	bals, err := s.st.GetCashBalances(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(bals))
	for _, b := range bals {
		out = append(out, map[string]any{
			"currency": string(b.Currency), "balance": b.Balance.String(), "locked_amount": b.Locked.String(),
			"is_paper": true, "sub_account_id": strOrNil(b.SubAccountID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"balances": out})
}

// cashTransactions lists the tenant's paper cash rows, newest first (customer).
func (s *Server) cashTransactions(w http.ResponseWriter, r *http.Request) {
	p, err := tenantPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	limit, ok := parseTxLimit(r.URL.Query().Get("limit"))
	if !ok {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "limit must be an integer from 1 to 200")
		return
	}
	txs, err := s.st.ListCashTransactions(r.Context(), p.TenantID, limit)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]cashTxDTO, 0, len(txs))
	for _, t := range txs {
		out = append(out, toCashTxDTO(t))
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": out, "next_cursor": nil})
}

// grantBody is the PaperCashGrantRequest.
type grantBody struct {
	TenantID string `json:"tenant_id"`
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
	IsPaper  *bool  `json:"is_paper"`
}

// grantPaperCash credits a tenant's starting paper cash, once per Idempotency-Key (internal).
func (s *Server) grantPaperCash(w http.ResponseWriter, r *http.Request) {
	if _, err := servicePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "Idempotency-Key header is required")
		return
	}
	var b grantBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if b.IsPaper == nil || !*b.IsPaper {
		writeErr(w, http.StatusUnprocessableEntity, "REAL_MONEY_DISABLED", "cash is paper-only until counsel clears custody")
		return
	}
	if _, err := uuid.Parse(b.TenantID); err != nil || domain.Currency(b.Currency) != domain.USD {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "tenant_id (uuid) and currency USD are required")
		return
	}
	amt, err := domain.ParseMoney(b.Amount)
	if err != nil || amt.Sign() <= 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a positive fixed-point decimal")
		return
	}
	tx, replayed, err := s.st.GrantPaperCash(r.Context(), b.TenantID, domain.USD, amt, key)
	if err != nil {
		serverError(w, err)
		return
	}
	if replayed && tx.Amount.Cmp(amt) != 0 {
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "key reused with different body")
		return
	}
	if !replayed {
		s.pub.PublishCashTx(tx)
	}
	writeJSON(w, http.StatusOK, toCashTxDTO(tx))
}

// settleBody is the SettleTradeRequest wire shape; money stays a string until domain parsing.
type settleBody struct {
	TradeID    string     `json:"trade_id"`
	ProductID  string     `json:"product_id"`
	CreditType string     `json:"credit_type"`
	Price      string     `json:"price"`
	Quantity   string     `json:"quantity"`
	Currency   string     `json:"currency"`
	Buyer      settleSide `json:"buyer"`
	Seller     settleSide `json:"seller"`
	IsPaper    *bool      `json:"is_paper"`
	ChainHash  string     `json:"chain_hash"`
}

// settleSide is one SettleParty on the wire.
type settleSide struct {
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	OrderID      string  `json:"order_id"`
	Fee          string  `json:"fee"`
	IsInternal   bool    `json:"is_internal"`
}

// party parses one side, requiring UUIDs where the contract (and the DB columns) do.
func (p settleSide) party() (domain.SettleParty, bool) {
	sub := ""
	if p.SubAccountID != nil {
		sub = *p.SubAccountID
		if _, err := uuid.Parse(sub); err != nil {
			return domain.SettleParty{}, false
		}
	}
	if _, err := uuid.Parse(p.TenantID); err != nil {
		return domain.SettleParty{}, false
	}
	if _, err := uuid.Parse(p.OrderID); err != nil {
		return domain.SettleParty{}, false
	}
	fee, err := domain.ParseMoney(p.Fee)
	if err != nil {
		return domain.SettleParty{}, false
	}
	return domain.SettleParty{TenantID: p.TenantID, SubAccountID: sub, OrderID: p.OrderID, Fee: fee, IsInternal: p.IsInternal}, true
}

// settleTrade settles one matched trade atomically (internal, matching-engine only).
func (s *Server) settleTrade(w http.ResponseWriter, r *http.Request) {
	if err := settlePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "settlement is restricted to the matching engine")
		return
	}
	var b settleBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if b.IsPaper == nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "is_paper is required")
		return
	}
	if _, err := uuid.Parse(b.TradeID); err != nil || r.Header.Get("Idempotency-Key") != b.TradeID {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "trade_id must be a uuid and equal the Idempotency-Key")
		return
	}
	buyer, okB := b.Buyer.party()
	seller, okS := b.Seller.party()
	price, errP := domain.ParseMoney(b.Price)
	qty, errQ := domain.ParseMoney(b.Quantity)
	if !okB || !okS || errP != nil || errQ != nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "invalid party, price or quantity")
		return
	}
	req := domain.SettleRequest{
		TradeID: b.TradeID, ProductID: b.ProductID, CreditType: domain.CreditType(b.CreditType), Price: price,
		Quantity: qty, Currency: domain.Currency(b.Currency), Buyer: buyer, Seller: seller, IsPaper: *b.IsPaper,
		EngineChainHash: b.ChainHash,
	}
	res, replayed, err := s.st.SettleTrade(r.Context(), req)
	switch {
	case errors.Is(err, domain.ErrRealMoneyDisabled):
		writeErr(w, http.StatusUnprocessableEntity, "REAL_MONEY_DISABLED", "real-money settlement is disabled pending counsel")
	case errors.Is(err, domain.ErrSettleSelfTrade):
		writeErr(w, http.StatusUnprocessableEntity, "SELF_TRADE", "buyer and seller are the same tenant")
	case errors.Is(err, domain.ErrInternalOnPaper):
		writeErr(w, http.StatusUnprocessableEntity, "INTERNAL_ON_PAPER", "internal accounts cannot trade with customer paper accounts")
	case errors.Is(err, domain.ErrBadSettle):
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "invalid settlement request")
	case errors.Is(err, store.ErrNotTradeable):
		writeErr(w, http.StatusUnprocessableEntity, "NOT_TRADEABLE", "credit type is not tradeable")
	case errors.Is(err, domain.ErrInsufficientCredit):
		writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDIT", "seller lacks the credits; nothing settled")
	case errors.Is(err, domain.ErrInsufficientCash):
		writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CASH", "buyer lacks the cash; nothing settled")
	case errors.Is(err, store.ErrSettleConflict):
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "trade already settled with a different body")
	case err != nil:
		serverError(w, err)
	default:
		if !replayed {
			for _, t := range res.Credit {
				s.pub.PublishTx(t)
			}
			for _, t := range res.Cash {
				s.pub.PublishCashTx(t)
			}
		}
		credit := make([]txDTO, 0, len(res.Credit))
		for _, t := range res.Credit {
			credit = append(credit, toTxDTO(t))
		}
		cash := make([]cashTxDTO, 0, len(res.Cash))
		for _, t := range res.Cash {
			cash = append(cash, toCashTxDTO(t))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"trade_id": res.TradeID, "notional": res.Notional.String(),
			"credit_transactions": credit, "cash_transactions": cash,
		})
	}
}
