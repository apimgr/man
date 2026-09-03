// Package auth provides authentication and session management.
package auth

import (
	"context"
)

// contextKey is a type for context keys.
type contextKey string

const adminContextKey contextKey = "admin"

// SetAdminContext stores an admin in the request context.
func SetAdminContext(ctx context.Context, admin *Admin) context.Context {
	return context.WithValue(ctx, adminContextKey, admin)
}

// GetAdminFromContext retrieves an admin from the request context.
func GetAdminFromContext(ctx context.Context) *Admin {
	admin, ok := ctx.Value(adminContextKey).(*Admin)
	if !ok {
		return nil
	}
	return admin
}
