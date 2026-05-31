package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDo covers a successful decode, the bearer header, and upstream errors → *APIError.
func TestDo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": "unauthorized", "message": "no token"})
			return
		}
		switch r.URL.Path {
		case "/ok":
			_ = json.NewEncoder(w).Encode(map[string]string{"hello": "world"})
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": "bad", "message": "nope"})
		}
	}))
	defer srv.Close()

	var out map[string]string
	if err := Do("GET", srv.URL, "/ok", "tok", nil, nil, &out); err != nil || out["hello"] != "world" {
		t.Fatalf("ok request: out=%v err=%v", out, err)
	}

	var ae *APIError
	if err := Do("GET", srv.URL, "/ok", "", nil, nil, nil); !errors.As(err, &ae) || ae.Status != 401 || ae.Code != "unauthorized" {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
	if err := Do("GET", srv.URL, "/bad", "tok", nil, nil, nil); !errors.As(err, &ae) || ae.Status != 422 {
		t.Fatalf("expected 422 APIError, got %v", err)
	}
}
