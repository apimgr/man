// Package main provides a tool to download man pages from multiple sources.
// It saves plain text files with content-hash deduplication via symlinks.
//
// Usage:
//
//	go run . -output src/data/man -sources linux,freebsd,openbsd,netbsd
//
// Directory structure:
//
//	src/data/man/
//	├── _shared/           # Unique content (hash-named files)
//	│   ├── a1b2c3d4e5f6   # Actual content
//	│   └── ...
//	├── linux/1/ls         # Symlink -> ../../_shared/a1b2c3d4e5f6
//	├── freebsd/1/ls       # Symlink -> same hash if identical
//	└── ...
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	outputDir   = flag.String("output", "src/data/man", "Output directory for man pages")
	sources     = flag.String("sources", "linux,freebsd,openbsd,netbsd", "Comma-separated list of sources to download")
	workers     = flag.Int("workers", 10, "Number of concurrent download workers per source")
	verbose     = flag.Bool("verbose", false, "Verbose output")
	rateLimit   = flag.Duration("rate", 100*time.Millisecond, "Rate limit between requests")
	timeout     = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
	skipExist   = flag.Bool("skip-existing", true, "Skip pages that already exist")
	listOnly    = flag.Bool("list-only", false, "Only list pages, don't download")
	sectionOnly = flag.String("section", "", "Only download specific section (1-9)")
)

// ManPage represents a man page to download.
type ManPage struct {
	Name     string
	Section  string
	Platform string
	URL      string
}

// Stats tracks download statistics.
type Stats struct {
	Total      int64
	Downloaded int64
	Skipped    int64
	Failed     int64
	Deduped    int64
}

func main() {
	flag.Parse()

	log.SetFlags(log.Ltime)

	// Create output directories
	sharedDir := filepath.Join(*outputDir, "_shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		log.Fatalf("Failed to create shared directory: %v", err)
	}

	// Parse sources
	sourceList := strings.Split(*sources, ",")
	for i := range sourceList {
		sourceList[i] = strings.TrimSpace(sourceList[i])
	}

	log.Printf("Downloading man pages from: %v", sourceList)
	log.Printf("Output directory: %s", *outputDir)
	log.Printf("Workers per source: %d", *workers)

	// Download from each source in parallel
	var wg sync.WaitGroup
	totalStats := &Stats{}

	for _, source := range sourceList {
		wg.Add(1)
		go func(src string) {
			defer wg.Done()
			stats := downloadSource(src)
			atomic.AddInt64(&totalStats.Total, stats.Total)
			atomic.AddInt64(&totalStats.Downloaded, stats.Downloaded)
			atomic.AddInt64(&totalStats.Skipped, stats.Skipped)
			atomic.AddInt64(&totalStats.Failed, stats.Failed)
			atomic.AddInt64(&totalStats.Deduped, stats.Deduped)
		}(source)
	}

	wg.Wait()

	log.Printf("=== Final Statistics ===")
	log.Printf("Total pages:   %d", totalStats.Total)
	log.Printf("Downloaded:    %d", totalStats.Downloaded)
	log.Printf("Deduplicated:  %d", totalStats.Deduped)
	log.Printf("Skipped:       %d", totalStats.Skipped)
	log.Printf("Failed:        %d", totalStats.Failed)
}

// downloadSource downloads man pages from a specific source.
func downloadSource(source string) *Stats {
	stats := &Stats{}

	log.Printf("[%s] Starting download...", source)

	// Get list of pages for this source
	pages, err := listPages(source)
	if err != nil {
		log.Printf("[%s] Failed to list pages: %v", source, err)
		return stats
	}

	stats.Total = int64(len(pages))
	log.Printf("[%s] Found %d pages", source, len(pages))

	if *listOnly {
		for _, p := range pages {
			fmt.Printf("%s/%s/%s\n", p.Platform, p.Section, p.Name)
		}
		return stats
	}

	// Create platform directory
	platformDir := filepath.Join(*outputDir, source)
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		log.Printf("[%s] Failed to create platform directory: %v", source, err)
		return stats
	}

	// Download pages using worker pool
	pageChan := make(chan ManPage, len(pages))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: *timeout}
			for page := range pageChan {
				result := downloadPage(client, page)
				switch result {
				case "downloaded":
					atomic.AddInt64(&stats.Downloaded, 1)
				case "deduped":
					atomic.AddInt64(&stats.Deduped, 1)
				case "skipped":
					atomic.AddInt64(&stats.Skipped, 1)
				case "failed":
					atomic.AddInt64(&stats.Failed, 1)
				}
				time.Sleep(*rateLimit)
			}
		}()
	}

	// Send pages to workers
	for _, page := range pages {
		pageChan <- page
	}
	close(pageChan)

	wg.Wait()

	log.Printf("[%s] Complete: %d downloaded, %d deduped, %d skipped, %d failed",
		source, stats.Downloaded, stats.Deduped, stats.Skipped, stats.Failed)

	return stats
}

// listPages returns a list of man pages for a source.
func listPages(source string) ([]ManPage, error) {
	switch source {
	case "linux":
		return listLinuxPages()
	case "freebsd":
		return listFreeBSDPages()
	case "openbsd":
		return listOpenBSDPages()
	case "netbsd":
		return listNetBSDPages()
	case "macos", "darwin":
		return listMacOSPages()
	default:
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

// listLinuxPages lists man pages from multiple Linux sources.
func listLinuxPages() ([]ManPage, error) {
	var pages []ManPage
	var mu sync.Mutex

	// Sections to download
	sections := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	if *sectionOnly != "" {
		sections = []string{*sectionOnly}
	}

	client := &http.Client{Timeout: *timeout}

	// Source 1: kernel.org man-pages (primarily sections 2, 3, 4, 5, 7)
	for _, section := range sections {
		url := fmt.Sprintf("https://git.kernel.org/pub/scm/docs/man-pages/man-pages.git/tree/man/man%s", section)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[linux] Failed to list kernel.org section %s: %v", section, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Pattern: href='/pub/scm/docs/man-pages/man-pages.git/tree/man/man1/getent.1'>getent.1</a>
		re := regexp.MustCompile(`href='[^']*man-pages\.git/tree/man/man` + section + `/([^']+)\.` + section + `'>`)
		matches := re.FindAllStringSubmatch(string(body), -1)

		seen := make(map[string]bool)
		for _, m := range matches {
			name := m[1]
			if seen[name] {
				continue
			}
			seen[name] = true

			mu.Lock()
			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "linux",
				URL:      fmt.Sprintf("https://git.kernel.org/pub/scm/docs/man-pages/man-pages.git/plain/man/man%s/%s.%s", section, name, section),
			})
			mu.Unlock()
		}

		log.Printf("[linux] kernel.org section %s: found %d pages", section, len(seen))
		time.Sleep(*rateLimit)
	}

	// Source 2: manpages.ubuntu.com for comprehensive coverage
	for _, section := range sections {
		url := fmt.Sprintf("https://manpages.ubuntu.com/manpages/jammy/man%s/", section)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[linux] Failed to list ubuntu section %s: %v", section, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Pattern: <a href="ls.1.html">ls.1.html</a>
		re := regexp.MustCompile(`<a href="([^"]+)\.` + section + `\.html">`)
		matches := re.FindAllStringSubmatch(string(body), -1)

		count := 0
		seen := make(map[string]bool)
		for _, m := range matches {
			name := m[1]
			if seen[name] || name == "" || strings.Contains(name, "/") {
				continue
			}
			seen[name] = true

			mu.Lock()
			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "linux",
				URL:      fmt.Sprintf("https://manpages.ubuntu.com/manpages/jammy/man%s/%s.%s.html", section, name, section),
			})
			mu.Unlock()
			count++
		}

		log.Printf("[linux] ubuntu section %s: found %d pages", section, count)
		time.Sleep(*rateLimit)
	}

	return pages, nil
}

// listFreeBSDPages lists man pages from man.freebsd.org.
func listFreeBSDPages() ([]ManPage, error) {
	var pages []ManPage

	sections := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if *sectionOnly != "" {
		sections = []string{*sectionOnly}
	}

	client := &http.Client{Timeout: *timeout}

	for _, section := range sections {
		// FreeBSD man page index
		url := fmt.Sprintf("https://man.freebsd.org/cgi/man.cgi?manpath=FreeBSD+14.0-RELEASE&apropos=2&sektion=%s", section)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[freebsd] Failed to list section %s: %v", section, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Parse HTML for man page links
		// Pattern: <a href="man.cgi?query=ls&amp;sektion=1...">ls(1)</a>
		re := regexp.MustCompile(`query=([^&"]+)&amp;sektion=` + section)
		matches := re.FindAllStringSubmatch(string(body), -1)

		seen := make(map[string]bool)
		for _, m := range matches {
			name := m[1]
			if seen[name] || name == "" {
				continue
			}
			seen[name] = true

			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "freebsd",
				URL:      fmt.Sprintf("https://man.freebsd.org/cgi/man.cgi?query=%s&sektion=%s&manpath=FreeBSD+14.0-RELEASE&format=ascii", name, section),
			})
		}

		log.Printf("[freebsd] Section %s: found %d pages", section, len(seen))
		time.Sleep(*rateLimit)
	}

	return pages, nil
}

// listOpenBSDPages lists man pages from man.openbsd.org.
func listOpenBSDPages() ([]ManPage, error) {
	var pages []ManPage

	sections := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if *sectionOnly != "" {
		sections = []string{*sectionOnly}
	}

	client := &http.Client{Timeout: *timeout}

	for _, section := range sections {
		// OpenBSD man page index - they have nice listings
		url := fmt.Sprintf("https://man.openbsd.org/?query=&sec=%s&manpath=OpenBSD-current&arch=amd64", section)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[openbsd] Failed to list section %s: %v", section, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Parse for man page links - pattern: <a href="/ls.1">ls(1)</a>
		re := regexp.MustCompile(`<a href="/([^"]+)\.` + section + `">`)
		matches := re.FindAllStringSubmatch(string(body), -1)

		seen := make(map[string]bool)
		for _, m := range matches {
			name := m[1]
			if seen[name] || name == "" || strings.Contains(name, "/") {
				continue
			}
			seen[name] = true

			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "openbsd",
				URL:      fmt.Sprintf("https://man.openbsd.org/%s.%s?query=&sec=%s&manpath=OpenBSD-current&arch=amd64&format=ascii", name, section, section),
			})
		}

		log.Printf("[openbsd] Section %s: found %d pages", section, len(seen))
		time.Sleep(*rateLimit)
	}

	return pages, nil
}

// listNetBSDPages lists man pages from man.netbsd.org.
func listNetBSDPages() ([]ManPage, error) {
	var pages []ManPage

	sections := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	if *sectionOnly != "" {
		sections = []string{*sectionOnly}
	}

	client := &http.Client{Timeout: *timeout}

	for _, section := range sections {
		// NetBSD man page search
		url := fmt.Sprintf("https://man.netbsd.org/cgi-bin/man-cgi?++NetBSD-current+%s", section)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[netbsd] Failed to list section %s: %v", section, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Parse for man page links
		re := regexp.MustCompile(`<a href="[^"]*\?([^+"]+)\+NetBSD-current\+` + section + `"`)
		matches := re.FindAllStringSubmatch(string(body), -1)

		seen := make(map[string]bool)
		for _, m := range matches {
			name := m[1]
			if seen[name] || name == "" {
				continue
			}
			seen[name] = true

			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "netbsd",
				URL:      fmt.Sprintf("https://man.netbsd.org/cgi-bin/man-cgi?%s+NetBSD-current+%s+text", name, section),
			})
		}

		log.Printf("[netbsd] Section %s: found %d pages", section, len(seen))
		time.Sleep(*rateLimit)
	}

	return pages, nil
}

// listMacOSPages lists man pages from Apple's open source.
func listMacOSPages() ([]ManPage, error) {
	// macOS man pages are harder to scrape directly
	// We'll use a curated list from common commands
	var pages []ManPage

	// Common macOS/Darwin commands
	commands := map[string][]string{
		"1": {"ls", "cat", "grep", "find", "chmod", "chown", "cp", "mv", "rm", "mkdir",
			"rmdir", "pwd", "echo", "date", "head", "tail", "wc", "sort", "uniq",
			"cut", "paste", "tr", "sed", "awk", "diff", "tar", "gzip", "ssh", "scp",
			"rsync", "curl", "ps", "kill", "man", "less", "more", "vi", "touch",
			"ln", "du", "df", "open", "pbcopy", "pbpaste", "say", "caffeinate",
			"defaults", "launchctl", "dscl", "hdiutil", "diskutil", "sw_vers"},
		"2": {"open", "read", "write", "close", "fork", "execve", "exit", "wait",
			"pipe", "dup", "socket", "bind", "listen", "accept", "connect",
			"send", "recv", "mmap", "munmap", "stat", "fstat", "lstat"},
		"3": {"printf", "scanf", "malloc", "free", "realloc", "calloc", "memcpy",
			"memmove", "memset", "strcmp", "strcpy", "strlen", "strcat", "strtok",
			"fopen", "fclose", "fread", "fwrite", "fgets", "fputs"},
		"5": {"passwd", "group", "hosts", "resolv.conf", "fstab", "shells"},
		"8": {"mount", "umount", "ifconfig", "route", "netstat", "ping", "traceroute",
			"shutdown", "reboot", "sysctl", "launchd", "kextload", "kextunload"},
	}

	if *sectionOnly != "" {
		if cmds, ok := commands[*sectionOnly]; ok {
			commands = map[string][]string{*sectionOnly: cmds}
		} else {
			commands = map[string][]string{}
		}
	}

	for section, names := range commands {
		for _, name := range names {
			pages = append(pages, ManPage{
				Name:     name,
				Section:  section,
				Platform: "macos",
				// Use keith.github.io/xcode-man-pages which mirrors Apple's docs
				URL: fmt.Sprintf("https://keith.github.io/xcode-man-pages/%s.%s.html", name, section),
			})
		}
		log.Printf("[macos] Section %s: %d pages", section, len(names))
	}

	return pages, nil
}

// downloadPage downloads a single man page and saves it with deduplication.
func downloadPage(client *http.Client, page ManPage) string {
	// Check if already exists
	targetPath := filepath.Join(*outputDir, page.Platform, page.Section, page.Name)

	if *skipExist {
		if _, err := os.Lstat(targetPath); err == nil {
			if *verbose {
				log.Printf("[%s] Skipping existing: %s(%s)", page.Platform, page.Name, page.Section)
			}
			return "skipped"
		}
	}

	if *verbose {
		log.Printf("[%s] Downloading: %s(%s)", page.Platform, page.Name, page.Section)
	}

	// Download the page
	resp, err := client.Get(page.URL)
	if err != nil {
		log.Printf("[%s] Failed to download %s: %v", page.Platform, page.Name, err)
		return "failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if *verbose {
			log.Printf("[%s] %s returned %d", page.Platform, page.Name, resp.StatusCode)
		}
		return "failed"
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] Failed to read %s: %v", page.Platform, page.Name, err)
		return "failed"
	}

	// Convert to plain text if needed
	text := extractPlainText(content, page)
	if text == "" {
		if *verbose {
			log.Printf("[%s] Empty content for %s", page.Platform, page.Name)
		}
		return "failed"
	}

	// Calculate content hash
	hash := sha256.Sum256([]byte(text))
	hashStr := hex.EncodeToString(hash[:])[:16] // Use first 16 chars

	// Save to shared directory
	sharedPath := filepath.Join(*outputDir, "_shared", hashStr)
	sharedExists := false

	if _, err := os.Stat(sharedPath); err == nil {
		sharedExists = true
	} else {
		if err := os.WriteFile(sharedPath, []byte(text), 0644); err != nil {
			log.Printf("[%s] Failed to write shared file: %v", page.Platform, err)
			return "failed"
		}
	}

	// Create section directory
	sectionDir := filepath.Join(*outputDir, page.Platform, page.Section)
	if err := os.MkdirAll(sectionDir, 0755); err != nil {
		log.Printf("[%s] Failed to create section dir: %v", page.Platform, err)
		return "failed"
	}

	// Remove existing file/symlink if any
	os.Remove(targetPath)

	// Create relative symlink
	// From: src/data/man/linux/1/ls
	// To:   src/data/man/_shared/abc123
	// Relative: ../../_shared/abc123
	relPath := filepath.Join("..", "..", "_shared", hashStr)

	if err := os.Symlink(relPath, targetPath); err != nil {
		log.Printf("[%s] Failed to create symlink: %v", page.Platform, err)
		return "failed"
	}

	if sharedExists {
		return "deduped"
	}
	return "downloaded"
}

// extractPlainText extracts plain text from raw content.
func extractPlainText(content []byte, page ManPage) string {
	text := string(content)

	// Check if it's HTML and convert
	if strings.Contains(text, "<html") || strings.Contains(text, "<HTML") ||
		strings.Contains(text, "<!DOCTYPE") || strings.Contains(text, "<pre>") {
		text = htmlToText(text)
	}

	// Check if it's groff/troff and convert
	if strings.HasPrefix(strings.TrimSpace(text), ".\\\"") ||
		strings.HasPrefix(strings.TrimSpace(text), ".TH") ||
		strings.HasPrefix(strings.TrimSpace(text), ".Dd") ||
		strings.HasPrefix(strings.TrimSpace(text), "'\\\"") {
		text = groffToText(text, page.Name, page.Section)
	}

	// Clean up the text
	text = cleanText(text)

	return text
}

// htmlToText converts HTML to plain text.
func htmlToText(html string) string {
	// Remove script and style tags
	reScript := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = reScript.ReplaceAllString(html, "")

	reStyle := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = reStyle.ReplaceAllString(html, "")

	// Extract content from <pre> tags if present (common for man pages)
	rePre := regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	if matches := rePre.FindStringSubmatch(html); len(matches) > 1 {
		html = matches[1]
	}

	// Remove HTML tags
	reTag := regexp.MustCompile(`<[^>]+>`)
	text := reTag.ReplaceAllString(html, "")

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return text
}

// groffToText converts groff/troff to plain text using groff command.
func groffToText(source, name, section string) string {
	// Try using groff if available
	cmd := exec.Command("groff", "-man", "-Tutf8", "-P-c")
	cmd.Stdin = strings.NewReader(source)

	output, err := cmd.Output()
	if err != nil {
		// If groff not available, do basic conversion
		return basicGroffToText(source)
	}

	return string(output)
}

// basicGroffToText does basic groff to text conversion without external tools.
func basicGroffToText(source string) string {
	lines := strings.Split(source, "\n")
	var result []string

	for _, line := range lines {
		// Skip comment lines
		if strings.HasPrefix(line, ".\\\"") || strings.HasPrefix(line, "'\\\"") {
			continue
		}

		// Handle common macros
		if strings.HasPrefix(line, ".TH ") || strings.HasPrefix(line, ".Dt ") {
			// Title header - extract title
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				result = append(result, fmt.Sprintf("%s(%s)", parts[1], parts[2]))
				result = append(result, "")
			}
			continue
		}

		if strings.HasPrefix(line, ".SH ") || strings.HasPrefix(line, ".Sh ") {
			// Section header
			header := strings.TrimPrefix(line, ".SH ")
			header = strings.TrimPrefix(header, ".Sh ")
			header = strings.Trim(header, "\"")
			result = append(result, "")
			result = append(result, strings.ToUpper(header))
			continue
		}

		if strings.HasPrefix(line, ".SS ") {
			// Subsection header
			header := strings.TrimPrefix(line, ".SS ")
			header = strings.Trim(header, "\"")
			result = append(result, "")
			result = append(result, "  "+header)
			continue
		}

		if strings.HasPrefix(line, ".TP") {
			// Tagged paragraph
			result = append(result, "")
			continue
		}

		if strings.HasPrefix(line, ".PP") || strings.HasPrefix(line, ".P") ||
			strings.HasPrefix(line, ".LP") || strings.HasPrefix(line, ".Pp") {
			// Paragraph
			result = append(result, "")
			continue
		}

		if strings.HasPrefix(line, ".BR ") || strings.HasPrefix(line, ".B ") ||
			strings.HasPrefix(line, ".I ") || strings.HasPrefix(line, ".IR ") {
			// Bold/Italic text - just extract the text
			text := strings.TrimPrefix(line, ".BR ")
			text = strings.TrimPrefix(text, ".B ")
			text = strings.TrimPrefix(text, ".I ")
			text = strings.TrimPrefix(text, ".IR ")
			text = strings.ReplaceAll(text, "\\fB", "")
			text = strings.ReplaceAll(text, "\\fI", "")
			text = strings.ReplaceAll(text, "\\fR", "")
			text = strings.ReplaceAll(text, "\\fP", "")
			result = append(result, text)
			continue
		}

		// Skip other macros
		if strings.HasPrefix(line, ".") {
			continue
		}

		// Regular text - clean up escapes
		line = strings.ReplaceAll(line, "\\fB", "")
		line = strings.ReplaceAll(line, "\\fI", "")
		line = strings.ReplaceAll(line, "\\fR", "")
		line = strings.ReplaceAll(line, "\\fP", "")
		line = strings.ReplaceAll(line, "\\-", "-")
		line = strings.ReplaceAll(line, "\\ ", " ")
		line = strings.ReplaceAll(line, "\\(co", "(c)")
		line = strings.ReplaceAll(line, "\\(em", "--")
		line = strings.ReplaceAll(line, "\\(en", "-")

		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// cleanText cleans up extracted text.
func cleanText(text string) string {
	// Normalize line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Remove excessive blank lines
	reBlank := regexp.MustCompile(`\n{3,}`)
	text = reBlank.ReplaceAllString(text, "\n\n")

	// Trim whitespace
	text = strings.TrimSpace(text)

	// Ensure single trailing newline
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	return text
}

// printStats prints statistics about the downloaded pages.
func printStats() {
	// Count files in each directory
	platforms := []string{"linux", "freebsd", "openbsd", "netbsd", "macos"}

	fmt.Println("\n=== Statistics ===")

	var totalPages, totalShared int

	// Count shared files
	sharedDir := filepath.Join(*outputDir, "_shared")
	if entries, err := os.ReadDir(sharedDir); err == nil {
		totalShared = len(entries)
	}

	for _, platform := range platforms {
		platformDir := filepath.Join(*outputDir, platform)
		if _, err := os.Stat(platformDir); os.IsNotExist(err) {
			continue
		}

		var count int
		filepath.Walk(platformDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				count++
			}
			return nil
		})

		fmt.Printf("%s: %d pages\n", platform, count)
		totalPages += count
	}

	fmt.Printf("\nTotal pages: %d\n", totalPages)
	fmt.Printf("Unique content files: %d\n", totalShared)

	if totalShared > 0 {
		dedupeRatio := float64(totalPages-totalShared) / float64(totalPages) * 100
		fmt.Printf("Deduplication ratio: %.1f%%\n", dedupeRatio)
	}
}

// listAllPages lists all downloaded pages.
func listAllPages() {
	platforms := []string{"linux", "freebsd", "openbsd", "netbsd", "macos"}

	var allPages []string

	for _, platform := range platforms {
		platformDir := filepath.Join(*outputDir, platform)
		if _, err := os.Stat(platformDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(platformDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				rel, _ := filepath.Rel(*outputDir, path)
				allPages = append(allPages, rel)
			}
			return nil
		})
	}

	sort.Strings(allPages)
	for _, p := range allPages {
		fmt.Println(p)
	}
}
