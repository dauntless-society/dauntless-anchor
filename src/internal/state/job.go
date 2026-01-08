package state

import "time"

type JobStatus string

const (
	StatusReceived     JobStatus = "RECEIVED"
	StatusValidated    JobStatus = "VALIDATED"
	StatusIPFSPrepared JobStatus = "IPFS_PREPARED"
	StatusIndexUpdated JobStatus = "INDEX_UPDATED"
	StatusFinalized    JobStatus = "FINALIZED"
	StatusAborted      JobStatus = "ABORTED"
	StatusFailed       JobStatus = "FAILED"
)

type AnchorJob struct {
	ID                   string
	DocumentHash         string
	DocumentHashSHA512   string
	DocumentHashSHA3_512 string
	CID                  string
	IndexCID             string
	IndexVersion         int
	TxID                 string
	Status               JobStatus
	Error                string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
