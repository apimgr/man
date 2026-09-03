// Package manpage provides parsing and rendering for man pages.
// Supports groff/troff and mdoc formats.
package manpage

import (
	"bufio"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Page represents a parsed man page.
type Page struct {
	Name         string
	Section      string
	Title        string
	Synopsis     string
	Description  string
	SeeAlso      []string
	SourceFormat string
	SourceRaw    string

	// Rendered content
	ContentHTML     string
	ContentText     string
	ContentMarkdown string
	SearchText      string

	// Metadata
	Platform string
	Distro   string
	Version  string
	Language string
}

// Parser handles man page parsing.
type Parser struct {
	// Current state
	currentSection string
	sections       map[string][]string
	inList         bool
	listType       string
}

// NewParser creates a new man page parser.
func NewParser() *Parser {
	return &Parser{
		sections: make(map[string][]string),
	}
}

// Parse parses raw man page content and returns a Page.
func (p *Parser) Parse(raw string) (*Page, error) {
	p.reset()

	page := &Page{
		SourceRaw:    raw,
		SourceFormat: p.detectFormat(raw),
	}

	// Parse based on format
	if page.SourceFormat == "mdoc" {
		p.parseMdoc(raw)
	} else {
		p.parseGroff(raw)
	}

	// Extract metadata from parsed sections
	page.Name = p.extractName()
	page.Section = p.extractSection()
	page.Title = p.extractTitle()
	page.Synopsis = p.renderSectionText("SYNOPSIS")
	page.Description = p.extractDescription()
	page.SeeAlso = p.extractSeeAlso()

	// Render to different formats
	page.ContentHTML = p.renderHTML()
	page.ContentText = p.renderText()
	page.ContentMarkdown = p.renderMarkdown()
	page.SearchText = p.buildSearchText()

	return page, nil
}

func (p *Parser) reset() {
	p.currentSection = ""
	p.sections = make(map[string][]string)
	p.inList = false
	p.listType = ""
}

// detectFormat determines if content is groff or mdoc format.
func (p *Parser) detectFormat(raw string) string {
	// mdoc uses .Dd, .Dt, .Os, .Nm, .Nd macros
	if strings.Contains(raw, ".Dd") || strings.Contains(raw, ".Dt") ||
		strings.Contains(raw, ".Nm") || strings.Contains(raw, ".Nd") {
		return "mdoc"
	}
	return "groff"
}

// parseGroff parses traditional groff/troff man page format.
func (p *Parser) parseGroff(raw string) {
	scanner := bufio.NewScanner(strings.NewReader(raw))

	for scanner.Scan() {
		line := scanner.Text()
		p.parseGroffLine(line)
	}
}

func (p *Parser) parseGroffLine(line string) {
	// Skip comments
	if strings.HasPrefix(line, ".\\\"") || strings.HasPrefix(line, "'\\\"") {
		return
	}

	// Handle macros
	if strings.HasPrefix(line, ".") {
		p.handleGroffMacro(line)
		return
	}

	// Regular text - add to current section
	if p.currentSection != "" {
		p.sections[p.currentSection] = append(p.sections[p.currentSection], line)
	}
}

func (p *Parser) handleGroffMacro(line string) {
	parts := splitMacroLine(line)
	if len(parts) == 0 {
		return
	}

	macro := parts[0]
	args := parts[1:]

	switch macro {
	case "TH":
		// Title header: .TH name section date source manual
		if len(args) >= 2 {
			p.sections["_NAME"] = []string{strings.ToLower(args[0])}
			p.sections["_SECTION"] = []string{args[1]}
		}
	case "SH":
		// Section header
		if len(args) > 0 {
			p.currentSection = strings.ToUpper(strings.Join(args, " "))
			p.currentSection = strings.Trim(p.currentSection, "\"")
		}
	case "SS":
		// Subsection - add as text with formatting
		if p.currentSection != "" && len(args) > 0 {
			subsection := strings.Join(args, " ")
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				fmt.Sprintf("__SUBSECTION__%s", subsection))
		}
	case "PP", "P", "LP":
		// Paragraph break
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "")
		}
	case "TP":
		// Tagged paragraph (for options)
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__TP__")
		}
	case "IP":
		// Indented paragraph
		if p.currentSection != "" {
			if len(args) > 0 {
				p.sections[p.currentSection] = append(p.sections[p.currentSection],
					fmt.Sprintf("__IP__%s", strings.Join(args, " ")))
			}
		}
	case "B":
		// Bold
		if p.currentSection != "" && len(args) > 0 {
			text := strings.Join(args, " ")
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				fmt.Sprintf("__BOLD__%s__/BOLD__", text))
		}
	case "I":
		// Italic
		if p.currentSection != "" && len(args) > 0 {
			text := strings.Join(args, " ")
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				fmt.Sprintf("__ITALIC__%s__/ITALIC__", text))
		}
	case "BR", "BI", "IB", "IR", "RB", "RI":
		// Alternating font macros
		if p.currentSection != "" && len(args) > 0 {
			text := p.handleAlternatingFonts(macro, args)
			p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
		}
	case "nf":
		// No-fill (preformatted)
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__PRE__")
		}
	case "fi":
		// Fill (end preformatted)
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__/PRE__")
		}
	case "RS":
		// Relative indent start
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__INDENT__")
		}
	case "RE":
		// Relative indent end
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__/INDENT__")
		}
	default:
		// Unknown macro - try to extract text
		if p.currentSection != "" && len(args) > 0 {
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				strings.Join(args, " "))
		}
	}
}

func (p *Parser) handleAlternatingFonts(macro string, args []string) string {
	var result strings.Builder
	firstBold := macro[1] == 'B'

	for i, arg := range args {
		bold := (i%2 == 0) == firstBold
		if bold {
			result.WriteString(fmt.Sprintf("__BOLD__%s__/BOLD__", arg))
		} else {
			result.WriteString(fmt.Sprintf("__ITALIC__%s__/ITALIC__", arg))
		}
	}

	return result.String()
}

// parseMdoc parses BSD mdoc format.
func (p *Parser) parseMdoc(raw string) {
	scanner := bufio.NewScanner(strings.NewReader(raw))

	for scanner.Scan() {
		line := scanner.Text()
		p.parseMdocLine(line)
	}
}

func (p *Parser) parseMdocLine(line string) {
	// Skip comments
	if strings.HasPrefix(line, ".\\\"") {
		return
	}

	// Handle macros
	if strings.HasPrefix(line, ".") {
		p.handleMdocMacro(line)
		return
	}

	// Regular text
	if p.currentSection != "" {
		p.sections[p.currentSection] = append(p.sections[p.currentSection], line)
	}
}

func (p *Parser) handleMdocMacro(line string) {
	parts := splitMacroLine(line)
	if len(parts) == 0 {
		return
	}

	macro := parts[0]
	args := parts[1:]

	switch macro {
	case "Dt":
		// Document title: .Dt NAME SECTION
		if len(args) >= 2 {
			p.sections["_NAME"] = []string{strings.ToLower(args[0])}
			p.sections["_SECTION"] = []string{args[1]}
		}
	case "Nm":
		// Name macro
		if len(args) > 0 {
			if p.currentSection == "" {
				p.sections["_NAME"] = []string{strings.ToLower(args[0])}
			} else {
				text := fmt.Sprintf("__BOLD__%s__/BOLD__", args[0])
				p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
			}
		}
	case "Nd":
		// Name description
		if len(args) > 0 {
			p.sections["_TITLE"] = []string{strings.Join(args, " ")}
		}
	case "Sh":
		// Section header
		if len(args) > 0 {
			p.currentSection = strings.ToUpper(strings.Join(args, " "))
		}
	case "Ss":
		// Subsection
		if p.currentSection != "" && len(args) > 0 {
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				fmt.Sprintf("__SUBSECTION__%s", strings.Join(args, " ")))
		}
	case "Pp":
		// Paragraph
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "")
		}
	case "Fl":
		// Flag
		if p.currentSection != "" && len(args) > 0 {
			text := fmt.Sprintf("__BOLD__-%s__/BOLD__", args[0])
			p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
		}
	case "Ar":
		// Argument
		if p.currentSection != "" && len(args) > 0 {
			text := fmt.Sprintf("__ITALIC__%s__/ITALIC__", strings.Join(args, " "))
			p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
		}
	case "Op":
		// Optional
		if p.currentSection != "" && len(args) > 0 {
			text := fmt.Sprintf("[%s]", strings.Join(args, " "))
			p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
		}
	case "Xr":
		// Cross-reference
		if p.currentSection != "" && len(args) >= 1 {
			ref := args[0]
			if len(args) >= 2 {
				ref = fmt.Sprintf("%s(%s)", args[0], args[1])
			}
			text := fmt.Sprintf("__XREF__%s__/XREF__", ref)
			p.sections[p.currentSection] = append(p.sections[p.currentSection], text)
		}
	case "Bl":
		// Begin list
		p.inList = true
		if len(args) > 0 {
			p.listType = args[0]
		}
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__LIST__")
		}
	case "El":
		// End list
		p.inList = false
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__/LIST__")
		}
	case "It":
		// List item
		if p.currentSection != "" {
			if len(args) > 0 {
				p.sections[p.currentSection] = append(p.sections[p.currentSection],
					fmt.Sprintf("__ITEM__%s", strings.Join(args, " ")))
			} else {
				p.sections[p.currentSection] = append(p.sections[p.currentSection], "__ITEM__")
			}
		}
	case "Bd":
		// Begin display (preformatted)
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__PRE__")
		}
	case "Ed":
		// End display
		if p.currentSection != "" {
			p.sections[p.currentSection] = append(p.sections[p.currentSection], "__/PRE__")
		}
	default:
		// Unknown macro - try to extract text
		if p.currentSection != "" && len(args) > 0 {
			p.sections[p.currentSection] = append(p.sections[p.currentSection],
				strings.Join(args, " "))
		}
	}
}

// splitMacroLine splits a macro line into parts, respecting quotes.
func splitMacroLine(line string) []string {
	// Remove leading dot
	line = strings.TrimPrefix(line, ".")

	var parts []string
	var current strings.Builder
	inQuote := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// Extraction methods
func (p *Parser) extractName() string {
	if names, ok := p.sections["_NAME"]; ok && len(names) > 0 {
		return names[0]
	}

	// Try to extract from NAME section
	if lines, ok := p.sections["NAME"]; ok {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Format: "name - description" or "name, name2 - description"
			if idx := strings.Index(line, " - "); idx > 0 {
				names := strings.Split(line[:idx], ",")
				if len(names) > 0 {
					return strings.TrimSpace(names[0])
				}
			}
			// Just return first word
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return strings.Trim(parts[0], "(),")
			}
		}
	}

	return ""
}

func (p *Parser) extractSection() string {
	if sections, ok := p.sections["_SECTION"]; ok && len(sections) > 0 {
		return sections[0]
	}
	return "1"
}

func (p *Parser) extractTitle() string {
	if titles, ok := p.sections["_TITLE"]; ok && len(titles) > 0 {
		return titles[0]
	}

	// Try to extract from NAME section
	if lines, ok := p.sections["NAME"]; ok {
		for _, line := range lines {
			line = p.cleanFormatting(line)
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Try various separator patterns:
			// "name - description"
			// "name \- description" (groff minus)
			// "name -- description"
			separators := []string{" - ", " \\- ", " -- ", "\\-", " \u2013 ", " \u2014 "}
			for _, sep := range separators {
				if idx := strings.Index(line, sep); idx > 0 {
					return strings.TrimSpace(line[idx+len(sep):])
				}
			}
		}
	}

	return ""
}

func (p *Parser) extractDescription() string {
	if lines, ok := p.sections["DESCRIPTION"]; ok {
		// Return first paragraph
		var desc strings.Builder
		for _, line := range lines {
			line = p.cleanFormatting(line)
			line = strings.TrimSpace(line)
			if line == "" && desc.Len() > 0 {
				break
			}
			if desc.Len() > 0 {
				desc.WriteString(" ")
			}
			desc.WriteString(line)
		}
		result := desc.String()
		// Limit length
		if len(result) > 500 {
			result = result[:497] + "..."
		}
		return result
	}
	return ""
}

func (p *Parser) extractSeeAlso() []string {
	var refs []string
	if lines, ok := p.sections["SEE ALSO"]; ok {
		refRegex := regexp.MustCompile(`(\w+)\((\d+[a-z]?)\)`)
		for _, line := range lines {
			line = p.cleanFormatting(line)
			matches := refRegex.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				refs = append(refs, fmt.Sprintf("%s(%s)", m[1], m[2]))
			}
		}
	}
	return refs
}

func (p *Parser) renderSectionText(section string) string {
	if lines, ok := p.sections[section]; ok {
		var result strings.Builder
		for _, line := range lines {
			line = p.cleanFormatting(line)
			line = strings.TrimSpace(line)
			if result.Len() > 0 && line != "" {
				result.WriteString(" ")
			}
			result.WriteString(line)
		}
		return result.String()
	}
	return ""
}

func (p *Parser) cleanFormatting(text string) string {
	// Remove our internal formatting markers
	text = strings.ReplaceAll(text, "__BOLD__", "")
	text = strings.ReplaceAll(text, "__/BOLD__", "")
	text = strings.ReplaceAll(text, "__ITALIC__", "")
	text = strings.ReplaceAll(text, "__/ITALIC__", "")
	text = strings.ReplaceAll(text, "__XREF__", "")
	text = strings.ReplaceAll(text, "__/XREF__", "")
	text = strings.ReplaceAll(text, "__PRE__", "")
	text = strings.ReplaceAll(text, "__/PRE__", "")
	text = strings.ReplaceAll(text, "__LIST__", "")
	text = strings.ReplaceAll(text, "__/LIST__", "")
	text = strings.ReplaceAll(text, "__INDENT__", "")
	text = strings.ReplaceAll(text, "__/INDENT__", "")
	text = strings.ReplaceAll(text, "__TP__", "")

	// Remove item and subsection markers but keep content
	text = regexp.MustCompile(`__ITEM__`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`__IP__`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`__SUBSECTION__`).ReplaceAllString(text, "")

	return text
}

// Rendering methods
func (p *Parser) renderHTML() string {
	var html strings.Builder

	html.WriteString("<div class=\"man-page\">\n")

	// Render each section in order
	sectionOrder := []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "EXAMPLES",
		"FILES", "ENVIRONMENT", "EXIT STATUS", "RETURN VALUE", "ERRORS",
		"NOTES", "BUGS", "AUTHORS", "HISTORY", "SEE ALSO"}

	rendered := make(map[string]bool)

	for _, section := range sectionOrder {
		if lines, ok := p.sections[section]; ok {
			html.WriteString(p.renderHTMLSection(section, lines))
			rendered[section] = true
		}
	}

	// Render any remaining sections
	for section, lines := range p.sections {
		if strings.HasPrefix(section, "_") {
			continue
		}
		if !rendered[section] {
			html.WriteString(p.renderHTMLSection(section, lines))
		}
	}

	html.WriteString("</div>\n")

	return html.String()
}

func (p *Parser) renderHTMLSection(name string, lines []string) string {
	var html strings.Builder

	html.WriteString(fmt.Sprintf("<section id=\"%s\">\n", strings.ToLower(strings.ReplaceAll(name, " ", "-"))))
	html.WriteString(fmt.Sprintf("<h2>%s</h2>\n", htmlEscape(name)))

	inPre := false
	inList := false
	inIndent := false
	inTP := false

	for _, line := range lines {
		switch {
		case line == "__PRE__":
			if !inPre {
				html.WriteString("<pre><code>")
				inPre = true
			}
		case line == "__/PRE__":
			if inPre {
				html.WriteString("</code></pre>\n")
				inPre = false
			}
		case line == "__LIST__":
			if !inList {
				html.WriteString("<ul>\n")
				inList = true
			}
		case line == "__/LIST__":
			if inList {
				html.WriteString("</ul>\n")
				inList = false
			}
		case strings.HasPrefix(line, "__ITEM__"):
			content := strings.TrimPrefix(line, "__ITEM__")
			content = p.formatHTMLInline(content)
			html.WriteString(fmt.Sprintf("<li>%s</li>\n", content))
		case line == "__INDENT__":
			if !inIndent {
				html.WriteString("<div class=\"indent\">\n")
				inIndent = true
			}
		case line == "__/INDENT__":
			if inIndent {
				html.WriteString("</div>\n")
				inIndent = false
			}
		case line == "__TP__":
			if inTP {
				html.WriteString("</dd>\n")
			}
			html.WriteString("<dl class=\"options\">\n")
			inTP = true
		case strings.HasPrefix(line, "__SUBSECTION__"):
			content := strings.TrimPrefix(line, "__SUBSECTION__")
			html.WriteString(fmt.Sprintf("<h3>%s</h3>\n", htmlEscape(content)))
		case strings.HasPrefix(line, "__IP__"):
			content := strings.TrimPrefix(line, "__IP__")
			content = p.formatHTMLInline(content)
			html.WriteString(fmt.Sprintf("<p class=\"tag\">%s</p>\n", content))
		case inPre:
			html.WriteString(htmlEscape(line) + "\n")
		case line == "":
			if !inPre {
				html.WriteString("<p></p>\n")
			}
		default:
			content := p.formatHTMLInline(line)
			if inTP {
				// Check if this looks like an option (starts with -)
				if strings.HasPrefix(strings.TrimSpace(line), "-") ||
					strings.HasPrefix(strings.TrimSpace(line), "__BOLD__-") {
					if inTP {
						html.WriteString("</dd>\n")
					}
					html.WriteString(fmt.Sprintf("<dt>%s</dt>\n<dd>", content))
				} else {
					html.WriteString(content + " ")
				}
			} else {
				html.WriteString(fmt.Sprintf("<p>%s</p>\n", content))
			}
		}
	}

	if inTP {
		html.WriteString("</dd></dl>\n")
	}
	if inPre {
		html.WriteString("</code></pre>\n")
	}
	if inList {
		html.WriteString("</ul>\n")
	}
	if inIndent {
		html.WriteString("</div>\n")
	}

	html.WriteString("</section>\n")

	return html.String()
}

func (p *Parser) formatHTMLInline(text string) string {
	// Escape HTML first
	text = htmlEscape(text)

	// Convert our markers to HTML
	text = strings.ReplaceAll(text, "__BOLD__", "<strong>")
	text = strings.ReplaceAll(text, "__/BOLD__", "</strong>")
	text = strings.ReplaceAll(text, "__ITALIC__", "<em>")
	text = strings.ReplaceAll(text, "__/ITALIC__", "</em>")

	// Handle cross-references as links
	xrefRegex := regexp.MustCompile(`__XREF__([^_]+)\(([^)]+)\)__/XREF__`)
	text = xrefRegex.ReplaceAllString(text, `<a href="/man/$2/$1">$1($2)</a>`)
	text = strings.ReplaceAll(text, "__XREF__", "")
	text = strings.ReplaceAll(text, "__/XREF__", "")

	return text
}

func (p *Parser) renderText() string {
	var text strings.Builder

	sectionOrder := []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "EXAMPLES",
		"FILES", "ENVIRONMENT", "EXIT STATUS", "RETURN VALUE", "ERRORS",
		"NOTES", "BUGS", "AUTHORS", "HISTORY", "SEE ALSO"}

	rendered := make(map[string]bool)

	for _, section := range sectionOrder {
		if lines, ok := p.sections[section]; ok {
			text.WriteString(p.renderTextSection(section, lines))
			rendered[section] = true
		}
	}

	for section, lines := range p.sections {
		if strings.HasPrefix(section, "_") {
			continue
		}
		if !rendered[section] {
			text.WriteString(p.renderTextSection(section, lines))
		}
	}

	return text.String()
}

func (p *Parser) renderTextSection(name string, lines []string) string {
	var text strings.Builder

	text.WriteString(name + "\n")
	text.WriteString(strings.Repeat("-", len(name)) + "\n")

	for _, line := range lines {
		clean := p.cleanFormatting(line)
		if strings.HasPrefix(line, "__SUBSECTION__") {
			clean = strings.TrimPrefix(line, "__SUBSECTION__")
			text.WriteString("\n  " + clean + "\n")
		} else if strings.HasPrefix(line, "__IP__") {
			clean = strings.TrimPrefix(line, "__IP__")
			text.WriteString("  " + clean + "\n")
		} else if clean != "" {
			text.WriteString("  " + clean + "\n")
		} else {
			text.WriteString("\n")
		}
	}

	text.WriteString("\n")
	return text.String()
}

func (p *Parser) renderMarkdown() string {
	var md strings.Builder

	sectionOrder := []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "EXAMPLES",
		"FILES", "ENVIRONMENT", "EXIT STATUS", "RETURN VALUE", "ERRORS",
		"NOTES", "BUGS", "AUTHORS", "HISTORY", "SEE ALSO"}

	rendered := make(map[string]bool)

	for _, section := range sectionOrder {
		if lines, ok := p.sections[section]; ok {
			md.WriteString(p.renderMarkdownSection(section, lines))
			rendered[section] = true
		}
	}

	for section, lines := range p.sections {
		if strings.HasPrefix(section, "_") {
			continue
		}
		if !rendered[section] {
			md.WriteString(p.renderMarkdownSection(section, lines))
		}
	}

	return md.String()
}

func (p *Parser) renderMarkdownSection(name string, lines []string) string {
	var md strings.Builder

	md.WriteString("## " + name + "\n\n")

	inPre := false

	for _, line := range lines {
		switch {
		case line == "__PRE__":
			md.WriteString("```\n")
			inPre = true
		case line == "__/PRE__":
			md.WriteString("```\n\n")
			inPre = false
		case strings.HasPrefix(line, "__SUBSECTION__"):
			content := strings.TrimPrefix(line, "__SUBSECTION__")
			md.WriteString("### " + content + "\n\n")
		case strings.HasPrefix(line, "__ITEM__"):
			content := strings.TrimPrefix(line, "__ITEM__")
			content = p.formatMarkdownInline(content)
			md.WriteString("- " + content + "\n")
		case strings.HasPrefix(line, "__IP__"):
			content := strings.TrimPrefix(line, "__IP__")
			content = p.formatMarkdownInline(content)
			md.WriteString("- " + content + "\n")
		case line == "__LIST__" || line == "__/LIST__" ||
			line == "__INDENT__" || line == "__/INDENT__" || line == "__TP__":
			// Skip structural markers
		case inPre:
			md.WriteString(line + "\n")
		case line == "":
			md.WriteString("\n")
		default:
			content := p.formatMarkdownInline(line)
			md.WriteString(content + "\n")
		}
	}

	md.WriteString("\n")
	return md.String()
}

func (p *Parser) formatMarkdownInline(text string) string {
	text = strings.ReplaceAll(text, "__BOLD__", "**")
	text = strings.ReplaceAll(text, "__/BOLD__", "**")
	text = strings.ReplaceAll(text, "__ITALIC__", "*")
	text = strings.ReplaceAll(text, "__/ITALIC__", "*")

	// Handle cross-references
	xrefRegex := regexp.MustCompile(`__XREF__([^_]+)__/XREF__`)
	text = xrefRegex.ReplaceAllString(text, "`$1`")
	text = strings.ReplaceAll(text, "__XREF__", "`")
	text = strings.ReplaceAll(text, "__/XREF__", "`")

	return text
}

func (p *Parser) buildSearchText() string {
	var search strings.Builder

	// Include name and title
	if names, ok := p.sections["_NAME"]; ok {
		search.WriteString(strings.Join(names, " ") + " ")
	}
	if titles, ok := p.sections["_TITLE"]; ok {
		search.WriteString(strings.Join(titles, " ") + " ")
	}

	// Include content from main sections
	for _, section := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS"} {
		if lines, ok := p.sections[section]; ok {
			for _, line := range lines {
				clean := p.cleanFormatting(line)
				search.WriteString(clean + " ")
			}
		}
	}

	return strings.TrimSpace(search.String())
}

// htmlEscape escapes HTML special characters.
func htmlEscape(s string) string {
	return html.EscapeString(s)
}
