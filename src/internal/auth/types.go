package auth

import "time"

type Role string

type Scope string

type IdentityStatus string

const (
	RoleReader  Role = "READER"
	RoleEditor  Role = "EDITOR"
	RoleFounder Role = "FOUNDER"
	RoleAdmin   Role = "ADMIN"
)

const (
	ScopePublic   Scope = "PUBLIC"
	ScopeInternal Scope = "INTERNAL"
)

const (
	StatusPending IdentityStatus = "PENDING"
	StatusActive  IdentityStatus = "ACTIVE"
	StatusRevoked IdentityStatus = "REVOKED"
)

type PublicKeyIdentity struct {
	ID        string    `json:"id"`
	PublicKey string    `json:"public_key"`
	Roles     []string  `json:"roles"`
	Scope     []string  `json:"scope"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthChallenge struct {
	ChallengeID string    `json:"challenge_id"`
	PublicKey   string    `json:"public_key"`
	Nonce       string    `json:"nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type AuthSession struct {
	Token     string    `json:"token"`
	PublicKey string    `json:"public_key"`
	Roles     []string  `json:"roles"`
	Scope     []string  `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
}
