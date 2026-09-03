// Package main provides a tool to download and process man pages.
// This is run at build time to populate the embedded database.
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/casapps/casman/src/manpage"
	"github.com/casapps/casman/src/server/model"
	"github.com/casapps/casman/src/server/store"
)

var (
	dbPath    = flag.String("db", "data/manpages.db", "Path to output database")
	sourceDir = flag.String("source", "", "Directory containing man pages to load")
	platform  = flag.String("platform", "linux", "Platform name for loaded pages")
	distro    = flag.String("distro", "", "Distribution name")
	version   = flag.String("version", "", "Version string")
	download  = flag.Bool("download", false, "Download man pages from online sources")
	verbose   = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ltime)

	// Create output directory
	dir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	// Open database
	db, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if *download {
		if err := downloadManPages(db); err != nil {
			log.Fatalf("Failed to download man pages: %v", err)
		}
	}

	if *sourceDir != "" {
		if err := loadFromDirectory(db, *sourceDir); err != nil {
			log.Fatalf("Failed to load from directory: %v", err)
		}
	}

	// If no source specified, load sample pages for testing
	if !*download && *sourceDir == "" {
		log.Println("No source specified, loading sample man pages...")
		if err := loadSamplePages(db); err != nil {
			log.Fatalf("Failed to load sample pages: %v", err)
		}
	}

	log.Println("Done!")
}

// loadFromDirectory loads man pages from a local directory.
func loadFromDirectory(db *store.DB, dir string) error {
	parser := manpage.NewParser()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip non-man files
		if !isManPageFile(path) {
			return nil
		}

		if *verbose {
			log.Printf("Loading: %s", path)
		}

		// Read file content
		content, err := readManPage(path)
		if err != nil {
			log.Printf("Warning: failed to read %s: %v", path, err)
			return nil
		}

		// Parse the man page
		page, err := parser.Parse(content)
		if err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			return nil
		}

		// Extract section from filename or path
		section := extractSectionFromPath(path)
		if page.Section == "" || page.Section == "1" {
			page.Section = section
		}

		// Set platform/distro/version
		page.Platform = *platform
		page.Distro = *distro
		page.Version = *version
		page.Language = "en"

		// If name is empty, extract from filename
		if page.Name == "" {
			page.Name = extractNameFromPath(path)
		}

		// Insert into database
		modelPage := &model.ManPage{
			Name:            page.Name,
			Section:         page.Section,
			Title:           page.Title,
			Platform:        page.Platform,
			Distro:          page.Distro,
			Version:         page.Version,
			Language:        page.Language,
			SourceFormat:    page.SourceFormat,
			SourceRaw:       page.SourceRaw,
			ContentHTML:     page.ContentHTML,
			ContentText:     page.ContentText,
			ContentMarkdown: page.ContentMarkdown,
			Synopsis:        page.Synopsis,
			Description:     page.Description,
			SearchText:      page.SearchText,
		}

		// Convert see also
		for _, ref := range page.SeeAlso {
			entry := model.SeeAlsoEntry{Name: ref}
			if idx := strings.Index(ref, "("); idx > 0 {
				entry.Name = ref[:idx]
				if endIdx := strings.Index(ref, ")"); endIdx > idx {
					entry.Section = ref[idx+1 : endIdx]
				}
			}
			modelPage.SeeAlso = append(modelPage.SeeAlso, entry)
		}

		if err := db.InsertManPage(modelPage); err != nil {
			log.Printf("Warning: failed to insert %s: %v", page.Name, err)
		}

		return nil
	})
}

// isManPageFile checks if a file is a man page.
func isManPageFile(path string) bool {
	ext := filepath.Ext(path)

	// Check for compressed files
	if ext == ".gz" || ext == ".bz2" || ext == ".xz" {
		base := strings.TrimSuffix(path, ext)
		ext = filepath.Ext(base)
	}

	// Man page sections
	sections := []string{".1", ".2", ".3", ".4", ".5", ".6", ".7", ".8", ".9", ".n", ".l"}
	for _, s := range sections {
		if ext == s {
			return true
		}
		// Also check subsections like .1p, .3pm
		if strings.HasPrefix(ext, s) {
			return true
		}
	}

	// Check if file is in a section directory (1/, 2/, 3/, etc.)
	// This handles our deduped structure: platform/section/name
	dir := filepath.Base(filepath.Dir(path))
	if len(dir) == 1 && ((dir[0] >= '1' && dir[0] <= '9') || dir[0] == 'n') {
		return true
	}

	return false
}

// readManPage reads a man page file, handling compression.
func readManPage(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var reader io.Reader = f

	// Handle gzip compression
	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// extractSectionFromPath extracts the section number from a file path.
func extractSectionFromPath(path string) string {
	// Try to find man[1-9n] in path
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, "man") && len(part) >= 4 {
			section := part[3:]
			if len(section) >= 1 && (section[0] >= '1' && section[0] <= '9' || section[0] == 'n') {
				return string(section[0])
			}
		}
	}

	// Check if parent directory is a section number (1/, 2/, 3/, etc.)
	// This handles our deduped structure: platform/section/name
	dir := filepath.Base(filepath.Dir(path))
	if len(dir) == 1 && ((dir[0] >= '1' && dir[0] <= '9') || dir[0] == 'n') {
		return dir
	}

	// Extract from file extension
	ext := filepath.Ext(path)
	if ext == ".gz" || ext == ".bz2" || ext == ".xz" {
		base := strings.TrimSuffix(path, ext)
		ext = filepath.Ext(base)
	}

	if len(ext) >= 2 && ext[0] == '.' {
		return string(ext[1])
	}

	return "1"
}

// extractNameFromPath extracts the page name from a file path.
func extractNameFromPath(path string) string {
	base := filepath.Base(path)

	// Remove compression extension
	if strings.HasSuffix(base, ".gz") {
		base = strings.TrimSuffix(base, ".gz")
	} else if strings.HasSuffix(base, ".bz2") {
		base = strings.TrimSuffix(base, ".bz2")
	} else if strings.HasSuffix(base, ".xz") {
		base = strings.TrimSuffix(base, ".xz")
	}

	// Remove section extension
	ext := filepath.Ext(base)
	if len(ext) >= 2 && ext[0] == '.' {
		base = strings.TrimSuffix(base, ext)
	}

	return base
}

// downloadManPages downloads man pages from online sources.
func downloadManPages(db *store.DB) error {
	log.Println("Downloading man pages from online sources...")

	// Download Linux man pages from man7.org
	if err := downloadMan7Pages(db); err != nil {
		log.Printf("Warning: man7.org download failed: %v", err)
	}

	return nil
}

// downloadMan7Pages downloads pages from man7.org.
func downloadMan7Pages(db *store.DB) error {
	// List of essential man pages to download
	pages := []struct {
		name    string
		section string
	}{
		{"ls", "1"}, {"cat", "1"}, {"grep", "1"}, {"find", "1"}, {"chmod", "1"},
		{"chown", "1"}, {"cp", "1"}, {"mv", "1"}, {"rm", "1"}, {"mkdir", "1"},
		{"rmdir", "1"}, {"pwd", "1"}, {"cd", "1"}, {"echo", "1"}, {"date", "1"},
		{"head", "1"}, {"tail", "1"}, {"wc", "1"}, {"sort", "1"}, {"uniq", "1"},
		{"cut", "1"}, {"paste", "1"}, {"tr", "1"}, {"sed", "1"}, {"awk", "1"},
		{"diff", "1"}, {"patch", "1"}, {"tar", "1"}, {"gzip", "1"}, {"bzip2", "1"},
		{"ssh", "1"}, {"scp", "1"}, {"rsync", "1"}, {"wget", "1"}, {"curl", "1"},
		{"ps", "1"}, {"top", "1"}, {"kill", "1"}, {"man", "1"}, {"less", "1"},
		{"more", "1"}, {"vi", "1"}, {"vim", "1"}, {"nano", "1"}, {"touch", "1"},
		{"ln", "1"}, {"du", "1"}, {"df", "1"}, {"mount", "8"}, {"umount", "8"},
		{"open", "2"}, {"read", "2"}, {"write", "2"}, {"close", "2"}, {"fork", "2"},
		{"exec", "3"}, {"exit", "3"}, {"malloc", "3"}, {"free", "3"}, {"printf", "3"},
	}

	client := &http.Client{Timeout: 30 * time.Second}
	parser := manpage.NewParser()

	for _, p := range pages {
		url := fmt.Sprintf("https://man7.org/linux/man-pages/man%s/%s.%s.html", p.section, p.name, p.section)

		if *verbose {
			log.Printf("Downloading: %s", url)
		}

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Warning: failed to download %s: %v", p.name, err)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			log.Printf("Warning: %s returned %d", p.name, resp.StatusCode)
			continue
		}

		content, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Warning: failed to read %s: %v", p.name, err)
			continue
		}

		// man7.org returns HTML, but we need the raw source
		// For now, store the HTML as-is and create a basic text version
		htmlContent := string(content)

		// Create a model page
		modelPage := &model.ManPage{
			Name:        p.name,
			Section:     p.section,
			Title:       fmt.Sprintf("%s - Linux man page", p.name),
			Platform:    "linux",
			Distro:      "man-pages",
			Language:    "en",
			SourceURL:   url,
			ContentHTML: htmlContent,
		}

		// Try to parse as groff to generate proper content
		// First, try to get the raw source
		rawURL := fmt.Sprintf("https://git.kernel.org/pub/scm/docs/man-pages/man-pages.git/plain/man%s/%s.%s", p.section, p.name, p.section)
		rawResp, err := client.Get(rawURL)
		if err == nil && rawResp.StatusCode == 200 {
			rawContent, _ := io.ReadAll(rawResp.Body)
			rawResp.Body.Close()

			page, err := parser.Parse(string(rawContent))
			if err == nil {
				modelPage.Title = page.Title
				modelPage.Synopsis = page.Synopsis
				modelPage.Description = page.Description
				modelPage.SourceFormat = page.SourceFormat
				modelPage.SourceRaw = page.SourceRaw
				modelPage.ContentHTML = page.ContentHTML
				modelPage.ContentText = page.ContentText
				modelPage.ContentMarkdown = page.ContentMarkdown
				modelPage.SearchText = page.SearchText

				for _, ref := range page.SeeAlso {
					entry := model.SeeAlsoEntry{Name: ref}
					if idx := strings.Index(ref, "("); idx > 0 {
						entry.Name = ref[:idx]
						if endIdx := strings.Index(ref, ")"); endIdx > idx {
							entry.Section = ref[idx+1 : endIdx]
						}
					}
					modelPage.SeeAlso = append(modelPage.SeeAlso, entry)
				}
			}
		} else if rawResp != nil {
			rawResp.Body.Close()
		}

		if err := db.InsertManPage(modelPage); err != nil {
			log.Printf("Warning: failed to insert %s: %v", p.name, err)
		}

		// Rate limit
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// loadSamplePages loads a set of sample man pages for testing.
func loadSamplePages(db *store.DB) error {
	parser := manpage.NewParser()

	// Sample man pages with real groff content
	samples := []struct {
		name     string
		section  string
		platform string
		raw      string
	}{
		{
			name:     "ls",
			section:  "1",
			platform: "linux",
			raw: `.TH LS 1 "2024-01-01" "GNU coreutils 9.4" "User Commands"
.SH NAME
ls \- list directory contents
.SH SYNOPSIS
.B ls
[\fIOPTION\fR]... [\fIFILE\fR]...
.SH DESCRIPTION
List information about the FILEs (the current directory by default).
Sort entries alphabetically if none of \fB\-cftuvSUX\fR nor \fB\-\-sort\fR is specified.
.PP
Mandatory arguments to long options are mandatory for short options too.
.SH OPTIONS
.TP
\fB\-a\fR, \fB\-\-all\fR
do not ignore entries starting with .
.TP
\fB\-A\fR, \fB\-\-almost\-all\fR
do not list implied . and ..
.TP
\fB\-\-author\fR
with \fB\-l\fR, print the author of each file
.TP
\fB\-b\fR, \fB\-\-escape\fR
print C-style escapes for nongraphic characters
.TP
\fB\-\-block\-size\fR=\fISIZE\fR
scale sizes by SIZE before printing them
.TP
\fB\-B\fR, \fB\-\-ignore\-backups\fR
do not list implied entries ending with ~
.TP
\fB\-c\fR
with \fB\-lt\fR: sort by, and show, ctime
.TP
\fB\-C\fR
list entries by columns
.TP
\fB\-\-color\fR[=\fIWHEN\fR]
colorize the output; WHEN can be 'always', 'auto', or 'never'
.TP
\fB\-d\fR, \fB\-\-directory\fR
list directories themselves, not their contents
.TP
\fB\-F\fR, \fB\-\-classify\fR
append indicator (one of */=>@|) to entries
.TP
\fB\-h\fR, \fB\-\-human\-readable\fR
with \fB\-l\fR and/or \fB\-s\fR, print human readable sizes
.TP
\fB\-i\fR, \fB\-\-inode\fR
print the index number of each file
.TP
\fB\-l\fR
use a long listing format
.TP
\fB\-n\fR, \fB\-\-numeric\-uid\-gid\fR
like \fB\-l\fR, but list numeric user and group IDs
.TP
\fB\-r\fR, \fB\-\-reverse\fR
reverse order while sorting
.TP
\fB\-R\fR, \fB\-\-recursive\fR
list subdirectories recursively
.TP
\fB\-s\fR, \fB\-\-size\fR
print the allocated size of each file, in blocks
.TP
\fB\-S\fR
sort by file size, largest first
.TP
\fB\-t\fR
sort by modification time, newest first
.TP
\fB\-\-help\fR
display this help and exit
.TP
\fB\-\-version\fR
output version information and exit
.SH EXAMPLES
.TP
ls \-la
List all files in long format, including hidden files.
.TP
ls \-lh
List files with human-readable sizes.
.TP
ls \-lt
List files sorted by modification time.
.SH SEE ALSO
.BR dir (1),
.BR vdir (1),
.BR chmod (1),
.BR chown (1)
.SH AUTHOR
Written by Richard M. Stallman and David MacKenzie.
`,
		},
		{
			name:     "cat",
			section:  "1",
			platform: "linux",
			raw: `.TH CAT 1 "2024-01-01" "GNU coreutils 9.4" "User Commands"
.SH NAME
cat \- concatenate files and print on the standard output
.SH SYNOPSIS
.B cat
[\fIOPTION\fR]... [\fIFILE\fR]...
.SH DESCRIPTION
Concatenate FILE(s) to standard output.
.PP
With no FILE, or when FILE is \-, read standard input.
.SH OPTIONS
.TP
\fB\-A\fR, \fB\-\-show\-all\fR
equivalent to \fB\-vET\fR
.TP
\fB\-b\fR, \fB\-\-number\-nonblank\fR
number nonempty output lines, overrides \fB\-n\fR
.TP
\fB\-e\fR
equivalent to \fB\-vE\fR
.TP
\fB\-E\fR, \fB\-\-show\-ends\fR
display $ at end of each line
.TP
\fB\-n\fR, \fB\-\-number\fR
number all output lines
.TP
\fB\-s\fR, \fB\-\-squeeze\-blank\fR
suppress repeated empty output lines
.TP
\fB\-t\fR
equivalent to \fB\-vT\fR
.TP
\fB\-T\fR, \fB\-\-show\-tabs\fR
display TAB characters as ^I
.TP
\fB\-v\fR, \fB\-\-show\-nonprinting\fR
use ^ and M- notation, except for LFD and TAB
.TP
\fB\-\-help\fR
display this help and exit
.TP
\fB\-\-version\fR
output version information and exit
.SH EXAMPLES
.TP
cat f \- g
Output f's contents, then standard input, then g's contents.
.TP
cat
Copy standard input to standard output.
.SH SEE ALSO
.BR tac (1),
.BR head (1),
.BR tail (1)
.SH AUTHOR
Written by Torbjorn Granlund and Richard M. Stallman.
`,
		},
		{
			name:     "grep",
			section:  "1",
			platform: "linux",
			raw: `.TH GREP 1 "2024-01-01" "GNU grep 3.11" "User Commands"
.SH NAME
grep \- print lines that match patterns
.SH SYNOPSIS
.B grep
[\fIOPTION\fR...] \fIPATTERN\fR [\fIFILE\fR...]
.SH DESCRIPTION
.B grep
searches for \fIPATTERN\fR in each \fIFILE\fR.
\fIPATTERN\fR is a basic regular expression (BRE).
.PP
A \fIFILE\fR of "\fB\-\fP" stands for standard input.
If no \fIFILE\fR is given, recursive searches examine the working directory,
and nonrecursive searches read standard input.
.SH OPTIONS
.SS Pattern Syntax
.TP
\fB\-E\fR, \fB\-\-extended\-regexp\fR
Interpret \fIPATTERN\fR as an extended regular expression (ERE).
.TP
\fB\-F\fR, \fB\-\-fixed\-strings\fR
Interpret \fIPATTERN\fR as a list of fixed strings.
.TP
\fB\-G\fR, \fB\-\-basic\-regexp\fR
Interpret \fIPATTERN\fR as a basic regular expression (BRE). This is the default.
.TP
\fB\-P\fR, \fB\-\-perl\-regexp\fR
Interpret \fIPATTERN\fR as a Perl-compatible regular expression (PCRE).
.SS Matching Control
.TP
\fB\-e\fR \fIPATTERN\fR, \fB\-\-regexp\fR=\fIPATTERN\fR
Use \fIPATTERN\fR as the pattern.
.TP
\fB\-f\fR \fIFILE\fR, \fB\-\-file\fR=\fIFILE\fR
Obtain patterns from \fIFILE\fR, one per line.
.TP
\fB\-i\fR, \fB\-\-ignore\-case\fR
Ignore case distinctions in patterns and input data.
.TP
\fB\-v\fR, \fB\-\-invert\-match\fR
Invert the sense of matching, to select non-matching lines.
.TP
\fB\-w\fR, \fB\-\-word\-regexp\fR
Select only those lines containing matches that form whole words.
.TP
\fB\-x\fR, \fB\-\-line\-regexp\fR
Select only those matches that exactly match the whole line.
.SS Output Control
.TP
\fB\-c\fR, \fB\-\-count\fR
Suppress normal output; instead print a count of matching lines.
.TP
\fB\-l\fR, \fB\-\-files\-with\-matches\fR
Suppress normal output; instead print the name of each input file from which output would normally have been printed.
.TP
\fB\-L\fR, \fB\-\-files\-without\-match\fR
Suppress normal output; instead print the name of each input file from which no output would normally have been printed.
.TP
\fB\-n\fR, \fB\-\-line\-number\fR
Prefix each line of output with the 1-based line number within its input file.
.TP
\fB\-o\fR, \fB\-\-only\-matching\fR
Print only the matched (non-empty) parts of a matching line.
.TP
\fB\-q\fR, \fB\-\-quiet\fR, \fB\-\-silent\fR
Quiet; do not write anything to standard output.
.TP
\fB\-r\fR, \fB\-\-recursive\fR
Read all files under each directory, recursively.
.TP
\fB\-H\fR, \fB\-\-with\-filename\fR
Print the file name for each match.
.TP
\fB\-h\fR, \fB\-\-no\-filename\fR
Suppress the prefixing of file names on output.
.SS Context Control
.TP
\fB\-A\fR \fINUM\fR, \fB\-\-after\-context\fR=\fINUM\fR
Print \fINUM\fR lines of trailing context after matching lines.
.TP
\fB\-B\fR \fINUM\fR, \fB\-\-before\-context\fR=\fINUM\fR
Print \fINUM\fR lines of leading context before matching lines.
.TP
\fB\-C\fR \fINUM\fR, \fB\-\-context\fR=\fINUM\fR
Print \fINUM\fR lines of output context.
.SH EXAMPLES
.TP
grep \-i 'hello world' menu.h main.c
Search for "hello world" case-insensitively in files.
.TP
grep \-r 'pattern' .
Recursively search for pattern in current directory.
.TP
grep \-v '^#' config.txt
Show lines not starting with #.
.SH SEE ALSO
.BR egrep (1),
.BR fgrep (1),
.BR sed (1),
.BR awk (1)
.SH AUTHOR
Mike Haertel wrote the main grep code.
`,
		},
		{
			name:     "chmod",
			section:  "1",
			platform: "linux",
			raw: `.TH CHMOD 1 "2024-01-01" "GNU coreutils 9.4" "User Commands"
.SH NAME
chmod \- change file mode bits
.SH SYNOPSIS
.B chmod
[\fIOPTION\fR]... \fIMODE\fR[,\fIMODE\fR]... \fIFILE\fR...
.br
.B chmod
[\fIOPTION\fR]... \fIOCTAL-MODE\fR \fIFILE\fR...
.br
.B chmod
[\fIOPTION\fR]... \fB\-\-reference\fR=\fIRFILE\fR \fIFILE\fR...
.SH DESCRIPTION
This manual page documents the GNU version of
.BR chmod .
.B chmod
changes the file mode bits of each given file according to
\fImode\fR,
which can be either a symbolic representation of changes to make, or
an octal number representing the bit pattern for the new mode bits.
.PP
The format of a symbolic mode is [\fBugoa\fR...][\fB-+=\fR][\fBrwxXst\fR...].
.SH OPTIONS
.TP
\fB\-c\fR, \fB\-\-changes\fR
like verbose but report only when a change is made
.TP
\fB\-f\fR, \fB\-\-silent\fR, \fB\-\-quiet\fR
suppress most error messages
.TP
\fB\-v\fR, \fB\-\-verbose\fR
output a diagnostic for every file processed
.TP
\fB\-\-no\-preserve\-root\fR
do not treat '/' specially (the default)
.TP
\fB\-\-preserve\-root\fR
fail to operate recursively on '/'
.TP
\fB\-\-reference\fR=\fIRFILE\fR
use RFILE's mode instead of MODE values
.TP
\fB\-R\fR, \fB\-\-recursive\fR
change files and directories recursively
.TP
\fB\-\-help\fR
display this help and exit
.TP
\fB\-\-version\fR
output version information and exit
.SH EXAMPLES
.TP
chmod 755 script.sh
Set read, write, execute for owner; read, execute for group and others.
.TP
chmod u+x script.sh
Add execute permission for the owner.
.TP
chmod \-R 644 directory/
Recursively set permissions on all files in directory.
.TP
chmod a+r file
Add read permission for all users.
.SH SEE ALSO
.BR chown (1),
.BR chgrp (1),
.BR ls (1)
.SH AUTHOR
Written by David MacKenzie and Jim Meyering.
`,
		},
		{
			name:     "ssh",
			section:  "1",
			platform: "linux",
			raw: `.TH SSH 1 "2024-01-01" "OpenSSH 9.6" "General Commands Manual"
.SH NAME
ssh \- OpenSSH remote login client
.SH SYNOPSIS
.B ssh
[\fB\-46AaCfGgKkMNnqsTtVvXxYy\fR]
[\fB\-B\fR \fIbind_interface\fR]
[\fB\-b\fR \fIbind_address\fR]
[\fB\-c\fR \fIcipher_spec\fR]
[\fB\-D\fR [\fIbind_address\fR:]\fIport\fR]
[\fB\-E\fR \fIlog_file\fR]
[\fB\-e\fR \fIescape_char\fR]
[\fB\-F\fR \fIconfigfile\fR]
[\fB\-I\fR \fIpkcs11\fR]
[\fB\-i\fR \fIidentity_file\fR]
[\fB\-J\fR \fIdestination\fR]
[\fB\-L\fR \fIaddress\fR]
[\fB\-l\fR \fIlogin_name\fR]
[\fB\-m\fR \fImac_spec\fR]
[\fB\-O\fR \fIctl_cmd\fR]
[\fB\-o\fR \fIoption\fR]
[\fB\-p\fR \fIport\fR]
[\fB\-Q\fR \fIquery_option\fR]
[\fB\-R\fR \fIaddress\fR]
[\fB\-S\fR \fIctl_path\fR]
[\fB\-W\fR \fIhost\fR:\fIport\fR]
[\fB\-w\fR \fIlocal_tun\fR[:\fIremote_tun\fR]]
\fIdestination\fR
[\fIcommand\fR [\fIargument ...\fR]]
.SH DESCRIPTION
.B ssh
(SSH client) is a program for logging into a remote machine and for
executing commands on a remote machine.
It is intended to provide secure encrypted communications between
two untrusted hosts over an insecure network.
X11 connections, arbitrary TCP ports and UNIX-domain sockets
can also be forwarded over the secure channel.
.PP
.B ssh
connects and logs into the specified
.IR destination ,
which may be specified as either
[\fIuser\fR@]\fIhostname\fR
or a URI of the form
ssh://[\fIuser\fR@]\fIhostname\fR[:\fIport\fR].
.SH OPTIONS
.TP
\fB\-4\fR
Forces ssh to use IPv4 addresses only.
.TP
\fB\-6\fR
Forces ssh to use IPv6 addresses only.
.TP
\fB\-A\fR
Enables forwarding of connections from an authentication agent.
.TP
\fB\-a\fR
Disables forwarding of the authentication agent connection.
.TP
\fB\-C\fR
Requests compression of all data.
.TP
\fB\-f\fR
Requests ssh to go to background just before command execution.
.TP
\fB\-i\fR \fIidentity_file\fR
Selects a file from which the identity (private key) for public key authentication is read.
.TP
\fB\-l\fR \fIlogin_name\fR
Specifies the user to log in as on the remote machine.
.TP
\fB\-N\fR
Do not execute a remote command. This is useful for just forwarding ports.
.TP
\fB\-o\fR \fIoption\fR
Can be used to give options in the format used in the configuration file.
.TP
\fB\-p\fR \fIport\fR
Port to connect to on the remote host.
.TP
\fB\-q\fR
Quiet mode. Causes most warning and diagnostic messages to be suppressed.
.TP
\fB\-T\fR
Disable pseudo-terminal allocation.
.TP
\fB\-t\fR
Force pseudo-terminal allocation.
.TP
\fB\-v\fR
Verbose mode. Causes ssh to print debugging messages about its progress.
.TP
\fB\-X\fR
Enables X11 forwarding.
.SH EXAMPLES
.TP
ssh user@hostname
Connect to hostname as user.
.TP
ssh \-p 2222 user@hostname
Connect to hostname on port 2222.
.TP
ssh \-i ~/.ssh/id_ed25519 user@hostname
Connect using a specific identity file.
.TP
ssh \-L 8080:localhost:80 user@hostname
Forward local port 8080 to remote localhost:80.
.SH SEE ALSO
.BR scp (1),
.BR sftp (1),
.BR ssh-keygen (1),
.BR ssh_config (5),
.BR sshd (8)
.SH AUTHORS
OpenSSH is a derivative of the original and free ssh 1.2.12 release by
Tatu Ylonen.
`,
		},
		{
			name:     "find",
			section:  "1",
			platform: "linux",
			raw: `.TH FIND 1 "2024-01-01" "GNU findutils 4.9" "User Commands"
.SH NAME
find \- search for files in a directory hierarchy
.SH SYNOPSIS
.B find
[\fB\-H\fR] [\fB\-L\fR] [\fB\-P\fR] [\fB\-D\fR \fIdebugopt\fR] [\fB\-O\fR\fIlevel\fR] [\fIstarting-point...\fR] [\fIexpression\fR]
.SH DESCRIPTION
This manual page documents the GNU version of
.BR find .
GNU
.B find
searches the directory tree rooted at each given starting-point by
evaluating the given expression from left to right, according to the
rules of precedence, until the outcome is known.
.PP
If no starting-point is specified, '.' is assumed.
.SH OPTIONS
.TP
\fB\-P\fR
Never follow symbolic links. This is the default behaviour.
.TP
\fB\-L\fR
Follow symbolic links.
.TP
\fB\-H\fR
Do not follow symbolic links, except while processing the command line arguments.
.SS Tests
.TP
\fB\-name\fR \fIpattern\fR
Base of file name matches shell pattern \fIpattern\fR.
.TP
\fB\-iname\fR \fIpattern\fR
Like \fB\-name\fR, but the match is case insensitive.
.TP
\fB\-path\fR \fIpattern\fR
File name matches shell pattern \fIpattern\fR.
.TP
\fB\-type\fR \fIc\fR
File is of type \fIc\fR: b (block), c (character), d (directory), f (regular file), l (symbolic link), p (named pipe), s (socket).
.TP
\fB\-size\fR \fIn\fR[\fBcwbkMG\fR]
File uses \fIn\fR units of space. Units: c (bytes), k (KiB), M (MiB), G (GiB).
.TP
\fB\-mtime\fR \fIn\fR
File was modified \fIn\fR*24 hours ago.
.TP
\fB\-newer\fR \fIfile\fR
File was modified more recently than \fIfile\fR.
.TP
\fB\-user\fR \fIuname\fR
File is owned by user \fIuname\fR.
.TP
\fB\-group\fR \fIgname\fR
File belongs to group \fIgname\fR.
.TP
\fB\-perm\fR \fImode\fR
File's permission bits match \fImode\fR.
.TP
\fB\-empty\fR
File is empty and is either a regular file or a directory.
.SS Actions
.TP
\fB\-print\fR
Print the full file name on the standard output, followed by a newline.
.TP
\fB\-print0\fR
Print the full file name, followed by a null character.
.TP
\fB\-exec\fR \fIcommand\fR ;
Execute \fIcommand\fR. The string '{}' is replaced by the current file name.
.TP
\fB\-exec\fR \fIcommand\fR {} +
Execute \fIcommand\fR with multiple file names at once.
.TP
\fB\-delete\fR
Delete files.
.SH EXAMPLES
.TP
find . \-name "*.txt"
Find all .txt files in current directory tree.
.TP
find /home \-type f \-size +10M
Find files larger than 10MB in /home.
.TP
find . \-mtime \-7 \-type f
Find files modified in the last 7 days.
.TP
find . \-name "*.log" \-delete
Delete all .log files.
.TP
find . \-type f \-exec chmod 644 {} \\;
Set permissions on all files.
.SH SEE ALSO
.BR locate (1),
.BR xargs (1),
.BR chmod (1)
.SH AUTHOR
Written by Eric B. Decker, James Youngman, and Kevin Dalley.
`,
		},
	}

	for _, s := range samples {
		log.Printf("Loading sample: %s(%s)", s.name, s.section)

		page, err := parser.Parse(s.raw)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", s.name, err)
		}

		if page.Name == "" {
			page.Name = s.name
		}
		if page.Section == "" {
			page.Section = s.section
		}

		modelPage := &model.ManPage{
			Name:            page.Name,
			Section:         page.Section,
			Title:           page.Title,
			Platform:        s.platform,
			Language:        "en",
			SourceFormat:    page.SourceFormat,
			SourceRaw:       page.SourceRaw,
			ContentHTML:     page.ContentHTML,
			ContentText:     page.ContentText,
			ContentMarkdown: page.ContentMarkdown,
			Synopsis:        page.Synopsis,
			Description:     page.Description,
			SearchText:      page.SearchText,
		}

		for _, ref := range page.SeeAlso {
			entry := model.SeeAlsoEntry{Name: ref}
			if idx := strings.Index(ref, "("); idx > 0 {
				entry.Name = ref[:idx]
				if endIdx := strings.Index(ref, ")"); endIdx > idx {
					entry.Section = ref[idx+1 : endIdx]
				}
			}
			modelPage.SeeAlso = append(modelPage.SeeAlso, entry)
		}

		if err := db.InsertManPage(modelPage); err != nil {
			return fmt.Errorf("inserting %s: %w", s.name, err)
		}
	}

	log.Printf("Loaded %d sample man pages", len(samples))
	return nil
}
