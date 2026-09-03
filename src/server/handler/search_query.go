// Search-prefix tokenizer per IDEA.md "Search Types".
//
// Strings like `os:linux section:1 copy` are split into a structured query
// where the unprefixed tokens are the actual search terms and the prefixed
// tokens become filter / mode hints. URL query parameters always win over
// inline prefixes — so a deliberate `?section=8` overrides any `section:1`
// embedded in `q`. This keeps shareable URLs predictable.

package handler

import "strings"

// SearchQuery is the parsed result of tokenizing a user-typed search string.
type SearchQuery struct {
	// Query is the remaining text after prefixes have been stripped.
	Query string
	// Section is "" or the requested section ID (numeric or letter).
	Section string
	// Platform is "" or the requested OS / platform ID.
	Platform string
	// Mode is "any" (default), "name", or "content" — driven by the
	// `name:` and `content:` prefixes. Currently informational; the storage
	// search has a single ranking strategy.
	Mode string
}

// ParseSearchQuery walks the query, lifting `name:`, `content:`, `section:`,
// `os:` prefixes (and bare leading section numbers like `1 ls`). Anything
// not matched is appended to the residual query.
func ParseSearchQuery(raw string) SearchQuery {
	out := SearchQuery{Mode: "any"}
	var residual []string

	for _, tok := range strings.Fields(raw) {
		lower := strings.ToLower(tok)
		switch {
		case strings.HasPrefix(lower, "name:"):
			val := tok[len("name:"):]
			if val != "" {
				residual = append(residual, val)
			}
			out.Mode = "name"
		case strings.HasPrefix(lower, "content:"):
			val := tok[len("content:"):]
			if val != "" {
				residual = append(residual, val)
			}
			out.Mode = "content"
		case strings.HasPrefix(lower, "section:"):
			out.Section = tok[len("section:"):]
		case strings.HasPrefix(lower, "os:"):
			out.Platform = tok[len("os:"):]
		case strings.HasPrefix(lower, "platform:"):
			out.Platform = tok[len("platform:"):]
		case isBareSection(tok) && out.Section == "":
			// `1 ls` style — first token is a section, everything else is
			// the actual query.
			out.Section = tok
		default:
			residual = append(residual, tok)
		}
	}
	out.Query = strings.Join(residual, " ")
	return out
}

// MergeURLParams overlays URL-level filter values on top of the parsed query.
// Empty URL values do not override prefixes; "any" is treated as empty so
// the legacy <select name="section"> default is not sticky.
func (q SearchQuery) MergeURLParams(section, platform string) SearchQuery {
	if section == "any" {
		section = ""
	}
	if platform == "any" {
		platform = ""
	}
	if section != "" {
		q.Section = section
	}
	if platform != "" {
		q.Platform = platform
	}
	return q
}

// isBareSection accepts single-character section identifiers (1-9, n, x).
func isBareSection(tok string) bool {
	if len(tok) != 1 {
		return false
	}
	c := tok[0]
	if c >= '1' && c <= '9' {
		return true
	}
	if c == 'n' || c == 'x' {
		return true
	}
	return false
}
