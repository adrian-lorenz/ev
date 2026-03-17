//go:build !darwin

package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// On non-macOS platforms the session key is stored in a sibling file
// ~/.envault/sessions/<project>.key with mode 0600.
// The key itself is wrapped with AES-256-GCM using a machine-derived secret,
// so it cannot be used on a different machine even if the file is copied.

func sessionKeyPath(project string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".envault", "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, safeProjectID(project)+".key"), nil
}

// machineSecret derives a stable, machine-specific 32-byte key used to wrap
// session keys at rest. It is NOT a cryptographic secret — its purpose is to
// bind stored session keys to the current machine so they cannot be trivially
// reused if the ~/.envault directory is copied elsewhere.
func machineSecret() []byte {
	// Linux: /etc/machine-id is a stable, unique machine identifier.
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		id := strings.TrimSpace(string(data))
		if len(id) >= 8 {
			h := sha256.Sum256([]byte("envault:session-key-wrap:" + id))
			return h[:]
		}
	}
	// Fallback: use hostname.
	if host, err := os.Hostname(); err == nil && len(host) >= 2 {
		h := sha256.Sum256([]byte("envault:session-key-wrap:" + host))
		return h[:]
	}
	// Last resort: no machine binding (same security as plain file).
	h := sha256.Sum256([]byte("envault:session-key-wrap:none"))
	return h[:]
}

func wrapKey(plainKey []byte) ([]byte, error) {
	wk := machineSecret()
	block, err := aes.NewCipher(wk)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, plainKey, nil)
	return []byte(hex.EncodeToString(sealed)), nil
}

func unwrapKey(data []byte) ([]byte, error) {
	sealed, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	wk := machineSecret()
	block, err := aes.NewCipher(wk)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return nil, fmt.Errorf("invalid key file")
	}
	return gcm.Open(nil, sealed[:nonceSize], sealed[nonceSize:], nil)
}

func storeSessionSecret(project string, key []byte) error {
	path, err := sessionKeyPath(project)
	if err != nil {
		return fmt.Errorf("session key path: %w", err)
	}
	wrapped, err := wrapKey(key)
	if err != nil {
		return fmt.Errorf("wrapping session key: %w", err)
	}
	return os.WriteFile(path, wrapped, 0600)
}

func loadSessionSecret(project string) ([]byte, error) {
	path, err := sessionKeyPath(project)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Try new wrapped format first.
	key, err := unwrapKey(raw)
	if err == nil {
		return key, nil
	}
	// Fall back to legacy plain-hex format and migrate transparently.
	key, err = hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("reading session key: unrecognised format")
	}
	_ = storeSessionSecret(project, key) // migrate to wrapped format
	return key, nil
}

func removeSessionSecret(project string) error {
	path, err := sessionKeyPath(project)
	if err != nil {
		return nil
	}
	_ = os.Remove(path)
	return nil
}
