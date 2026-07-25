package handlers

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"api.dauntless-society.com/anchor/internal/hashing"
	"api.dauntless-society.com/anchor/internal/state"
)

type fakeIPFS struct {
	prepareErr error
}

func (f *fakeIPFS) Prepare(_ []byte) (string, error) {
	if f.prepareErr != nil {
		return "", f.prepareErr
	}
	return "bafyTEST", nil
}

func (f *fakeIPFS) Abort(_ string) error { return nil }

func TestAnchorHandler_MethodNotAllowed(t *testing.T) {
	svc := &AnchorService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/anchor", nil)
	svc.AnchorHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestAnchorHandler_MissingFile(t *testing.T) {
	svc := &AnchorService{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/anchor", bytes.NewReader(nil))
	r.Header.Set("Content-Type", "multipart/form-data")
	svc.AnchorHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestAnchorHandler_HashMismatchRejected(t *testing.T) {
	// This should fail before any IPFS/state side effects.
	svc := &AnchorService{}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "doc.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = fw.Write([]byte("hello"))
	_ = mw.WriteField("sha256", "deadbeef")
	_ = mw.WriteField("sha512", "deadbeef")
	_ = mw.WriteField("sha3_512", "deadbeef")
	_ = mw.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/anchor", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	svc.AnchorHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAnchorHandler_AcceptsCorrectHashesUpToIPFS(t *testing.T) {
	// Verify it passes hash validation and reaches IPFS, returning a clean 500 on IPFS failure.
	jobDir := filepath.Join(t.TempDir(), "jobs")
	state.SetJobDir(jobDir)

	svc := &AnchorService{IPFS: &fakeIPFS{prepareErr: errors.New("ipfs down")}}

	data := []byte("hello")
	d := hashing.Compute(data)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "doc.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = fw.Write(data)
	_ = mw.WriteField("sha256", d.SHA256)
	_ = mw.WriteField("sha512", d.SHA512)
	_ = mw.WriteField("sha3_512", d.SHA3_512)
	_ = mw.Close()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/anchor", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())

	svc.AnchorHandler(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
}
