package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ParsesKeyValuesAndDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.env")
	content := "# comment\n\nLISTEN_ADDR=0.0.0.0\nLISTEN_PORT=8080\nIPFS_BIN=/usr/local/bin/ipfs\nANCHOR_STATE_DIR=/tmp/anchor\nUNKNOWN=ignored\nMALFORMED\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0" || cfg.ListenPort != "8080" {
		t.Fatalf("listen mismatch: %#v", cfg)
	}
	if cfg.IPFSBin != "/usr/local/bin/ipfs" {
		t.Fatalf("ipfs bin mismatch: %q", cfg.IPFSBin)
	}
	if cfg.AnchorStateDir != "/tmp/anchor" {
		t.Fatalf("anchor state dir mismatch: %q", cfg.AnchorStateDir)
	}
	if cfg.IPFSPath == "" || cfg.JWTKeyFile == "" {
		t.Fatalf("expected defaults to be set")
	}
}
