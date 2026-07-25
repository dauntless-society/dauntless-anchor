package jwt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSigner_IssueAndParse_RoundTrip(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "jwt.key")
	key := []byte(strings.Repeat("k", 64))
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	s, err := NewSignerFromKeyFile(keyFile, 2*time.Minute)
	if err != nil {
		t.Fatalf("NewSignerFromKeyFile: %v", err)
	}

	now := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)
	tok, exp, err := s.Issue("pk1", []string{"EDITOR"}, []string{"PUBLIC"}, now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tok == "" {
		t.Fatalf("expected token")
	}
	if exp.IsZero() {
		t.Fatalf("expected exp")
	}
	claims, err := s.ParseAndVerify(tok, now)
	if err != nil {
		t.Fatalf("ParseAndVerify: %v", err)
	}

	if claims.PublicKey != "pk1" {
		t.Errorf("PublicKey: got %q, want %q", claims.PublicKey, "pk1")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "EDITOR" {
		t.Errorf("Roles: got %v, want %v", claims.Roles, []string{"EDITOR"})
	}
	if len(claims.Scope) != 1 || claims.Scope[0] != "PUBLIC" {
		t.Errorf("Scope: got %v, want %v", claims.Scope, []string{"PUBLIC"})
	}
}

func TestSigner_NewSignerFromKeyFile_Errors(t *testing.T) {
	_, err := NewSignerFromKeyFile("", 0)
	if err == nil {
		t.Fatalf("expected error for empty path")
	}

	keyFile := filepath.Join(t.TempDir(), "jwt.key")
	if err := os.WriteFile(keyFile, []byte("shortkey"), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	_, err = NewSignerFromKeyFile(keyFile, 0)
	if err == nil {
		t.Fatalf("expected error for short key")
	}
}

func TestSigner_ParseAndVerify_InvalidToken(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "jwt.key")
	key := []byte(strings.Repeat("k", 64))
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	s, err := NewSignerFromKeyFile(keyFile, 2*time.Minute)
	if err != nil {
		t.Fatalf("NewSignerFromKeyFile: %v", err)
	}

	_, err = s.ParseAndVerify("not-a-jwt", time.Now().UTC())
	if err == nil {
		t.Fatalf("expected error for invalid token")
	}
}
