package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"

	"api.dauntless-society.com/anchor/handlers"
	authhandlers "api.dauntless-society.com/anchor/internal/auth/handlers"
	authjwt "api.dauntless-society.com/anchor/internal/auth/jwt"
	authstate "api.dauntless-society.com/anchor/internal/auth/state"
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

	state.SetJobDir(filepath.Join(cfg.AnchorStateDir, "jobs"))
	indexStore := canonicalindex.NewStore(filepath.Join(cfg.AnchorStateDir, "index"))
	authStore := authstate.New(filepath.Join(cfg.AnchorStateDir, "auth"))
	authSigner, err := authjwt.NewSignerFromKeyFile(cfg.JWTKeyFile, 20*time.Minute)
	if err != nil {
		log.Fatalf("Failed to load JWT key: %v", err)
	}
	authService := &authhandlers.Service{Store: authStore, JWT: authSigner, ChallengeTTL: 5 * time.Minute}

	service := &handlers.AnchorService{
		IPFS:  ipfsClient,
		Index: indexStore,
	}

	mux := http.NewServeMux()
	authService.RegisterRoutes(mux)

	// Read-only (READER implicit)
	mux.HandleFunc("/api/v1/index/latest", service.LatestIndexHandler)
	mux.HandleFunc("/api/v1/document/", service.DocumentLookupHandler)

	// Mutating endpoints require JWT + EDITOR or higher
	mux.Handle("/api/v1/anchor", authService.Middleware(authhandlers.RequireRoles(http.HandlerFunc(service.AnchorHandler), "EDITOR", "FOUNDER", "ADMIN")))

	addr := cfg.ListenAddr + ":" + cfg.ListenPort
	log.Printf("Dauntless Anchor API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
