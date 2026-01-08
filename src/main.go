package main

import (
	"log"
	"net/http"

	"api.dauntless-society.com/anchor/handlers"
	"api.dauntless-society.com/anchor/internal/bitcoin"
	"api.dauntless-society.com/anchor/internal/config"
	"api.dauntless-society.com/anchor/internal/ipfs"
)

func main() {
	cfg, err := config.Load("/etc/dauntless-anchor/api.conf")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ipfsClient := ipfs.New(cfg.IPFSBin)
	btcClient := bitcoin.New(
		cfg.BitcoinCLI,
		cfg.BitcoinDataDir,
		cfg.BitcoinAddress,
		cfg.BitcoinFeeBTC,
	)

	service := &handlers.AnchorService{
		IPFS:    ipfsClient,
		Bitcoin: btcClient,
	}

	http.HandleFunc("/api/v1/anchor", service.AnchorHandler)

	addr := cfg.ListenAddr + ":" + cfg.ListenPort
	log.Printf("Dauntless Anchor API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
