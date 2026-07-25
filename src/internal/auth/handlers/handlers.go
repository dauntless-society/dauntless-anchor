package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"api.dauntless-society.com/anchor/internal/auth"
	"api.dauntless-society.com/anchor/internal/auth/crypto"
	"api.dauntless-society.com/anchor/internal/auth/jwt"
	"api.dauntless-society.com/anchor/internal/auth/state"
)

type Service struct {
	Store        *state.Store
	JWT          *jwt.Signer
	ChallengeTTL time.Duration
}

type registerRequest struct {
	PublicKey      string   `json:"public_key"`
	RequestedRoles []string `json:"requested_roles"`
}

type registerResponse struct {
	RegistrationID string `json:"registration_id"`
	Status         string `json:"status"`
}

func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	regID, err := s.Store.RegisterPending(req.PublicKey, req.RequestedRoles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(registerResponse{RegistrationID: regID, Status: string(auth.StatusPending)})
}

type approveRequest struct {
	PublicKey     string   `json:"public_key"`
	ApprovedRoles []string `json:"approved_roles"`
}

func (s *Service) Approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.Store.Approve(req.PublicKey, req.ApprovedRoles); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type revokeRequest struct {
	PublicKey string `json:"public_key"`
}

func (s *Service) Revoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req revokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := s.Store.Revoke(req.PublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type challengeRequest struct {
	PublicKey string `json:"public_key"`
}

type challengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (s *Service) Challenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req challengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if _, err := s.Store.RequireActiveIdentity(req.PublicKey); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	ch, err := s.Store.CreateChallenge(req.PublicKey, s.ChallengeTTL)
	if err != nil {
		http.Error(w, "failed to create challenge", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(challengeResponse{ChallengeID: ch.ChallengeID, Nonce: ch.Nonce, ExpiresAt: ch.ExpiresAt})
}

type verifyRequest struct {
	PublicKey string `json:"public_key"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

type verifyResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) Verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	identity, err := s.Store.RequireActiveIdentity(req.PublicKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Verify signature first so invalid signatures don't consume the nonce.
	if err := crypto.VerifySignature(req.PublicKey, req.Nonce, req.Signature); err != nil {
		http.Error(w, "signature invalid", http.StatusUnauthorized)
		return
	}
	if err := s.Store.ConsumeChallenge(req.PublicKey, req.Nonce); err != nil {
		http.Error(w, "challenge invalid", http.StatusUnauthorized)
		return
	}

	tok, expiresAt, err := s.JWT.Issue(identity.PublicKey, identity.Roles, identity.Scope, time.Now().UTC())
	if err != nil {
		http.Error(w, "failed to issue token", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(verifyResponse{Token: tok, ExpiresAt: expiresAt})
}

// Middleware

type contextKey string

const ctxClaimsKey contextKey = "authClaims"

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		claims, err := s.JWT.ParseAndVerify(parts[1], time.Now().UTC())
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// Revocation is immediate: ensure identity is still ACTIVE.
		if _, err := s.Store.RequireActiveIdentity(claims.PublicKey); err != nil {
			http.Error(w, "identity not active", http.StatusUnauthorized)
			return
		}

		ctx := contextWithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRoles(next http.Handler, allowed ...string) http.Handler {
	allowedSet := map[string]bool{}
	for _, a := range allowed {
		allowedSet[a] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, "missing auth context", http.StatusUnauthorized)
			return
		}
		for _, role := range claims.Roles {
			if allowedSet[role] {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}

func RequireScope(next http.Handler, required string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromContext(r.Context())
		if claims == nil {
			http.Error(w, "missing auth context", http.StatusUnauthorized)
			return
		}
		for _, s := range claims.Scope {
			if s == required {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	})
}
