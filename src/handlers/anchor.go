package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"api.dauntless-society.com/anchor/internal/canonicalindex"
	"api.dauntless-society.com/anchor/internal/hashing"
	"api.dauntless-society.com/anchor/internal/state"

	"github.com/google/uuid"
)

func (s *AnchorService) AnchorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	author := r.FormValue("author")

	expectedSHA256 := strings.ToLower(strings.TrimSpace(r.FormValue("sha256")))
	expectedSHA512 := strings.ToLower(strings.TrimSpace(r.FormValue("sha512")))
	expectedSHA3_512 := strings.ToLower(strings.TrimSpace(r.FormValue("sha3_512")))

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	digests := hashing.Compute(data)

	// Validate client-provided hashes to prevent pollution in transit.
	if expectedSHA256 == "" || expectedSHA512 == "" || expectedSHA3_512 == "" {
		http.Error(w, "sha256, sha512, and sha3_512 are required", http.StatusBadRequest)
		return
	}
	if expectedSHA256 != strings.ToLower(digests.SHA256) {
		http.Error(w, "sha256 mismatch", http.StatusBadRequest)
		return
	}
	if expectedSHA512 != strings.ToLower(digests.SHA512) {
		http.Error(w, "sha512 mismatch", http.StatusBadRequest)
		return
	}
	if expectedSHA3_512 != strings.ToLower(digests.SHA3_512) {
		http.Error(w, "sha3_512 mismatch", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()

	job := state.AnchorJob{
		ID:                   id,
		DocumentHash:         digests.SHA256,
		DocumentHashSHA512:   digests.SHA512,
		DocumentHashSHA3_512: digests.SHA3_512,
		Status:               state.StatusValidated,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}
	_ = state.Save(job)

	// IPFS prepare (content anchoring)
	cid, err := s.IPFS.Prepare(data)
	if err != nil {
		job.Status = state.StatusFailed
		job.Error = err.Error()
		_ = state.Save(job)
		http.Error(w, "ipfs prepare failed", http.StatusInternalServerError)
		return
	}

	job.CID = cid
	job.Status = state.StatusIPFSPrepared
	_ = state.Save(job)

	if s.Index == nil {
		job.Status = state.StatusFailed
		job.Error = "canonical index store not configured"
		_ = state.Save(job)
		_ = s.IPFS.Abort(cid)
		http.Error(w, "canonical index not configured", http.StatusInternalServerError)
		return
	}

	indexCID, indexVersion, err := s.Index.Append(s.IPFS, canonicalindex.Entry{
		DocumentID:       id,
		DocumentVersion:  1,
		DocumentSHA256:   digests.SHA256,
		DocumentSHA512:   digests.SHA512,
		DocumentSHA3_512: digests.SHA3_512,
		DocumentCID:      cid,
		Author:           author,
		Timestamp:        time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		job.Status = state.StatusFailed
		job.Error = err.Error()
		_ = state.Save(job)
		_ = s.IPFS.Abort(cid)
		http.Error(w, "canonical index update failed", http.StatusInternalServerError)
		return
	}

	job.IndexCID = indexCID
	job.IndexVersion = indexVersion
	job.Status = state.StatusIndexUpdated

	if err := state.Save(job); err != nil {
		_ = s.IPFS.Abort(indexCID)
		_ = s.IPFS.Abort(cid)
		http.Error(w, "failed to persist job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
