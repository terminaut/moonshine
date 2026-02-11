package middleware

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type contextKey string

const userIDKey contextKey = "userID"

var errUnauthorized = errors.New("unauthorized")

// ContextWithUserID returns a new context with the given user ID set.
// This is intended for use in tests and middleware.
func ContextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	v := ctx.Value(userIDKey)
	if v == nil {
		return uuid.Nil, errUnauthorized
	}

	switch id := v.(type) {
	case uuid.UUID:
		return id, nil
	case string:
		parsed, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, errUnauthorized
		}
		return parsed, nil
	default:
		return uuid.Nil, errUnauthorized
	}
}
