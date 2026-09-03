//go:build windows
// +build windows

// Package updater implements self-update functionality.
// This file contains Windows-specific implementations.
package updater

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ReplaceBinary replaces the running binary (Windows).
// Windows cannot delete/rename a running executable, so we:
// 1. Rename current binary to .old
// 2. Move new binary to current path
// 3. The .old file will be cleaned up on next update or manually
func ReplaceBinary(currentPath, newBinaryPath string) error {
	oldPath := currentPath + ".old"

	// Remove any existing .old file from previous update
	os.Remove(oldPath)

	// Rename running binary to .old (this works on Windows)
	if err := os.Rename(currentPath, oldPath); err != nil {
		return fmt.Errorf("failed to rename current binary: %w", err)
	}

	// Move new binary to current path
	if err := os.Rename(newBinaryPath, currentPath); err != nil {
		// Try to restore original
		os.Rename(oldPath, currentPath)
		return fmt.Errorf("failed to move new binary: %w", err)
	}

	return nil
}

// RestartSelf starts a new instance and exits (Windows).
// Windows doesn't support exec() replacement, so we spawn new process and exit.
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Start new process
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}

	// Give the new process time to start
	time.Sleep(100 * time.Millisecond)

	// Exit current process
	os.Exit(0)
	return nil
}
