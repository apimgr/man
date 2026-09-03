package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/casapps/casman/src/server/model"
)

func sampleDoc() *Document {
	return &Document{
		Title:    "ls(1)",
		Subtitle: "Linux man pages",
		Pages: []*model.ManPage{{
			Name:            "ls",
			Section:         "1",
			Platform:        "linux",
			Title:           "list directory contents",
			Synopsis:        "ls [OPTION]... [FILE]...",
			ContentText:     "List information about the FILEs (the current directory by default).",
			ContentHTML:     "<p>List information about the <em>FILEs</em>.</p>",
			ContentMarkdown: "List information about the *FILEs*.",
		}},
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]Format{
		"pdf": FormatPDF, ".pdf": FormatPDF, "PDF": FormatPDF,
		"epub": FormatEPUB, ".EPUB": FormatEPUB,
		"html": FormatHTML,
	}
	for in, want := range cases {
		got, err := ParseFormat(in)
		if err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseFormat("mobi"); err == nil {
		t.Error("expected mobi to be rejected")
	}
}

func TestBuildPDF_Smoke(t *testing.T) {
	doc := sampleDoc()
	out, ct, err := Build(doc, FormatPDF)
	if err != nil {
		t.Fatalf("BuildPDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output is not a PDF (first bytes: %q)", out[:8])
	}
	if ct != "application/pdf" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestBuildEPUB_Smoke(t *testing.T) {
	doc := sampleDoc()
	out, ct, err := Build(doc, FormatEPUB)
	if err != nil {
		t.Fatalf("BuildEPUB: %v", err)
	}
	// EPUB is a ZIP, so it starts with PK signature.
	if !bytes.HasPrefix(out, []byte{0x50, 0x4B, 0x03, 0x04}) {
		t.Errorf("output is not a ZIP/EPUB (first bytes: % x)", out[:4])
	}
	if ct != "application/epub+zip" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestBuildHTML_Smoke(t *testing.T) {
	doc := sampleDoc()
	out, ct, err := Build(doc, FormatHTML)
	if err != nil {
		t.Fatalf("BuildHTML: %v", err)
	}
	if !strings.HasPrefix(string(out), "<!DOCTYPE html>") {
		t.Errorf("output is not standalone HTML (first 30 bytes: %q)", out[:30])
	}
	if !strings.Contains(string(out), "ls(1)") {
		t.Error("output missing page title")
	}
	if ct != "text/html; charset=utf-8" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestBuild_EmptyDoc(t *testing.T) {
	if _, _, err := Build(&Document{Title: "empty"}, FormatPDF); err == nil {
		t.Error("expected error for empty doc")
	}
}

func TestBuild_UnknownFormat(t *testing.T) {
	if _, _, err := Build(sampleDoc(), Format("xyz")); err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestCatalog_HasAll(t *testing.T) {
	cat := Catalog()
	if len(cat) != len(Available()) {
		t.Errorf("Catalog has %d entries, Available has %d", len(cat), len(Available()))
	}
	ids := map[string]bool{}
	for _, fi := range cat {
		ids[fi.ID] = true
	}
	for _, want := range []string{"pdf", "epub", "html"} {
		if !ids[want] {
			t.Errorf("catalog missing %q", want)
		}
	}
}
