package canonicalindex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeIPFS struct {
	cid        string
	abortCIDs  []string
	prepareErr error
}

func (f *fakeIPFS) Prepare(_ []byte) (string, error) {
	if f.prepareErr != nil {
		return "", f.prepareErr
	}
	return f.cid, nil
}

func (f *fakeIPFS) Abort(cid string) error {
	f.abortCIDs = append(f.abortCIDs, cid)
	return nil
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestStore_LoadLatest_NotInitialized(t *testing.T) {
	s := NewStore(t.TempDir())
	idx, cid, err := s.LoadLatest()
	if err != nil {
		t.Fatalf("LoadLatest: %v", err)
	}
	if idx != nil || cid != "" {
		t.Fatalf("expected nil index and empty cid")
	}
}

func TestStore_Append_CreatesVersionedAndLatestFiles(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	ipfs := &fakeIPFS{cid: "bafyTESTCID1"}

	entry1 := Entry{
		DocumentID:       "doc1",
		DocumentVersion:  1,
		DocumentSHA256:   strings.Repeat("a", 64),
		DocumentSHA512:   strings.Repeat("b", 128),
		DocumentSHA3_512: strings.Repeat("c", 128),
		DocumentCID:      "bafyDOC1",
		Timestamp:        "2026-01-09T12:00:00Z",
	}

	cid1, v1, err := s.Append(ipfs, entry1)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if cid1 != "bafyTESTCID1" || v1 != 1 {
		t.Fatalf("unexpected cid/version: %q %d", cid1, v1)
	}

	latestJSON := mustRead(t, filepath.Join(dir, "latest.json"))
	latestCID := strings.TrimSpace(mustRead(t, filepath.Join(dir, "latest.cid")))
	if latestCID != cid1 {
		t.Fatalf("latest.cid mismatch: %q", latestCID)
	}

	var parsed Index
	if err := json.Unmarshal([]byte(latestJSON), &parsed); err != nil {
		t.Fatalf("unmarshal latest.json: %v", err)
	}
	if parsed.IndexVersion != 1 || parsed.PreviousIndexCID != "" {
		t.Fatalf("unexpected index header: %#v", parsed)
	}
	if len(parsed.Entries) != 1 || parsed.Entries[0].DocumentID != "doc1" {
		t.Fatalf("unexpected entries: %#v", parsed.Entries)
	}

	if _, err := os.Stat(filepath.Join(dir, "index-v000001.json")); err != nil {
		t.Fatalf("missing versioned json: %v", err)
	}
	if strings.TrimSpace(mustRead(t, filepath.Join(dir, "index-v000001.cid"))) != cid1 {
		t.Fatalf("versioned cid mismatch")
	}
}

func TestStore_Append_AbortsOnCIDTempWriteFailure(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	ipfs := &fakeIPFS{cid: "bafyWILLABORT"}

	// Force a failure when writing the CID tmp file by pre-creating it as a non-empty directory.
	poison := filepath.Join(dir, "index-v000001.cid.tmp")
	if err := os.Mkdir(poison, 0755); err != nil {
		t.Fatalf("mkdir poison tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(poison, "keep"), []byte("x"), 0644); err != nil {
		t.Fatalf("poison keep: %v", err)
	}

	entry := Entry{
		DocumentID:       "doc1",
		DocumentVersion:  1,
		DocumentSHA256:   strings.Repeat("a", 64),
		DocumentSHA512:   strings.Repeat("b", 128),
		DocumentSHA3_512: strings.Repeat("c", 128),
		DocumentCID:      "bafyDOC1",
		Timestamp:        "2026-01-09T12:00:00Z",
	}

	_, _, err := s.Append(ipfs, entry)
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(ipfs.abortCIDs) != 1 || ipfs.abortCIDs[0] != "bafyWILLABORT" {
		t.Fatalf("expected abort to be called, got: %#v", ipfs.abortCIDs)
	}

	if _, err := os.Stat(filepath.Join(dir, "latest.json")); err == nil {
		t.Fatalf("expected latest.json to not exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "index-v000001.json")); err == nil {
		t.Fatalf("expected versioned json to not exist")
	}
}
