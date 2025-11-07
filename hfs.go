package hfs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HFSpace represents a client to a Hugging Face Space.
// I is the input type, O is the output type. Use `any` if there are different types.
// Use NewHfs() to create an instance.
type HFSpace[I any, O any] struct {
	BaseURL string
	Headers map[string]string
	client  *http.Client
}

// NewHfs creates a new HFSpace with a default HTTP client.
// I is the input type, O is the output type. Use `any` if there are different types.
func NewHfs[I, O any](Name string) *HFSpace[I, O] {
	return &HFSpace[I, O]{
		BaseURL: "https://" + Name + ".hf.space/gradio_api/call",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		client: http.DefaultClient,
	}
}

// WithHeader sets a custom header.
func (h *HFSpace[I, O]) WithHeader(key, value string) *HFSpace[I, O] {
	h.Headers[key] = value
	return h
}

// WithBearerToken adds an Authorization Bearer token.
func (h *HFSpace[I, O]) WithBearerToken(token string) *HFSpace[I, O] {
	return h.WithHeader("Authorization", "Bearer "+token)
}

// WithTimeout sets a custom timeout on the underlying HTTP client.
func (h *HFSpace[I, O]) WithTimeout(d time.Duration) *HFSpace[I, O] {
	h.client.Timeout = d
	return h
}

// WithUserAgent sets a custom User-Agent.
func (h *HFSpace[I, O]) WithUserAgent(agent string) *HFSpace[I, O] {
	return h.WithHeader("User-Agent", agent)
}

// WithHTTPClient allows setting a custom http.Client.
func (h *HFSpace[I, O]) WithHTTPClient(client *http.Client) *HFSpace[I, O] {
	h.client = client
	return h
}

// Do performs the full request + follow-up GET using the event ID.
func (h *HFSpace[I, O]) Do(endpoint string, params ...I) ([]O, error) {
	fullURL := fmt.Sprintf("%s/%s", h.BaseURL, strings.TrimLeft(endpoint, "/"))

	// Step 1: POST request
	payload := map[string]any{
		"data": params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("request body marshall: %w", err)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("post request create: %w", err)
	}
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request exec: %w", err)
	}
	defer resp.Body.Close()

	// Decode event ID
	var idResp struct {
		Eventid string `json:"event_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&idResp); err != nil {
		return nil, fmt.Errorf("event ID decode: %w", err)
	}
	eventID := idResp.Eventid

	// Step 2: GET request to fetch final result
	streamURL := fmt.Sprintf("%s/%s", fullURL, eventID)

	getReq, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("get request create: %w", err)
	}
	for k, v := range h.Headers {
		getReq.Header.Set(k, v)
	}

	resp2, err := h.client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("get request send: %w", err)
	}
	defer resp2.Body.Close()

	res2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, fmt.Errorf("get response read: %w", err)
	}

	lines := strings.Split(string(res2), "\n")
	var dataLine string

	for _, line := range lines {
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(line[len("data:"):])
			break
		}
	}

	if len(dataLine) == 0 {
		return nil, fmt.Errorf("no data in response")
	}

	// Final result
	var Result []O
	if err := json.Unmarshal([]byte(dataLine), &Result); err != nil {
		return nil, fmt.Errorf("decode final response: %w", err)
	}

	return Result, nil
}

// Gradio-compatible FileData structure.
// Usually used for images, audio, video, etc.
type FileData struct {
	Path     string         `json:"path,omitempty"`
	URL      string         `json:"url,omitempty"`
	Size     int64          `json:"size,omitempty"`
	OrigName string         `json:"orig_name,omitempty"`
	MimeType string         `json:"mime_type,omitempty"`
	IsStream bool           `json:"is_stream"`
	Meta     map[string]any `json:"meta,omitempty"`
}

func CreateFileData(url, origName, mimeType string, size int64) *FileData {
	return &FileData{
		Path:     "",
		URL:      url,
		Size:     size,
		OrigName: origName,
		MimeType: mimeType,
		IsStream: false,
		Meta:     map[string]any{"_type": "gradio.FileData"},
	}
}

// Create FileData from bytes, base64 string, or URL.
func ToFileData[T []byte | string](fileData T, filename, mimeType string) *FileData {
	if fd, ok := any(fileData).([]byte); ok {
		// If it's a byte slice, encode it to base64
		if len(fd) == 0 {
			return nil
		}
		b64 := base64.StdEncoding.EncodeToString(fd)
		size := int64(len(fd))
		return CreateFileData(b64, filename, mimeType, size)
	} else if fd, ok := any(fileData).(string); ok {
		file_url := strings.TrimSpace(fd)
		// If it's a valid url, return a FileData with that URL
		if _, err := url.ParseRequestURI(file_url); err == nil {
			return CreateFileData(file_url, filename, mimeType, 0)
		}
		// Otherwise, treat it as a base64 string
		b64 := strings.TrimPrefix(fd, "data:")
		b64 = strings.Split(b64, ",")[1] // Remove the prefix if present
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil // Invalid base64 string
		}
		size := int64(len(decoded))
		return CreateFileData(b64, filename, mimeType, size)
	}
	return nil // Unsupported type
}

// Check if src is a FileData.
// Download content from FileData's URL if so.
func GetFileData(src any) ([]byte, error) {
	var fd *FileData
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", src)), &fd); err != nil {
		return nil, fmt.Errorf("not filedata: %w", err)
	}
	return FileDataDownload(fd, 30*time.Second)
}

// Download content from a FileData's HTTPS URL.
// Use on output FileData.
func FileDataDownload(fileData *FileData, timeout time.Duration) ([]byte, error) {
	// Validate input
	if fileData == nil {
		return nil, fmt.Errorf("filedata is nil")
	}

	if fileData.URL == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout * time.Second,
	}

	// Create the request
	req, err := http.NewRequest("GET", fileData.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	// Read the response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("downloaded content is empty")
	}

	return content, nil
}
