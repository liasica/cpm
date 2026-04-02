package cookie

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

var (
	dllCrypt32  = syscall.NewLazyDLL("Crypt32.dll")
	dllKernel32 = syscall.NewLazyDLL("Kernel32.dll")
	procDecrypt = dllCrypt32.NewProc("CryptUnprotectData")
	procFree    = dllKernel32.NewProc("LocalFree")
)

type dataBlob struct {
	Size uint32
	Data *byte
}

var (
	aesKeyOnce sync.Once
	aesKey     []byte
	aesKeyErr  error
)

func decrypt(encrypted []byte) (string, error) {
	if len(encrypted) <= 3 {
		return "", fmt.Errorf("encrypted value too short")
	}

	prefix := string(encrypted[:3])
	if prefix != "v10" && prefix != "v20" {
		return string(encrypted), nil
	}

	// Windows Chromium v80+ 使用 AES-256-GCM
	key, err := getAESKey()
	if err != nil {
		return "", err
	}

	ciphertext := encrypted[3:]
	if len(ciphertext) < 12+16 {
		return "", fmt.Errorf("ciphertext too short")
	}

	// 前 12 字节是 nonce，最后 16 字节是 auth tag（包含在 ciphertext 中）
	nonce := ciphertext[:12]
	ciphertext = ciphertext[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("AES-GCM decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// getAESKey 从 Local State 文件获取 DPAPI 加密的 AES 密钥
func getAESKey() ([]byte, error) {
	aesKeyOnce.Do(func() {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			aesKeyErr = fmt.Errorf("APPDATA not set")
			return
		}

		localStatePath := filepath.Join(appData, "Claude", "Local State")
		data, err := os.ReadFile(localStatePath)
		if err != nil {
			aesKeyErr = fmt.Errorf("failed to read Local State: %w", err)
			return
		}

		var state struct {
			OSCrypt struct {
				EncryptedKey string `json:"encrypted_key"`
			} `json:"os_crypt"`
		}
		if err = json.Unmarshal(data, &state); err != nil {
			aesKeyErr = fmt.Errorf("failed to parse Local State: %w", err)
			return
		}

		keyBytes, err := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
		if err != nil {
			aesKeyErr = fmt.Errorf("failed to decode encrypted key: %w", err)
			return
		}

		// 去掉 "DPAPI" 前缀 (5 bytes)
		if len(keyBytes) < 5 || string(keyBytes[:5]) != "DPAPI" {
			aesKeyErr = fmt.Errorf("unexpected key format")
			return
		}

		aesKey, aesKeyErr = dpApiDecrypt(keyBytes[5:])
	})

	return aesKey, aesKeyErr
}

func dpApiDecrypt(data []byte) ([]byte, error) {
	in := dataBlob{
		Size: uint32(len(data)),
		Data: &data[0],
	}
	var out dataBlob

	r, _, err := procDecrypt.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	defer procFree.Call(uintptr(unsafe.Pointer(out.Data)))

	result := make([]byte, out.Size)
	copy(result, unsafe.Slice(out.Data, out.Size))

	return result, nil
}
