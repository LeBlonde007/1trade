package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Usage is the token accounting returned at the end of a (streamed) completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// streamChunk is the slice of an OpenAI chat.completion.chunk the CLI renders: the incremental content
// delta, plus the usage that may ride on the final chunk.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *Usage `json:"usage"`
}

// StreamChat POSTs a streaming chat completion (the body should set "stream": true) and calls onToken
// for each content delta as it arrives — the typewriter feel. It returns the final Usage when the
// provider sends it. Server-Sent Events are `data: {json}` lines terminated by `data: [DONE]`; a
// non-2xx response is surfaced as an *APIError (the same shape as Do), so errors stay actionable.
func StreamChat(baseURL, path, token string, body any, onToken func(string)) (Usage, error) {
	var usage Usage
	b, err := json.Marshal(body)
	if err != nil {
		return usage, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(baseURL, "/")+path, bytes.NewReader(b))
	if err != nil {
		return usage, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return usage, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return usage, apiErrorFromBody(resp)
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // SSE lines can be large
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk streamChunk
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue // tolerate keep-alives / non-JSON comments
		}
		for _, c := range chunk.Choices {
			if c.Delta.Content != "" {
				onToken(c.Delta.Content)
			}
		}
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
	}
	return usage, sc.Err()
}

// apiErrorFromBody builds an *APIError from a non-2xx streaming response body (the {code,message} shape).
func apiErrorFromBody(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
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
		ae.Message = fmt.Sprintf("stream failed (%d)", resp.StatusCode)
	}
	return ae
}
