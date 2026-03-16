package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
