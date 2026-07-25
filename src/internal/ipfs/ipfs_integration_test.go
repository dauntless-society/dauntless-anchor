//go:build integration

package ipfs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIPFSClient_PrepareAndAbort_RoundTrip(t *testing.T) {
	ipfsBin := os.Getenv("IPFS_BIN")
	if ipfsBin == "" {
		ipfsBin = "ipfs"
	}
	if _, err := exec.LookPath(ipfsBin); err != nil {
		t.Skipf("ipfs binary not available (%s): %v", ipfsBin, err)
	}

	repo := filepath.Join(t.TempDir(), "ipfsrepo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	// Init a fresh repo.
	initCmd := exec.Command(ipfsBin, "init")
	initCmd.Env = append(os.Environ(), "IPFS_PATH="+repo)
	out, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ipfs init failed: %v\n%s", err, string(out))
	}

	c := New(ipfsBin, repo)
	cid, err := c.Prepare([]byte("hello dauntless"))
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	cid = strings.TrimSpace(cid)
	if cid == "" {
		t.Fatalf("expected non-empty cid")
	}
	if !strings.HasPrefix(cid, "b") {
		// CIDv1 base32 typically starts with 'b'. This is a heuristic, not a strict CID validator.
		t.Fatalf("expected cidv1 base32-ish cid, got %q", cid)
	}

	// Ensure the CID is pinned by default (since Prepare uses --pin=true).
	pinLs := exec.Command(ipfsBin, "pin", "ls", cid)
	pinLs.Env = append(os.Environ(), "IPFS_PATH="+repo)
	pinOut, pinErr := pinLs.CombinedOutput()
	if pinErr != nil {
		t.Fatalf("ipfs pin ls failed: %v\n%s", pinErr, string(pinOut))
	}
	if !bytes.Contains(pinOut, []byte(cid)) {
		t.Fatalf("expected pin ls output to include cid, got: %s", string(pinOut))
	}

	if err := c.Abort(cid); err != nil {
		t.Fatalf("abort: %v", err)
	}
}
