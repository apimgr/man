// Package server provides the HTTP server for casman.
package server

import (
	"net/http"
	"strings"

	"github.com/casapps/casman/src/common/theme"
)

// ThemePreference represents a user's theme preference.
type ThemePreference struct {
	Theme theme.ThemeName `json:"theme"`
}

// GetThemeFromRequest determines the theme from the request.
// Priority: 1. Query param 2. Cookie 3. System (defaults to dark)
// See AI.md PART 16 for details.
func GetThemeFromRequest(r *http.Request) theme.ThemeName {
	// Check query param
	if t := r.URL.Query().Get("theme"); t != "" {
		return normalizeTheme(t)
	}

	// Check cookie
	if cookie, err := r.Cookie("theme"); err == nil {
		return normalizeTheme(cookie.Value)
	}

	// Default to dark
	return theme.ThemeDark
}

// SetThemeCookie sets the theme preference cookie.
func SetThemeCookie(w http.ResponseWriter, themeName theme.ThemeName) {
	http.SetCookie(w, &http.Cookie{
		Name:     "theme",
		Value:    string(themeName),
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

// normalizeTheme normalizes a theme string to a ThemeName.
func normalizeTheme(s string) theme.ThemeName {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "light":
		return theme.ThemeLight
	case "auto", "system":
		return theme.ThemeAuto
	default:
		return theme.ThemeDark
	}
}

// ThemeCSS returns the CSS for the current theme.
func ThemeCSS(themeName theme.ThemeName) string {
	palette := theme.GetPalette(themeName)
	return `:root {` + palette.CSSVariables() + `}`
}

// ThemeClass returns the CSS class for the current theme.
func ThemeClass(themeName theme.ThemeName) string {
	switch themeName {
	case theme.ThemeLight:
		return "theme-light"
	case theme.ThemeAuto:
		return "theme-auto"
	default:
		return "theme-dark"
	}
}
