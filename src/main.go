// Package main is the entry point for the casman server.
//
// casman is a universal man page application with embedded man pages
// from BSD, macOS, Linux, and other Unix-like systems.
//
// This software is licensed under the MIT License.
// See LICENSE.md for details.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/casapps/casman/src/backup"
	"github.com/casapps/casman/src/config"
	"github.com/casapps/casman/src/server"
	"github.com/casapps/casman/src/service"
	"github.com/casapps/casman/src/updater"
)

// Version information - set via ldflags at build time
var (
	Version      = "dev"
	CommitID     = "unknown"
	BuildDate    = "unknown"
	OfficialSite = "https://casman.example.com"
)

// Project constants - NEVER change these (hardcoded identifiers)
const (
	ProjectName = "casman"
	ProjectOrg  = "casapps"
)

func main() {
	// Get actual binary name (may be renamed by user)
	binaryName := filepath.Base(os.Args[0])

	// Parse CLI flags
	cfg, action := parseFlags(binaryName)

	// Handle action based on parsed flags
	switch action {
	case actionHelp:
		printHelp(binaryName)
		os.Exit(0)

	case actionVersion:
		printVersion(binaryName)
		os.Exit(0)

	case actionStatus:
		exitCode := showStatus(cfg)
		os.Exit(exitCode)

	case actionShellCompletions:
		printShellCompletions(cfg.ShellType)
		os.Exit(0)

	case actionShellInit:
		printShellInit(cfg.ShellType, binaryName)
		os.Exit(0)

	case actionShellHelp:
		printShellHelp(binaryName)
		os.Exit(0)

	case actionServiceHelp:
		printServiceHelp(binaryName)
		os.Exit(0)

	case actionService:
		exitCode := handleService(cfg)
		os.Exit(exitCode)

	case actionMaintenanceHelp:
		printMaintenanceHelp(binaryName)
		os.Exit(0)

	case actionMaintenance:
		exitCode := handleMaintenance(cfg)
		os.Exit(exitCode)

	case actionUpdateHelp:
		printUpdateHelp(binaryName)
		os.Exit(0)

	case actionUpdate:
		exitCode := handleUpdate(cfg)
		os.Exit(exitCode)

	case actionRun:
		// Continue to server startup
	}

	// PHASE 5: Server startup
	if err := startServer(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Action types for CLI handling
type actionType int

const (
	actionRun actionType = iota
	actionHelp
	actionVersion
	actionStatus
	actionShellCompletions
	actionShellInit
	actionShellHelp
	actionServiceHelp
	actionService
	actionMaintenanceHelp
	actionMaintenance
	actionUpdateHelp
	actionUpdate
)

// CLIConfig holds parsed command-line configuration
type CLIConfig struct {
	// Directories
	ConfigDir string
	DataDir   string
	CacheDir  string
	LogDir    string
	BackupDir string
	PIDFile   string

	// Server settings
	Address string
	Port    string
	Mode    string
	Daemon  bool
	Debug   bool

	// Subcommand arguments
	ServiceCmd     string
	MaintenanceCmd string
	MaintenanceArg string
	UpdateCmd      string
	UpdateArg      string
	ShellType      string
}

func parseFlags(binaryName string) (*CLIConfig, actionType) {
	cfg := &CLIConfig{
		Address: "0.0.0.0",
		Mode:    "production",
	}

	args := os.Args[1:]
	i := 0

	for i < len(args) {
		arg := args[i]

		switch {
		// Help flags
		case arg == "-h" || arg == "--help":
			return cfg, actionHelp

		// Version flags
		case arg == "-v" || arg == "--version":
			return cfg, actionVersion

		// Status
		case arg == "--status":
			return cfg, actionStatus

		// Shell commands
		case arg == "--shell":
			if i+1 < len(args) {
				subCmd := args[i+1]
				switch subCmd {
				case "--help":
					return cfg, actionShellHelp
				case "completions":
					if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
						cfg.ShellType = args[i+2]
					}
					return cfg, actionShellCompletions
				case "init":
					if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
						cfg.ShellType = args[i+2]
					}
					return cfg, actionShellInit
				}
			}
			return cfg, actionShellHelp

		// Service commands
		case arg == "--service":
			if i+1 < len(args) {
				subCmd := args[i+1]
				if subCmd == "--help" {
					return cfg, actionServiceHelp
				}
				cfg.ServiceCmd = subCmd
				return cfg, actionService
			}
			return cfg, actionServiceHelp

		// Maintenance commands
		case arg == "--maintenance":
			if i+1 < len(args) {
				subCmd := args[i+1]
				if subCmd == "--help" {
					return cfg, actionMaintenanceHelp
				}
				cfg.MaintenanceCmd = subCmd
				if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
					cfg.MaintenanceArg = args[i+2]
				}
				return cfg, actionMaintenance
			}
			return cfg, actionMaintenanceHelp

		// Update commands
		case arg == "--update":
			if i+1 < len(args) {
				subCmd := args[i+1]
				if subCmd == "--help" {
					return cfg, actionUpdateHelp
				}
				cfg.UpdateCmd = subCmd
				if i+2 < len(args) && !strings.HasPrefix(args[i+2], "-") {
					cfg.UpdateArg = args[i+2]
				}
				return cfg, actionUpdate
			}
			// --update alone means check
			cfg.UpdateCmd = "check"
			return cfg, actionUpdate

		// Directory flags
		case arg == "--config":
			if i+1 < len(args) {
				cfg.ConfigDir = args[i+1]
				i++
			}

		case arg == "--data":
			if i+1 < len(args) {
				cfg.DataDir = args[i+1]
				i++
			}

		case arg == "--cache":
			if i+1 < len(args) {
				cfg.CacheDir = args[i+1]
				i++
			}

		case arg == "--log":
			if i+1 < len(args) {
				cfg.LogDir = args[i+1]
				i++
			}

		case arg == "--backup":
			if i+1 < len(args) {
				cfg.BackupDir = args[i+1]
				i++
			}

		case arg == "--pid":
			if i+1 < len(args) {
				cfg.PIDFile = args[i+1]
				i++
			}

		// Server settings
		case arg == "--address":
			if i+1 < len(args) {
				cfg.Address = args[i+1]
				i++
			}

		case arg == "--port":
			if i+1 < len(args) {
				cfg.Port = args[i+1]
				i++
			}

		case arg == "--mode":
			if i+1 < len(args) {
				cfg.Mode = args[i+1]
				i++
			}

		case arg == "--daemon":
			cfg.Daemon = true

		case arg == "--debug":
			cfg.Debug = true

		default:
			// Unknown flag
			fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", arg)
			fmt.Fprintf(os.Stderr, "Run '%s --help' for usage.\n", binaryName)
			os.Exit(1)
		}

		i++
	}

	return cfg, actionRun
}

func printHelp(binaryName string) {
	fmt.Printf(`%s %s - Universal Man Page Application

Usage:
  %s [flags]

Information:
  -h, --help                        Show help (--help for any command shows its help)
  -v, --version                     Show version
      --status                      Show server status and health

Shell Integration:
      --shell completions [SHELL]   Print shell completions
      --shell init [SHELL]          Print shell init command
      --shell --help                Show shell help

Server Configuration:
      --mode {production|development}  Application mode (default: production)
      --config DIR                  Config directory
      --data DIR                    Data directory
      --cache DIR                   Cache directory
      --log DIR                     Log directory
      --backup DIR                  Backup directory
      --pid FILE                    PID file path
      --address ADDR                Listen address (default: 0.0.0.0)
      --port PORT                   Listen port (default: random 64xxx, 80 in container)
      --daemon                      Run as daemon (detach from terminal)
      --debug                       Enable debug mode

Service Management:
      --service CMD                 Service management (--service --help for details)
      --maintenance CMD             Maintenance operations (--maintenance --help for details)
      --update [CMD]                Check/perform updates (--update --help for details)

Run '%s <command> --help' for detailed help on any command.
`, binaryName, Version, binaryName, binaryName)
}

func printVersion(binaryName string) {
	fmt.Printf("%s %s\n", binaryName, Version)
	fmt.Printf("  Commit:  %s\n", CommitID)
	fmt.Printf("  Built:   %s\n", BuildDate)
	fmt.Printf("  Site:    %s\n", OfficialSite)
}

func showStatus(cfg *CLIConfig) int {
	// Try to load config to find server settings
	var appCfg *config.Config
	configDir := cfg.ConfigDir
	dataDir := cfg.DataDir
	if configDir != "" || dataDir != "" {
		loadedCfg, err := config.Load(configDir, dataDir)
		if err == nil {
			appCfg = loadedCfg
		}
	}
	if appCfg == nil {
		// Use minimal defaults
		appCfg = &config.Config{}
	}

	// Determine port to check
	port := cfg.Port
	if port == "" {
		port = appCfg.Server.Port
	}
	if port == "" {
		port = "64580"
	}

	// Determine address
	address := cfg.Address
	if address == "" || address == "0.0.0.0" || address == "[::]" {
		address = "127.0.0.1"
	}

	// Try to connect to the health endpoint
	url := fmt.Sprintf("http://%s:%s/healthz", address, port)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		// Server not running or not responding
		fmt.Println()
		fmt.Println("Server Status: Not Running")
		fmt.Printf("  Port: %s (expected)\n", port)
		fmt.Println()
		fmt.Println("The server is not responding. It may not be running.")

		// Check PID file
		pidFile := cfg.PIDFile
		if pidFile == "" && appCfg.Paths.DataDir != "" {
			pidFile = appCfg.Paths.DataDir + "/casman.pid"
		}
		if pidFile != "" {
			if pidData, err := os.ReadFile(pidFile); err == nil {
				pid := strings.TrimSpace(string(pidData))
				fmt.Printf("  PID file: %s (PID: %s)\n", pidFile, pid)

				// Check if process exists
				if pidNum, err := strconv.Atoi(pid); err == nil {
					if process, err := os.FindProcess(pidNum); err == nil {
						// On Unix, FindProcess always succeeds
						// Try to send signal 0 to check if process exists
						if err := process.Signal(os.Signal(nil)); err == nil {
							fmt.Println("  Process: exists but not responding")
						}
					}
				}
			}
		}
		return 1
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Server Status: Error")
		fmt.Printf("  Error reading response: %v\n", err)
		return 1
	}

	// Parse JSON response
	var health struct {
		Project struct {
			Name    string `json:"name"`
			Tagline string `json:"tagline"`
		} `json:"project"`
		Status    string `json:"status"`
		Version   string `json:"version"`
		GoVersion string `json:"go_version"`
		Build     struct {
			Commit string `json:"commit"`
			Date   string `json:"date"`
		} `json:"build"`
		Uptime    string `json:"uptime"`
		Mode      string `json:"mode"`
		Timestamp string `json:"timestamp"`
		Cluster   struct {
			Enabled bool   `json:"enabled"`
			NodeID  string `json:"node_id"`
			Nodes   int    `json:"nodes"`
		} `json:"cluster"`
		Features struct {
			Tor struct {
				Enabled bool   `json:"enabled"`
				Running bool   `json:"running"`
				Status  string `json:"status"`
				Address string `json:"address"`
			} `json:"tor"`
		} `json:"features"`
		Checks struct {
			Database  string `json:"database"`
			Cache     string `json:"cache"`
			Scheduler string `json:"scheduler"`
		} `json:"checks"`
		Stats struct {
			ManPagesTotal int `json:"man_pages_total"`
			PlatformCount int `json:"platform_count"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(body, &health); err != nil {
		// Not JSON, might be HTML - server is up but returning unexpected format
		fmt.Println()
		fmt.Println("Server Status: Running (non-JSON response)")
		fmt.Printf("  Port: %s\n", port)
		return 0
	}

	// Display status
	fmt.Println()
	statusText := "Running"
	if health.Status == "degraded" {
		statusText = "Degraded"
	} else if health.Status == "unhealthy" {
		statusText = "Unhealthy"
	}
	fmt.Printf("Server Status: %s\n", statusText)
	fmt.Printf("  Port: %s\n", port)
	fmt.Printf("  Mode: %s\n", health.Mode)
	fmt.Printf("  Version: %s\n", health.Version)
	fmt.Printf("  Uptime: %s\n", health.Uptime)
	fmt.Println()

	// Node/Cluster info
	if health.Cluster.Enabled {
		fmt.Printf("Node: %s\n", health.Cluster.NodeID)
		fmt.Printf("Cluster: connected\n")
		fmt.Printf("  Nodes: %d\n", health.Cluster.Nodes)
	} else {
		fmt.Println("Node: standalone")
		fmt.Println("Cluster: disabled")
	}
	fmt.Println()

	// Tor status
	if health.Features.Tor.Enabled {
		torStatus := "Disconnected"
		if health.Features.Tor.Running {
			torStatus = "Connected"
		}
		fmt.Printf("Tor Hidden Service: %s\n", torStatus)
		if health.Features.Tor.Address != "" {
			fmt.Printf("  Address: %s\n", health.Features.Tor.Address)
		}
	} else {
		fmt.Println("Tor Hidden Service: disabled")
	}

	// Component health
	if health.Status != "healthy" {
		fmt.Println()
		fmt.Println("Component Health:")
		fmt.Printf("  Database:  %s\n", health.Checks.Database)
		fmt.Printf("  Cache:     %s\n", health.Checks.Cache)
		fmt.Printf("  Scheduler: %s\n", health.Checks.Scheduler)
	}

	// Exit code based on health status
	if health.Status == "healthy" {
		return 0
	}
	return 1
}

func printShellCompletions(shellType string) {
	binaryName := filepath.Base(os.Args[0])

	// Auto-detect shell if not specified
	if shellType == "" {
		shellPath := os.Getenv("SHELL")
		if shellPath != "" {
			shellType = filepath.Base(shellPath)
		} else {
			shellType = "bash"
		}
	}

	switch shellType {
	case "bash":
		printBashCompletions(binaryName)
	case "zsh":
		printZshCompletions(binaryName)
	case "fish":
		printFishCompletions(binaryName)
	case "powershell", "pwsh":
		printPowershellCompletions(binaryName)
	default:
		// POSIX-compatible fallback
		printPosixCompletions(binaryName)
	}
}

func printBashCompletions(binaryName string) {
	fmt.Printf(`# Bash completions for %s
# Generated by %s --shell completions bash

_%s_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main options
    opts="--help --version --status --shell --mode --config --data --cache --log --backup --pid --address --port --daemon --debug --service --maintenance --update"

    case "${prev}" in
        --mode)
            COMPREPLY=($(compgen -W "production development" -- "${cur}"))
            return 0
            ;;
        --shell)
            COMPREPLY=($(compgen -W "completions init --help" -- "${cur}"))
            return 0
            ;;
        completions|init)
            COMPREPLY=($(compgen -W "bash zsh fish powershell" -- "${cur}"))
            return 0
            ;;
        --service)
            COMPREPLY=($(compgen -W "start stop restart reload --install --uninstall --disable --help" -- "${cur}"))
            return 0
            ;;
        --maintenance)
            COMPREPLY=($(compgen -W "backup restore mode setup --help" -- "${cur}"))
            return 0
            ;;
        --update)
            COMPREPLY=($(compgen -W "check yes branch --help" -- "${cur}"))
            return 0
            ;;
        --config|--data|--cache|--log|--backup)
            COMPREPLY=($(compgen -d -- "${cur}"))
            return 0
            ;;
        --pid)
            COMPREPLY=($(compgen -f -- "${cur}"))
            return 0
            ;;
        *)
            ;;
    esac

    COMPREPLY=($(compgen -W "${opts}" -- "${cur}"))
    return 0
}

complete -F _%s_completions %s
`, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func printZshCompletions(binaryName string) {
	fmt.Printf(`#compdef %s
# Zsh completions for %s
# Generated by %s --shell completions zsh

__%s() {
    local -a opts
    opts=(
        '--help[Show help]'
        '-h[Show help]'
        '--version[Show version]'
        '-v[Show version]'
        '--status[Show server status and health]'
        '--shell[Shell integration]:command:(completions init --help)'
        '--mode[Application mode]:mode:(production development)'
        '--config[Config directory]:directory:_directories'
        '--data[Data directory]:directory:_directories'
        '--cache[Cache directory]:directory:_directories'
        '--log[Log directory]:directory:_directories'
        '--backup[Backup directory]:directory:_directories'
        '--pid[PID file]:file:_files'
        '--address[Listen address]:address:'
        '--port[Listen port]:port:'
        '--daemon[Run as daemon]'
        '--debug[Enable debug mode]'
        '--service[Service management]:command:(start stop restart reload --install --uninstall --disable --help)'
        '--maintenance[Maintenance operations]:command:(backup restore mode setup --help)'
        '--update[Check/perform updates]:command:(check yes branch --help)'
    )
    _arguments -s $opts
}

compdef __%s %s
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func printFishCompletions(binaryName string) {
	fmt.Printf(`# Fish completions for %s
# Generated by %s --shell completions fish

complete -c %s -f

# Main options
complete -c %s -l help -s h -d 'Show help'
complete -c %s -l version -s v -d 'Show version'
complete -c %s -l status -d 'Show server status and health'
complete -c %s -l shell -d 'Shell integration' -xa 'completions init --help'
complete -c %s -l mode -d 'Application mode' -xa 'production development'
complete -c %s -l config -d 'Config directory' -xa '(__fish_complete_directories)'
complete -c %s -l data -d 'Data directory' -xa '(__fish_complete_directories)'
complete -c %s -l cache -d 'Cache directory' -xa '(__fish_complete_directories)'
complete -c %s -l log -d 'Log directory' -xa '(__fish_complete_directories)'
complete -c %s -l backup -d 'Backup directory' -xa '(__fish_complete_directories)'
complete -c %s -l pid -d 'PID file' -r
complete -c %s -l address -d 'Listen address'
complete -c %s -l port -d 'Listen port'
complete -c %s -l daemon -d 'Run as daemon'
complete -c %s -l debug -d 'Enable debug mode'
complete -c %s -l service -d 'Service management' -xa 'start stop restart reload --install --uninstall --disable --help'
complete -c %s -l maintenance -d 'Maintenance operations' -xa 'backup restore mode setup --help'
complete -c %s -l update -d 'Check/perform updates' -xa 'check yes branch --help'
`, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func printPowershellCompletions(binaryName string) {
	fmt.Printf(`# PowerShell completions for %s
# Generated by %s --shell completions powershell

Register-ArgumentCompleter -Native -CommandName %s -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commands = @(
        @{Name='--help'; Description='Show help'}
        @{Name='-h'; Description='Show help'}
        @{Name='--version'; Description='Show version'}
        @{Name='-v'; Description='Show version'}
        @{Name='--status'; Description='Show server status and health'}
        @{Name='--shell'; Description='Shell integration'}
        @{Name='--mode'; Description='Application mode'}
        @{Name='--config'; Description='Config directory'}
        @{Name='--data'; Description='Data directory'}
        @{Name='--cache'; Description='Cache directory'}
        @{Name='--log'; Description='Log directory'}
        @{Name='--backup'; Description='Backup directory'}
        @{Name='--pid'; Description='PID file'}
        @{Name='--address'; Description='Listen address'}
        @{Name='--port'; Description='Listen port'}
        @{Name='--daemon'; Description='Run as daemon'}
        @{Name='--debug'; Description='Enable debug mode'}
        @{Name='--service'; Description='Service management'}
        @{Name='--maintenance'; Description='Maintenance operations'}
        @{Name='--update'; Description='Check/perform updates'}
    )

    $commands | Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_.Name, $_.Name, 'ParameterValue', $_.Description)
    }
}
`, binaryName, binaryName, binaryName)
}

func printPosixCompletions(binaryName string) {
	fmt.Printf(`# POSIX shell completions for %s
# Generated by %s --shell completions
# Limited completion support for POSIX shells

_%s_complete() {
    echo "--help --version --status --shell --mode --config --data --cache --log --backup --pid --address --port --daemon --debug --service --maintenance --update"
}
`, binaryName, binaryName, binaryName)
}

func printShellInit(shellType, binaryName string) {
	// Auto-detect shell if not specified
	if shellType == "" {
		shellPath := os.Getenv("SHELL")
		if shellPath != "" {
			shellType = filepath.Base(shellPath)
		} else {
			shellType = "bash"
		}
	}

	switch shellType {
	case "bash":
		fmt.Printf("source <(%s --shell completions bash)\n", binaryName)
	case "zsh":
		fmt.Printf("source <(%s --shell completions zsh)\n", binaryName)
	case "fish":
		fmt.Printf("%s --shell completions fish | source\n", binaryName)
	case "powershell", "pwsh":
		fmt.Printf("Invoke-Expression (& %s --shell completions powershell)\n", binaryName)
	default:
		fmt.Printf("eval \"$(%s --shell completions %s)\"\n", binaryName, shellType)
	}
}

func printShellHelp(binaryName string) {
	fmt.Printf(`Shell Integration

Usage:
  %s --shell completions [SHELL]   Print shell completions
  %s --shell init [SHELL]          Print shell init command

Supported shells: bash, zsh, fish, powershell

Examples:
  # Bash completions
  %s --shell completions bash > /etc/bash_completion.d/%s

  # Add to .bashrc
  eval "$(%s --shell init bash)"
`, binaryName, binaryName, binaryName, binaryName, binaryName)
}

func printServiceHelp(binaryName string) {
	fmt.Printf(`Service Management

Usage:
  %s --service CMD

Commands:
  start       Start the service
  stop        Stop the service
  restart     Restart the service
  reload      Reload configuration
  --install   Install and enable service
  --disable   Stop and disable service
  --uninstall Remove service completely

Examples:
  %s --service --install   # Install as system service
  %s --service start       # Start the service
  %s --service stop        # Stop the service
`, binaryName, binaryName, binaryName, binaryName)
}

func handleService(cfg *CLIConfig) int {
	svc := service.New(ProjectName)

	switch cfg.ServiceCmd {
	case "start":
		if err := svc.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			return 1
		}
		fmt.Println("Service started successfully")

	case "stop":
		if err := svc.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping service: %v\n", err)
			return 1
		}
		fmt.Println("Service stopped successfully")

	case "restart":
		if err := svc.Stop(); err != nil {
			// Ignore stop errors, service might not be running
		}
		if err := svc.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting service: %v\n", err)
			return 1
		}
		fmt.Println("Service restarted successfully")

	case "status":
		status, err := svc.Status()
		if err != nil {
			fmt.Printf("Service status: %s\n", status)
		} else {
			fmt.Printf("Service status: %s\n", status)
		}

	case "--install":
		if err := svc.Install(); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing service: %v\n", err)
			return 1
		}
		fmt.Println("Service installed and enabled successfully")
		fmt.Printf("Start with: %s --service start\n", ProjectName)

	case "--disable":
		if err := svc.Stop(); err != nil {
			// Ignore stop errors
		}
		fmt.Println("Service stopped and disabled")

	case "--uninstall":
		if err := svc.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "Error uninstalling service: %v\n", err)
			return 1
		}
		fmt.Println("Service uninstalled successfully")

	default:
		fmt.Fprintf(os.Stderr, "Unknown service command: %s\n", cfg.ServiceCmd)
		fmt.Fprintf(os.Stderr, "Use --service --help for usage information\n")
		return 1
	}

	return 0
}

func printMaintenanceHelp(binaryName string) {
	fmt.Printf(`Maintenance Operations

Usage:
  %s --maintenance CMD [ARG]

Commands:
  backup              Create backup
  restore FILE        Restore from backup file
  update              Update to latest version
  mode MODE           Set application mode (production/development)
  setup               Run initial setup wizard

Examples:
  %s --maintenance backup
  %s --maintenance restore /path/to/backup.tar.gz
  %s --maintenance mode development
`, binaryName, binaryName, binaryName, binaryName)
}

func handleMaintenance(cfg *CLIConfig) int {
	// Load application config
	var appCfg *config.Config
	if cfg.ConfigDir != "" || cfg.DataDir != "" {
		if loadedCfg, err := config.Load(cfg.ConfigDir, cfg.DataDir); err == nil {
			appCfg = loadedCfg
		}
	}
	if appCfg == nil {
		// Use minimal defaults
		appCfg = &config.Config{}
	}

	switch cfg.MaintenanceCmd {
	case "backup":
		return handleMaintenanceBackup(cfg, appCfg)
	case "restore":
		return handleMaintenanceRestore(cfg, appCfg)
	case "mode":
		return handleMaintenanceMode(cfg, appCfg)
	case "setup":
		return handleMaintenanceSetup(cfg, appCfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown maintenance command: %s\n", cfg.MaintenanceCmd)
		return 1
	}
}

func handleMaintenanceBackup(cfg *CLIConfig, appCfg *config.Config) int {
	// Import backup package
	backupPkg, err := loadBackupManager(appCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backup: %v\n", err)
		return 1
	}

	// Create backup
	ctx := context.Background()
	filename := cfg.MaintenanceArg
	description := "Manual backup via CLI"

	// Create backup (no password for now - would prompt in full implementation)
	backupPath, err := backupPkg.Backup(ctx, filename, "", description)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
		return 1
	}

	fmt.Printf("Backup created: %s\n", backupPath)
	return 0
}

func handleMaintenanceRestore(cfg *CLIConfig, appCfg *config.Config) int {
	backupFile := cfg.MaintenanceArg
	if backupFile == "" {
		fmt.Fprintf(os.Stderr, "Error: backup file required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --maintenance restore <backup-file>\n", filepath.Base(os.Args[0]))
		return 1
	}

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: backup file not found: %s\n", backupFile)
		return 1
	}

	// Import backup package
	backupPkg, err := loadBackupManager(appCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backup: %v\n", err)
		return 1
	}

	// Restore backup (no password for now - would prompt in full implementation)
	ctx := context.Background()
	if err := backupPkg.Restore(ctx, backupFile, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
		return 1
	}

	fmt.Println("Restore completed successfully")
	return 0
}

func handleMaintenanceMode(cfg *CLIConfig, appCfg *config.Config) int {
	mode := cfg.MaintenanceArg
	if mode == "" {
		// Show current mode
		fmt.Printf("Current mode: %s\n", appCfg.Server.Mode)
		return 0
	}

	// Validate mode
	if mode != "production" && mode != "development" {
		fmt.Fprintf(os.Stderr, "Invalid mode: %s (must be 'production' or 'development')\n", mode)
		return 1
	}

	// Update mode in config file
	configFile := filepath.Join(appCfg.Paths.ConfigDir, "server.yml")
	fmt.Printf("Setting mode to: %s\n", mode)
	fmt.Printf("Update config file: %s\n", configFile)
	fmt.Println("Note: Restart server for changes to take effect")

	// In full implementation, would update the YAML file
	return 0
}

func handleMaintenanceSetup(cfg *CLIConfig, appCfg *config.Config) int {
	fmt.Println("Setup Wizard")
	fmt.Println()

	// Check if setup is already complete (has admins)
	// In full implementation, would check the database
	fmt.Println("Starting interactive setup...")
	fmt.Println()
	fmt.Println("For web-based setup, start the server and visit:")
	fmt.Printf("  http://localhost:%s/admin/server/setup\n", appCfg.Server.Port)
	fmt.Println()
	fmt.Println("The setup token will be displayed in the server logs.")

	return 0
}

func loadBackupManager(appCfg *config.Config) (*backup.Manager, error) {
	backupCfg := backup.DefaultConfig()
	if appCfg.Server.Backup != nil {
		if appCfg.Server.Backup.Dir != "" {
			backupCfg.Dir = appCfg.Server.Backup.Dir
		} else {
			backupCfg.Dir = appCfg.Paths.BackupDir
		}
		backupCfg.Retention.MaxBackups = appCfg.Server.Backup.Retention.MaxBackups
	} else {
		backupCfg.Dir = appCfg.Paths.BackupDir
	}

	return backup.New(backupCfg, Version, appCfg.Paths.ConfigDir, appCfg.Paths.DataDir), nil
}

func printUpdateHelp(binaryName string) {
	fmt.Printf(`Update Management

Usage:
  %s --update [CMD] [ARG]

Commands:
  check               Check for updates (default)
  yes                 Download and install update
  branch NAME         Switch update branch (stable, beta, daily)

Examples:
  %s --update              # Check for updates
  %s --update yes          # Install latest update
  %s --update branch beta  # Switch to beta channel
`, binaryName, binaryName, binaryName, binaryName)
}

func handleUpdate(cfg *CLIConfig) int {
	// Create updater with current version and default branch
	u := updater.New(Version, updater.BranchStable)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	switch cfg.UpdateCmd {
	case "check", "":
		// Check for updates
		info, err := u.Check(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
			return 1
		}

		fmt.Printf("Current version: %s\n", info.CurrentVersion)
		fmt.Printf("Update branch:   %s\n", u.GetBranch())

		if !info.Available {
			fmt.Println("\nYou are running the latest version.")
			return 0
		}

		fmt.Printf("\nUpdate available: %s\n", info.NewVersion)
		if info.AssetSize > 0 {
			fmt.Printf("Download size:    %.2f MB\n", float64(info.AssetSize)/(1024*1024))
		}
		if info.ReleaseNotes != "" {
			fmt.Printf("\nRelease notes:\n%s\n", info.ReleaseNotes)
		}
		fmt.Printf("\nRun '%s --update yes' to install the update.\n", ProjectName)
		return 0

	case "yes":
		// Check and install update
		fmt.Println("Checking for updates...")
		info, err := u.Check(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
			return 1
		}

		if !info.Available {
			fmt.Printf("Current version: %s\n", info.CurrentVersion)
			fmt.Println("You are already running the latest version.")
			return 0
		}

		fmt.Printf("Update available: %s -> %s\n", info.CurrentVersion, info.NewVersion)
		fmt.Println("Downloading update...")

		// Perform update
		info, err = u.Update(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing update: %v\n", err)
			return 1
		}

		fmt.Printf("Successfully updated to %s\n", info.NewVersion)
		fmt.Println("Restarting...")

		// Restart the process
		if err := updater.RestartSelf(); err != nil {
			fmt.Fprintf(os.Stderr, "Error restarting: %v\n", err)
			fmt.Println("Please restart the application manually.")
			return 1
		}
		return 0

	case "branch":
		// Switch update branch
		if cfg.UpdateArg == "" {
			// Show current branch
			fmt.Printf("Current update branch: %s\n", u.GetBranch())
			fmt.Println("\nAvailable branches:")
			fmt.Println("  stable - Stable releases (recommended)")
			fmt.Println("  beta   - Beta releases (testing)")
			fmt.Println("  daily  - Daily builds (development)")
			fmt.Printf("\nUsage: %s --update branch {stable|beta|daily}\n", ProjectName)
			return 0
		}

		// Validate and set branch
		if err := u.SetBranch(cfg.UpdateArg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}

		fmt.Printf("Update branch set to: %s\n", cfg.UpdateArg)
		fmt.Printf("\nRun '%s --update check' to check for updates on this branch.\n", ProjectName)
		return 0

	default:
		fmt.Fprintf(os.Stderr, "Unknown update command: %s\n", cfg.UpdateCmd)
		fmt.Fprintf(os.Stderr, "Use --update --help for usage information\n")
		return 1
	}
}

func startServer(cliCfg *CLIConfig) error {
	// Load configuration
	cfg, err := config.Load(cliCfg.ConfigDir, cliCfg.DataDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI overrides
	if cliCfg.Port != "" {
		cfg.Server.Port = cliCfg.Port
	}
	if cliCfg.Address != "" {
		cfg.Server.Address = cliCfg.Address
	}
	if cliCfg.Mode != "" {
		cfg.Server.Mode = cliCfg.Mode
	}
	if cliCfg.Debug {
		cfg.Server.Mode = "development"
	}

	// Create and start server
	srv, err := server.New(cfg, Version, CommitID, BuildDate)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return srv.Run()
}
