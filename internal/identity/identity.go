package identity

import (
	"context"
	"time"
)

// Token is a short-lived workload identity credential.
// Value must never be logged, printed, or serialized into status output.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// TokenSource mints workload identity tokens for a caller-chosen audience.
type TokenSource interface {
	Token(ctx context.Context, audience string) (Token, error)
}
