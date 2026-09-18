package config

import "testing"

func TestParseBool(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		defaultVal bool
		want       bool
		wantErr    bool
	}{
		{"empty uses default true", "", true, true, false},
		{"empty uses default false", "", false, false, false},
		{"truthy lowercase", "true", false, true, false},
		{"truthy uppercase", "TRUE", false, true, false},
		{"truthy with whitespace", "  yes  ", false, true, false},
		{"truthy numeric", "1", false, true, false},
		{"truthy synonym", "aye", false, true, false},
		{"falsy lowercase", "false", true, false, false},
		{"falsy uppercase", "FALSE", true, false, false},
		{"falsy numeric", "0", true, false, false},
		{"falsy synonym", "nope", true, false, false},
		{"invalid value", "maybe", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseBool(tc.in, tc.defaultVal)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseBool(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ParseBool(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestMustParseBool(t *testing.T) {
	if !MustParseBool("true", false) {
		t.Error("MustParseBool(true) should be true")
	}
	if MustParseBool("", false) {
		t.Error("MustParseBool empty should use default")
	}
}

func TestMustParseBool_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on invalid value")
		}
	}()
	MustParseBool("notabool", false)
}

func TestIsTruthy(t *testing.T) {
	if !IsTruthy("YES") {
		t.Error("IsTruthy(YES) should be true")
	}
	if IsTruthy("") {
		t.Error("IsTruthy(empty) should be false")
	}
	if IsTruthy("nonsense") {
		t.Error("IsTruthy(nonsense) should be false")
	}
	if IsTruthy("false") {
		t.Error("IsTruthy(false) should be false")
	}
}

func TestIsFalsy(t *testing.T) {
	if !IsFalsy("NO") {
		t.Error("IsFalsy(NO) should be true")
	}
	if IsFalsy("") {
		t.Error("IsFalsy(empty) should be false")
	}
	if IsFalsy("nonsense") {
		t.Error("IsFalsy(nonsense) should be false")
	}
	if IsFalsy("true") {
		t.Error("IsFalsy(true) should be false")
	}
}
