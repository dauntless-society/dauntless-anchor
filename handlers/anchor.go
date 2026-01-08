package handlers

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "io"
    "net/http"
    "time"

    "api.dauntless-society.com/anchor/internal/bitcoin"
    "api.dauntless-society.com/anchor/internal/ipfs"
    "api.dauntless-society.com/anchor/internal/state"

    "github.com/google/uuid"
)

func AnchorHandler(w http.ResponseWriter, r *http.Request) {
    job := state.AnchorJob{
        ID:        uuid.NewString(),
        Status:    state.StatusReceived,
        CreatedAt: time.Now(),
    }

    data, err := io.ReadAll(r.Body)
    if err != nil {
        job.Status = state.StatusFailed
        job.Error = err.Error()
        json.NewEncoder(w).Encode(job)
        return
    }

    hash := sha256.Sum256(data)
    job.DocumentHash = hex.EncodeToString(hash[:])
    job.Status = state.StatusValidated

    cid, err := ipfs.Prepare(data)
    if err != nil {
        job.Status = state.StatusFailed
        job.Error = err.Error()
        json.NewEncoder(w).Encode(job)
        return
    }

    job.CID = cid
    job.Status = state.StatusIPFSPrepared

    txid, err := bitcoin.Commit(job.DocumentHash)
    if err != nil {
        _ = ipfs.Abort(cid)
        job.Status = state.StatusAborted
        job.Error = err.Error()
        json.NewEncoder(w).Encode(job)
        return
    }

    job.TxID = txid
    job.Status = state.StatusFinalized
    job.UpdatedAt = time.Now()

    json.NewEncoder(w).Encode(job)
}
