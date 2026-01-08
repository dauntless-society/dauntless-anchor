package state

import "time"

type JobStatus string

const (
    StatusReceived        JobStatus = "RECEIVED"
    StatusValidated       JobStatus = "VALIDATED"
    StatusIPFSPrepared    JobStatus = "IPFS_PREPARED"
    StatusFinalized       JobStatus = "FINALIZED"
    StatusAborted         JobStatus = "ABORTED"
    StatusFailed          JobStatus = "FAILED"
)

type AnchorJob struct {
    ID           string
    DocumentHash string
    CID          string
    TxID         string
    Status       JobStatus
    Error        string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
