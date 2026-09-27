package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/trade1/credit-ledger/internal/domain"
	"github.com/trade1/credit-ledger/internal/store"
)

// reservationDTO is the contract shape of a Reservation (credit.yaml v1.2).
type reservationDTO struct {
	OrderID      string  `json:"order_id"`
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	AssetKind    string  `json:"asset_kind"`
	Asset        string  `json:"asset"`
	Amount       string  `json:"amount"`
	Remaining    string  `json:"remaining"`
	State        string  `json:"state"`
	IsPaper      bool    `json:"is_paper"`
}

// toReservationDTO maps a store reservation to its contract shape.
func toReservationDTO(r store.Reservation) reservationDTO {
	return reservationDTO{
		OrderID: r.OrderID, TenantID: r.TenantID, SubAccountID: strOrNil(r.SubAccountID), AssetKind: r.AssetKind,
		Asset: r.Asset, Amount: r.Amount.String(), Remaining: r.Remaining.String(), State: r.State, IsPaper: true,
	}
}

// reserveBody is the ReserveRequest wire shape.
type reserveBody struct {
	OrderID      string  `json:"order_id"`
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	AssetKind    string  `json:"asset_kind"`
	Asset        string  `json:"asset"`
	Amount       string  `json:"amount"`
	IsPaper      *bool   `json:"is_paper"`
}

// reserve holds what an order could spend (internal, matching-engine only).
func (s *Server) reserve(w http.ResponseWriter, r *http.Request) {
	if err := settlePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "reservations are restricted to the matching engine")
		return
	}
	var b reserveBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if b.IsPaper == nil || !*b.IsPaper {
		writeErr(w, http.StatusUnprocessableEntity, "REAL_MONEY_DISABLED", "reservations are paper-only until counsel clears real money")
		return
	}
	sub := ""
	if b.SubAccountID != nil {
		sub = *b.SubAccountID
		if _, err := uuid.Parse(sub); err != nil {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "sub_account_id must be a uuid")
			return
		}
	}
	_, errO := uuid.Parse(b.OrderID)
	_, errT := uuid.Parse(b.TenantID)
	if errO != nil || errT != nil || r.Header.Get("Idempotency-Key") != b.OrderID {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "order_id and tenant_id must be uuids; Idempotency-Key must equal order_id")
		return
	}
	if (b.AssetKind != "credit" && b.AssetKind != "cash") || b.Asset == "" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "asset_kind must be credit or cash, with an asset")
		return
	}
	amt, err := domain.ParseMoney(b.Amount)
	if err != nil || amt.Sign() <= 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a positive fixed-point decimal")
		return
	}
	res, _, err := s.st.Reserve(r.Context(), store.Reservation{
		OrderID: b.OrderID, TenantID: b.TenantID, SubAccountID: sub, AssetKind: b.AssetKind, Asset: b.Asset, Amount: amt,
	})
	switch {
	case errors.Is(err, domain.ErrInsufficientCredit):
		writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDIT", "not enough available credits to reserve")
	case errors.Is(err, domain.ErrInsufficientCash):
		writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CASH", "not enough available cash to reserve")
	case errors.Is(err, store.ErrReserveConflict):
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "order already reserved with a different body")
	case errors.Is(err, store.ErrBadAsset):
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "unknown asset for this asset kind")
	case err != nil:
		serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, toReservationDTO(res))
	}
}

// release unlocks what an order's reservation still holds (internal, matching-engine only).
func (s *Server) release(w http.ResponseWriter, r *http.Request) {
	if err := settlePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "reservations are restricted to the matching engine")
		return
	}
	var b struct {
		OrderID string `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if _, err := uuid.Parse(b.OrderID); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "order_id must be a uuid")
		return
	}
	res, err := s.st.Release(r.Context(), b.OrderID)
	switch {
	case errors.Is(err, store.ErrNoReservation):
		writeErr(w, http.StatusNotFound, "not_found", "no reservation for this order")
	case err != nil:
		serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, toReservationDTO(res))
	}
}
