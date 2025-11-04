package hfs

import (
	"encoding/base64"
	"testing"
	"time"
)

var test_hf_token = "your-token"
var test_input = []byte{}   // your test input bytes
var test_mime = "image/png" // your test input mime type
var test_prompt = "make it happy"

func Test_FileDataIO(t *testing.T) {
	t.Parallel()

	hfs := NewHfs[any, any]("zerogpu-aoti-flux-1-kontext-dev")
	hfs.WithTimeout(300 * time.Second)
	hfs.WithBearerToken(test_hf_token)

	fdi := ToFileData(test_input, "input.png", test_mime)

	res, err := hfs.Do("/infer", fdi, test_prompt, 0, true, 2.5, 28)
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
