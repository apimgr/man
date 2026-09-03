// Package theme provides project-wide theme definitions for casman.
// These colors are the single source of truth used across Web, TUI, CLI, and GUI.
// See AI.md PART 16 for details.
package theme

// ThemePalette defines the color scheme for the application.
type ThemePalette struct {
	Background string `json:"background"`
	Foreground string `json:"foreground"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Success    string `json:"success"`
	Warning    string `json:"warning"`
	Error      string `json:"error"`
	Info       string `json:"info"`
	Surface    string `json:"surface"`
	SurfaceAlt string `json:"surface_alt"`
	Border     string `json:"border"`
	Muted      string `json:"muted"`
}

// ThemePaletteDark is the dark theme color palette (default).
var ThemePaletteDark = ThemePalette{
	Background: "#1a1b26",
	Foreground: "#c0caf5",
	Primary:    "#7aa2f7",
	Secondary:  "#9ece6a",
	Accent:     "#bb9af7",
	Success:    "#9ece6a",
	Warning:    "#e0af68",
	Error:      "#f7768e",
	Info:       "#7dcfff",
	Surface:    "#24283b",
	SurfaceAlt: "#1f2335",
	Border:     "#414868",
	Muted:      "#565f89",
}

// ThemePaletteLight is the light theme color palette.
var ThemePaletteLight = ThemePalette{
	Background: "#ffffff",
	Foreground: "#1a1b26",
	Primary:    "#2e7de9",
	Secondary:  "#587539",
	Accent:     "#7847bd",
	Success:    "#587539",
	Warning:    "#8c6c3e",
	Error:      "#c64343",
	Info:       "#007197",
	Surface:    "#f5f5f5",
	SurfaceAlt: "#e9e9ec",
	Border:     "#c0caf5",
	Muted:      "#6172b0",
}

// ThemeName represents a theme name.
type ThemeName string

const (
	// ThemeDark is the dark theme.
	ThemeDark ThemeName = "dark"
	// ThemeLight is the light theme.
	ThemeLight ThemeName = "light"
	// ThemeAuto follows system preference.
	ThemeAuto ThemeName = "auto"
)

// GetPalette returns the palette for the given theme name.
func GetPalette(name ThemeName) ThemePalette {
	switch name {
	case ThemeLight:
		return ThemePaletteLight
	default:
		return ThemePaletteDark
	}
}

// CSSVariables returns CSS custom properties for the palette.
func (p ThemePalette) CSSVariables() string {
	return `
  --color-background: ` + p.Background + `;
  --color-foreground: ` + p.Foreground + `;
  --color-primary: ` + p.Primary + `;
  --color-secondary: ` + p.Secondary + `;
  --color-accent: ` + p.Accent + `;
  --color-success: ` + p.Success + `;
  --color-warning: ` + p.Warning + `;
  --color-error: ` + p.Error + `;
  --color-info: ` + p.Info + `;
  --color-surface: ` + p.Surface + `;
  --color-surface-alt: ` + p.SurfaceAlt + `;
  --color-border: ` + p.Border + `;
  --color-muted: ` + p.Muted + `;
`
}
