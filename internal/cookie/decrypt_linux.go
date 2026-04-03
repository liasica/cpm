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

	prefix := string(encrypted[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(encrypted), nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext := encrypted[3:]
	if len(ciphertext) < aes.BlockSize || len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid ciphertext length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}

	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)

	if padding := int(plaintext[len(plaintext)-1]); padding > 0 && padding <= aes.BlockSize {
		plaintext = plaintext[:len(plaintext)-padding]
	}

	return string(plaintext), nil
}

func getEncryptionKey() ([]byte, error) {
	encKeyOnce.Do(func() {
		// Try GNOME Keyring first
		out, err := exec.Command("secret-tool", "lookup", "application", "claude").Output()
		if err == nil && len(out) > 0 {
			password := strings.TrimSpace(string(out))
			encKey = pbkdf2.Key([]byte(password), []byte("saltysalt"), 1, 16, sha1.New)
			return
		}

		// Fall back to Chromium default password
		encKey = pbkdf2.Key([]byte("peanuts"), []byte("saltysalt"), 1, 16, sha1.New)
	})

	return encKey, encKeyErr
}
