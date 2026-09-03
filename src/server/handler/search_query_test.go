package handler

import "testing"

func TestParseSearchQuery_Plain(t *testing.T) {
	q := ParseSearchQuery("ls files")
	if q.Query != "ls files" || q.Section != "" || q.Platform != "" || q.Mode != "any" {
		t.Errorf("plain parse mismatched: %+v", q)
	}
}

func TestParseSearchQuery_NamePrefix(t *testing.T) {
	q := ParseSearchQuery("name:ls")
	if q.Query != "ls" || q.Mode != "name" {
		t.Errorf("name prefix: %+v", q)
	}
}

func TestParseSearchQuery_ContentPrefix(t *testing.T) {
	q := ParseSearchQuery("content:directory")
	if q.Query != "directory" || q.Mode != "content" {
		t.Errorf("content prefix: %+v", q)
	}
}

func TestParseSearchQuery_FilterPrefixes(t *testing.T) {
	q := ParseSearchQuery("os:linux section:1 copy")
	if q.Query != "copy" || q.Section != "1" || q.Platform != "linux" {
		t.Errorf("filter prefixes: %+v", q)
	}
}

func TestParseSearchQuery_BareSectionLeading(t *testing.T) {
	q := ParseSearchQuery("1 ls")
	if q.Query != "ls" || q.Section != "1" {
		t.Errorf("bare section: %+v", q)
	}
}

func TestParseSearchQuery_BareSectionLetter(t *testing.T) {
	q := ParseSearchQuery("x xterm")
	if q.Query != "xterm" || q.Section != "x" {
		t.Errorf("bare section letter: %+v", q)
	}
}

func TestParseSearchQuery_PrefixSectionWinsOverBare(t *testing.T) {
	q := ParseSearchQuery("section:8 1 ifconfig")
	// section: prefix should claim the slot; the bare `1` becomes part of
	// the residual.
	if q.Section != "8" {
		t.Errorf("section: %q, want 8", q.Section)
	}
	if q.Query != "1 ifconfig" {
		t.Errorf("query: %q, want '1 ifconfig'", q.Query)
	}
}

func TestParseSearchQuery_PlatformAlias(t *testing.T) {
	q := ParseSearchQuery("platform:freebsd zfs")
	if q.Platform != "freebsd" || q.Query != "zfs" {
		t.Errorf("platform alias: %+v", q)
	}
}

func TestMergeURLParams_OverridesPrefixes(t *testing.T) {
	q := ParseSearchQuery("os:linux section:1 ls").MergeURLParams("8", "freebsd")
	if q.Section != "8" || q.Platform != "freebsd" {
		t.Errorf("URL override failed: %+v", q)
	}
}

func TestMergeURLParams_AnyTreatedAsEmpty(t *testing.T) {
	q := ParseSearchQuery("os:linux section:1 ls").MergeURLParams("any", "any")
	if q.Section != "1" || q.Platform != "linux" {
		t.Errorf("'any' should not override: %+v", q)
	}
}
