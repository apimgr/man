// Package admin provides admin panel functionality for casman.
// See AI.md PART 10 for details.
package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Admin represents the admin panel.
type Admin struct {
	basePath string
	version  string
}

// New creates a new Admin instance.
func New(basePath, version string) *Admin {
	if basePath == "" {
		basePath = "admin"
	}
	return &Admin{
		basePath: basePath,
		version:  version,
	}
}

// Routes returns the admin panel routes.
func (a *Admin) Routes() chi.Router {
	r := chi.NewRouter()

	// Dashboard
	r.Get("/", a.handleDashboard)

	// Server settings
	r.Route("/server", func(r chi.Router) {
		r.Get("/", a.handleServerSettings)
		r.Get("/ssl", a.handleSSLSettings)
		r.Get("/backup", a.handleBackupSettings)
		r.Get("/scheduler", a.handleSchedulerSettings)
	})

	// Database
	r.Route("/database", func(r chi.Router) {
		r.Get("/", a.handleDatabaseOverview)
		r.Get("/browse", a.handleDatabaseBrowse)
	})

	// Users
	r.Route("/users", func(r chi.Router) {
		r.Get("/", a.handleUsersList)
		r.Get("/sessions", a.handleUserSessions)
	})

	// Logs
	r.Get("/logs", a.handleLogs)

	// API
	r.Route("/api", func(r chi.Router) {
		r.Get("/stats", a.handleAPIStats)
		r.Get("/health", a.handleAPIHealth)
	})

	return r
}

// handleDashboard renders the admin dashboard.
func (a *Admin) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Dashboard - casman</title>
    <style>
        body { font-family: system-ui, sans-serif; margin: 0; padding: 20px; background: #1a1b26; color: #c0caf5; }
        h1 { color: #7aa2f7; }
        .card { background: #24283b; padding: 20px; border-radius: 8px; margin: 10px 0; }
        a { color: #7dcfff; text-decoration: none; }
        a:hover { text-decoration: underline; }
        ul { list-style: none; padding: 0; }
        li { padding: 8px 0; }
    </style>
</head>
<body>
    <h1>Admin Dashboard</h1>
    <div class="card">
        <h2>Navigation</h2>
        <ul>
            <li><a href="server">Server Settings</a></li>
            <li><a href="database">Database</a></li>
            <li><a href="users">Users</a></li>
            <li><a href="logs">Logs</a></li>
        </ul>
    </div>
    <div class="card">
        <h2>Quick Stats</h2>
        <p>Server is running</p>
    </div>
</body>
</html>`))
}

// handleServerSettings renders server settings page.
func (a *Admin) handleServerSettings(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Server Settings", "Server configuration options")
}

// handleSSLSettings renders SSL settings page.
func (a *Admin) handleSSLSettings(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "SSL/TLS Settings", "Certificate and Let's Encrypt configuration")
}

// handleBackupSettings renders backup settings page.
func (a *Admin) handleBackupSettings(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Backup Settings", "Backup schedule and retention configuration")
}

// handleSchedulerSettings renders scheduler settings page.
func (a *Admin) handleSchedulerSettings(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Scheduler Settings", "Background task configuration")
}

// handleDatabaseOverview renders database overview page.
func (a *Admin) handleDatabaseOverview(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Database Overview", "Database statistics and health")
}

// handleDatabaseBrowse renders database browse page.
func (a *Admin) handleDatabaseBrowse(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Browse Database", "Browse database tables")
}

// handleUsersList renders users list page.
func (a *Admin) handleUsersList(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Users", "User management")
}

// handleUserSessions renders user sessions page.
func (a *Admin) handleUserSessions(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Active Sessions", "View and manage active sessions")
}

// handleLogs renders logs page.
func (a *Admin) handleLogs(w http.ResponseWriter, r *http.Request) {
	a.renderPage(w, "Logs", "View server logs")
}

// handleAPIStats returns admin API stats.
func (a *Admin) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// handleAPIHealth returns admin API health.
func (a *Admin) handleAPIHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"healthy"}`))
}

// renderPage renders a simple admin page.
func (a *Admin) renderPage(w http.ResponseWriter, title, description string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + ` - casman Admin</title>
    <style>
        body { font-family: system-ui, sans-serif; margin: 0; padding: 20px; background: #1a1b26; color: #c0caf5; }
        h1 { color: #7aa2f7; }
        .breadcrumb { margin-bottom: 20px; }
        .breadcrumb a { color: #7dcfff; text-decoration: none; }
        .card { background: #24283b; padding: 20px; border-radius: 8px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="breadcrumb"><a href="../">Admin</a> / ` + title + `</div>
    <h1>` + title + `</h1>
    <div class="card">
        <p>` + description + `</p>
        <p><em>Coming soon</em></p>
    </div>
</body>
</html>`))
}
