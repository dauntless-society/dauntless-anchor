package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"api.dauntless-society.com/anchor/internal/auth"
)

func TestStore_RegisterPending_Idempotent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	id1, err := s.RegisterPending(" pk1 ", []string{string(auth.RoleEditor)})
	if err != nil {
		t.Fatalf("RegisterPending: %v", err)
	}
	if id1 == "" {
		t.Fatalf("expected registration id")
	}

	id2, err := s.RegisterPending("pk1", []string{string(auth.RoleEditor)})
	if err != nil {
		t.Fatalf("RegisterPending(2): %v", err)
	}
	if id2 != id1 {
		t.Fatalf("expected idempotent id: got %q want %q", id2, id1)
	}

	got, err := s.GetIdentity("pk1")
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got == nil {
		t.Fatalf("expected identity")
	}
	if got.ID != id1 {
		t.Fatalf("id mismatch: %q", got.ID)
	}
	if got.Status != string(auth.StatusPending) {
		t.Fatalf("status mismatch: %q", got.Status)
	}
	if len(got.Roles) != 0 {
		t.Fatalf("expected empty roles until approved: %#v", got.Roles)
	}
	if len(got.Scope) != 1 || got.Scope[0] != string(auth.ScopePublic) {
		t.Fatalf("scope mismatch: %#v", got.Scope)
	}

	// Ensure persistence.
	s2 := New(dir)
	got2, err := s2.GetIdentity("pk1")
	if err != nil {
		t.Fatalf("GetIdentity(persisted): %v", err)
	}
	if got2 == nil || got2.ID != id1 {
		t.Fatalf("expected persisted identity")
	}

	if _, err := os.Stat(filepath.Join(dir, "identities.json")); err != nil {
		t.Fatalf("identities.json missing: %v", err)
	}
}

func TestStore_ApproveAndRevoke(t *testing.T) {
	s := New(t.TempDir())

	_, err := s.RegisterPending("pk1", []string{string(auth.RoleEditor)})
	if err != nil {
		t.Fatalf("RegisterPending: %v", err)
	}

	if err := s.Approve("pk1", []string{string(auth.RoleEditor), string(auth.RoleFounder)}); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	id, err := s.RequireActiveIdentity("pk1")
	if err != nil {
		t.Fatalf("RequireActiveIdentity: %v", err)
	}
	if id.Status != string(auth.StatusActive) {
		t.Fatalf("status mismatch: %q", id.Status)
	}
	if len(id.Roles) != 2 {
		t.Fatalf("roles mismatch: %#v", id.Roles)
	}

	if err := s.Approve("pk1", []string{string(auth.RoleEditor)}); err == nil {
		t.Fatalf("expected error approving already-active identity")
	}

	if err := s.Revoke("pk1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := s.RequireActiveIdentity("pk1"); err == nil {
		t.Fatalf("expected error for revoked identity")
	}
}

func TestStore_ChallengeLifecycle(t *testing.T) {
	s := New(t.TempDir())

	_, err := s.RegisterPending("pk1", []string{string(auth.RoleEditor)})
	if err != nil {
		t.Fatalf("RegisterPending: %v", err)
	}
	if _, err := s.CreateChallenge("pk1", time.Minute); err == nil {
		t.Fatalf("expected error creating challenge for non-active identity")
	}

	if err := s.Approve("pk1", []string{string(auth.RoleEditor)}); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	ch1, err := s.CreateChallenge("pk1", time.Minute)
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
	if ch1.PublicKey != "pk1" || ch1.Nonce == "" || ch1.ChallengeID == "" {
		t.Fatalf("unexpected challenge: %#v", ch1)
	}

	ch2, err := s.CreateChallenge("pk1", time.Minute)
	if err != nil {
		t.Fatalf("CreateChallenge(2): %v", err)
	}
	if ch2.Nonce == ch1.Nonce {
		t.Fatalf("expected different nonce")
	}

	if err := s.ConsumeChallenge("pk1", ch1.Nonce); err == nil {
		t.Fatalf("expected old challenge to be gone")
	}

	if err := s.ConsumeChallenge("pk1", ch2.Nonce); err != nil {
		t.Fatalf("ConsumeChallenge: %v", err)
	}
	if err := s.ConsumeChallenge("pk1", ch2.Nonce); err == nil {
		t.Fatalf("expected consume to be single-use")
	}
}
