package handlers

import (
	"api.dauntless-society.com/anchor/internal/canonicalindex"
)

type IPFSClient interface {
	Prepare(data []byte) (string, error)
	Abort(cid string) error
}

type AnchorService struct {
	IPFS  IPFSClient
	Index *canonicalindex.Store
}
