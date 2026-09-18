package manpage

import (
	"strings"
	"testing"
)

const groffSample = `.\" comment line
.TH TESTCMD 1 "2024-01-01" "Test Source" "Test Manual"
.SH NAME
testcmd \- do something useful
.SH SYNOPSIS
.B testcmd
.I file
.SH DESCRIPTION
testcmd does the thing.
.PP
It has more than one paragraph.
.SS Subsection Title
.TP
.B \-v
Enable verbose output
.IP item1
An indented item
.nf
preformatted line 1
preformatted line 2
.fi
.RS
indented block
.RE
.BR bold italic
.SH SEE ALSO
.BR other (1),
.BR another (5)
`

const mdocSample = `.Dd January 1, 2024
.Dt TESTCMD 1
.Os
.Sh NAME
.Nm testcmd
.Nd do something useful
.Sh SYNOPSIS
.Nm testcmd
.Op Fl v
.Ar file
.Sh DESCRIPTION
.Nm
does the thing.
.Pp
Another paragraph.
.Ss Subsection
.Bl -tag
.It Fl v
verbose flag
.El
.Bd -literal
preformatted
.Ed
.Xr other 1
.Sh SEE ALSO
.Xr another 5
`

func TestDetectFormat(t *testing.T) {
	p := NewParser()
	if got := p.detectFormat(groffSample); got != "groff" {
		t.Errorf("detectFormat(groff) = %q, want groff", got)
	}
	if got := p.detectFormat(mdocSample); got != "mdoc" {
		t.Errorf("detectFormat(mdoc) = %q, want mdoc", got)
	}
}

func TestParse_Groff(t *testing.T) {
	p := NewParser()
	page, err := p.Parse(groffSample)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if page.SourceFormat != "groff" {
		t.Errorf("SourceFormat = %q, want groff", page.SourceFormat)
	}
	if page.Name != "testcmd" {
		t.Errorf("Name = %q, want testcmd", page.Name)
	}
	if page.Section != "1" {
		t.Errorf("Section = %q, want 1", page.Section)
	}
	if page.Title == "" {
		t.Error("Title should not be empty")
	}
	if !strings.Contains(page.ContentHTML, "<h2>NAME</h2>") {
		t.Error("ContentHTML missing NAME section header")
	}
	if !strings.Contains(page.ContentHTML, "<strong>") {
		t.Error("ContentHTML missing bold rendering")
	}
	if !strings.Contains(page.ContentHTML, "<pre><code>") {
		t.Error("ContentHTML missing preformatted block")
	}
	if !strings.Contains(page.ContentText, "NAME") {
		t.Error("ContentText missing NAME section")
	}
	if !strings.Contains(page.ContentMarkdown, "## NAME") {
		t.Error("ContentMarkdown missing NAME heading")
	}
	if len(page.SeeAlso) != 2 {
		t.Errorf("SeeAlso = %v, want 2 entries", page.SeeAlso)
	}
	if page.SearchText == "" {
		t.Error("SearchText should not be empty")
	}
}

func TestParse_Mdoc(t *testing.T) {
	p := NewParser()
	page, err := p.Parse(mdocSample)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if page.SourceFormat != "mdoc" {
		t.Errorf("SourceFormat = %q, want mdoc", page.SourceFormat)
	}
	if page.Name != "testcmd" {
		t.Errorf("Name = %q, want testcmd", page.Name)
	}
	if page.Title != "do something useful" {
		t.Errorf("Title = %q, want %q", page.Title, "do something useful")
	}
	if !strings.Contains(page.ContentHTML, "<h2>SYNOPSIS</h2>") {
		t.Error("ContentHTML missing SYNOPSIS section")
	}
	if !strings.Contains(page.ContentHTML, "<pre><code>") {
		t.Error("ContentHTML missing preformatted block from Bd/Ed")
	}
	if len(page.SeeAlso) != 1 || page.SeeAlso[0] != "another(5)" {
		t.Errorf("SeeAlso = %v, want [another(5)]", page.SeeAlso)
	}
}

func TestParse_ResetsBetweenCalls(t *testing.T) {
	p := NewParser()
	if _, err := p.Parse(groffSample); err != nil {
		t.Fatalf("first Parse: %v", err)
	}
	page2, err := p.Parse(".TH OTHER 3\n.SH NAME\nother \\- another tool\n")
	if err != nil {
		t.Fatalf("second Parse: %v", err)
	}
	if page2.Name != "other" {
		t.Errorf("Name = %q, want other (state should reset between calls)", page2.Name)
	}
	if page2.Section != "3" {
		t.Errorf("Section = %q, want 3", page2.Section)
	}
	if strings.Contains(page2.ContentHTML, "testcmd") {
		t.Error("second parse should not contain content from first parse")
	}
}

func TestParse_EmptyInput(t *testing.T) {
	p := NewParser()
	page, err := p.Parse("")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if page.Name != "" {
		t.Errorf("Name = %q, want empty", page.Name)
	}
	if page.Section != "1" {
		t.Errorf("Section = %q, want default 1", page.Section)
	}
}

func TestSplitMacroLine(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{".TH NAME 1", []string{"TH", "NAME", "1"}},
		{`.SH "SEE ALSO"`, []string{"SH", "SEE ALSO"}},
		{".B", []string{"B"}},
		{"", nil},
	}
	for _, c := range cases {
		got := splitMacroLine(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitMacroLine(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitMacroLine(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestHandleAlternatingFonts(t *testing.T) {
	p := NewParser()
	// macro[1] == 'B' selects bold-first alternation.
	got := p.handleAlternatingFonts("RB", []string{"one", "two"})
	want := "__BOLD__one__/BOLD____ITALIC__two__/ITALIC__"
	if got != want {
		t.Errorf("handleAlternatingFonts(RB) = %q, want %q", got, want)
	}

	got2 := p.handleAlternatingFonts("BR", []string{"one", "two"})
	want2 := "__ITALIC__one__/ITALIC____BOLD__two__/BOLD__"
	if got2 != want2 {
		t.Errorf("handleAlternatingFonts(BR) = %q, want %q", got2, want2)
	}
}

func TestExtractName_FromNameSection(t *testing.T) {
	p := NewParser()
	p.sections["NAME"] = []string{"mytool, mytool2 - does a thing"}
	if got := p.extractName(); got != "mytool" {
		t.Errorf("extractName = %q, want mytool", got)
	}
}

func TestExtractName_NoSeparator(t *testing.T) {
	p := NewParser()
	p.sections["NAME"] = []string{"(justaword)"}
	if got := p.extractName(); got != "justaword" {
		t.Errorf("extractName = %q, want justaword", got)
	}
}

func TestExtractName_Empty(t *testing.T) {
	p := NewParser()
	if got := p.extractName(); got != "" {
		t.Errorf("extractName = %q, want empty", got)
	}
}

func TestExtractTitle_Separators(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"tool - short description", "short description"},
		{`tool \- backslash dash description`, "backslash dash description"},
		{"tool -- double dash description", "double dash description"},
	}
	for _, c := range cases {
		p := NewParser()
		p.sections["NAME"] = []string{c.line}
		if got := p.extractTitle(); got != c.want {
			t.Errorf("extractTitle(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestExtractTitle_NoMatch(t *testing.T) {
	p := NewParser()
	p.sections["NAME"] = []string{"just a plain line"}
	if got := p.extractTitle(); got != "" {
		t.Errorf("extractTitle = %q, want empty", got)
	}
}

func TestExtractDescription_TruncatesLongText(t *testing.T) {
	p := NewParser()
	long := strings.Repeat("word ", 200)
	p.sections["DESCRIPTION"] = []string{long}
	got := p.extractDescription()
	if len(got) > 500 {
		t.Errorf("extractDescription length = %d, want <= 500", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("expected truncated description to end with ...")
	}
}

func TestExtractDescription_StopsAtBlankLine(t *testing.T) {
	p := NewParser()
	p.sections["DESCRIPTION"] = []string{"first paragraph", "", "second paragraph"}
	got := p.extractDescription()
	if strings.Contains(got, "second") {
		t.Errorf("extractDescription should stop at blank line, got %q", got)
	}
}

func TestExtractSeeAlso(t *testing.T) {
	p := NewParser()
	p.sections["SEE ALSO"] = []string{"foo(1), bar(5)", "baz(8)"}
	refs := p.extractSeeAlso()
	if len(refs) != 3 {
		t.Fatalf("extractSeeAlso = %v, want 3 entries", refs)
	}
	want := map[string]bool{"foo(1)": true, "bar(5)": true, "baz(8)": true}
	for _, r := range refs {
		if !want[r] {
			t.Errorf("unexpected ref %q", r)
		}
	}
}

func TestCleanFormatting(t *testing.T) {
	p := NewParser()
	in := "__BOLD__bold__/BOLD__ __ITALIC__it__/ITALIC__ __TP__ __IP__x __SUBSECTION__y __XREF__z__/XREF__"
	got := p.cleanFormatting(in)
	for _, marker := range []string{"__BOLD__", "__/BOLD__", "__ITALIC__", "__/ITALIC__", "__TP__", "__IP__", "__SUBSECTION__", "__XREF__", "__/XREF__"} {
		if strings.Contains(got, marker) {
			t.Errorf("cleanFormatting left marker %q in %q", marker, got)
		}
	}
}

func TestFormatHTMLInline_Xref(t *testing.T) {
	p := NewParser()
	got := p.formatHTMLInline("__XREF__foo(1)__/XREF__")
	want := `<a href="/man/1/foo">foo(1)</a>`
	if got != want {
		t.Errorf("formatHTMLInline = %q, want %q", got, want)
	}
}

func TestFormatHTMLInline_EscapesHTML(t *testing.T) {
	p := NewParser()
	got := p.formatHTMLInline("<script>")
	if strings.Contains(got, "<script>") {
		t.Errorf("formatHTMLInline did not escape HTML: %q", got)
	}
}

func TestFormatMarkdownInline(t *testing.T) {
	p := NewParser()
	got := p.formatMarkdownInline("__BOLD__b__/BOLD__ __ITALIC__i__/ITALIC__")
	if got != "**b** *i*" {
		t.Errorf("formatMarkdownInline = %q, want %q", got, "**b** *i*")
	}
}

func TestHtmlEscape(t *testing.T) {
	if got := htmlEscape("<a & b>"); !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") {
		t.Errorf("htmlEscape = %q", got)
	}
}

func TestBuildSearchText(t *testing.T) {
	p := NewParser()
	p.sections["_NAME"] = []string{"tool"}
	p.sections["_TITLE"] = []string{"does things"}
	p.sections["NAME"] = []string{"tool - does things"}
	got := p.buildSearchText()
	if !strings.Contains(got, "tool") || !strings.Contains(got, "does things") {
		t.Errorf("buildSearchText = %q", got)
	}
}
