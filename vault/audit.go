package vault

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HashName returns a short, non-reversible identifier for a project or key name
// so that audit log entries are useful for correlation without exposing plaintext names.
func HashName(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:6]) // 12 hex chars, 48-bit prefix
}

// Audit appends a structured entry to ~/.envault/audit.log.
// Failures are silently ignored so that audit errors never break normal operations.
func Audit(action, detail string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".envault")
	_ = os.MkdirAll(dir, 0700)
	logPath := filepath.Join(dir, "audit.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	fmt.Fprintf(f, "%s\t%s\t%s\n", ts, action, detail)
}
