// Package admin provides admin panel functionality for casman.
// See AI.md PART 10 for details.
//
// Note: Authentication is handled by the auth package (src/auth/auth.go).
// This package provides admin-specific API response helpers.
package admin

import (
	"encoding/json"
	"net/http"

	"github.com/casapps/casman/src/auth"
)

// APIResponse represents an API response.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondSuccess writes a success response.
func respondSuccess(w http.ResponseWriter, data interface{}) {
	respondJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// respondError writes an error response.
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}

// The Admin type and constructor live in admin.go. This file only exposes
// the JSON response helpers and a top-level RequireAuth shim retained for
// backward compatibility — the real middleware lives in src/auth/auth.go.

// RequireAuth is middleware that requires authentication.
// Use auth.Middleware.RequireAuth instead - this is kept for compatibility.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if admin is in context (set by auth.Middleware)
		admin := auth.GetAdminFromContext(r.Context())
		if admin == nil {
			respondError(w, http.StatusUnauthorized, "Authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
