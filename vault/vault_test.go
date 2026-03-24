package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- ValidateName ---

func TestValidateName_Valid(t *testing.T) {
	cases := []string{
		"KEY", "DB_HOST", "my.key", "a-b", "A1",
		"UPPER_LOWER_123", strings.Repeat("x", 128),
	}
	for _, name := range cases {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", name, err)
		}
	}
}

func TestValidateName_Invalid(t *testing.T) {
	cases := []struct {
		name string
		desc string
	}{
		{"", "empty"},
		{strings.Repeat("x", 129), "129 chars"},
		{"KEY NAME", "space"},
		{"KEY=VAL", "equals sign"},
		{"KEY\nFOO", "newline"},
		{"KEY!FOO", "exclamation"},
		{"KEY$FOO", "dollar"},
		{"KEY/FOO", "slash"},
		{"KEY@FOO", "at sign"},
	}
	for _, tc := range cases {
		if err := ValidateName(tc.name); err == nil {
			t.Errorf("ValidateName(%q) [%s]: expected error, got nil", tc.name, tc.desc)
		}
	}
}

func TestValidateName_BoundaryLength(t *testing.T) {
	if err := ValidateName(strings.Repeat("a", 128)); err != nil {
		t.Errorf("128-char name should be valid: %v", err)
	}
	if err := ValidateName(strings.Repeat("a", 129)); err == nil {
		t.Error("129-char name should be invalid")
	}
}

// --- Open ---

func TestOpen_NonExistentFile(t *testing.T) {
	v, err := Open(filepath.Join(t.TempDir(), "vault.json"), "anypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v == nil {
		t.Fatal("expected non-nil vault")
	}
	if projs := v.ListProjects(); len(projs) != 0 {
		t.Fatalf("expected empty vault, got projects: %v", projs)
	}
}

func TestOpen_WrongPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	v, _ := Open(path, "correctpassword")
	v.Set("p", "K", "v")
	if err := v.Save(path, "correctpassword"); err != nil {
		t.Fatal(err)
	}

	_, err := Open(path, "wrongpassword")
	if err == nil {
		t.Fatal("expected error with wrong password, got nil")
	}
	if !strings.Contains(err.Error(), "invalid password") {
		t.Errorf("error should mention 'invalid password', got: %v", err)
	}
}

func TestOpen_CorruptedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	os.WriteFile(path, []byte("not valid json {{{"), 0600)

	_, err := Open(path, "anypassword")
	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}
}

func TestOpen_WrongCiphertext(t *testing.T) {
	// Save a valid vault, then flip bytes in the ciphertext so GCM auth fails.
	path := filepath.Join(t.TempDir(), "vault.json")
	v, _ := Open(path, "pw")
	v.Set("p", "K", "val")
	if err := v.Save(path, "pw"); err != nil {
		t.Fatal(err)
	}

	// Read and corrupt one byte of the ciphertext in the raw JSON.
	data, _ := os.ReadFile(path)
	// Flip a byte somewhere in the middle of the file.
	mid := len(data) / 2
	data[mid] ^= 0xFF
	os.WriteFile(path, data, 0600)

	_, err := Open(path, "pw")
	if err == nil {
		t.Fatal("expected error for corrupted ciphertext, got nil")
	}
}

// --- Save + Open roundtrip ---

func TestSaveAndOpen_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	const password = "testpassword"

	v1, _ := Open(path, password)
	v1.Set("myproject", "DB_HOST", "localhost")
	v1.Set("myproject", "API_KEY", "secret123")
	v1.Set("myproject", "EMPTY_VAL", "")
	if err := v1.Save(path, password); err != nil {
		t.Fatal(err)
	}

	v2, err := Open(path, password)
	if err != nil {
		t.Fatalf("Open after Save: %v", err)
	}
	cases := map[string]string{
		"DB_HOST":   "localhost",
		"API_KEY":   "secret123",
		"EMPTY_VAL": "",
	}
	for k, want := range cases {
		got, ok := v2.Get("myproject", k)
		if !ok {
			t.Errorf("key %q not found after roundtrip", k)
			continue
		}
		if got != want {
			t.Errorf("key %q: got %q, want %q", k, got, want)
		}
	}
}

func TestSaveAndOpen_SpecialCharValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")

	v1, _ := Open(path, "pw")
	special := "spaces and\nnewlines\ttabs\"quotes'apostrophes\\backslash"
	v1.Set("p", "SPECIAL", special)
	v1.Save(path, "pw")

	v2, _ := Open(path, "pw")
	got, ok := v2.Get("p", "SPECIAL")
	if !ok || got != special {
		t.Errorf("got %q, want %q", got, special)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	v, _ := Open(path, "pw")
	v.Set("p", "K", "v")
	if err := v.Save(path, "pw"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600, got %04o", perm)
	}
}

func TestSave_UniqueCiphertextEachTime(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "v1.json")
	path2 := filepath.Join(dir, "v2.json")

	v, _ := Open(path1, "pw")
	v.Set("p", "K", "val")
	v.Save(path1, "pw")
	v.Save(path2, "pw")

	b1, _ := os.ReadFile(path1)
	b2, _ := os.ReadFile(path2)
	if string(b1) == string(b2) {
		t.Error("two saves produced identical ciphertext — salt/nonce reuse suspected")
	}
}

func TestSave_NoTempFileLeft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	v, _ := Open(path, "pw")
	v.Set("p", "K", "v")
	v.Save(path, "pw")

	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !errors.Is(err, os.ErrNotExist) {
		t.Error("temp file should be cleaned up after Save")
	}
}

// --- CRUD ---

func TestVault_SetGetDelete(t *testing.T) {
	v, _ := Open(filepath.Join(t.TempDir(), "v.json"), "pw")

	v.Set("proj", "KEY", "value")

	got, ok := v.Get("proj", "KEY")
	if !ok || got != "value" {
		t.Errorf("Get after Set: got %q, %v", got, ok)
	}

	if err := v.Delete("proj", "KEY"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := v.Get("proj", "KEY"); ok {
		t.Error("key should not exist after Delete")
	}
}

func TestVault_Delete_UnknownKey(t *testing.T) {
	v, _ := Open(filepath.Join(t.TempDir(), "v.json"), "pw")
	v.Set("proj", "KEY", "v")

	if err := v.Delete("proj", "MISSING"); err == nil {
		t.Error("expected error deleting missing key")
	}
	if err := v.Delete("noproj", "KEY"); err == nil {
		t.Error("expected error deleting from missing project")
	}
}

func TestVault_GetAll_IsolatedCopy(t *testing.T) {
	v, _ := Open(filepath.Join(t.TempDir(), "v.json"), "pw")
	v.Set("proj", "A", "1")
	v.Set("proj", "B", "2")

	all := v.GetAll("proj")
	all["A"] = "mutated"

	got, _ := v.Get("proj", "A")
	if got != "1" {
		t.Error("GetAll should return an isolated copy, not a reference")
	}
}

func TestVault_MultipleProjects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.json")
	v, _ := Open(path, "pw")
	v.Set("alpha", "KEY", "alpha-value")
	v.Set("beta", "KEY", "beta-value")
	v.Save(path, "pw")

	v2, _ := Open(path, "pw")
	if val, _ := v2.Get("alpha", "KEY"); val != "alpha-value" {
		t.Errorf("alpha: got %q", val)
	}
	if val, _ := v2.Get("beta", "KEY"); val != "beta-value" {
		t.Errorf("beta: got %q", val)
	}
}
