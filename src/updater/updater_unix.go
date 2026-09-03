//go:build !windows
// +build !windows

// Package updater implements self-update functionality.
// This file contains Unix-specific implementations.
package updater

import (
	"fmt"
	"os"
	"syscall"
)

// ReplaceBinary replaces the running binary (Unix).
// On Unix, we can replace a running binary - the old binary stays in memory
// until the process exits, then the new one takes over on next start.
func ReplaceBinary(currentPath, newBinaryPath string) error {
	// Get current binary permissions
	info, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("failed to stat current binary: %w", err)
	}

	// Atomic rename: new binary replaces current
	// This works because Unix allows renaming over a running executable
	if err := os.Rename(newBinaryPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Restore permissions
	if err := os.Chmod(currentPath, info.Mode()); err != nil {
		return fmt.Errorf("failed to restore permissions: %w", err)
	}

	return nil
}

// RestartSelf re-executes the current process (Unix).
func RestartSelf() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// syscall.Exec replaces the current process
	return syscall.Exec(exe, os.Args, os.Environ())
}
