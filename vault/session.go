package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const SessionLocalFile = ".envault.session" // kept for .gitignore compat, no longer used

// sessionFile is stored at ~/.envault/sessions/<project>.json
type sessionFile struct {
	Project    string    `json:"project"`
	Expires    time.Time `json:"expires"`
	Nonce      []byte    `json:"nonce"`
	Ciphertext []byte    `json:"ciphertext"` // AES-GCM encrypted JSON vars
}

var (
	putSessionSecret    = storeSessionSecret
	getSessionSecret    = loadSessionSecret
	deleteSessionSecret = removeSessionSecret
)

// safeProjectID returns a collision-resistant, filename-safe identifier for
// the given project name. Using a SHA-256 hash prevents path traversal
// collisions (e.g. "foo" and "../foo" must not share the same session file).
func safeProjectID(project string) string {
	h := sha256.Sum256([]byte(project))
	return hex.EncodeToString(h[:])
}

func sessionPath(project string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".envault", "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, safeProjectID(project)+".json"), nil
}

// CreateSession encrypts the project vars and stores them in ~/.envault/sessions/<project>.json.
// No files are written to the project directory — the session is found by project name alone.
func CreateSession(project string, vars map[string]string, ttl time.Duration) (time.Time, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return time.Time{}, fmt.Errorf("generating session key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return time.Time{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return time.Time{}, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return time.Time{}, err
	}

	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return time.Time{}, err
	}

	expires := time.Now().Add(ttl).UTC().Truncate(time.Second)

	sf := sessionFile{
		Project:    project,
		Expires:    expires,
		Nonce:      nonce,
		Ciphertext: gcm.Seal(nil, nonce, varsJSON, nil),
	}

	data, err := json.Marshal(sf)
	if err != nil {
		return time.Time{}, err
	}

	path, err := sessionPath(project)
	if err != nil {
		return time.Time{}, err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return time.Time{}, fmt.Errorf("writing session: %w", err)
	}
	if err := putSessionSecret(project, key); err != nil {
		_ = os.Remove(path)
		return time.Time{}, fmt.Errorf("storing session secret: %w", err)
	}

	return expires, nil
}

// LoadSession reads the session for the given project and returns decrypted vars.
// Returns nil, nil if no valid session exists — caller should fall back to password.
func LoadSession(project string) (map[string]string, error) {
	path, err := sessionPath(project)
	if err != nil {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // no session
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, nil
	}

	if time.Now().After(sf.Expires) {
		os.Remove(path)
		deleteSessionSecret(project)
		return nil, nil
	}

	key, err := getSessionSecret(project)
	if err != nil {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	varsJSON, err := gcm.Open(nil, sf.Nonce, sf.Ciphertext, nil)
	if err != nil {
		return nil, nil
	}

	var vars map[string]string
	if err := json.Unmarshal(varsJSON, &vars); err != nil {
		return nil, err
	}

	return vars, nil
}

// RefreshSession updates the vars in an existing session while keeping the original expiry.
// If no active session exists, it does nothing.
func RefreshSession(project string, vars map[string]string) error {
	expires, ok := SessionInfo(project)
	if !ok {
		return nil // no active session — nothing to refresh
	}
	ttl := time.Until(expires)
	if ttl <= 0 {
		return nil
	}
	_, err := CreateSession(project, vars, ttl)
	return err
}

// DeleteSession removes the session for the given project.
func DeleteSession(project string) {
	if path, err := sessionPath(project); err == nil {
		os.Remove(path)
	}
	deleteSessionSecret(project)
}

// SessionInfo returns metadata about the current session without decrypting vars.
func SessionInfo(project string) (expires time.Time, ok bool) {
	path, err := sessionPath(project)
	if err != nil {
		return time.Time{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, false
	}

	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return time.Time{}, false
	}

	if time.Now().After(sf.Expires) {
		os.Remove(path)
		deleteSessionSecret(project)
		return time.Time{}, false
	}

	return sf.Expires, true
}
