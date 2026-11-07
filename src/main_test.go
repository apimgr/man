package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseFlagsImmediateActions(t *testing.T) {
	tests := []struct {
		args []string
		want actionType
	}{
		{[]string{"--help"}, actionHelp}, {[]string{"-h"}, actionHelp},
		{[]string{"--version"}, actionVersion}, {[]string{"-v"}, actionVersion},
		{[]string{"--status"}, actionStatus},
		{[]string{"--shell"}, actionShellHelp}, {[]string{"--shell", "--help"}, actionShellHelp},
		{[]string{"--shell", "completions", "zsh"}, actionShellCompletions},
		{[]string{"--shell", "init", "fish"}, actionShellInit},
		{[]string{"--service"}, actionServiceHelp}, {[]string{"--service", "--help"}, actionServiceHelp},
		{[]string{"--service", "start"}, actionService},
		{[]string{"--maintenance"}, actionMaintenanceHelp}, {[]string{"--maintenance", "--help"}, actionMaintenanceHelp},
		{[]string{"--maintenance", "backup", "file.tar.gz"}, actionMaintenance},
		{[]string{"--update"}, actionUpdate}, {[]string{"--update", "--help"}, actionUpdateHelp},
		{[]string{"--update", "branch", "stable"}, actionUpdate},
		{[]string{"--config", "/config", "--data", "/data", "--cache", "/cache", "--log", "/logs", "--backup", "/backup", "--pid", "/run/app.pid", "--address", "127.0.0.1", "--port", "8090", "--mode", "development", "--daemon", "--debug"}, actionRun},
		{nil, actionRun},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, "_"), func(t *testing.T) {
			oldArgs := os.Args
			os.Args = append([]string{"casman"}, tt.args...)
			t.Cleanup(func() { os.Args = oldArgs })
			cfg, got := parseFlags("casman")
			if got != tt.want {
				t.Fatalf("parseFlags() action = %v, want %v", got, tt.want)
			}
			if len(tt.args) > 0 && tt.args[0] == "--config" {
				if cfg.ConfigDir != "/config" || cfg.DataDir != "/data" || cfg.CacheDir != "/cache" || cfg.LogDir != "/logs" || cfg.BackupDir != "/backup" || cfg.PIDFile != "/run/app.pid" || cfg.Address != "127.0.0.1" || cfg.Port != "8090" || cfg.Mode != "development" || !cfg.Daemon || !cfg.Debug {
					t.Fatalf("parsed config flags incorrectly: %+v", cfg)
				}
			}
		})
	}
}

func TestCLIHelpAndShellOutput(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"casman"}
	t.Cleanup(func() { os.Args = oldArgs })

	originalStdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = originalStdout })

	printHelp("casman")
	printVersion("casman")
	printShellCompletions("bash")
	printShellCompletions("zsh")
	printShellCompletions("fish")
	printShellCompletions("powershell")
	printShellCompletions("unknown")
	printShellInit("bash", "casman")
	printShellInit("zsh", "casman")
	printShellInit("fish", "casman")
	printShellInit("pwsh", "casman")
	printShellInit("unknown", "casman")
	printShellHelp("casman")
	printServiceHelp("casman")
	printMaintenanceHelp("casman")
	printUpdateHelp("casman")
	if err := write.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Universal Man Page Application") || !strings.Contains(string(output), "POSIX shell completions") {
		t.Fatalf("help output did not contain expected command and shell documentation")
	}
}
