package state

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"api.dauntless-society.com/anchor/internal/auth"

	"github.com/google/uuid"
)

var b64 = base64.RawURLEncoding

type Store struct {
	Dir string
}

func New(dir string) *Store { return &Store{Dir: dir} }

func (s *Store) Ensure() error {
	if s.Dir == "" {
		return errors.New("auth state dir required")
	}
	return os.MkdirAll(s.Dir, 0755)
}

func (s *Store) lockPath() string       { return filepath.Join(s.Dir, "auth.lock") }
func (s *Store) identitiesPath() string { return filepath.Join(s.Dir, "identities.json") }
func (s *Store) challengesPath() string { return filepath.Join(s.Dir, "challenges.json") }

type identitiesFile struct {
	Schema     string                   `json:"schema"`
	UpdatedAt  time.Time                `json:"updated_at"`
	Identities []auth.PublicKeyIdentity `json:"identities"`
}

type challengesFile struct {
	Schema    string               `json:"schema"`
	UpdatedAt time.Time            `json:"updated_at"`
	Items     []auth.AuthChallenge `json:"items"`
}

func (s *Store) withLock(fn func() error) error {
	if err := s.Ensure(); err != nil {
		return err
	}
	f, err := os.OpenFile(s.lockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

func readJSON[T any](path string, dst *T) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dst)
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func normalizePublicKey(pk string) string { return strings.TrimSpace(pk) }

func validateRoles(roles []string) error {
	allowed := map[string]bool{
		string(auth.RoleEditor):  true,
		string(auth.RoleFounder): true,
		string(auth.RoleAdmin):   true,
	}
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			return errors.New("empty role")
		}
		if !allowed[r] {
			return errors.New("invalid role")
		}
	}
	return nil
}

func (s *Store) RegisterPending(publicKey string, requestedRoles []string) (registrationID string, err error) {
	publicKey = normalizePublicKey(publicKey)
	if publicKey == "" {
		return "", errors.New("public_key required")
	}
	if err := validateRoles(requestedRoles); err != nil {
		return "", err
	}

	registrationID = uuid.New().String()
	return registrationID, s.withLock(func() error {
		var f identitiesFile
		_ = readJSON(s.identitiesPath(), &f)
		if f.Schema == "" {
			f.Schema = "dauntless-anchor.auth.identities.v1"
		}

		for _, existing := range f.Identities {
			if existing.PublicKey != publicKey {
				continue
			}
			if existing.Status == string(auth.StatusPending) {
				registrationID = existing.ID
				return nil
			}
			return errors.New("public key already registered")
		}

		f.Identities = append(f.Identities, auth.PublicKeyIdentity{
			ID:        registrationID,
			PublicKey: publicKey,
			Roles:     []string{},
			Scope:     []string{string(auth.ScopePublic)},
			Status:    string(auth.StatusPending),
			CreatedAt: time.Now().UTC(),
		})
		f.UpdatedAt = time.Now().UTC()
		return writeJSONAtomic(s.identitiesPath(), f)
	})
}

func (s *Store) GetIdentity(publicKey string) (*auth.PublicKeyIdentity, error) {
	publicKey = normalizePublicKey(publicKey)
	if publicKey == "" {
		return nil, errors.New("public_key required")
	}

	var out *auth.PublicKeyIdentity
	err := s.withLock(func() error {
		var f identitiesFile
		if err := readJSON(s.identitiesPath(), &f); err != nil {
			return err
		}
		for _, existing := range f.Identities {
			if existing.PublicKey == publicKey {
				e := existing
				out = &e
				return nil
			}
		}
		return nil
	})
	return out, err
}

func (s *Store) RequireActiveIdentity(publicKey string) (*auth.PublicKeyIdentity, error) {
	id, err := s.GetIdentity(publicKey)
	if err != nil {
		return nil, err
	}
	if id == nil {
		return nil, errors.New("identity not found")
	}
	if id.Status != string(auth.StatusActive) {
		return nil, errors.New("identity not active")
	}
	return id, nil
}

func (s *Store) Approve(publicKey string, approvedRoles []string) error {
	publicKey = normalizePublicKey(publicKey)
	if publicKey == "" {
		return errors.New("public_key required")
	}
	if len(approvedRoles) == 0 {
		return errors.New("approved_roles required")
	}
	if err := validateRoles(approvedRoles); err != nil {
		return err
	}

	return s.withLock(func() error {
		var f identitiesFile
		_ = readJSON(s.identitiesPath(), &f)
		for i := range f.Identities {
			if f.Identities[i].PublicKey != publicKey {
				continue
			}
			switch f.Identities[i].Status {
			case string(auth.StatusActive):
				return errors.New("identity already active")
			case string(auth.StatusRevoked):
				return errors.New("identity revoked")
			}
			f.Identities[i].Roles = append([]string{}, approvedRoles...)
			f.Identities[i].Status = string(auth.StatusActive)
			f.UpdatedAt = time.Now().UTC()
			return writeJSONAtomic(s.identitiesPath(), f)
		}
		return errors.New("identity not found")
	})
}

func (s *Store) Revoke(publicKey string) error {
	publicKey = normalizePublicKey(publicKey)
	if publicKey == "" {
		return errors.New("public_key required")
	}

	return s.withLock(func() error {
		var f identitiesFile
		_ = readJSON(s.identitiesPath(), &f)
		for i := range f.Identities {
			if f.Identities[i].PublicKey != publicKey {
				continue
			}
			f.Identities[i].Status = string(auth.StatusRevoked)
			f.UpdatedAt = time.Now().UTC()
			return writeJSONAtomic(s.identitiesPath(), f)
		}
		return errors.New("identity not found")
	})
}

func randomNonceB64(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b64.EncodeToString(b), nil
}

func (s *Store) CreateChallenge(publicKey string, ttl time.Duration) (*auth.AuthChallenge, error) {
	publicKey = normalizePublicKey(publicKey)
	if publicKey == "" {
		return nil, errors.New("public_key required")
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	nonce, err := randomNonceB64(32)
	if err != nil {
		return nil, err
	}
	ch := auth.AuthChallenge{
		ChallengeID: uuid.New().String(),
		PublicKey:   publicKey,
		Nonce:       nonce,
		ExpiresAt:   time.Now().UTC().Add(ttl),
	}

	return &ch, s.withLock(func() error {
		// Ensure the identity is ACTIVE at the time of challenge creation.
		var ids identitiesFile
		_ = readJSON(s.identitiesPath(), &ids)
		active := false
		for _, id := range ids.Identities {
			if id.PublicKey == publicKey && id.Status == string(auth.StatusActive) {
				active = true
				break
			}
		}
		if !active {
			return errors.New("identity not active")
		}

		var f challengesFile
		_ = readJSON(s.challengesPath(), &f)
		if f.Schema == "" {
			f.Schema = "dauntless-anchor.auth.challenges.v1"
		}

		now := time.Now().UTC()
		items := make([]auth.AuthChallenge, 0, len(f.Items)+1)
		for _, item := range f.Items {
			if !item.ExpiresAt.IsZero() && item.ExpiresAt.Before(now) {
				continue
			}
			if item.PublicKey == publicKey {
				continue
			}
			items = append(items, item)
		}
		items = append(items, ch)
		f.Items = items
		f.UpdatedAt = now
		return writeJSONAtomic(s.challengesPath(), f)
	})
}

func (s *Store) ConsumeChallenge(publicKey string, nonce string) error {
	publicKey = normalizePublicKey(publicKey)
	nonce = strings.TrimSpace(nonce)
	if publicKey == "" || nonce == "" {
		return errors.New("public_key and nonce required")
	}

	return s.withLock(func() error {
		var f challengesFile
		_ = readJSON(s.challengesPath(), &f)

		now := time.Now().UTC()
		kept := make([]auth.AuthChallenge, 0, len(f.Items))
		found := false
		for _, item := range f.Items {
			if !item.ExpiresAt.IsZero() && item.ExpiresAt.Before(now) {
				continue
			}
			if item.PublicKey == publicKey && item.Nonce == nonce {
				found = true
				continue
			}
			kept = append(kept, item)
		}

		f.Items = kept
		f.UpdatedAt = now
		if err := writeJSONAtomic(s.challengesPath(), f); err != nil {
			return err
		}
		if !found {
			return errors.New("challenge not found")
		}
		return nil
	})
}
