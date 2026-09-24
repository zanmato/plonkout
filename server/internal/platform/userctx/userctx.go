// Package userctx carries the authenticated user through a context.
//
// It is a leaf package so the database hook and the API layer can both depend
// on it without depending on each other.
package userctx

import (
	"context"

	"github.com/gofrs/uuid/v5"
)

type key struct{}

// With returns a context acting for the user.
func With(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, key{}, userID)
}

// From returns the user the context acts for.
func From(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(key{}).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}
