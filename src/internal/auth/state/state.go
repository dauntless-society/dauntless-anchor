package state

import "context"

// Principal represents the authenticated identity and permissions for a request.
type Principal struct {
	PublicKey string
	Roles     []string
	Scope     []string
	Token     string // optional: raw bearer token, if you want to keep it
}

type principalKey struct{}

// WithPrincipal stores the principal on the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFromContext returns the principal from the context.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	v := ctx.Value(principalKey{})
	if v == nil {
		return Principal{}, false
	}
	p, ok := v.(Principal)
	return p, ok && p.PublicKey != ""
}

// PublicKeyFromContext returns the public key if present.
func PublicKeyFromContext(ctx context.Context) (string, bool) {
	p, ok := PrincipalFromContext(ctx)
	if !ok {
		return "", false
	}
	return p.PublicKey, true
}
