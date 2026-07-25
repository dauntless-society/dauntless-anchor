package handlers

import (
	"context"

	"api.dauntless-society.com/anchor/internal/auth/jwt"
)

func contextWithClaims(ctx context.Context, claims *jwt.Claims) context.Context {
	return context.WithValue(ctx, ctxClaimsKey, claims)
}

func claimsFromContext(ctx context.Context) *jwt.Claims {
	v := ctx.Value(ctxClaimsKey)
	c, _ := v.(*jwt.Claims)
	return c
}
