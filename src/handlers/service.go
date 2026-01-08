package handlers

import (
	"api.dauntless-society.com/anchor/internal/bitcoin"
	"api.dauntless-society.com/anchor/internal/canonicalindex"
	"api.dauntless-society.com/anchor/internal/ipfs"
)

type AnchorService struct {
	IPFS    *ipfs.Client
	Bitcoin *bitcoin.Client
	Index   *canonicalindex.Store
}
