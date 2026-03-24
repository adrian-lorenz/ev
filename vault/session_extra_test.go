package vault

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

// installMockKeychain replaces the keychain hooks with an in-memory store
// and restores the originals via t.Cleanup.
func installMockKeychain(t *testing.T) {
	t.Helper()
	var storedProject string
	var storedKey []byte

	origPut := putSessionSecret
	origGet := getSessionSecret
	origDel := deleteSessionSecret

	putSessionSecret = func(project string, key []byte) error {
		storedProject = project
		storedKey = append([]byte(nil), key...)
		return nil
	}
	getSessionSecret = func(project string) ([]byte, error) {
		if project != storedProject {
			return nil, errors.New("no key stored for project")
		}
		return append([]byte(nil), storedKey...), nil
	}
	deleteSessionSecret = func(project string) error {
		storedProject = ""
		storedKey = nil
		return nil
	}
	t.Cleanup(func() {
		putSessionSecret = origPut
		getSessionSecret = origGet
		deleteSessionSecret = origDel
	})
}

// --- safeProjectID ---

func TestSafeProjectID_Deterministic(t *testing.T) {
	a := safeProjectID("myproject")
	b := safeProjectID("myproject")
	if a != b {
		t.Error("safeProjectID should be deterministic")
	}
}

func TestSafeProjectID_DifferentInputs(t *testing.T) {
	a := safeProjectID("foo")
	b := safeProjectID("bar")
	if a == b {
		t.Error("different project names should produce different IDs")
	}
}

func TestSafeProjectID_PathTraversal(t *testing.T) {
	normal := safeProjectID("foo")
	traversal := safeProjectID("../foo")
	if normal == traversal {
		t.Error("'foo' and '../foo' must produce different IDs to prevent path traversal")
	}
}

func TestSafeProjectID_IsSafeForFilename(t *testing.T) {
	cases := []string{"foo/bar", "foo\x00bar", "foo bar", "..", "."}
	for _, name := range cases {
		id := safeProjectID(name)
		for _, ch := range id {
			if ch == '/' || ch == '\x00' || ch == ' ' {
				t.Errorf("safeProjectID(%q) contains unsafe char %q in %q", name, ch, id)
			}
		}
	}
}

// --- Expired session ---

func TestLoadSession_ExpiredReturnsNilAndDeletesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	// Write a session file with an expiry in the past
	path, err := sessionPath("expired-proj")
	if err != nil {
		t.Fatal(err)
	}
	sf := sessionFile{
		Project:    "expired-proj",
		Expires:    time.Now().Add(-time.Hour),
		Nonce:      make([]byte, 12),
		Ciphertext: []byte("garbage"),
	}
	data, _ := json.Marshal(sf)
	os.WriteFile(path, data, 0600)

	vars, err := LoadSession("expired-proj")
	if err != nil || vars != nil {
		t.Errorf("expired session should return nil,nil; got vars=%v, err=%v", vars, err)
	}

	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Error("expired session file should be deleted")
	}
}

func TestSessionInfo_ExpiredReturnsNotOk(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	path, _ := sessionPath("proj")
	sf := sessionFile{
		Project: "proj",
		Expires: time.Now().Add(-time.Minute),
	}
	data, _ := json.Marshal(sf)
	os.WriteFile(path, data, 0600)

	_, ok := SessionInfo("proj")
	if ok {
		t.Error("SessionInfo should return ok=false for expired session")
	}
}

func TestSessionInfo_NoSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	_, ok := SessionInfo("no-such-project")
	if ok {
		t.Error("SessionInfo should return ok=false when no session exists")
	}
}

// --- RefreshSession ---

func TestRefreshSession_NoSession_IsNoOp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	// No session file exists — RefreshSession should return nil and not create one
	if err := RefreshSession("noexist", map[string]string{"K": "v"}); err != nil {
		t.Fatalf("RefreshSession with no session: %v", err)
	}
	if _, ok := SessionInfo("noexist"); ok {
		t.Error("RefreshSession with no prior session should not create one")
	}
}

func TestRefreshSession_PreservesExpiry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	original, err := CreateSession("proj", map[string]string{"A": "1"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// Refresh with new vars
	if err := RefreshSession("proj", map[string]string{"A": "updated", "B": "new"}); err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}

	// Expiry should be very close to the original (within 2 seconds)
	refreshed, ok := SessionInfo("proj")
	if !ok {
		t.Fatal("session should still be active after refresh")
	}
	diff := refreshed.Sub(original)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expiry changed by %v after refresh; expected it to be preserved", diff)
	}

	// Updated vars should be accessible
	vars, _ := LoadSession("proj")
	if vars["A"] != "updated" || vars["B"] != "new" {
		t.Errorf("refreshed vars not updated: %v", vars)
	}
}

// --- DeleteSession ---

func TestDeleteSession_RemovesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	installMockKeychain(t)

	if _, err := CreateSession("proj", map[string]string{"K": "v"}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, ok := SessionInfo("proj"); !ok {
		t.Fatal("session should exist before delete")
	}

	DeleteSession("proj")

	if _, ok := SessionInfo("proj"); ok {
		t.Error("session should not exist after DeleteSession")
	}
}
