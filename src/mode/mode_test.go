package mode

import "testing"

func TestDetect_DefaultsToProduction(t *testing.T) {
	t.Setenv("MODE", "")
	t.Setenv("DEBUG", "")
	s := Detect("", false)
	if s.Mode != Production {
		t.Errorf("Mode = %v, want Production", s.Mode)
	}
	if s.Debug {
		t.Error("Debug should be false by default")
	}
}

func TestDetect_CLIModeOverridesEnv(t *testing.T) {
	t.Setenv("MODE", "development")
	s := Detect("production", false)
	if s.Mode != Production {
		t.Errorf("CLI flag should win over env, got %v", s.Mode)
	}
}

func TestDetect_EnvModeUsedWhenNoCLI(t *testing.T) {
	t.Setenv("MODE", "dev")
	s := Detect("", false)
	if s.Mode != Development {
		t.Errorf("Mode = %v, want Development from env", s.Mode)
	}
}

func TestDetect_CLIDebugOverridesEnv(t *testing.T) {
	t.Setenv("DEBUG", "false")
	s := Detect("", true)
	if !s.Debug {
		t.Error("CLI debug flag should win over env")
	}
}

func TestDetect_EnvDebugUsedWhenNoCLI(t *testing.T) {
	t.Setenv("DEBUG", "true")
	s := Detect("", false)
	if !s.Debug {
		t.Error("Debug should be true from env")
	}
}

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"dev":         Development,
		"development": Development,
		"DEV":         Development,
		"  dev  ":     Development,
		"prod":        Production,
		"production":  Production,
		"":            Production,
		"garbage":     Production,
	}
	for in, want := range cases {
		if got := parseMode(in); got != want {
			t.Errorf("parseMode(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestState_IsProduction_IsDevelopment(t *testing.T) {
	prod := State{Mode: Production}
	dev := State{Mode: Development}

	if !prod.IsProduction() || prod.IsDevelopment() {
		t.Error("prod state should report IsProduction true, IsDevelopment false")
	}
	if dev.IsProduction() || !dev.IsDevelopment() {
		t.Error("dev state should report IsProduction false, IsDevelopment true")
	}
}

func TestState_IsDebug(t *testing.T) {
	if (State{Debug: true}).IsDebug() != true {
		t.Error("IsDebug should be true")
	}
	if (State{Debug: false}).IsDebug() != false {
		t.Error("IsDebug should be false")
	}
}

func TestState_String(t *testing.T) {
	s := State{Mode: Production, Debug: false}
	if s.String() != "production" {
		t.Errorf("String() = %q, want %q", s.String(), "production")
	}

	s = State{Mode: Development, Debug: true}
	if s.String() != "development [debugging]" {
		t.Errorf("String() = %q, want %q", s.String(), "development [debugging]")
	}
}

func TestState_LogPrefix(t *testing.T) {
	cases := []struct {
		state State
		want  string
	}{
		{State{Mode: Production, Debug: false}, "🔒"},
		{State{Mode: Production, Debug: true}, "🔒"},
		{State{Mode: Development, Debug: false}, "🔧"},
		{State{Mode: Development, Debug: true}, "🔧"},
	}
	for _, tc := range cases {
		if got := tc.state.LogPrefix(); got != tc.want {
			t.Errorf("LogPrefix(%+v) = %q, want %q", tc.state, got, tc.want)
		}
	}
}
