package main

import (
	"log"
	"net/http"
	"path/filepath"

	"api.dauntless-society.com/anchor/handlers"
	"api.dauntless-society.com/anchor/internal/bitcoin"
	"api.dauntless-society.com/anchor/internal/canonicalindex"
	"api.dauntless-society.com/anchor/internal/config"
	"api.dauntless-society.com/anchor/internal/ipfs"
	"api.dauntless-society.com/anchor/internal/state"
)

func main() {
	cfg, err := config.Load("/etc/dauntless-anchor/api.conf")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ipfsClient := ipfs.New(cfg.IPFSBin, cfg.IPFSPath)
	btcClient := bitcoin.New(
		cfg.BitcoinCLI,
		cfg.BitcoinDataDir,
		cfg.BitcoinAddress,
		cfg.BitcoinFeeBTC,
	)

	state.SetJobDir(filepath.Join(cfg.AnchorStateDir, "jobs"))
	indexStore := canonicalindex.NewStore(filepath.Join(cfg.AnchorStateDir, "index"))

	service := &handlers.AnchorService{
		IPFS:    ipfsClient,
		Bitcoin: btcClient,
		Index:   indexStore,
	}

	http.HandleFunc("/api/v1/anchor", service.AnchorHandler)
	http.HandleFunc("/api/v1/index/latest", service.LatestIndexHandler)
	http.HandleFunc("/api/v1/document/", service.DocumentLookupHandler)

	addr := cfg.ListenAddr + ":" + cfg.ListenPort
	log.Printf("Dauntless Anchor API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
