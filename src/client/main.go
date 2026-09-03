// Package main is the entry point for the casman CLI client.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/casapps/casman/src/client/cmd"
	"github.com/casapps/casman/src/client/tui"
)

// Version information - set via ldflags at build time
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = "https://casman.example.com"
)

// Display mode types
const (
	modeCLI   = "cli"
	modeTUI   = "tui"
	modePlain = "plain"
)

func main() {
	binaryName := filepath.Base(os.Args[0])

	// Parse global flags
	args := os.Args[1:]
	cfg := &cmd.Config{
		ServerURL: getEnv("CASMAN_SERVER", "http://localhost:64580"),
		Token:     getEnv("CASMAN_TOKEN", ""),
		Pager:     true,
	}

	// Extract global flags and detect mode
	var cmdArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			printHelp(binaryName)
			return
		case arg == "-v" || arg == "--version":
			printVersion(binaryName)
			return
		case arg == "--server" && i+1 < len(args):
			cfg.ServerURL = args[i+1]
			i++
		case strings.HasPrefix(arg, "--server="):
			cfg.ServerURL = strings.TrimPrefix(arg, "--server=")
		case arg == "--token" && i+1 < len(args):
			cfg.Token = args[i+1]
			i++
		case strings.HasPrefix(arg, "--token="):
			cfg.Token = strings.TrimPrefix(arg, "--token=")
		case arg == "--no-pager":
			cfg.Pager = false
		default:
			cmdArgs = append(cmdArgs, arg)
		}
	}

	// Detect display mode per PART 33
	mode := detectMode(cmdArgs)

	// Handle based on mode
	switch mode {
	case modeTUI:
		// Launch TUI application
		if err := tui.Run(cfg.ServerURL, cfg.Token); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return

	case modePlain, modeCLI:
		// If no command provided, show help
		if len(cmdArgs) == 0 {
			printHelp(binaryName)
			return
		}

		// Execute CLI command
		executeCLI(cfg, cmdArgs, binaryName)
	}
}

// detectMode determines the display mode based on environment and arguments.
// Per PART 33: Auto-detect TUI vs CLI based on terminal and command presence.
func detectMode(args []string) string {
	// Not a terminal = plain output (piped/redirected)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return modePlain
	}

	// Config-only flags don't prevent TUI
	configFlags := map[string]bool{
		"--config": true, "--server": true, "--token": true, "--debug": true,
	}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			// Command/arg provided = CLI mode
			return modeCLI
		}
		flag := strings.Split(arg, "=")[0]
		if !configFlags[flag] {
			// Action flag = CLI mode
			return modeCLI
		}
	}

	// No args or config-only flags = TUI mode (interactive terminal)
	return modeTUI
}

// executeCLI runs CLI commands
func executeCLI(cfg *cmd.Config, cmdArgs []string, binaryName string) {
	// Create command runner
	runner := cmd.New(cfg)

	// Execute command
	command := cmdArgs[0]
	cmdArgs = cmdArgs[1:]

	var err error
	switch command {
	case "man":
		if len(cmdArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: man [SECTION] NAME")
			os.Exit(1)
		}
		name, section := parseManArgs(cmdArgs)
		err = runner.Man(name, section)

	case "search":
		if len(cmdArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: search QUERY")
			os.Exit(1)
		}
		query := strings.Join(cmdArgs, " ")
		err = runner.Search(query, "", "")

	case "whatis":
		if len(cmdArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: whatis NAME")
			os.Exit(1)
		}
		err = runner.Whatis(cmdArgs[0])

	case "apropos":
		if len(cmdArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: apropos QUERY")
			os.Exit(1)
		}
		query := strings.Join(cmdArgs, " ")
		err = runner.Apropos(query)

	case "stats":
		err = runner.Stats()

	case "sections":
		err = runner.Sections()

	case "platforms":
		err = runner.Platforms()

	case "health":
		err = runner.HealthCheck()

	default:
		// Treat unknown command as man page name
		name, section := parseManArgs(cmdArgs)
		if command != "" {
			if name != "" {
				name = command + " " + name
			} else {
				name = command
			}
		}
		err = runner.Man(name, section)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// parseManArgs parses man command arguments.
// Supports: NAME, SECTION NAME, NAME(SECTION)
func parseManArgs(args []string) (name, section string) {
	if len(args) == 0 {
		return "", ""
	}

	if len(args) == 1 {
		arg := args[0]
		// Check for name(section) format
		if idx := strings.Index(arg, "("); idx > 0 {
			if endIdx := strings.Index(arg, ")"); endIdx > idx {
				return arg[:idx], arg[idx+1 : endIdx]
			}
		}
		return arg, ""
	}

	// Check if first arg looks like a section number
	first := args[0]
	if len(first) <= 2 && (first[0] >= '1' && first[0] <= '9' || first == "n" || first == "l") {
		return args[1], first
	}

	// Otherwise, treat as name with possible section
	return args[0], ""
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func printHelp(binaryName string) {
	fmt.Printf(`%s %s - casman CLI Client

Usage:
  %s [global-options] <command> [arguments]
  %s                          Launch interactive TUI

Global Options:
  -h, --help           Show help
  -v, --version        Show version
  --server URL         Server URL (default: $CASMAN_SERVER or http://localhost:64580)
  --token TOKEN        API token (default: $CASMAN_TOKEN)
  --no-pager           Don't use pager for output

Commands:
  man [SECTION] NAME   View man page (default command)
  search QUERY         Search man pages
  whatis NAME          Show one-line description
  apropos QUERY        Search descriptions
  stats                Show database statistics
  sections             List all sections
  platforms            List all platforms
  health               Check server health

Examples:
  %s                            Launch TUI mode
  %s ls                          View ls(1) man page
  %s 5 passwd                    View passwd(5) man page
  %s man grep                    View grep man page
  %s search "file copy"          Search for file copy
  %s whatis chmod                Show chmod description
  %s apropos permission          Search for permission-related pages
  %s --server https://example.com man ls
`, binaryName, Version, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func printVersion(binaryName string) {
	fmt.Printf("%s %s\n", binaryName, Version)
	fmt.Printf("  Commit:  %s\n", CommitID)
	fmt.Printf("  Built:   %s\n", BuildDate)
	fmt.Printf("  Site:    %s\n", OfficialSite)
}
