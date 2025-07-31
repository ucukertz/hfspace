package hfspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HFSpace represents a client to a Hugging Face Space.
type HFSpace[T any, O any] struct {
	BaseURL string
	Headers map[string]string
	client  *http.Client
}

// NewHFSpace creates a new HFSpace with a default HTTP client.
func NewHFSpace[T, O any](baseURL string) *HFSpace[T, O] {
	return &HFSpace[T, O]{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		client: http.DefaultClient,
	}
}

// WithHeader sets a custom header.
func (h *HFSpace[T, O]) WithHeader(key, value string) *HFSpace[T, O] {
	h.Headers[key] = value
	return h
}

// WithBearerToken adds an Authorization Bearer token.
func (h *HFSpace[T, O]) WithBearerToken(token string) *HFSpace[T, O] {
	return h.WithHeader("Authorization", "Bearer "+token)
}

// WithTimeout sets a custom timeout on the underlying HTTP client.
func (h *HFSpace[T, O]) WithTimeout(d time.Duration) *HFSpace[T, O] {
	h.client.Timeout = d
	return h
}

// WithUserAgent sets a custom User-Agent.
func (h *HFSpace[T, O]) WithUserAgent(agent string) *HFSpace[T, O] {
	return h.WithHeader("User-Agent", agent)
}

// WithHTTPClient allows setting a custom http.Client.
func (h *HFSpace[T, O]) WithHTTPClient(client *http.Client) *HFSpace[T, O] {
	h.client = client
	return h
}

// Do performs the full request + follow-up GET using the event ID.
func (h *HFSpace[T, O]) Do(endpoint string, params ...T) ([]O, error) {
	fullURL := fmt.Sprintf("%s/%s", h.BaseURL, strings.TrimLeft(endpoint, "/"))

	// Step 1: POST request
	payload := map[string]any{
		"data": params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %w", err)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating POST request: %w", err)
	}
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	// Decode event ID
	var idResp struct {
		Data []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&idResp); err != nil {
		return nil, fmt.Errorf("error decoding event ID response: %w", err)
	}
	if len(idResp.Data) != 1 {
		return nil, fmt.Errorf("expected one event ID, got: %v", idResp.Data)
	}
	eventID := idResp.Data[0]

	// Step 2: GET request to fetch final result
	streamURL := fmt.Sprintf("%s/%s", fullURL, eventID)

	getReq, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request: %w", err)
	}
	for k, v := range h.Headers {
		getReq.Header.Set(k, v)
	}

	resp2, err := h.client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("GET request failed: %w", err)
	}
	defer resp2.Body.Close()

	// Final result
	var finalResp struct {
		Data []O `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&finalResp); err != nil {
		return nil, fmt.Errorf("error decoding final response: %w", err)
	}

	return finalResp.Data, nil
}
