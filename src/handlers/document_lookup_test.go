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

func writeLatestIndex(t *testing.T, dir string, idx canonicalindex.Index, cid string) {
	t.Helper()
	b, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), b, 0644); err != nil {
		t.Fatalf("write latest.json: %v", err)
	}
	if cid != "" {
		if err := os.WriteFile(filepath.Join(dir, "latest.cid"), []byte(cid+"\n"), 0644); err != nil {
			t.Fatalf("write latest.cid: %v", err)
		}
	}
}

func TestDocumentLookupHandler_BadDocumentID(t *testing.T) {
	svc := &AnchorService{Index: canonicalindex.NewStore(t.TempDir())}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/document/", nil)

	svc.DocumentLookupHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("code=%d", w.Code)
	}
}

func TestDocumentLookupHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	idx := canonicalindex.Index{Schema: canonicalindex.SchemaV2, IndexVersion: 1, CreatedAt: "2026-01-09T12:00:00Z", Entries: []canonicalindex.Entry{{DocumentID: "doc1", DocumentCID: "c1", DocumentVersion: 1, DocumentSHA256: "x", DocumentSHA512: "y", DocumentSHA3_512: "z", Timestamp: "t"}}}
	writeLatestIndex(t, dir, idx, "bafyINDEX")

	svc := &AnchorService{Index: canonicalindex.NewStore(dir)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/document/missing", nil)

	svc.DocumentLookupHandler(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDocumentLookupHandler_ReturnsMostRecentEntry(t *testing.T) {
	dir := t.TempDir()
	idx := canonicalindex.Index{
		Schema:       canonicalindex.SchemaV2,
		IndexVersion: 2,
		CreatedAt:    "2026-01-09T12:00:00Z",
		Entries: []canonicalindex.Entry{
			{DocumentID: "doc1", DocumentCID: "old", DocumentVersion: 1, DocumentSHA256: "x", DocumentSHA512: "y", DocumentSHA3_512: "z", Timestamp: "t1"},
			{DocumentID: "doc1", DocumentCID: "new", DocumentVersion: 2, DocumentSHA256: "x2", DocumentSHA512: "y2", DocumentSHA3_512: "z2", Timestamp: "t2"},
		},
	}
	writeLatestIndex(t, dir, idx, "bafyINDEX")

	svc := &AnchorService{Index: canonicalindex.NewStore(dir)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/document/doc1", nil)

	svc.DocumentLookupHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}

	var resp documentLookupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.DocumentID != "doc1" || resp.IndexCID != "bafyINDEX" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Entry == nil || resp.Entry.DocumentCID != "new" {
		t.Fatalf("expected most recent entry, got: %#v", resp.Entry)
	}
}
