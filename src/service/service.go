// Package service provides system service management for casman.
// See AI.md PART 24 for details.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ServiceManager handles system service operations.
type ServiceManager struct {
	name string
}

// New creates a new ServiceManager.
func New(name string) *ServiceManager {
	return &ServiceManager{name: name}
}

// Install installs the application as a system service.
func (s *ServiceManager) Install() error {
	switch runtime.GOOS {
	case "linux":
		return s.installSystemd()
	case "darwin":
		return s.installLaunchd()
	case "windows":
		return s.installWindowsService()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Uninstall removes the system service.
func (s *ServiceManager) Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return s.uninstallSystemd()
	case "darwin":
		return s.uninstallLaunchd()
	case "windows":
		return s.uninstallWindowsService()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Start starts the system service.
func (s *ServiceManager) Start() error {
	switch runtime.GOOS {
	case "linux":
		return s.runCommand("systemctl", "start", s.name)
	case "darwin":
		return s.runCommand("launchctl", "start", s.name)
	case "windows":
		return s.runCommand("sc", "start", s.name)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Stop stops the system service.
func (s *ServiceManager) Stop() error {
	switch runtime.GOOS {
	case "linux":
		return s.runCommand("systemctl", "stop", s.name)
	case "darwin":
		return s.runCommand("launchctl", "stop", s.name)
	case "windows":
		return s.runCommand("sc", "stop", s.name)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// Status returns the service status.
func (s *ServiceManager) Status() (string, error) {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("systemctl", "is-active", s.name).Output()
		return strings.TrimSpace(string(out)), err
	case "darwin":
		out, err := exec.Command("launchctl", "list", s.name).Output()
		if err != nil {
			return "not running", nil
		}
		return strings.TrimSpace(string(out)), nil
	case "windows":
		out, err := exec.Command("sc", "query", s.name).Output()
		return strings.TrimSpace(string(out)), err
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// IsRunning returns true if the service is running.
func (s *ServiceManager) IsRunning() bool {
	status, err := s.Status()
	if err != nil {
		return false
	}
	return status == "active" || strings.Contains(status, "RUNNING")
}

// installSystemd creates and enables a systemd service.
func (s *ServiceManager) installSystemd() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	unit := fmt.Sprintf(`[Unit]
Description=%s service
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, s.name, binaryPath)

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", s.name)
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
		return err
	}

	if err := s.runCommand("systemctl", "daemon-reload"); err != nil {
		return err
	}

	return s.runCommand("systemctl", "enable", s.name)
}

// uninstallSystemd removes the systemd service.
func (s *ServiceManager) uninstallSystemd() error {
	_ = s.runCommand("systemctl", "stop", s.name)
	_ = s.runCommand("systemctl", "disable", s.name)

	unitPath := fmt.Sprintf("/etc/systemd/system/%s.service", s.name)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return s.runCommand("systemctl", "daemon-reload")
}

// installLaunchd creates and enables a launchd service.
func (s *ServiceManager) installLaunchd() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`, s.name, binaryPath)

	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", s.name)
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}

	return s.runCommand("launchctl", "load", plistPath)
}

// uninstallLaunchd removes the launchd service.
func (s *ServiceManager) uninstallLaunchd() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", s.name)

	_ = s.runCommand("launchctl", "unload", plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// installWindowsService creates a Windows service.
func (s *ServiceManager) installWindowsService() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	return s.runCommand("sc", "create", s.name, "binPath=", binaryPath, "start=", "auto")
}

// uninstallWindowsService removes a Windows service.
func (s *ServiceManager) uninstallWindowsService() error {
	_ = s.runCommand("sc", "stop", s.name)
	return s.runCommand("sc", "delete", s.name)
}

// runCommand executes a command.
func (s *ServiceManager) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
