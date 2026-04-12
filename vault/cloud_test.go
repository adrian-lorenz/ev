package vault

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ─── CloudConfigPath ──────────────────────────────────────────────────────────

func TestCloudConfigPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		vaultPath string
		wantBase  string
	}{
		{"/home/user/.envault/vault.json", "cloud.json"},
		{"/home/user/.envault/work.json", "work.cloud.json"},
		{"/home/user/.envault/private.json", "private.cloud.json"},
		{"/home/user/.envault/my-company.json", "my-company.cloud.json"},
	}

	for _, tt := range tests {
		got := CloudConfigPath(tt.vaultPath)
		if filepath.Base(got) != tt.wantBase {
			t.Errorf("CloudConfigPath(%q) base = %q, want %q", tt.vaultPath, filepath.Base(got), tt.wantBase)
		}
		if filepath.Dir(got) != filepath.Dir(tt.vaultPath) {
			t.Errorf("CloudConfigPath(%q) dir = %q, want %q", tt.vaultPath, filepath.Dir(got), filepath.Dir(tt.vaultPath))
		}
	}
}

// ─── LoadCloudConfig / SaveCloudConfig / DeleteCloudConfig ───────────────────

func TestCloudConfigRoundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")

	// Not found → returns nil, no error
	cfg, err := LoadCloudConfig(vaultPath)
	if err != nil {
		t.Fatalf("LoadCloudConfig on missing file: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}

	// Save and reload
	want := &CloudConfig{
		URL:     "https://git.example.com",
		StoreID: "12345678-1234-1234-1234-123456789abc",
		Token:   "es_abc123",
	}
	if err := SaveCloudConfig(vaultPath, want); err != nil {
		t.Fatalf("SaveCloudConfig: %v", err)
	}

	got, err := LoadCloudConfig(vaultPath)
	if err != nil {
		t.Fatalf("LoadCloudConfig after save: %v", err)
	}
	if got == nil {
		t.Fatal("expected config, got nil")
	}
	if got.URL != want.URL || got.StoreID != want.StoreID || got.Token != want.Token {
		t.Errorf("got %+v, want %+v", got, want)
	}

	// Config file must have 0600 permissions
	cfgPath := CloudConfigPath(vaultPath)
	fi, _ := os.Stat(cfgPath)
	if fi.Mode() != 0600 {
		t.Errorf("cloud config permissions = %o, want 0600", fi.Mode())
	}
}

func TestCloudConfigDeleteNonExistentIsNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")
	if err := DeleteCloudConfig(vaultPath); err != nil {
		t.Errorf("DeleteCloudConfig on non-existent: %v", err)
	}
}

func TestCloudConfigEmptyFieldsReturnsNil(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")
	cfgPath := CloudConfigPath(vaultPath)

	// Write config with empty URL
	raw, _ := json.Marshal(CloudConfig{URL: "", StoreID: "abc", Token: "es_x"})
	os.WriteFile(cfgPath, raw, 0600)

	cfg, err := LoadCloudConfig(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil for incomplete config")
	}
}

// ─── LocalHash ────────────────────────────────────────────────────────────────

func TestLocalHash(t *testing.T) {
	t.Parallel()

	data := []byte("test data")
	h1 := LocalHash(data)
	h2 := LocalHash(data)
	h3 := LocalHash([]byte("other"))

	if h1 != h2 {
		t.Error("LocalHash is not deterministic")
	}
	if h1 == h3 {
		t.Error("different data must produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("LocalHash length = %d, want 64 (hex SHA-256)", len(h1))
	}
}

// ─── CloudConfig.Push / Pull (HTTP mock) ─────────────────────────────────────

func TestCloudPushAndPull(t *testing.T) {
	t.Parallel()

	// Simulate a GitWall envault sync server
	storedData := ""
	storedHash := ""
	version := int64(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			var body struct {
				Data string `json:"data"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			raw, err := base64.StdEncoding.DecodeString(body.Data)
			if err != nil {
				http.Error(w, "bad base64", http.StatusBadRequest)
				return
			}
			storedData = body.Data
			storedHash = LocalHash(raw)
			version++
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"version":    version,
				"hash":       storedHash,
				"updated_at": "2026-04-12T10:00:00Z",
			})

		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"version":    version,
				"hash":       storedHash,
				"data":       storedData,
				"updated_at": "2026-04-12T10:00:00Z",
			})
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")

	// Write a fake vault file
	vaultContent := []byte(`{"version":1,"salt":"dGVzdA==","nonce":"dGVzdA==","ciphertext":"dGVzdA=="}`)
	if err := os.WriteFile(vaultPath, vaultContent, 0600); err != nil {
		t.Fatal(err)
	}

	cfg := &CloudConfig{
		URL:     srv.URL,
		StoreID: "00000000-0000-0000-0000-000000000001",
		Token:   "es_testtoken",
	}

	// Push
	if err := cfg.Push(vaultPath); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if storedData == "" {
		t.Error("server should have received data after Push")
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}

	// Pull when already in sync → no file change
	updated, err := cfg.Pull(vaultPath)
	if err != nil {
		t.Fatalf("Pull (in-sync): %v", err)
	}
	if updated {
		t.Error("Pull should return updated=false when already in sync")
	}

	// Modify server data so Pull sees a difference
	storedHash = "differenthash"
	updated, err = cfg.Pull(vaultPath)
	if err != nil {
		t.Fatalf("Pull (out-of-sync): %v", err)
	}
	if !updated {
		t.Error("Pull should return updated=true when cloud has different data")
	}
}

func TestCloudPullEmptyStore(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"version":    0,
			"hash":       "",
			"data":       "",
			"updated_at": "2026-04-12T10:00:00Z",
		})
	}))
	defer srv.Close()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")

	cfg := &CloudConfig{URL: srv.URL, StoreID: "uuid", Token: "es_tok"}
	updated, err := cfg.Pull(vaultPath)
	if err != nil {
		t.Fatalf("Pull on empty store: %v", err)
	}
	if updated {
		t.Error("empty store should not update local file")
	}
}

func TestAutoPullNoConfigIsNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")
	// No cloud config exists — AutoPull must not panic or error
	AutoPull(vaultPath) // should be silent
}

func TestAutoPushNoConfigIsNoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "vault.json")
	// No cloud config exists — AutoPush must not panic or error
	AutoPush(vaultPath) // should be silent
}

func TestMultipleVaultConfigs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	defaultVault := filepath.Join(dir, "vault.json")
	workVault := filepath.Join(dir, "work.json")
	privateVault := filepath.Join(dir, "private.json")

	cfgDefault := &CloudConfig{URL: "https://git.example.com", StoreID: "id1", Token: "es_1"}
	cfgWork := &CloudConfig{URL: "https://work.example.com", StoreID: "id2", Token: "es_2"}

	SaveCloudConfig(defaultVault, cfgDefault)
	SaveCloudConfig(workVault, cfgWork)

	// Each vault has its own config
	gotDefault, _ := LoadCloudConfig(defaultVault)
	gotWork, _ := LoadCloudConfig(workVault)
	gotPrivate, _ := LoadCloudConfig(privateVault)

	if gotDefault == nil || gotDefault.URL != "https://git.example.com" {
		t.Error("default vault config mismatch")
	}
	if gotWork == nil || gotWork.URL != "https://work.example.com" {
		t.Error("work vault config mismatch")
	}
	if gotPrivate != nil {
		t.Error("private vault should have no config")
	}

	// Configs must be in separate files
	if CloudConfigPath(defaultVault) == CloudConfigPath(workVault) {
		t.Error("default and work vault must have separate config paths")
	}
}
