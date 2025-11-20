package hfs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	ENDPOINT = "https://qu.ax/upload.php"
)

type Quax struct {
	Client   *http.Client
	Userhash string
}

type File struct {
	URL string `json:"url"`
}

type QuaxResponse struct {
	Success bool   `json:"success"`
	Files   []File `json:"files"`
}

func NewQuax(client *http.Client) *Quax {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Quax{
		Client: client,
	}
}

// Upload file or URI to the Quax. It returns an URL string and error.
func (quax *Quax) Upload(v ...any) (string, error) {
	if len(v) == 0 {
		return "", fmt.Errorf(`must specify file path or byte slice`)
	}

	switch t := v[0].(type) {
	case string:
		path := t
		parse := func(s string, _ error) (string, error) {
			uri, err := url.Parse(s)
			if err != nil {
				return "", err
			}
			return uri.String(), nil
		}
		switch {
		case FileExists(path):
			return parse(quax.fileUpload(path))
		default:
			return "", errors.New(`path invalid`)
		}
	case []byte:
		if len(v) != 2 {
			return "", fmt.Errorf(`must specify file name`)
		}
		return quax.rawUpload(t, v[1].(string))
	}
	return "", fmt.Errorf(`unhandled`)
}

func (quax *Quax) rawUpload(b []byte, name string) (string, error) {
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		defer w.Close()
		defer m.Close()

		m.WriteField("reqtype", "fileupload")
		m.WriteField("userhash", quax.Userhash)
		part, err := m.CreateFormFile("files[]", filepath.Base(name))
		if err != nil {
			return
		}
		if _, err = io.Copy(part, bytes.NewBuffer(b)); err != nil {
			return
		}
	}()
	req, _ := http.NewRequest(http.MethodPost, ENDPOINT, r)
	req.Header.Add("Content-Type", m.FormDataContentType())

	resp, err := quax.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var qr QuaxResponse
	err = json.Unmarshal([]byte(body), &qr)
	if err != nil {
		return "", fmt.Errorf("quax upload response unmarshal: %w", err)
	}
	if !qr.Success || len(qr.Files) == 0 {
		return "", fmt.Errorf("quax upload failed")
	}

	return qr.Files[0].URL, nil
}

func (quax *Quax) fileUpload(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if size := FileSize(path); size > 209715200 {
		return "", fmt.Errorf("file too large, size: %d MB", size/1024/1024)
	}

	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		defer w.Close()
		defer m.Close()

		m.WriteField("reqtype", "fileupload")
		m.WriteField("userhash", quax.Userhash)
		part, err := m.CreateFormFile("files[]", filepath.Base(file.Name()))
		if err != nil {
			return
		}

		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()

	req, _ := http.NewRequest(http.MethodPost, ENDPOINT, r)
	req.Header.Add("Content-Type", m.FormDataContentType())

	resp, err := quax.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var qr QuaxResponse
	err = json.Unmarshal([]byte(body), &qr)
	if err != nil {
		return "", fmt.Errorf("quax upload response unmarshal: %w", err)
	}
	if !qr.Success || len(qr.Files) == 0 {
		return "", fmt.Errorf("quax upload failed")
	}

	return qr.Files[0].URL, nil
}

// FileSeze returns file attritubes of size about an inode, and
// it's unit alway is bytes.
func FileSize(filepath string) int64 {
	f, err := os.Stat(filepath)
	if err != nil {
		return 0
	}

	return f.Size()
}

// FileExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
