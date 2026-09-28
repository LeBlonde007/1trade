package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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
	CreatedAt    string  `json:"created_at"`
}

// toReservationDTO maps a store reservation to its contract shape.
func toReservationDTO(r store.Reservation) reservationDTO {
	return reservationDTO{
		OrderID: r.OrderID, TenantID: r.TenantID, SubAccountID: strOrNil(r.SubAccountID), AssetKind: r.AssetKind,
		Asset: r.Asset, Amount: r.Amount.String(), Remaining: r.Remaining.String(), State: r.State, IsPaper: true,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano),
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
	case errors.Is(err, store.ErrReservationClosed):
		writeErr(w, http.StatusConflict, "RESERVATION_CLOSED", "this order_id's reservation is already released; use a new order_id")
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
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if _, err := uuid.Parse(b.OrderID); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "order_id must be a uuid")
		return
	}
	if b.Reason != "" && b.Reason != store.ReleaseOrderClosed && b.Reason != store.ReleaseOrphaned {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "reason must be order_closed or orphaned")
		return
	}
	res, err := s.st.Release(r.Context(), b.OrderID, b.Reason)
	switch {
	case errors.Is(err, store.ErrNoReservation):
		writeErr(w, http.StatusNotFound, "not_found", "no reservation for this order")
	case err != nil:
		serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, toReservationDTO(res))
	}
}

// reservationPage is the ReservationPage wire shape (credit.yaml v1.3).
type reservationPage struct {
	Data       []reservationDTO `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

// listReservations pages through open reservations for the engine's reconciler (internal,
// matching-engine only, credit.yaml v1.3).
func (s *Server) listReservations(w http.ResponseWriter, r *http.Request) {
	if err := settlePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusForbidden, "forbidden", "reservations are restricted to the matching engine")
		return
	}
	q := r.URL.Query()
	if q.Get("state") != "open" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "state=open is required")
		return
	}
	minAge, limit := 0, 100
	var err error
	if v := q.Get("min_age_seconds"); v != "" {
		if minAge, err = strconv.Atoi(v); err != nil || minAge < 0 || minAge > 604800 {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "min_age_seconds must be 0..604800")
			return
		}
	}
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 || limit > 500 {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "limit must be 1..500")
			return
		}
	}
	after := q.Get("after")
	if after != "" {
		if _, err := uuid.Parse(after); err != nil {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "after must be a uuid")
			return
		}
	}
	rows, next, err := s.st.ListOpenReservations(r.Context(), time.Duration(minAge)*time.Second, after, limit)
	if err != nil {
		serverError(w, err)
		return
	}
	page := reservationPage{Data: make([]reservationDTO, 0, len(rows))}
	for _, res := range rows {
		page.Data = append(page.Data, toReservationDTO(res))
	}
	if next != "" {
		page.NextCursor = &next
	}
	writeJSON(w, http.StatusOK, page)
}
