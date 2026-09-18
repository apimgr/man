package template

import (
	"bytes"
	"strings"
	"testing"

	"github.com/casapps/casman/src/server/model"
)

func TestNew(t *testing.T) {
	tmpl, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if tmpl == nil || tmpl.templates == nil {
		t.Fatal("New returned nil templates")
	}
}

func TestRender_Home(t *testing.T) {
	tmpl, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var buf bytes.Buffer
	data := HomeData{
		Title: "casman",
		Stats: model.Stats{TotalPages: 5},
	}
	if err := tmpl.Render(&buf, "home.html", data); err != nil {
		t.Fatalf("Render(home.html): %v", err)
	}
	if !strings.Contains(buf.String(), "casman") {
		t.Errorf("rendered output missing title: %q", buf.String())
	}
}

func TestRender_Error(t *testing.T) {
	tmpl, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var buf bytes.Buffer
	data := ErrorData{Code: 404, Message: "not found"}
	if err := tmpl.Render(&buf, "error.html", data); err != nil {
		t.Fatalf("Render(error.html): %v", err)
	}
	if !strings.Contains(buf.String(), "not found") {
		t.Errorf("rendered output missing message: %q", buf.String())
	}
}

func TestRender_UnknownTemplate(t *testing.T) {
	tmpl, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Render(&buf, "does-not-exist.html", nil); err == nil {
		t.Error("expected error for unknown template, got nil")
	}
}
