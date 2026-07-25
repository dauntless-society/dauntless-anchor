package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSave_WritesJobJSON(t *testing.T) {
	dir := t.TempDir()
	SetJobDir(dir)

	job := AnchorJob{
		ID:           "job1",
		Status:       StatusValidated,
		DocumentHash: "abc",
		CreatedAt:    time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC),
	}
	if err := Save(job); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path := filepath.Join(dir, "job1.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read job file: %v", err)
	}

	var parsed AnchorJob
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.ID != "job1" {
		t.Fatalf("id mismatch: %q", parsed.ID)
	}
	if parsed.Status != StatusValidated {
		t.Fatalf("status mismatch: %q", parsed.Status)
	}
	if parsed.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be set")
	}
}
