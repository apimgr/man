// Package auth provides authentication and session management.
// Per AI.md PART 10, PART 11, PART 17 specifications.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionNotFound    = errors.New("session not found")
	ErrAdminExists        = errors.New("admin already exists")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrInvalidToken       = errors.New("invalid token")
)

// Argon2id parameters per AI.md spec
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
)

// Session configuration
const (
	sessionCookieName = "casman_admin_session"
	sessionDuration   = 24 * time.Hour
	maxFailedAttempts = 5
	lockoutDuration   = 15 * time.Minute
)

// Admin represents a server admin account.
// Per AI.md: Server Admin manages the app (NOT a privileged OS user).
type Admin struct {
	ID             int64
	Username       string
	Email          string
	Role           string // superadmin, admin, readonly
	Enabled        bool
	Source         string // local, oidc:{provider}, ldap
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastLogin      *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
}

// Session represents an admin login session.
type Session struct {
	ID         string
	AdminID    int64
	IPAddress  string
	UserAgent  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastActive time.Time
}

// Store handles admin authentication database operations.
type Store struct {
	db *sql.DB
}

// NewStore creates a new auth store.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

// initSchema creates the authentication tables.
func (s *Store) initSchema() error {
	schema := `
	-- Admin accounts per AI.md PART 10
	CREATE TABLE IF NOT EXISTS admins (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		username        TEXT NOT NULL UNIQUE,
		password        TEXT NOT NULL,
		email           TEXT,
		role            TEXT NOT NULL DEFAULT 'admin',
		enabled         INTEGER NOT NULL DEFAULT 1,
		api_token_hash  TEXT,
		created_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		updated_at      INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		last_login      INTEGER,
		failed_attempts INTEGER NOT NULL DEFAULT 0,
		locked_until    INTEGER,
		source          TEXT NOT NULL DEFAULT 'local',
		external_id     TEXT,
		groups          TEXT,
		last_sync       INTEGER
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_admins_username ON admins(username);

	-- Admin sessions per AI.md PART 10
	CREATE TABLE IF NOT EXISTS admin_sessions (
		id          TEXT PRIMARY KEY,
		admin_id    INTEGER NOT NULL,
		ip_address  TEXT NOT NULL,
		user_agent  TEXT,
		created_at  INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		expires_at  INTEGER NOT NULL,
		last_active INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_admin_sessions_admin ON admin_sessions(admin_id);
	CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires ON admin_sessions(expires_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateAdmin creates a new admin account.
// Per AI.md: Primary Admin is first admin, cannot be deleted.
func (s *Store) CreateAdmin(username, password, email, role string) (*Admin, error) {
	// Check if admin exists
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM admins WHERE username = ?", username).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrAdminExists
	}

	// Hash password with Argon2id
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// First admin becomes superadmin
	var existingCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&existingCount); err != nil {
		return nil, err
	}
	if existingCount == 0 {
		role = "superadmin"
	}

	// Insert admin
	result, err := s.db.Exec(`
		INSERT INTO admins (username, password, email, role)
		VALUES (?, ?, ?, ?)
	`, username, hash, email, role)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetAdminByID(id)
}

// GetAdminByID retrieves an admin by ID.
func (s *Store) GetAdminByID(id int64) (*Admin, error) {
	return s.scanAdmin(s.db.QueryRow(`
		SELECT id, username, email, role, enabled, source,
		       created_at, updated_at, last_login, failed_attempts, locked_until
		FROM admins WHERE id = ?
	`, id))
}

// GetAdminByUsername retrieves an admin by username.
func (s *Store) GetAdminByUsername(username string) (*Admin, error) {
	return s.scanAdmin(s.db.QueryRow(`
		SELECT id, username, email, role, enabled, source,
		       created_at, updated_at, last_login, failed_attempts, locked_until
		FROM admins WHERE username = ?
	`, username))
}

func (s *Store) scanAdmin(row *sql.Row) (*Admin, error) {
	var admin Admin
	var createdAt, updatedAt int64
	var lastLogin, lockedUntil sql.NullInt64
	var email sql.NullString

	err := row.Scan(
		&admin.ID, &admin.Username, &email, &admin.Role, &admin.Enabled, &admin.Source,
		&createdAt, &updatedAt, &lastLogin, &admin.FailedAttempts, &lockedUntil,
	)
	if err == sql.ErrNoRows {
		return nil, ErrAdminNotFound
	}
	if err != nil {
		return nil, err
	}

	admin.CreatedAt = time.Unix(createdAt, 0)
	admin.UpdatedAt = time.Unix(updatedAt, 0)
	if email.Valid {
		admin.Email = email.String
	}
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		admin.LastLogin = &t
	}
	if lockedUntil.Valid {
		t := time.Unix(lockedUntil.Int64, 0)
		admin.LockedUntil = &t
	}

	return &admin, nil
}

// Authenticate validates credentials and returns an admin.
// Per AI.md: Rate limiting, lockout after failed attempts.
func (s *Store) Authenticate(username, password string) (*Admin, error) {
	// Get admin with password hash
	var admin Admin
	var passwordHash string
	var createdAt, updatedAt int64
	var lastLogin, lockedUntil sql.NullInt64
	var email sql.NullString

	err := s.db.QueryRow(`
		SELECT id, username, password, email, role, enabled, source,
		       created_at, updated_at, last_login, failed_attempts, locked_until
		FROM admins WHERE username = ?
	`, username).Scan(
		&admin.ID, &admin.Username, &passwordHash, &email, &admin.Role,
		&admin.Enabled, &admin.Source, &createdAt, &updatedAt,
		&lastLogin, &admin.FailedAttempts, &lockedUntil,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	admin.CreatedAt = time.Unix(createdAt, 0)
	admin.UpdatedAt = time.Unix(updatedAt, 0)
	if email.Valid {
		admin.Email = email.String
	}
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		admin.LastLogin = &t
	}
	if lockedUntil.Valid {
		t := time.Unix(lockedUntil.Int64, 0)
		admin.LockedUntil = &t
	}

	// Check if account is disabled
	if !admin.Enabled {
		return nil, ErrAccountDisabled
	}

	// Check if account is locked
	if admin.LockedUntil != nil && time.Now().Before(*admin.LockedUntil) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if !VerifyPassword(password, passwordHash) {
		// Increment failed attempts
		s.incrementFailedAttempts(admin.ID)
		return nil, ErrInvalidCredentials
	}

	// Success - reset failed attempts and update last login
	s.db.Exec(`
		UPDATE admins SET
			failed_attempts = 0,
			locked_until = NULL,
			last_login = strftime('%s', 'now'),
			updated_at = strftime('%s', 'now')
		WHERE id = ?
	`, admin.ID)

	return &admin, nil
}

func (s *Store) incrementFailedAttempts(adminID int64) {
	// Get current count
	var attempts int
	s.db.QueryRow("SELECT failed_attempts FROM admins WHERE id = ?", adminID).Scan(&attempts)
	attempts++

	// Lock if too many failures
	if attempts >= maxFailedAttempts {
		lockUntil := time.Now().Add(lockoutDuration).Unix()
		s.db.Exec(`
			UPDATE admins SET
				failed_attempts = ?,
				locked_until = ?,
				updated_at = strftime('%s', 'now')
			WHERE id = ?
		`, attempts, lockUntil, adminID)
	} else {
		s.db.Exec(`
			UPDATE admins SET
				failed_attempts = ?,
				updated_at = strftime('%s', 'now')
			WHERE id = ?
		`, attempts, adminID)
	}
}

// CreateSession creates a new admin session.
func (s *Store) CreateSession(adminID int64, ipAddress, userAgent string) (*Session, error) {
	// Generate secure session ID
	sessionID, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(sessionDuration)

	_, err = s.db.Exec(`
		INSERT INTO admin_sessions (id, admin_id, ip_address, user_agent, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, adminID, ipAddress, userAgent, expiresAt.Unix())
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        sessionID,
		AdminID:   adminID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
	}, nil
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(sessionID string) (*Session, error) {
	var session Session
	var createdAt, expiresAt, lastActive int64

	err := s.db.QueryRow(`
		SELECT id, admin_id, ip_address, user_agent, created_at, expires_at, last_active
		FROM admin_sessions WHERE id = ?
	`, sessionID).Scan(
		&session.ID, &session.AdminID, &session.IPAddress, &session.UserAgent,
		&createdAt, &expiresAt, &lastActive,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	session.CreatedAt = time.Unix(createdAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)
	session.LastActive = time.Unix(lastActive, 0)

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}

	// Update last active
	s.db.Exec("UPDATE admin_sessions SET last_active = strftime('%s', 'now') WHERE id = ?", sessionID)

	return &session, nil
}

// DeleteSession removes a session.
func (s *Store) DeleteSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM admin_sessions WHERE id = ?", sessionID)
	return err
}

// DeleteAdminSessions removes all sessions for an admin.
func (s *Store) DeleteAdminSessions(adminID int64) error {
	_, err := s.db.Exec("DELETE FROM admin_sessions WHERE admin_id = ?", adminID)
	return err
}

// CleanupExpiredSessions removes expired sessions.
func (s *Store) CleanupExpiredSessions() (int64, error) {
	result, err := s.db.Exec("DELETE FROM admin_sessions WHERE expires_at < strftime('%s', 'now')")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// AdminCount returns the number of admin accounts.
func (s *Store) AdminCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	return count, err
}

// ListAdminEmails returns the email addresses of every enabled admin with a
// non-empty email field. Used by the notify package (PART 18) to fan out
// server notifications such as backup status and SSL renewal alerts.
func (s *Store) ListAdminEmails() ([]string, error) {
	rows, err := s.db.Query(`SELECT email FROM admins WHERE enabled = 1 AND email IS NOT NULL AND email != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var emails []string
	for rows.Next() {
		var e sql.NullString
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		if e.Valid && e.String != "" {
			emails = append(emails, e.String)
		}
	}
	return emails, rows.Err()
}

// UpdatePassword updates an admin's password.
func (s *Store) UpdatePassword(adminID int64, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		UPDATE admins SET
			password = ?,
			updated_at = strftime('%s', 'now')
		WHERE id = ?
	`, hash, adminID)
	return err
}

// HashPassword hashes a password using Argon2id.
// Per AI.md: NEVER use bcrypt - use Argon2id.
func HashPassword(password string) (string, error) {
	// Generate salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Generate hash
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode as: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword verifies a password against an Argon2id hash.
func VerifyPassword(password, encodedHash string) bool {
	// Parse the hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var version int
	fmt.Sscanf(parts[2], "v=%d", &version)
	if version != argon2.Version {
		return false
	}

	var memory, time uint32
	var threads uint8
	fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

	// Constant-time comparison
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateAPIToken generates an API token with proper prefix.
// Per AI.md: Format is {prefix}_{random_32_alphanumeric}
func GenerateAPIToken(prefix string) (string, string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 24) // Will be 32 chars in base64
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	// Convert to alphanumeric (base62-like encoding)
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 32)
	for i := range result {
		result[i] = charset[int(bytes[i%len(bytes)])%len(charset)]
	}

	token := prefix + string(result)

	// Hash for storage
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])

	return token, hashStr, nil
}

// VerifyAPIToken verifies an API token against a stored hash.
func VerifyAPIToken(token, storedHash string) bool {
	hash := sha256.Sum256([]byte(token))
	hashStr := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(storedHash)) == 1
}

// Middleware provides HTTP middleware for admin authentication.
type Middleware struct {
	store      *Store
	debugMode  bool
	setupToken string
}

// NewMiddleware creates authentication middleware.
func NewMiddleware(store *Store, debugMode bool, setupToken string) *Middleware {
	return &Middleware{
		store:      store,
		debugMode:  debugMode,
		setupToken: setupToken,
	}
}

// RequireAuth is middleware that requires admin authentication.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// In debug mode, bypass auth for development
		if m.debugMode {
			next.ServeHTTP(w, r)
			return
		}

		// Check session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil {
			session, err := m.store.GetSession(cookie.Value)
			if err == nil {
				// Valid session - get admin
				admin, err := m.store.GetAdminByID(session.AdminID)
				if err == nil && admin.Enabled {
					// Store admin in context
					ctx := SetAdminContext(r.Context(), admin)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// Check setup token (for initial setup)
		if m.setupToken != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if subtle.ConstantTimeCompare([]byte(token), []byte(m.setupToken)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
		}

		// Unauthorized
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// SetSessionCookie sets the session cookie.
// The request is used to auto-detect whether to set the Secure flag.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, session *Session) {
	// Auto-detect secure flag per AI.md PART 11
	// Secure=true for HTTPS, false for HTTP
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

// ClearSessionCookie removes the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
