// Package mode handles application mode detection and configuration.
// See AI.md PART 6 for details.
package mode

import (
	"os"
	"strings"

	"github.com/casapps/casman/src/config"
)

// Mode represents the application mode.
type Mode string

const (
	// Production is the default production mode.
	Production Mode = "production"
	// Development is the development mode.
	Development Mode = "development"
)

// State holds the current application mode and debug state.
type State struct {
	Mode  Mode
	Debug bool
}

// Detect determines the application mode and debug state.
// Priority for Mode: CLI flag > MODE env var > default (production)
// Priority for Debug: CLI flag > DEBUG env var > default (false)
// See AI.md PART 6 for details.
func Detect(cliMode string, cliDebug bool) State {
	state := State{
		Mode:  Production,
		Debug: false,
	}

	// Determine mode
	if cliMode != "" {
		state.Mode = parseMode(cliMode)
	} else if envMode := os.Getenv("MODE"); envMode != "" {
		state.Mode = parseMode(envMode)
	}

	// Determine debug
	if cliDebug {
		state.Debug = true
	} else if envDebug := os.Getenv("DEBUG"); envDebug != "" {
		state.Debug = config.IsTruthy(envDebug)
	}

	return state
}

// parseMode normalizes mode string to Mode type.
func parseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dev", "development":
		return Development
	case "prod", "production":
		return Production
	default:
		return Production
	}
}

// IsProduction returns true if in production mode.
func (s State) IsProduction() bool {
	return s.Mode == Production
}

// IsDevelopment returns true if in development mode.
func (s State) IsDevelopment() bool {
	return s.Mode == Development
}

// IsDebug returns true if debug is enabled.
func (s State) IsDebug() bool {
	return s.Debug
}

// String returns a human-readable representation of the state.
func (s State) String() string {
	result := string(s.Mode)
	if s.Debug {
		result += " [debugging]"
	}
	return result
}

// LogPrefix returns the appropriate log prefix emoji.
func (s State) LogPrefix() string {
	if s.Debug {
		if s.IsProduction() {
			return "🔒"
		}
		return "🔧"
	}
	if s.IsProduction() {
		return "🔒"
	}
	return "🔧"
}
