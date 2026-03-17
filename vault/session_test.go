package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateSessionStoresSecretOutsideSessionFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var storedProject string
	var storedKey []byte

	origPut := putSessionSecret
	origGet := getSessionSecret
	origDelete := deleteSessionSecret
	putSessionSecret = func(project string, key []byte) error {
		storedProject = project
		storedKey = append([]byte(nil), key...)
		return nil
	}
	getSessionSecret = func(project string) ([]byte, error) {
		if project != storedProject {
			t.Fatalf("unexpected project lookup: %s", project)
		}
		return append([]byte(nil), storedKey...), nil
	}
	deleteSessionSecret = func(project string) error { return nil }
	t.Cleanup(func() {
		putSessionSecret = origPut
		getSessionSecret = origGet
		deleteSessionSecret = origDelete
	})

	expires, err := CreateSession("demo", map[string]string{"DB_PASSWORD": "secret"}, time.Hour)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if expires.IsZero() {
		t.Fatal("expected non-zero expiry")
	}

	path := filepath.Join(os.Getenv("HOME"), ".envault", "sessions", safeProjectID("demo")+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var sf map[string]any
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if _, ok := sf["key"]; ok {
		t.Fatal("session file must not contain the encryption key")
	}

	vars, err := LoadSession("demo")
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if got := vars["DB_PASSWORD"]; got != "secret" {
		t.Fatalf("unexpected secret value: %q", got)
	}
}
