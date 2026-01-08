package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"api.dauntless-society.com/anchor/internal/canonicalindex"
)

type documentLookupResponse struct {
	DocumentID string                `json:"document_id"`
	IndexCID   string                `json:"index_cid"`
	Entry      *canonicalindex.Entry `json:"entry"`
}

func (s *AnchorService) DocumentLookupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	if s.Index == nil {
		http.Error(w, "canonical index not configured", http.StatusInternalServerError)
		return
	}

	// Route: /api/v1/document/{document_id}
	documentID := strings.TrimPrefix(r.URL.Path, "/api/v1/document/")
	documentID = strings.TrimSpace(documentID)
	if documentID == "" || strings.Contains(documentID, "/") {
		http.Error(w, "document_id required", http.StatusBadRequest)
		return
	}

	idx, indexCID, err := s.Index.LoadLatest()
	if err != nil {
		http.Error(w, "failed to load canonical index", http.StatusInternalServerError)
		return
	}
	if idx == nil {
		http.Error(w, "canonical index not initialized", http.StatusNotFound)
		return
	}

	var found *canonicalindex.Entry
	for i := len(idx.Entries) - 1; i >= 0; i-- {
		if idx.Entries[i].DocumentID == documentID {
			e := idx.Entries[i]
			found = &e
			break
		}
	}
	if found == nil {
		http.Error(w, "document not found in canonical index", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(documentLookupResponse{
		DocumentID: documentID,
		IndexCID:   indexCID,
		Entry:      found,
	})
}
