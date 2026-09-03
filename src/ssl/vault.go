// Vault stores DNS provider credentials encrypted with the server master key.
// Backing store is the users.db `ssl_credentials` table; values are JSON-
// encoded field maps sealed with secret.Vault. See AI.md PART 15.

package ssl

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/casapps/casman/src/secret"
)

// ErrCredentialsNotFound is returned when no credentials exist for a provider.
var ErrCredentialsNotFound = errors.New("ssl: credentials not found")

// Vault persists per-provider credential bags.
type Vault struct {
	db     *sql.DB
	sealer *secret.Vault
}

// NewVault initializes the ssl_credentials table and returns a ready-to-use
// vault. Both arguments are required.
func NewVault(db *sql.DB, sealer *secret.Vault) (*Vault, error) {
	if db == nil {
		return nil, errors.New("ssl: nil db")
	}
	if sealer == nil {
		return nil, errors.New("ssl: nil sealer")
	}
	v := &Vault{db: db, sealer: sealer}
	if err := v.initSchema(); err != nil {
		return nil, err
	}
	return v, nil
}

func (v *Vault) initSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS ssl_credentials (
		provider   TEXT PRIMARY KEY,
		ciphertext BLOB NOT NULL,
		updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	);`
	_, err := v.db.Exec(schema)
	return err
}

// Save encrypts and upserts the credential bag for the named provider.
func (v *Vault) Save(provider string, fields map[string]string) error {
	if provider == "" {
		return errors.New("ssl: empty provider")
	}
	if fields == nil {
		fields = map[string]string{}
	}
	plain, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	sealed, err := v.sealer.Encrypt(plain)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}
	_, err = v.db.Exec(`
		INSERT INTO ssl_credentials (provider, ciphertext, updated_at)
		VALUES (?, ?, strftime('%s','now'))
		ON CONFLICT(provider) DO UPDATE SET
			ciphertext = excluded.ciphertext,
			updated_at = excluded.updated_at
	`, provider, sealed)
	return err
}

// Load decrypts and returns the credential bag for the named provider.
func (v *Vault) Load(provider string) (map[string]string, error) {
	var sealed []byte
	err := v.db.QueryRow(
		`SELECT ciphertext FROM ssl_credentials WHERE provider = ?`,
		provider,
	).Scan(&sealed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCredentialsNotFound
	}
	if err != nil {
		return nil, err
	}
	plain, err := v.sealer.Decrypt(sealed)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}
	return out, nil
}

// List returns the providers that have stored credentials, ordered by name.
func (v *Vault) List() ([]string, error) {
	rows, err := v.db.Query(`SELECT provider FROM ssl_credentials ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// Delete removes the credentials for the named provider; missing rows are not
// an error.
func (v *Vault) Delete(provider string) error {
	_, err := v.db.Exec(`DELETE FROM ssl_credentials WHERE provider = ?`, provider)
	return err
}
