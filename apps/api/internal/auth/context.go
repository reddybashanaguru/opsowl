package auth

import "context"

type contextKey string

const claimsKey contextKey = "auth_claims"

func ContextWithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFromCtx(ctx context.Context) *Claims {
	c, _ := ctx.Value(claimsKey).(*Claims)
	return c
}
