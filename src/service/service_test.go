package service

import "testing"

func TestNew(t *testing.T) {
	s := New("test-service-name")
	if s.name != "test-service-name" {
		t.Errorf("name = %q, want test-service-name", s.name)
	}
}

func TestStatus_UnknownService(t *testing.T) {
	s := New("casman-definitely-does-not-exist-service")
	status, err := s.Status()
	if err == nil && status == "active" {
		t.Error("unexpected active status for nonexistent service")
	}
}

func TestIsRunning_UnknownService(t *testing.T) {
	s := New("casman-definitely-does-not-exist-service")
	if s.IsRunning() {
		t.Error("expected IsRunning false for nonexistent service")
	}
}

func TestRunCommand_Success(t *testing.T) {
	s := New("test")
	if err := s.runCommand("true"); err != nil {
		t.Errorf("runCommand(true) = %v, want nil", err)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	s := New("test")
	if err := s.runCommand("false"); err == nil {
		t.Error("runCommand(false) should return an error")
	}
}

func TestRunCommand_NotFound(t *testing.T) {
	s := New("test")
	if err := s.runCommand("casman-nonexistent-binary-xyz"); err == nil {
		t.Error("runCommand should error for a nonexistent binary")
	}
}
