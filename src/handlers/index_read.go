package handlers

import (
	"encoding/json"
	"net/http"

	"api.dauntless-society.com/anchor/internal/canonicalindex"
)

type latestIndexResponse struct {
	IndexCID     string                `json:"index_cid"`
	IndexVersion int                   `json:"index_version"`
	Index        *canonicalindex.Index `json:"index"`
}

func (s *AnchorService) LatestIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	if s.Index == nil {
		http.Error(w, "canonical index not configured", http.StatusInternalServerError)
		return
	}

	idx, cid, err := s.Index.LoadLatest()
	if err != nil {
		http.Error(w, "failed to load canonical index", http.StatusInternalServerError)
		return
	}
	if idx == nil {
		http.Error(w, "canonical index not initialized", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(latestIndexResponse{
		IndexCID:     cid,
		IndexVersion: idx.IndexVersion,
		Index:        idx,
	})
}
