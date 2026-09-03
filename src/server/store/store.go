// Package store provides SQLite database operations for casman.
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/casapps/casman/src/server/model"
)

// DB wraps the SQL database connection.
type DB struct {
	conn *sql.DB
}

// New creates a new database connection.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enabling WAL mode: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the database tables if they don't exist.
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS manpages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		section TEXT NOT NULL,
		title TEXT NOT NULL,
		platform TEXT NOT NULL,
		distro TEXT DEFAULT '',
		version TEXT DEFAULT '',
		language TEXT DEFAULT 'en',
		source_format TEXT DEFAULT 'groff',
		source_raw TEXT,
		source_url TEXT DEFAULT '',
		content_html TEXT,
		content_text TEXT,
		content_markdown TEXT,
		synopsis TEXT DEFAULT '',
		description TEXT DEFAULT '',
		see_also TEXT DEFAULT '',
		search_text TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, section, platform, language)
	);

	CREATE INDEX IF NOT EXISTS idx_manpages_name ON manpages(name);
	CREATE INDEX IF NOT EXISTS idx_manpages_section ON manpages(section);
	CREATE INDEX IF NOT EXISTS idx_manpages_platform ON manpages(platform);
	CREATE INDEX IF NOT EXISTS idx_manpages_language ON manpages(language);

	CREATE VIRTUAL TABLE IF NOT EXISTS manpages_fts USING fts5(
		name,
		title,
		synopsis,
		description,
		search_text,
		content='manpages',
		content_rowid='id'
	);

	CREATE TABLE IF NOT EXISTS page_views (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manpage_id INTEGER NOT NULL,
		viewed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (manpage_id) REFERENCES manpages(id)
	);

	CREATE INDEX IF NOT EXISTS idx_page_views_manpage ON page_views(manpage_id);
	CREATE INDEX IF NOT EXISTS idx_page_views_date ON page_views(viewed_at);

	-- Config key-value storage per AI.md PART 10
	CREATE TABLE IF NOT EXISTS config (
		key         TEXT PRIMARY KEY,
		value       TEXT NOT NULL,
		type        TEXT NOT NULL DEFAULT 'string',
		updated_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	);

	-- Config metadata for change detection per AI.md PART 10
	CREATE TABLE IF NOT EXISTS config_meta (
		id          INTEGER PRIMARY KEY CHECK (id = 1),
		version     INTEGER NOT NULL DEFAULT 1,
		updated_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	);

	-- Initialize metadata row
	INSERT OR IGNORE INTO config_meta (id, version) VALUES (1, 1);

	CREATE INDEX IF NOT EXISTS idx_config_key_prefix ON config(key);

	-- Rate limiting per AI.md PART 11
	CREATE TABLE IF NOT EXISTS rate_limits (
		key         TEXT PRIMARY KEY,
		count       INTEGER NOT NULL DEFAULT 1,
		window_start INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		updated_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	);

	CREATE INDEX IF NOT EXISTS idx_rate_limits_window ON rate_limits(window_start);

	-- Audit log per AI.md PART 11
	CREATE TABLE IF NOT EXISTS audit_log (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp   INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		level       TEXT NOT NULL DEFAULT 'info',
		category    TEXT NOT NULL,
		action      TEXT NOT NULL,
		actor_type  TEXT,
		actor_id    TEXT,
		actor_ip    TEXT,
		target_type TEXT,
		target_id   TEXT,
		details     TEXT,
		success     INTEGER NOT NULL DEFAULT 1
	);

	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_category ON audit_log(category);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor_type, actor_id);

	-- Scheduler tasks per AI.md PART 19
	CREATE TABLE IF NOT EXISTS scheduler_tasks (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		task_type   TEXT NOT NULL DEFAULT 'global',
		enabled     INTEGER NOT NULL DEFAULT 1,
		schedule    TEXT NOT NULL,
		last_run    INTEGER,
		next_run    INTEGER,
		last_status TEXT,
		last_error  TEXT,
		run_count   INTEGER NOT NULL DEFAULT 0,
		fail_count  INTEGER NOT NULL DEFAULT 0,
		locked_by   TEXT,
		locked_at   INTEGER
	);

	CREATE TABLE IF NOT EXISTS scheduler_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id     TEXT NOT NULL,
		started_at  INTEGER NOT NULL,
		finished_at INTEGER,
		status      TEXT NOT NULL,
		error       TEXT,
		duration_ms INTEGER
	);

	CREATE INDEX IF NOT EXISTS idx_scheduler_history_task ON scheduler_history(task_id);
	CREATE INDEX IF NOT EXISTS idx_scheduler_history_started ON scheduler_history(started_at);

	-- Backups per AI.md PART 22
	CREATE TABLE IF NOT EXISTS backups (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		filename    TEXT NOT NULL UNIQUE,
		filepath    TEXT NOT NULL,
		size_bytes  INTEGER NOT NULL,
		type        TEXT NOT NULL DEFAULT 'auto',
		created_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		checksum    TEXT,
		notes       TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_backups_created ON backups(created_at);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetManPage retrieves a man page by platform, section, and name.
func (db *DB) GetManPage(platform, section, name string) (*model.ManPage, error) {
	query := `
		SELECT id, name, section, title, platform, distro, version, language,
		       source_format, source_raw, source_url,
		       content_html, content_text, content_markdown,
		       synopsis, description, see_also, search_text,
		       updated_at, created_at
		FROM manpages
		WHERE name = ? AND section = ? AND platform = ?
		LIMIT 1
	`

	row := db.conn.QueryRow(query, name, section, platform)
	return db.scanManPage(row)
}

// GetManPageByName retrieves the best matching man page by name with optional section/platform.
func (db *DB) GetManPageByName(name, section, platform string) (*model.ManPage, error) {
	var query string
	var args []interface{}

	if platform != "" && section != "" {
		return db.GetManPage(platform, section, name)
	}

	if section != "" {
		query = `
			SELECT id, name, section, title, platform, distro, version, language,
			       source_format, source_raw, source_url,
			       content_html, content_text, content_markdown,
			       synopsis, description, see_also, search_text,
			       updated_at, created_at
			FROM manpages
			WHERE name = ? AND section = ?
			ORDER BY
				CASE WHEN platform = 'linux' THEN 0 ELSE 1 END
			LIMIT 1
		`
		args = []interface{}{name, section}
	} else {
		// Priority: linux section 1, then any linux, then any platform
		query = `
			SELECT id, name, section, title, platform, distro, version, language,
			       source_format, source_raw, source_url,
			       content_html, content_text, content_markdown,
			       synopsis, description, see_also, search_text,
			       updated_at, created_at
			FROM manpages
			WHERE name = ?
			ORDER BY
				CASE WHEN platform = 'linux' AND section = '1' THEN 0
				     WHEN platform = 'linux' THEN 1
				     WHEN section = '1' THEN 2
				     ELSE 3
				END,
				section ASC
			LIMIT 1
		`
		args = []interface{}{name}
	}

	row := db.conn.QueryRow(query, args...)
	return db.scanManPage(row)
}

func (db *DB) scanManPage(row *sql.Row) (*model.ManPage, error) {
	var page model.ManPage
	var seeAlsoStr string
	var sourceRaw, contentHTML, contentText, contentMD sql.NullString

	err := row.Scan(
		&page.ID, &page.Name, &page.Section, &page.Title,
		&page.Platform, &page.Distro, &page.Version, &page.Language,
		&page.SourceFormat, &sourceRaw, &page.SourceURL,
		&contentHTML, &contentText, &contentMD,
		&page.Synopsis, &page.Description, &seeAlsoStr, &page.SearchText,
		&page.UpdatedAt, &page.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if sourceRaw.Valid {
		page.SourceRaw = sourceRaw.String
		page.ContentRaw = sourceRaw.String
	}
	if contentHTML.Valid {
		page.ContentHTML = contentHTML.String
	}
	if contentText.Valid {
		page.ContentText = contentText.String
	}
	if contentMD.Valid {
		page.ContentMarkdown = contentMD.String
	}

	// Parse see_also as comma-separated into SeeAlsoEntry slice
	if seeAlsoStr != "" {
		parts := strings.Split(seeAlsoStr, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			entry := model.SeeAlsoEntry{Name: part}
			// Try to parse name(section) format
			if idx := strings.Index(part, "("); idx > 0 {
				entry.Name = part[:idx]
				if endIdx := strings.Index(part, ")"); endIdx > idx {
					entry.Section = part[idx+1 : endIdx]
				}
			}
			entry.URL = fmt.Sprintf("/man/%s", entry.Name)
			if entry.Section != "" {
				entry.URL = fmt.Sprintf("/man/%s/%s", entry.Section, entry.Name)
			}
			page.SeeAlso = append(page.SeeAlso, entry)
		}
	}

	return &page, nil
}

// GetOtherPlatforms returns other platforms that have this man page.
func (db *DB) GetOtherPlatforms(name, section, excludePlatform string) ([]string, error) {
	query := `
		SELECT DISTINCT platform
		FROM manpages
		WHERE name = ? AND section = ? AND platform != ?
		ORDER BY platform
	`

	rows, err := db.conn.Query(query, name, section, excludePlatform)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		platforms = append(platforms, p)
	}

	return platforms, nil
}

// Search performs a full-text search across man pages.
func (db *DB) Search(query, section, platform string, page, limit int) ([]model.SearchResult, int, error) {
	offset := (page - 1) * limit

	// Build the search query
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "manpages_fts MATCH ?")
	args = append(args, query)

	if section != "" && section != "any" {
		conditions = append(conditions, "m.section = ?")
		args = append(args, section)
	}
	if platform != "" && platform != "any" {
		conditions = append(conditions, "m.platform = ?")
		args = append(args, platform)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM manpages m
		JOIN manpages_fts ON m.id = manpages_fts.rowid
		WHERE %s
	`, whereClause)

	var total int
	if err := db.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get results
	searchQuery := fmt.Sprintf(`
		SELECT m.name, m.section, m.title, m.platform, m.distro,
		       snippet(manpages_fts, 4, '<mark>', '</mark>', '...', 32) as snippet,
		       bm25(manpages_fts) as score
		FROM manpages m
		JOIN manpages_fts ON m.id = manpages_fts.rowid
		WHERE %s
		ORDER BY score
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.conn.Query(searchQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []model.SearchResult
	for rows.Next() {
		var r model.SearchResult
		if err := rows.Scan(&r.Name, &r.Section, &r.Title, &r.Platform, &r.Distro, &r.Snippet, &r.Score); err != nil {
			return nil, 0, err
		}
		r.URL = fmt.Sprintf("/man/%s/%s/%s", r.Platform, r.Section, r.Name)
		results = append(results, r)
	}

	return results, total, nil
}

// Autocomplete returns quick name matches for autocomplete.
func (db *DB) Autocomplete(query string, limit int) ([]model.ManPageSummary, error) {
	sqlQuery := `
		SELECT DISTINCT name, section, title, platform
		FROM manpages
		WHERE name LIKE ? || '%'
		ORDER BY
			CASE WHEN name = ? THEN 0
			     WHEN name LIKE ? || '%' THEN 1
			     ELSE 2
			END,
			LENGTH(name),
			name
		LIMIT ?
	`

	rows, err := db.conn.Query(sqlQuery, query, query, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ManPageSummary
	for rows.Next() {
		var r model.ManPageSummary
		if err := rows.Scan(&r.Name, &r.Section, &r.Title, &r.Platform); err != nil {
			return nil, err
		}
		r.URL = fmt.Sprintf("/man/%s/%s/%s", r.Platform, r.Section, r.Name)
		results = append(results, r)
	}

	return results, nil
}

// GetSections returns all sections with page counts.
func (db *DB) GetSections() ([]model.Section, error) {
	query := `
		SELECT section, COUNT(*) as count
		FROM manpages
		GROUP BY section
		ORDER BY section
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionMap := make(map[string]int)
	for rows.Next() {
		var section string
		var count int
		if err := rows.Scan(&section, &count); err != nil {
			return nil, err
		}
		sectionMap[section] = count
	}

	// Merge with predefined sections
	result := make([]model.Section, len(model.Sections))
	for i, s := range model.Sections {
		result[i] = s
		if count, ok := sectionMap[s.ID]; ok {
			result[i].Count = count
		}
	}

	return result, nil
}

// GetPlatforms returns all platforms with page counts.
func (db *DB) GetPlatforms() ([]model.Platform, error) {
	query := `
		SELECT platform, COUNT(*) as count
		FROM manpages
		GROUP BY platform
		ORDER BY count DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	platformMap := make(map[string]int)
	for rows.Next() {
		var platform string
		var count int
		if err := rows.Scan(&platform, &count); err != nil {
			return nil, err
		}
		platformMap[platform] = count
	}

	// Merge with predefined platforms
	result := make([]model.Platform, len(model.Platforms))
	for i, p := range model.Platforms {
		result[i] = p
		if count, ok := platformMap[p.ID]; ok {
			result[i].Count = count
		}
	}

	return result, nil
}

// GetStats returns database statistics.
func (db *DB) GetStats() (model.Stats, error) {
	stats := model.Stats{
		BySection:  make(map[string]int),
		ByPlatform: make(map[string]int),
	}

	// Total pages
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM manpages").Scan(&stats.TotalPages); err != nil {
		return stats, err
	}

	// Sections count
	if err := db.conn.QueryRow("SELECT COUNT(DISTINCT section) FROM manpages").Scan(&stats.TotalSections); err != nil {
		return stats, err
	}

	// Platforms count
	if err := db.conn.QueryRow("SELECT COUNT(DISTINCT platform) FROM manpages").Scan(&stats.TotalPlatforms); err != nil {
		return stats, err
	}

	// Languages count
	if err := db.conn.QueryRow("SELECT COUNT(DISTINCT language) FROM manpages").Scan(&stats.TotalLanguages); err != nil {
		return stats, err
	}

	// By section
	rows, err := db.conn.Query("SELECT section, COUNT(*) FROM manpages GROUP BY section")
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var section string
		var count int
		if err := rows.Scan(&section, &count); err != nil {
			rows.Close()
			return stats, err
		}
		stats.BySection[section] = count
	}
	rows.Close()

	// By platform
	rows, err = db.conn.Query("SELECT platform, COUNT(*) FROM manpages GROUP BY platform")
	if err != nil {
		return stats, err
	}
	for rows.Next() {
		var platform string
		var count int
		if err := rows.Scan(&platform, &count); err != nil {
			rows.Close()
			return stats, err
		}
		stats.ByPlatform[platform] = count
	}
	rows.Close()

	// Last updated
	var lastUpdated sql.NullTime
	if err := db.conn.QueryRow("SELECT MAX(updated_at) FROM manpages").Scan(&lastUpdated); err != nil {
		return stats, err
	}
	if lastUpdated.Valid {
		stats.LastUpdated = lastUpdated.Time
	} else {
		stats.LastUpdated = time.Now()
	}

	return stats, nil
}

// GetPopular returns the most viewed man pages.
func (db *DB) GetPopular(limit int) ([]model.ManPageSummary, error) {
	query := `
		SELECT m.name, m.section, m.title, m.platform, COUNT(pv.id) as views
		FROM manpages m
		LEFT JOIN page_views pv ON m.id = pv.manpage_id
		GROUP BY m.id
		ORDER BY views DESC, m.name
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ManPageSummary
	for rows.Next() {
		var p model.ManPageSummary
		var views int
		if err := rows.Scan(&p.Name, &p.Section, &p.Title, &p.Platform, &views); err != nil {
			return nil, err
		}
		p.URL = fmt.Sprintf("/man/%s/%s/%s", p.Platform, p.Section, p.Name)
		results = append(results, p)
	}

	return results, nil
}

// RecordView records a page view for popularity tracking.
func (db *DB) RecordView(pageID int64) error {
	_, err := db.conn.Exec("INSERT INTO page_views (manpage_id) VALUES (?)", pageID)
	return err
}

// Whatis returns one-line descriptions matching the name.
func (db *DB) Whatis(name string) ([]model.ManPageSummary, error) {
	query := `
		SELECT name, section, title, platform
		FROM manpages
		WHERE name = ?
		ORDER BY platform, section
	`

	rows, err := db.conn.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ManPageSummary
	for rows.Next() {
		var r model.ManPageSummary
		if err := rows.Scan(&r.Name, &r.Section, &r.Title, &r.Platform); err != nil {
			return nil, err
		}
		r.URL = fmt.Sprintf("/man/%s/%s/%s", r.Platform, r.Section, r.Name)
		results = append(results, r)
	}

	return results, nil
}

// Apropos searches man page descriptions.
func (db *DB) Apropos(query string) ([]model.ManPageSummary, error) {
	sqlQuery := `
		SELECT name, section, title, platform
		FROM manpages
		WHERE title LIKE '%' || ? || '%'
		   OR description LIKE '%' || ? || '%'
		ORDER BY
			CASE WHEN title LIKE ? || '%' THEN 0
			     WHEN title LIKE '%' || ? || '%' THEN 1
			     ELSE 2
			END,
			name
		LIMIT 100
	`

	rows, err := db.conn.Query(sqlQuery, query, query, query, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ManPageSummary
	for rows.Next() {
		var r model.ManPageSummary
		if err := rows.Scan(&r.Name, &r.Section, &r.Title, &r.Platform); err != nil {
			return nil, err
		}
		r.URL = fmt.Sprintf("/man/%s/%s/%s", r.Platform, r.Section, r.Name)
		results = append(results, r)
	}

	return results, nil
}

// Browse lists man pages with optional filtering.
func (db *DB) Browse(section, platform string, page, limit int) ([]model.ManPageSummary, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}

	if section != "" && section != "any" {
		conditions = append(conditions, "section = ?")
		args = append(args, section)
	}
	if platform != "" && platform != "any" {
		conditions = append(conditions, "platform = ?")
		args = append(args, platform)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM manpages %s", whereClause)
	var total int
	if err := db.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get pages
	selectQuery := fmt.Sprintf(`
		SELECT name, section, title, platform
		FROM manpages
		%s
		ORDER BY name, section
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.conn.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pages []model.ManPageSummary
	for rows.Next() {
		var p model.ManPageSummary
		if err := rows.Scan(&p.Name, &p.Section, &p.Title, &p.Platform); err != nil {
			return nil, 0, err
		}
		p.URL = fmt.Sprintf("/man/%s/%s/%s", p.Platform, p.Section, p.Name)
		pages = append(pages, p)
	}

	return pages, total, nil
}

// Compare gets all versions of a man page across platforms.
func (db *DB) Compare(name, section string) (*model.CompareResult, error) {
	query := `
		SELECT platform, section, title, synopsis, content_html
		FROM manpages
		WHERE name = ?
	`
	args := []interface{}{name}

	if section != "" {
		query += " AND section = ?"
		args = append(args, section)
	}
	query += " ORDER BY platform"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &model.CompareResult{
		Name:    name,
		Section: section,
	}

	for rows.Next() {
		var cp model.ComparePlatform
		var contentHTML sql.NullString
		if err := rows.Scan(&cp.Platform, &result.Section, &cp.Title, &cp.Synopsis, &contentHTML); err != nil {
			return nil, err
		}
		if contentHTML.Valid {
			cp.ContentHTML = contentHTML.String
		}
		cp.Available = true
		result.Platforms = append(result.Platforms, cp)
	}

	if len(result.Platforms) == 0 {
		return nil, fmt.Errorf("no pages found for %s", name)
	}

	return result, nil
}

// GetTLDR generates a TLDR summary for a man page.
func (db *DB) GetTLDR(name, section string) (*model.TLDR, error) {
	var page *model.ManPage
	var err error

	if section != "" {
		page, err = db.GetManPageByName(name, section, "")
	} else {
		page, err = db.GetManPageByName(name, "", "")
	}

	if err != nil || page == nil {
		return nil, fmt.Errorf("page not found: %s", name)
	}

	tldr := &model.TLDR{
		Name:        page.Name,
		Section:     page.Section,
		OneLiner:    page.Title,
		Source:      "auto",
		GeneratedAt: time.Now(),
	}

	// Extract key options from synopsis
	tldr.KeyOptions = extractKeyOptions(page.Synopsis, page.ContentText)

	// Generate common examples based on the command
	tldr.CommonExamples = generateExamples(page.Name, page.Section, page.Synopsis)

	return tldr, nil
}

// extractKeyOptions extracts key options from synopsis and content.
func extractKeyOptions(synopsis, content string) []model.TLDROption {
	var options []model.TLDROption

	// Common option patterns to look for
	commonOpts := map[string]string{
		"-h":      "show help",
		"--help":  "show help",
		"-v":      "verbose output",
		"-V":      "show version",
		"-a":      "all",
		"-l":      "long format",
		"-r":      "recursive",
		"-R":      "recursive",
		"-f":      "force",
		"-i":      "interactive",
		"-n":      "dry run / numeric",
		"-q":      "quiet",
		"-d":      "directory / debug",
	}

	// Check which common options appear in synopsis or content
	for flag, desc := range commonOpts {
		if strings.Contains(synopsis, flag) || strings.Contains(content, flag+" ") || strings.Contains(content, flag+",") {
			options = append(options, model.TLDROption{Flag: flag, Description: desc})
			if len(options) >= 5 {
				break
			}
		}
	}

	return options
}

// generateExamples generates common usage examples for a command.
func generateExamples(name, section, synopsis string) []model.TLDRExample {
	var examples []model.TLDRExample

	// Section 1 = user commands
	if section == "1" {
		switch name {
		case "ls":
			examples = []model.TLDRExample{
				{Command: "ls -la", Description: "List all files with details"},
				{Command: "ls -lh", Description: "List with human-readable sizes"},
				{Command: "ls -lt", Description: "List sorted by modification time"},
				{Command: "ls *.txt", Description: "List only .txt files"},
			}
		case "grep":
			examples = []model.TLDRExample{
				{Command: "grep pattern file", Description: "Search for pattern in file"},
				{Command: "grep -r pattern dir/", Description: "Search recursively in directory"},
				{Command: "grep -i pattern file", Description: "Case-insensitive search"},
				{Command: "grep -n pattern file", Description: "Show line numbers"},
			}
		case "find":
			examples = []model.TLDRExample{
				{Command: "find . -name '*.txt'", Description: "Find files by name"},
				{Command: "find . -type d", Description: "Find directories only"},
				{Command: "find . -mtime -7", Description: "Find files modified in last 7 days"},
				{Command: "find . -size +10M", Description: "Find files larger than 10MB"},
			}
		case "chmod":
			examples = []model.TLDRExample{
				{Command: "chmod +x file", Description: "Make file executable"},
				{Command: "chmod 755 file", Description: "Set rwxr-xr-x permissions"},
				{Command: "chmod -R 644 dir/", Description: "Set permissions recursively"},
			}
		case "cat":
			examples = []model.TLDRExample{
				{Command: "cat file", Description: "Display file contents"},
				{Command: "cat file1 file2", Description: "Concatenate multiple files"},
				{Command: "cat -n file", Description: "Display with line numbers"},
			}
		case "ssh":
			examples = []model.TLDRExample{
				{Command: "ssh user@host", Description: "Connect to remote host"},
				{Command: "ssh -p 2222 user@host", Description: "Connect on custom port"},
				{Command: "ssh -i key.pem user@host", Description: "Connect with specific key"},
				{Command: "ssh -L 8080:localhost:80 user@host", Description: "Local port forwarding"},
			}
		default:
			// Generic example
			examples = []model.TLDRExample{
				{Command: name + " --help", Description: "Show help message"},
			}
		}
	} else if section == "8" {
		// Admin commands
		examples = []model.TLDRExample{
			{Command: "sudo " + name + " --help", Description: "Show help (requires root)"},
		}
	}

	return examples
}

// GetRecentPages returns recently updated pages for feeds.
func (db *DB) GetRecentPages(platform, section string, limit int) ([]model.FeedEntry, error) {
	query := `
		SELECT name, section, title, platform, description, updated_at
		FROM manpages
		WHERE 1=1
	`
	args := []interface{}{}

	if platform != "" {
		query += " AND platform = ?"
		args = append(args, platform)
	}
	if section != "" {
		query += " AND section = ?"
		args = append(args, section)
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.FeedEntry
	for rows.Next() {
		var e model.FeedEntry
		var desc sql.NullString
		if err := rows.Scan(&e.Name, &e.Section, &e.Title, &e.Platform, &desc, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			e.Summary = desc.String
		}
		e.URL = fmt.Sprintf("/man/%s/%s/%s", e.Platform, e.Section, e.Name)
		entries = append(entries, e)
	}

	return entries, nil
}

// GetAllPageURLs returns all page URLs for sitemap.
func (db *DB) GetAllPageURLs() ([]model.ManPageSummary, error) {
	query := `
		SELECT name, section, title, platform
		FROM manpages
		ORDER BY name, section, platform
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []model.ManPageSummary
	for rows.Next() {
		var p model.ManPageSummary
		if err := rows.Scan(&p.Name, &p.Section, &p.Title, &p.Platform); err != nil {
			return nil, err
		}
		p.URL = fmt.Sprintf("/man/%s/%s/%s", p.Platform, p.Section, p.Name)
		pages = append(pages, p)
	}

	return pages, nil
}

// InsertManPage inserts or updates a man page.
func (db *DB) InsertManPage(page *model.ManPage) error {
	// Convert SeeAlso entries to comma-separated string
	var seeAlsoParts []string
	for _, e := range page.SeeAlso {
		if e.Section != "" {
			seeAlsoParts = append(seeAlsoParts, fmt.Sprintf("%s(%s)", e.Name, e.Section))
		} else {
			seeAlsoParts = append(seeAlsoParts, e.Name)
		}
	}
	seeAlso := strings.Join(seeAlsoParts, ", ")

	query := `
		INSERT INTO manpages (
			name, section, title, platform, distro, version, language,
			source_format, source_raw, source_url,
			content_html, content_text, content_markdown,
			synopsis, description, see_also, search_text,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(name, section, platform, language) DO UPDATE SET
			title = excluded.title,
			distro = excluded.distro,
			version = excluded.version,
			source_format = excluded.source_format,
			source_raw = excluded.source_raw,
			source_url = excluded.source_url,
			content_html = excluded.content_html,
			content_text = excluded.content_text,
			content_markdown = excluded.content_markdown,
			synopsis = excluded.synopsis,
			description = excluded.description,
			see_also = excluded.see_also,
			search_text = excluded.search_text,
			updated_at = CURRENT_TIMESTAMP
	`

	result, err := db.conn.Exec(query,
		page.Name, page.Section, page.Title, page.Platform,
		page.Distro, page.Version, page.Language,
		page.SourceFormat, page.SourceRaw, page.SourceURL,
		page.ContentHTML, page.ContentText, page.ContentMarkdown,
		page.Synopsis, page.Description, seeAlso, page.SearchText,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	if id == 0 {
		// On update, we need to get the existing ID
		err = db.conn.QueryRow(
			"SELECT id FROM manpages WHERE name = ? AND section = ? AND platform = ? AND language = ?",
			page.Name, page.Section, page.Platform, page.Language,
		).Scan(&id)
		if err != nil {
			return err
		}
	}
	page.ID = id

	// Update FTS index - delete first then insert (FTS5 doesn't support UPSERT)
	_, _ = db.conn.Exec("DELETE FROM manpages_fts WHERE rowid = ?", page.ID)
	_, err = db.conn.Exec(`
		INSERT INTO manpages_fts(rowid, name, title, synopsis, description, search_text)
		VALUES (?, ?, ?, ?, ?, ?)
	`, page.ID, page.Name, page.Title, page.Synopsis, page.Description, page.SearchText)

	return err
}

// AuditEvent represents an audit log entry per AI.md PART 11.
type AuditEvent struct {
	Level      string // info, warning, error, security
	Category   string // auth, config, admin, api, system
	Action     string // login, logout, config_change, etc.
	ActorType  string // admin, api_key, system, anonymous
	ActorID    string
	ActorIP    string
	TargetType string // user, config, api_key, etc.
	TargetID   string
	Details    string // JSON with additional context
	Success    bool
}

// LogAudit records an audit event per AI.md PART 11.
func (db *DB) LogAudit(event AuditEvent) error {
	successVal := 1
	if !event.Success {
		successVal = 0
	}

	_, err := db.conn.Exec(`
		INSERT INTO audit_log (level, category, action, actor_type, actor_id, actor_ip, target_type, target_id, details, success)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.Level, event.Category, event.Action, event.ActorType, event.ActorID,
		event.ActorIP, event.TargetType, event.TargetID, event.Details, successVal)

	return err
}

// GetAuditLogs retrieves recent audit log entries.
func (db *DB) GetAuditLogs(limit, offset int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := db.conn.Query(`
		SELECT id, timestamp, level, category, action, actor_type, actor_id, actor_ip,
		       target_type, target_id, details, success
		FROM audit_log
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id, timestamp, success int64
		var level, category, action string
		var actorType, actorID, actorIP, targetType, targetID, details sql.NullString

		err := rows.Scan(&id, &timestamp, &level, &category, &action,
			&actorType, &actorID, &actorIP, &targetType, &targetID, &details, &success)
		if err != nil {
			continue
		}

		entry := map[string]interface{}{
			"id":        id,
			"timestamp": time.Unix(timestamp, 0).Format(time.RFC3339),
			"level":     level,
			"category":  category,
			"action":    action,
			"success":   success == 1,
		}

		if actorType.Valid {
			entry["actor_type"] = actorType.String
		}
		if actorID.Valid {
			entry["actor_id"] = actorID.String
		}
		if actorIP.Valid {
			entry["actor_ip"] = actorIP.String
		}
		if targetType.Valid {
			entry["target_type"] = targetType.String
		}
		if targetID.Valid {
			entry["target_id"] = targetID.String
		}
		if details.Valid {
			entry["details"] = details.String
		}

		logs = append(logs, entry)
	}

	return logs, nil
}
