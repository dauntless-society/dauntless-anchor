package jwt

import (
	"errors"
	"os"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

type Signer struct {
	key []byte
	TTL time.Duration
}

type Claims struct {
	PublicKey string   `json:"public_key"`
	Roles     []string `json:"roles"`
	Scope     []string `json:"scope"`
	jwtv5.RegisteredClaims
}

func NewSignerFromKeyFile(path string, ttl time.Duration) (*Signer, error) {
	if path == "" {
		return nil, errors.New("jwt key file path required")
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(key) < 32 {
		return nil, errors.New("jwt key must be at least 32 bytes")
	}
	if ttl <= 0 {
		ttl = 20 * time.Minute
	}
	return &Signer{key: key, TTL: ttl}, nil
}

func (s *Signer) Issue(publicKey string, roles []string, scope []string, now time.Time) (token string, expiresAt time.Time, err error) {
	if publicKey == "" {
		return "", time.Time{}, errors.New("public_key required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	expiresAt = now.Add(s.TTL)

	claims := Claims{
		PublicKey: publicKey,
		Roles:     roles,
		Scope:     scope,
		RegisteredClaims: jwtv5.RegisteredClaims{
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
		},
	}

	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	ss, err := tok.SignedString(s.key)
	if err != nil {
		return "", time.Time{}, err
	}
	return ss, expiresAt, nil
}

func (s *Signer) ParseAndVerify(tokenString string, now time.Time) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	parsed, err := jwtv5.ParseWithClaims(tokenString, &Claims{}, func(token *jwtv5.Token) (any, error) {
		if token.Method != jwtv5.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return s.key, nil
	}, jwtv5.WithLeeway(30*time.Second), jwtv5.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid jwt")
	}
	if claims.PublicKey == "" {
		return nil, errors.New("jwt missing public_key")
	}
	return claims, nil
}
