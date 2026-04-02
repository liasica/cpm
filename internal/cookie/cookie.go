package cookie

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// ReadFromProfile 从指定 profile 目录的 Cookies 数据库读取 claude.ai 的 cookies
func ReadFromProfile(profileDir string) ([]*http.Cookie, error) {
	dbPath := filepath.Join(profileDir, "Cookies")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("cookies database not found: %s", dbPath)
	}

	return readFromDB(dbPath)
}

// ReadFromClaudeDir 从 Claude 应用数据目录读取 cookies
func ReadFromClaudeDir(claudeDir string) ([]*http.Cookie, error) {
	return readFromDB(filepath.Join(claudeDir, "Cookies"))
}

func readFromDB(dbPath string) ([]*http.Cookie, error) {
	// 使用临时副本避免锁冲突
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

		// 解密加密的 cookie 值
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

		// 过滤含有非法 HTTP header 字符的 cookie
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

// isValidCookieValue 检查 cookie 值是否只包含合法的 HTTP header 字符
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

	// 复制主数据库文件
	data, err := os.ReadFile(src)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if err = os.WriteFile(tmpDB, data, 0600); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	// 复制 WAL 和 SHM 文件（Chromium 使用 WAL 模式，数据可能在 WAL 中）
	for _, suffix := range []string{"-wal", "-shm"} {
		if walData, rdErr := os.ReadFile(src + suffix); rdErr == nil {
			_ = os.WriteFile(tmpDB+suffix, walData, 0600)
		}
	}

	return tmpDB, nil
}
