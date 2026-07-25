package handlers_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	authhandlers "api.dauntless-society.com/anchor/internal/auth/handlers"
	authjwt "api.dauntless-society.com/anchor/internal/auth/jwt"
	authstate "api.dauntless-society.com/anchor/internal/auth/state"
)

var b64 = base64.RawURLEncoding

func mustJSON(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func doJSON(t *testing.T, mux http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, mustJSON(t, body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestAuthFlow_RegisterApproveChallengeVerify_AndRevocationIsImmediate(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := authstate.New(filepath.Join(baseDir, "auth"))

	keyFile := filepath.Join(baseDir, "jwt.key")
	if err := os.WriteFile(keyFile, bytes.Repeat([]byte{0x42}, 32), 0600); err != nil {
		t.Fatalf("write jwt key: %v", err)
	}
	signer, err := authjwt.NewSignerFromKeyFile(keyFile, 2*time.Minute)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	svc := &authhandlers.Service{Store: store, JWT: signer, ChallengeTTL: 2 * time.Minute}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// Seed an ADMIN identity so we can hit the protected approve/revoke endpoints.
	adminPk := "admin-public-key"
	if _, err := store.RegisterPending(adminPk, []string{"ADMIN"}); err != nil {
		t.Fatalf("register admin: %v", err)
	}
	if err := store.Approve(adminPk, []string{"ADMIN"}); err != nil {
		t.Fatalf("approve admin: %v", err)
	}
	adminToken, _, err := signer.Issue(adminPk, []string{"ADMIN"}, []string{"PUBLIC"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}

	// Register a new editor key.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519 keygen: %v", err)
	}
	userPk := b64.EncodeToString(pub)

	w := doJSON(t, mux, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"public_key":      userPk,
		"requested_roles": []string{"EDITOR"},
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, mux, http.MethodPost, "/api/v1/auth/approve", map[string]any{
		"public_key":     userPk,
		"approved_roles": []string{"EDITOR"},
	}, map[string]string{"Authorization": "Bearer " + adminToken})
	if w.Code != http.StatusNoContent {
		t.Fatalf("approve status=%d body=%s", w.Code, w.Body.String())
	}

	// Challenge.
	w = doJSON(t, mux, http.MethodPost, "/api/v1/auth/challenge", map[string]any{
		"public_key": userPk,
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("challenge status=%d body=%s", w.Code, w.Body.String())
	}
	var ch struct {
		ChallengeID string    `json:"challenge_id"`
		Nonce       string    `json:"nonce"`
		ExpiresAt   time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ch); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}
	if ch.Nonce == "" {
		t.Fatalf("expected nonce")
	}

	// Regression: invalid signature must NOT consume the challenge.
	badSig := ed25519.Sign(priv, []byte("different-nonce"))
	w = doJSON(t, mux, http.MethodPost, "/api/v1/auth/verify", map[string]any{
		"public_key": userPk,
		"nonce":      ch.Nonce,
		"signature":  b64.EncodeToString(badSig),
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("verify(bad) status=%d body=%s", w.Code, w.Body.String())
	}

	nonceBytes, err := b64.DecodeString(ch.Nonce)
	if err != nil {
		t.Fatalf("decode nonce: %v", err)
	}
	goodSig := ed25519.Sign(priv, nonceBytes)

	w = doJSON(t, mux, http.MethodPost, "/api/v1/auth/verify", map[string]any{
		"public_key": userPk,
		"nonce":      ch.Nonce,
		"signature":  b64.EncodeToString(goodSig),
	}, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("verify(good) status=%d body=%s", w.Code, w.Body.String())
	}
	var vr struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &vr); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if vr.Token == "" {
		t.Fatalf("expected token")
	}

	// Mount a protected endpoint and ensure the freshly-issued token works.
	mux.Handle("/protected", svc.Middleware(authhandlers.RequireRoles(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "EDITOR")))

	w = doJSON(t, mux, http.MethodGet, "/protected", nil, map[string]string{"Authorization": "Bearer " + vr.Token})
	if w.Code != http.StatusOK {
		t.Fatalf("protected status=%d body=%s", w.Code, w.Body.String())
	}

	// Regression: revocation must be immediate (token should stop working).
	w = doJSON(t, mux, http.MethodPost, "/api/v1/auth/revoke", map[string]any{
		"public_key": userPk,
	}, map[string]string{"Authorization": "Bearer " + adminToken})
	if w.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", w.Code, w.Body.String())
	}

	w = doJSON(t, mux, http.MethodGet, "/protected", nil, map[string]string{"Authorization": "Bearer " + vr.Token})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("protected-after-revoke status=%d body=%s", w.Code, w.Body.String())
	}
}
