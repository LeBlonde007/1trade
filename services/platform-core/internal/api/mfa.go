package api

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// errMFAOff: the user does not have two-factor authentication on.
var errMFAOff = errors.New("2FA is not on for this user")

// mfaRoutes registers two-factor authentication (platform-core.yaml v1.7).
func (s *Server) mfaRoutes() {
	s.mux.HandleFunc("GET /v1/auth/2fa", s.mfaStatus)
	s.mux.HandleFunc("POST /v1/auth/2fa/setup", s.mfaSetup)
	s.mux.HandleFunc("POST /v1/auth/2fa/enable", s.mfaEnable)
	s.mux.HandleFunc("POST /v1/auth/2fa/disable", s.mfaDisable)
	s.mux.HandleFunc("POST /v1/auth/login/2fa", s.loginSecondFactor) // the challenge token is the credential
}

// mfaBox builds the at-rest cipher: MFA_ENC_KEY when set (base64, 32 bytes), else a key derived from
// the JWT secret. A malformed key is a startup bug, so it panics rather than silently weakening.
func mfaBox(cfg config.Config) domain.SecretBox {
	key := domain.DeriveMFAKey(cfg.JWTSecret)
	if cfg.MFAKey != "" {
		k, err := base64.StdEncoding.DecodeString(cfg.MFAKey)
		if err != nil || len(k) != 32 {
			panic("MFA_ENC_KEY must be base64 of 32 bytes")
		}
		key = k
	}
	b, err := domain.NewSecretBox(key)
	if err != nil {
		panic(err)
	}
	return b
}

// secondFactor checks a TOTP code or a recovery code for a user with 2FA on. locked means too many
// wrong codes recently; every wrong code counts toward the lock. A TOTP code is claimed atomically, so
// it works once.
func (s *Server) secondFactor(r *http.Request, userID, code, recovery string) (ok, locked bool, err error) {
	st, err := s.st.GetMFA(r.Context(), userID)
	if err != nil {
		return false, false, err
	}
	if !st.Enabled {
		return false, false, errMFAOff
	}
	if st.LockedUntil != nil && time.Now().Before(*st.LockedUntil) {
		return false, true, nil
	}
	if recovery != "" {
		if c := domain.NormalizeRecoveryCode(recovery); c != "" {
			ok, err = s.st.UseRecoveryCode(r.Context(), userID, domain.HashAPIKey(c))
		}
	} else {
		secret, oerr := s.mfaBox.Open(st.Secret)
		if oerr != nil {
			return false, false, oerr
		}
		if step, match := domain.VerifyTOTP(secret, code, time.Now(), st.LastStep); match {
			ok, err = s.st.ClaimMFAStep(r.Context(), userID, step)
		}
	}
	if err != nil {
		return false, false, err
	}
	if !ok {
		if ferr := s.st.RecordMFAFailure(r.Context(), userID); ferr != nil {
			slog.Error("record 2FA failure", "err", ferr)
		}
	}
	return ok, false, nil
}

// mfaStatus serves GET /v1/auth/2fa: whether 2FA is on and how many recovery codes are left.
func (s *Server) mfaStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	st, err := s.st.GetMFA(r.Context(), p.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": st.Enabled, "recovery_codes_left": st.RecoveryLeft})
}

// mfaSetup serves POST /v1/auth/2fa/setup: a fresh secret for the authenticator app. Nothing changes
// until a code from it is confirmed at /enable.
func (s *Server) mfaSetup(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	st, err := s.st.GetMFA(r.Context(), p.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	secret, err := domain.NewTOTPSecret()
	if err != nil {
		serverError(w, err)
		return
	}
	sealed, err := s.mfaBox.Seal(secret)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := s.st.SetPendingMFA(r.Context(), p.UserID, sealed); errors.Is(err, store.ErrMFAEnabled) {
		writeErr(w, http.StatusConflict, "mfa_enabled", "two-factor authentication is already on")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"secret": domain.TOTPSecretText(secret), "otpauth_uri": domain.TOTPURI(secret, st.Email)})
}

// mfaEnable serves POST /v1/auth/2fa/enable {code}: confirms the app works, turns 2FA on, and returns
// ten recovery codes — shown this once.
func (s *Server) mfaEnable(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b struct {
		Code string `json:"code"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	st, err := s.st.GetMFA(r.Context(), p.UserID)
	if err != nil {
		serverError(w, err)
		return
	}
	if st.Enabled || st.Pending == nil {
		writeErr(w, http.StatusConflict, "no_setup", "start two-factor setup first")
		return
	}
	secret, err := s.mfaBox.Open(st.Pending)
	if err != nil {
		serverError(w, err)
		return
	}
	step, match := domain.VerifyTOTP(secret, b.Code, time.Now(), 0)
	if !match {
		writeErr(w, http.StatusUnprocessableEntity, "bad_code", "that code does not match — check the time on your device and try again")
		return
	}
	codes, err := domain.NewRecoveryCodes(10)
	if err != nil {
		serverError(w, err)
		return
	}
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = domain.HashAPIKey(c)
	}
	if err := s.st.EnableMFA(r.Context(), p.UserID, step, hashes); errors.Is(err, store.ErrMFANotPending) {
		writeErr(w, http.StatusConflict, "no_setup", "start two-factor setup first")
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "user.2fa.enable",
		TargetType: "user", TargetID: p.UserID, IsPaper: p.IsPaper})
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "recovery_codes": codes})
}

// mfaDisable serves POST /v1/auth/2fa/disable {code | recovery_code}: a current second factor is
// required, so a stolen session alone cannot turn 2FA off.
func (s *Server) mfaDisable(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b struct {
		Code         string `json:"code"`
		RecoveryCode string `json:"recovery_code"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	good, locked, err := s.secondFactor(r, p.UserID, b.Code, b.RecoveryCode)
	switch {
	case errors.Is(err, errMFAOff):
		writeErr(w, http.StatusConflict, "mfa_off", "two-factor authentication is not on")
		return
	case err != nil:
		serverError(w, err)
		return
	case locked:
		writeErr(w, http.StatusTooManyRequests, "mfa_locked", "too many wrong codes — try again in 15 minutes")
		return
	case !good:
		writeErr(w, http.StatusUnprocessableEntity, "bad_code", "that code is not valid")
		return
	}
	if err := s.st.DisableMFA(r.Context(), p.UserID); err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "user.2fa.disable",
		TargetType: "user", TargetID: p.UserID, IsPaper: p.IsPaper})
	w.WriteHeader(http.StatusNoContent)
}

// loginSecondFactor serves POST /v1/auth/login/2fa {mfa_token, code | recovery_code}: redeems the
// challenge from /login for a session.
func (s *Server) loginSecondFactor(w http.ResponseWriter, r *http.Request) {
	var b struct {
		MFAToken     string `json:"mfa_token"`
		Code         string `json:"code"`
		RecoveryCode string `json:"recovery_code"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	userID, err := domain.VerifyMFAChallenge(s.cfg.JWTSecret, b.MFAToken)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid_challenge", "sign in again — the two-factor step expired")
		return
	}
	good, locked, err := s.secondFactor(r, userID, b.Code, b.RecoveryCode)
	switch {
	case errors.Is(err, errMFAOff), errors.Is(err, store.ErrNotMember):
		writeErr(w, http.StatusUnauthorized, "invalid_challenge", "sign in again")
		return
	case err != nil:
		serverError(w, err)
		return
	case locked:
		writeErr(w, http.StatusTooManyRequests, "mfa_locked", "too many wrong codes — try again in 15 minutes")
		return
	case !good:
		writeErr(w, http.StatusUnauthorized, "bad_code", "that code is not valid")
		return
	}
	idn, found, err := s.st.GetUserByID(r.Context(), userID)
	if err != nil || !found {
		writeErr(w, http.StatusUnauthorized, "invalid_challenge", "sign in again")
		return
	}
	s.issue(w, http.StatusOK, idn.UserID, domain.Claims{
		TenantID: idn.TenantID, OrgID: idn.OrgID, SubAccountID: idn.SubAccountID, Roles: idn.Roles, IsPaper: idn.IsPaper,
	})
}
