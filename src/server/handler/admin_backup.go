// Admin backup form + API per AI.md PART 22.
//
// The web form supports three actions on the admin panel:
//   - action=backup: trigger an immediate backup; password is optional
//     unless compliance mode requires encryption.
//   - action=restore: restore from a named backup file in the backup dir.
//   - action=delete: remove a named backup file (admin-only housekeeping).
//
// The matching API endpoints under /api/v1/{admin_path}/server/backup speak
// JSON. Both surfaces are protected by the existing admin auth middleware
// registered in src/server/server.go.

package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/casapps/casman/src/backup"
)

// BackupBackend is the subset of the server the handler needs. The server
// satisfies it by exposing its *backup.Manager.
type BackupBackend interface {
	BackupManager() *backup.Manager
}

var backupBackend BackupBackend

// SetBackupBackend wires the backup manager into the handler package. Called
// once from src/server/server.go during initialization.
func SetBackupBackend(b BackupBackend) { backupBackend = b }

// AdminBackup renders the GET form and the list of existing backup files.
// The previous placeholder handler at handler.go:1579 now delegates here so
// the route registration in server.go does not need to change.
func (h *Handlers) AdminBackup(w http.ResponseWriter, r *http.Request) {
	csrf := getCSRFToken(r)

	var listing strings.Builder
	if backupBackend == nil || backupBackend.BackupManager() == nil {
		listing.WriteString(`<p><em>Backup manager unavailable.</em></p>`)
	} else {
		backups, err := backupBackend.BackupManager().ListBackups()
		switch {
		case err != nil:
			fmt.Fprintf(&listing, `<p class="error">Error listing backups: %s</p>`, html.EscapeString(err.Error()))
		case len(backups) == 0:
			listing.WriteString(`<p><em>No backups yet.</em></p>`)
		default:
			listing.WriteString(`<table class="pages-table"><thead><tr><th>Name</th><th>Size</th><th>Created</th><th>Encrypted</th><th>Actions</th></tr></thead><tbody>`)
			for _, b := range backups {
				fmt.Fprintf(&listing, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>
<form method="POST" action="/admin/server/backup" style="display:inline" onsubmit="return confirm('Restore %s? This overwrites current state.');">
  <input type="hidden" name="_csrf" value="%s">
  <input type="hidden" name="action" value="restore">
  <input type="hidden" name="filename" value="%s">
  <input type="password" name="password" placeholder="encryption password (if any)">
  <button type="submit">Restore</button>
</form>
<form method="POST" action="/admin/server/backup" style="display:inline" onsubmit="return confirm('Delete %s?');">
  <input type="hidden" name="_csrf" value="%s">
  <input type="hidden" name="action" value="delete">
  <input type="hidden" name="filename" value="%s">
  <button type="submit">Delete</button>
</form>
</td></tr>`,
					html.EscapeString(b.Name),
					formatBytes(b.Size),
					b.CreatedAt.UTC().Format("2006-01-02 15:04:05Z"),
					yesNo(b.Encrypted),
					html.EscapeString(b.Name),
					html.EscapeString(csrf),
					html.EscapeString(b.Name),
					html.EscapeString(b.Name),
					html.EscapeString(csrf),
					html.EscapeString(b.Name),
				)
			}
			listing.WriteString(`</tbody></table>`)
		}
	}

	body := `<p>Create backups (server.yml, server.db, users.db, custom templates) and restore from previous backups. Per AI.md PART 22.</p>`
	body += `<form method="POST" action="/admin/server/backup" autocomplete="off">`
	body += fmt.Sprintf(`<input type="hidden" name="_csrf" value="%s">`, html.EscapeString(csrf))
	body += `<input type="hidden" name="action" value="backup">`
	body += `<p><label>Encryption password (optional, required if compliance mode is enabled): <input type="password" name="password"></label></p>`
	body += `<p><button type="submit">Create backup now</button></p>`
	body += `</form><h2>Existing backups</h2>` + listing.String()

	h.adminPage(w, "Backup & Restore", body)
}

// AdminBackupSave processes POST /admin/server/backup. Switches on the
// `action` form field — backup, restore, or delete — and redirects back to
// the form with no body (the caller refreshes to see the new state).
func (h *Handlers) AdminBackupSave(w http.ResponseWriter, r *http.Request) {
	if backupBackend == nil || backupBackend.BackupManager() == nil {
		http.Error(w, "backup backend not initialized", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	mgr := backupBackend.BackupManager()

	switch r.FormValue("action") {
	case "backup":
		password := r.FormValue("password")
		path, err := mgr.Backup(r.Context(), "", password, "admin")
		if err != nil {
			http.Error(w, "backup failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = mgr.ApplyRetention()
		w.Header().Set("X-Backup-Path", path)
	case "restore":
		filename := safeBackupFilename(r.FormValue("filename"))
		if filename == "" {
			http.Error(w, "missing or unsafe filename", http.StatusBadRequest)
			return
		}
		path, err := resolveBackupPath(mgr, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := mgr.Restore(r.Context(), path, r.FormValue("password")); err != nil {
			http.Error(w, "restore failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case "delete":
		filename := safeBackupFilename(r.FormValue("filename"))
		if filename == "" {
			http.Error(w, "missing or unsafe filename", http.StatusBadRequest)
			return
		}
		path, err := resolveBackupPath(mgr, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.Remove(path); err != nil {
			http.Error(w, "delete failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/server/backup", http.StatusSeeOther)
}

// APIAdminBackupList handles GET /api/v1/{admin_path}/server/backup.
func (h *Handlers) APIAdminBackupList(w http.ResponseWriter, r *http.Request) {
	if backupBackend == nil || backupBackend.BackupManager() == nil {
		h.jsonResponse(w, map[string]string{"error": "backup not available"}, http.StatusServiceUnavailable)
		return
	}
	backups, err := backupBackend.BackupManager().ListBackups()
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	h.jsonResponse(w, map[string]interface{}{"backups": backups}, http.StatusOK)
}

// APIAdminBackupCreate handles POST /api/v1/{admin_path}/server/backup per
// AI.md PART 22. JSON body: {"password": "..."}.
func (h *Handlers) APIAdminBackupCreate(w http.ResponseWriter, r *http.Request) {
	if backupBackend == nil || backupBackend.BackupManager() == nil {
		h.jsonResponse(w, map[string]string{"error": "backup not available"}, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			h.jsonResponse(w, map[string]string{"error": "bad JSON: " + err.Error()}, http.StatusBadRequest)
			return
		}
	}
	path, err := backupBackend.BackupManager().Backup(r.Context(), body.Filename, body.Password, "admin-api")
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	_ = backupBackend.BackupManager().ApplyRetention()
	h.jsonResponse(w, map[string]string{"path": path}, http.StatusCreated)
}

// APIAdminBackupRestore handles POST /api/v1/{admin_path}/server/backup/restore.
// JSON body: {"filename": "...", "password": "..."}.
func (h *Handlers) APIAdminBackupRestore(w http.ResponseWriter, r *http.Request) {
	if backupBackend == nil || backupBackend.BackupManager() == nil {
		h.jsonResponse(w, map[string]string{"error": "backup not available"}, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Filename string `json:"filename"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.jsonResponse(w, map[string]string{"error": "bad JSON: " + err.Error()}, http.StatusBadRequest)
		return
	}
	filename := safeBackupFilename(body.Filename)
	if filename == "" {
		h.jsonResponse(w, map[string]string{"error": "missing or unsafe filename"}, http.StatusBadRequest)
		return
	}
	path, err := resolveBackupPath(backupBackend.BackupManager(), filename)
	if err != nil {
		h.jsonResponse(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return
	}
	if err := backupBackend.BackupManager().Restore(r.Context(), path, body.Password); err != nil {
		h.jsonResponse(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	h.jsonResponse(w, map[string]string{"status": "restored", "path": path}, http.StatusOK)
}

// safeBackupFilename rejects empty values and any path-traversal attempts.
// Per AI.md "NEVER use regex for path validation": this uses path.Clean and
// an allowlist of expected prefixes/suffixes, never regex. Filenames are
// expected to look like casman_backup_YYYY-MM-DD.tar.gz[.enc] OR the
// incremental filenames produced by the backup task.
func safeBackupFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, `/\`) {
		return ""
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return ""
	}
	if !strings.HasPrefix(name, "casman") {
		return ""
	}
	if !(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tar.gz.enc")) {
		return ""
	}
	return name
}

// resolveBackupPath joins the validated filename with the backup manager's
// configured directory and re-checks the result is still inside that
// directory after filepath.Clean (defence-in-depth against TOCTOU).
func resolveBackupPath(mgr *backup.Manager, filename string) (string, error) {
	dir := mgr.Dir()
	if dir == "" {
		return "", fmt.Errorf("backup dir not configured")
	}
	full := filepath.Clean(filepath.Join(dir, filename))
	cleanDir := filepath.Clean(dir)
	if !strings.HasPrefix(full, cleanDir+string(filepath.Separator)) && full != cleanDir {
		return "", fmt.Errorf("path traversal blocked")
	}
	return full, nil
}

func formatBytes(n int64) string {
	const k = 1024
	switch {
	case n < k:
		return fmt.Sprintf("%d B", n)
	case n < k*k:
		return fmt.Sprintf("%.1f KB", float64(n)/k)
	case n < k*k*k:
		return fmt.Sprintf("%.1f MB", float64(n)/(k*k))
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/(k*k*k))
	}
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
