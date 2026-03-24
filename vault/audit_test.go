package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashName_Deterministic(t *testing.T) {
	a := HashName("myproject")
	b := HashName("myproject")
	if a != b {
		t.Error("HashName should be deterministic")
	}
}

func TestHashName_DifferentInputs(t *testing.T) {
	if HashName("foo") == HashName("bar") {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashName_NonReversible(t *testing.T) {
	h := HashName("secretproject")
	if strings.Contains(h, "secretproject") {
		t.Error("HashName should not contain the original name")
	}
}

func TestHashName_NonEmpty(t *testing.T) {
	if HashName("") == "" {
		t.Error("HashName of empty string should still produce a non-empty hash")
	}
	if HashName("foo") == "" {
		t.Error("HashName should not be empty")
	}
}

func TestAudit_WritesEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	Audit("test-action", "detail=value")

	logPath := filepath.Join(os.Getenv("HOME"), ".envault", "audit.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("audit log not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "test-action") {
		t.Errorf("audit log missing action; content: %q", content)
	}
	if !strings.Contains(content, "detail=value") {
		t.Errorf("audit log missing detail; content: %q", content)
	}
}

func TestAudit_AppendsMultipleEntries(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	Audit("action-one", "first")
	Audit("action-two", "second")

	logPath := filepath.Join(os.Getenv("HOME"), ".envault", "audit.log")
	data, _ := os.ReadFile(logPath)
	content := string(data)

	if !strings.Contains(content, "action-one") || !strings.Contains(content, "action-two") {
		t.Errorf("both audit entries should be present; content: %q", content)
	}
	// Should be two lines
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(lines))
	}
}

func TestAudit_SilentFailure(t *testing.T) {
	// Set HOME to a non-writable path — Audit should not panic or return error
	t.Setenv("HOME", "/nonexistent/path/that/does/not/exist")

	// Should not panic
	Audit("action", "detail")
}

func TestAudit_FilePermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	Audit("action", "detail")

	logPath := filepath.Join(os.Getenv("HOME"), ".envault", "audit.log")
	fi, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600 on audit.log, got %04o", perm)
	}
}
