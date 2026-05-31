// Package client is the exascale CLI's tiny JSON HTTP helper for the platform services. It attaches
// the bearer token and surfaces upstream errors (code + message) so the CLI prints something useful.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIError carries an upstream non-2xx response (the contract's {code, message} shape).
type APIError struct {
	Status  int
	Code    string
	Message string
}

// Error renders the API error for the CLI.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s (%d %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("%s (%d)", e.Message, e.Status)
}

// httpClient has a generous timeout (inference can take a while).
var httpClient = &http.Client{Timeout: 120 * time.Second}

// Do performs a JSON request against baseURL+path, attaching the bearer token and any extra headers,
// and decoding the response into out (when non-nil). A non-2xx becomes an *APIError.
func Do(method, baseURL, path, token string, headers map[string]string, body, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ae := &APIError{Status: resp.StatusCode}
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &e) == nil {
			ae.Code, ae.Message = e.Code, e.Message
		}
		if ae.Message == "" {
			ae.Message = strings.TrimSpace(string(data))
		}
		if ae.Message == "" {
			ae.Message = http.StatusText(resp.StatusCode)
		}
		return ae
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
