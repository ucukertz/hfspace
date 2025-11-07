package hfs

import (
	"encoding/base64"
	"io"
	"net/http"
	"testing"
	"time"
)

var test_hf_token = "your-token"
var test_name = "zerogpu-aoti-flux-1-kontext-dev" // your HF Space name
var test_endpoint = "/infer"                      // your HF Space endpoint, e.g. "/predict" or "/infer"
var test_input_url = "http://"                    // your test input url, will get downloaded when testing byte input
var test_mime = "image/png"                       // your test input mime type
var test_prompt = "make it happy"

func Test_FileDataFromURL(t *testing.T) {
	t.Parallel()

	hfs := NewHfs[any, any](test_name)
	hfs.WithTimeout(300 * time.Second)
	hfs.WithBearerToken(test_hf_token)

	fdi := ToFileData(test_input_url, "input.png", test_mime)

	res, err := hfs.Do(test_endpoint, fdi, test_prompt, 0, true, 2.5, 28)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected at least one result from Do()")
	}

	var out []byte
	out, err = GetFileData(res[0])
	if err != nil {
		t.Fatalf("GetFileData() returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty output")
	}

	b64 := base64.StdEncoding.EncodeToString(out)
	t.Logf("Test_FileDataIO OK: %s", b64)
}

func Test_FileDataFromBytes(t *testing.T) {
	t.Parallel()
	hfs := NewHfs[any, any](test_name)
	hfs.WithTimeout(300 * time.Second)
	hfs.WithBearerToken(test_hf_token)
	resp, err := http.Get(test_input_url)
	if err != nil {
		t.Fatalf("http.Get() returned error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("http.Get() returned status code %d, expected 200", resp.StatusCode)
	}
	data_reader := resp.Body
	data, err := io.ReadAll(data_reader)
	if err != nil {
		t.Fatalf("io.ReadAll() returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty input data")
	}

	fdi := ToFileData(data, "input.png", test_mime)
	res, err := hfs.Do(test_endpoint, fdi, test_prompt, 0, true, 2.5, 28)
	if err != nil {
		t.Fatalf("Do() returned error: %v", err)
	}
	if len(res) == 0 {
		t.Fatalf("expected at least one result from Do()")
	}
	var out []byte
	out, err = GetFileData(res[0])
	if err != nil {
		t.Fatalf("GetFileData() returned error: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty output")
	}
	b64 := base64.StdEncoding.EncodeToString(out)
	t.Logf("Test_FileDataFromBytes OK: %s", b64)
}
