package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"api.dauntless-society.com/anchor/internal/canonicalindex"
)

func TestLatestIndexHandler_MethodNotAllowed(t *testing.T) {
	svc := &AnchorService{Index: canonicalindex.NewStore(t.TempDir())}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/index/latest", nil)
	w := httptest.NewRecorder()

	svc.LatestIndexHandler(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestLatestIndexHandler_NotInitialized(t *testing.T) {
	svc := &AnchorService{Index: canonicalindex.NewStore(t.TempDir())}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/index/latest", nil)
	w := httptest.NewRecorder()

	svc.LatestIndexHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestLatestIndexHandler_ReturnsJSON(t *testing.T) {
	dir := t.TempDir()
	idx := canonicalindex.Index{
		Schema:       canonicalindex.SchemaV2,
		IndexVersion: 7,
		CreatedAt:    "2026-01-09T12:00:00Z",
		Entries: []canonicalindex.Entry{{
			DocumentID:       "doc1",
			DocumentCID:      "bafyDOC1",
			DocumentVersion:  1,
			DocumentSHA256:   "x",
			DocumentSHA512:   "y",
			DocumentSHA3_512: "z",
			Timestamp:        "2026-01-09T12:00:00Z",
		}},
	}
	b, _ := json.Marshal(idx)
	b = append(b, '\n')
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), b, 0644); err != nil {
		t.Fatalf("write latest.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "latest.cid"), []byte("bafyINDEX\n"), 0644); err != nil {
		t.Fatalf("write latest.cid: %v", err)
	}

	svc := &AnchorService{Index: canonicalindex.NewStore(dir)}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/index/latest", nil)
	w := httptest.NewRecorder()

	svc.LatestIndexHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp latestIndexResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.IndexCID != "bafyINDEX" || resp.IndexVersion != 7 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Index == nil || resp.Index.IndexVersion != 7 {
		t.Fatalf("missing index")
	}
}
