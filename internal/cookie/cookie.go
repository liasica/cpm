package cookie

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// ReadFromProfile reads claude.ai cookies from the Cookies database in a profile directory
func ReadFromProfile(profileDir string) ([]*http.Cookie, error) {
	dbPath := filepath.Join(profileDir, "Cookies")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("cookies database not found: %s", dbPath)
	}

	return readFromDB(dbPath)
}

// ReadFromClaudeDir reads cookies from the Claude application data directory
func ReadFromClaudeDir(claudeDir string) ([]*http.Cookie, error) {
	return readFromDB(filepath.Join(claudeDir, "Cookies"))
}

func readFromDB(dbPath string) ([]*http.Cookie, error) {
	// Copy to temp file to avoid lock conflicts
	tmp, err := copyToTemp(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to copy cookies database: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(tmp))

	db, err := sql.Open("sqlite", tmp)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookies database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT name, encrypted_value, value, host_key, path, is_secure, is_httponly
		 FROM cookies WHERE host_key IN ('.claude.ai', 'claude.ai')`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []*http.Cookie
	for rows.Next() {
		var name, value, hostKey, path string
		var encValue []byte
		var isSecure, isHTTPOnly int

		if err = rows.Scan(&name, &encValue, &value, &hostKey, &path, &isSecure, &isHTTPOnly); err != nil {
			continue
		}

		// Decrypt encrypted cookie value
		if len(encValue) > 0 {
			var decrypted string
			decrypted, err = decrypt(encValue)
			if err != nil {
				continue
			}
			value = decrypted
		}

		if value == "" {
			continue
		}

		// Filter out cookies with invalid HTTP header characters
		if !isValidCookieValue(value) {
			continue
		}

		cookies = append(cookies, &http.Cookie{
			Name:     name,
			Value:    value,
			Domain:   hostKey,
			Path:     path,
			Secure:   isSecure == 1,
			HttpOnly: isHTTPOnly == 1,
		})
	}

	return cookies, nil
}

// isValidCookieValue checks that a cookie value only contains valid HTTP header characters
func isValidCookieValue(v string) bool {
	for _, c := range v {
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func copyToTemp(src string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "cpm-cookies-*")
	if err != nil {
		return "", err
	}

	dbName := filepath.Base(src)
	tmpDB := filepath.Join(tmpDir, dbName)

	// Copy main database file
	data, err := os.ReadFile(src)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err = os.WriteFile(tmpDB, data, 0600); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	// Copy WAL and SHM files if present (Chromium uses WAL mode)
	for _, suffix := range []string{"-wal", "-shm"} {
		if walData, rdErr := os.ReadFile(src + suffix); rdErr == nil {
			_ = os.WriteFile(tmpDB+suffix, walData, 0600)
		}
	}

	return tmpDB, nil
}
