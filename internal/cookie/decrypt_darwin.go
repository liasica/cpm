package cookie

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/crypto/pbkdf2"
)

var (
	encKeyOnce sync.Once
	encKey     []byte
	encKeyErr  error
)

func decrypt(encrypted []byte) (string, error) {
	if len(encrypted) <= 3 {
		return "", fmt.Errorf("encrypted value too short")
	}

	// Return unencrypted values as-is
	prefix := string(encrypted[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(encrypted), nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext := encrypted[3:]

	// Newer Electron/Chromium format: 32-byte header (last 16 bytes = IV) + ciphertext
	if len(ciphertext) > 32 {
		ct := ciphertext[32:]
		if len(ct) >= aes.BlockSize && len(ct)%aes.BlockSize == 0 {
			iv := ciphertext[16:32]
			if result, decErr := decryptCBC(key, iv, ct); decErr == nil && isValidCookieValue(result) {
				return result, nil
			}
		}
	}

	// Legacy format: 16-byte space IV + ciphertext
	if len(ciphertext) >= aes.BlockSize && len(ciphertext)%aes.BlockSize == 0 {
		iv := make([]byte, aes.BlockSize)
		for i := range iv {
			iv[i] = ' '
		}
		return decryptCBC(key, iv, ciphertext)
	}

	return "", fmt.Errorf("invalid ciphertext length")
}

func decryptCBC(key, iv, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	// PKCS#7 unpadding
	if padding := int(plaintext[len(plaintext)-1]); padding > 0 && padding <= aes.BlockSize {
		plaintext = plaintext[:len(plaintext)-padding]
	}

	return string(plaintext), nil
}

func getEncryptionKey() ([]byte, error) {
	encKeyOnce.Do(func() {
		out, err := exec.Command("security", "find-generic-password", "-s", "Claude Safe Storage", "-w").Output()
		if err != nil {
			encKeyErr = fmt.Errorf("failed to read Keychain (Claude Safe Storage): %w", err)
			return
		}

		password := strings.TrimSpace(string(out))
		encKey = pbkdf2.Key([]byte(password), []byte("saltysalt"), 1003, 16, sha1.New)
	})

	return encKey, encKeyErr
}
