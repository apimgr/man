package theme

import (
	"strings"
	"testing"
)

func TestGetPalette(t *testing.T) {
	if got := GetPalette(ThemeLight); got != ThemePaletteLight {
		t.Errorf("GetPalette(light) = %+v, want %+v", got, ThemePaletteLight)
	}
	if got := GetPalette(ThemeDark); got != ThemePaletteDark {
		t.Errorf("GetPalette(dark) = %+v, want %+v", got, ThemePaletteDark)
	}
	if got := GetPalette(ThemeAuto); got != ThemePaletteDark {
		t.Errorf("GetPalette(auto) = %+v, want dark (default)", got)
	}
	if got := GetPalette("unknown"); got != ThemePaletteDark {
		t.Errorf("GetPalette(unknown) = %+v, want dark (default)", got)
	}
}

func TestCSSVariables(t *testing.T) {
	css := ThemePaletteDark.CSSVariables()
	if css == "" {
		t.Fatal("expected non-empty CSS")
	}
	for _, want := range []string{
		"--color-background: " + ThemePaletteDark.Background,
		"--color-primary: " + ThemePaletteDark.Primary,
		"--color-error: " + ThemePaletteDark.Error,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("CSSVariables missing %q", want)
		}
	}
}
