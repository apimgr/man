package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/casapps/casman/src/common/theme"
)

func TestNormalizeTheme(t *testing.T) {
	cases := map[string]theme.ThemeName{
		"light":  theme.ThemeLight,
		"LIGHT":  theme.ThemeLight,
		" auto ": theme.ThemeAuto,
		"system": theme.ThemeAuto,
		"dark":   theme.ThemeDark,
		"":       theme.ThemeDark,
		"bogus":  theme.ThemeDark,
	}
	for in, want := range cases {
		if got := normalizeTheme(in); got != want {
			t.Errorf("normalizeTheme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetThemeFromRequest_QueryParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/?theme=light", nil)
	if got := GetThemeFromRequest(r); got != theme.ThemeLight {
		t.Errorf("GetThemeFromRequest(query) = %q, want light", got)
	}
}

func TestGetThemeFromRequest_Cookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "theme", Value: "auto"})
	if got := GetThemeFromRequest(r); got != theme.ThemeAuto {
		t.Errorf("GetThemeFromRequest(cookie) = %q, want auto", got)
	}
}

func TestGetThemeFromRequest_DefaultsToDark(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := GetThemeFromRequest(r); got != theme.ThemeDark {
		t.Errorf("GetThemeFromRequest(none) = %q, want dark", got)
	}
}

func TestGetThemeFromRequest_QueryWinsOverCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/?theme=light", nil)
	r.AddCookie(&http.Cookie{Name: "theme", Value: "auto"})
	if got := GetThemeFromRequest(r); got != theme.ThemeLight {
		t.Errorf("GetThemeFromRequest(query+cookie) = %q, want light (query wins)", got)
	}
}

func TestSetThemeCookie(t *testing.T) {
	w := httptest.NewRecorder()
	SetThemeCookie(w, theme.ThemeLight)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "theme" || cookies[0].Value != "light" {
		t.Errorf("unexpected cookies: %+v", cookies)
	}
}

func TestThemeCSS(t *testing.T) {
	css := ThemeCSS(theme.ThemeDark)
	if !strings.HasPrefix(css, ":root {") || !strings.Contains(css, "--color-background") {
		t.Errorf("ThemeCSS = %q", css)
	}
}

func TestThemeClass(t *testing.T) {
	cases := map[theme.ThemeName]string{
		theme.ThemeLight: "theme-light",
		theme.ThemeAuto:  "theme-auto",
		theme.ThemeDark:  "theme-dark",
	}
	for in, want := range cases {
		if got := ThemeClass(in); got != want {
			t.Errorf("ThemeClass(%q) = %q, want %q", in, got, want)
		}
	}
}
