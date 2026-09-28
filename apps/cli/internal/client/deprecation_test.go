package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDeprecationWarning: a deprecated model's response warns on stderr with the sunset date and the
// successor; an ordinary response prints nothing.
func TestDeprecationWarning(t *testing.T) {
	var buf bytes.Buffer
	prev := warnOut
	warnOut = &buf
	defer func() { warnOut = prev }()
	deprecated := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if deprecated {
			w.Header().Set("Deprecation", "@1790000000")
			w.Header().Set("Sunset", "Wed, 28 Oct 2026 12:00:00 GMT")
			w.Header().Set("Link", `</v1/models/llama-3.1-8b>; rel="successor-version"`)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if err := Do("POST", srv.URL, "/v1/chat/completions", "t", nil, map[string]any{}, nil); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "warning: this model is deprecated and stops serving on 2026-10-28; switch to llama-3.1-8b\n" {
		t.Fatalf("warning = %q", got)
	}
	buf.Reset()
	deprecated = false
	_ = Do("GET", srv.URL, "/v1/models", "t", nil, nil, nil)
	if buf.Len() != 0 {
		t.Fatalf("warned without a Deprecation header: %q", buf.String())
	}
}
