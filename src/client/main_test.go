package main

import (
	"os"
	"testing"
)

func TestParseManArgs(t *testing.T) {
	cases := []struct {
		args        []string
		wantName    string
		wantSection string
	}{
		{nil, "", ""},
		{[]string{}, "", ""},
		{[]string{"ls"}, "ls", ""},
		{[]string{"ls(1)"}, "ls", "1"},
		{[]string{"1", "ls"}, "ls", "1"},
		{[]string{"n", "ls"}, "ls", "n"},
		{[]string{"l", "ls"}, "ls", "l"},
		{[]string{"ls", "extra"}, "ls", ""},
	}
	for _, c := range cases {
		name, section := parseManArgs(c.args)
		if name != c.wantName || section != c.wantSection {
			t.Errorf("parseManArgs(%v) = (%q, %q), want (%q, %q)", c.args, name, section, c.wantName, c.wantSection)
		}
	}
}

func TestGetEnv_Default(t *testing.T) {
	os.Unsetenv("CASMAN_TEST_VAR")
	if got := getEnv("CASMAN_TEST_VAR", "fallback"); got != "fallback" {
		t.Errorf("getEnv = %q, want fallback", got)
	}
}

func TestGetEnv_Set(t *testing.T) {
	t.Setenv("CASMAN_TEST_VAR", "actual")
	if got := getEnv("CASMAN_TEST_VAR", "fallback"); got != "actual" {
		t.Errorf("getEnv = %q, want actual", got)
	}
}

func TestDetectMode_NonTerminalIsPlain(t *testing.T) {
	// go test's stdout is not a terminal, so this always resolves to plain
	// regardless of args, exercising the IsTerminal(false) branch.
	if got := detectMode([]string{}); got != modePlain {
		t.Errorf("detectMode([]) = %q, want %q", got, modePlain)
	}
	if got := detectMode([]string{"search", "ls"}); got != modePlain {
		t.Errorf("detectMode(search ls) = %q, want %q", got, modePlain)
	}
}

func TestPrintHelp_NoPanic(t *testing.T) {
	printHelp("man")
}

func TestPrintVersion_NoPanic(t *testing.T) {
	printVersion("man")
}
