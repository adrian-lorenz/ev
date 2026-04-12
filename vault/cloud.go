package vault

// cloud.go – GitWall cloud sync for ev vaults.
//
// Config is stored in ~/.envault/cloud.json.  The file contains the GitWall
// instance URL, the store ID (UUID) and the es_ store token.  The vault blob
// is transmitted as base64-encoded raw bytes (the same JSON that is written to
// disk); GitWall stores it opaquely — it never sees the plaintext.

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CloudConfig holds the connection details for one GitWall store.
type CloudConfig struct {
	URL     string `json:"url"`      // e.g. "https://git.example.com"
	StoreID string `json:"store_id"` // UUID of the envault_stores row
	Token   string `json:"token"`    // es_ token (secret)
}

// SyncInfo is the metadata returned by the server about the current blob.
type SyncInfo struct {
	Version   int64  `json:"version"`
	Hash      string `json:"hash"`
	UpdatedAt string `json:"updated_at"`
}

// ─── Config I/O ───────────────────────────────────────────────────────────────

// CloudConfigPath derives the cloud config path from the vault file path.
// Convention:
//
//	~/.envault/vault.json   → ~/.envault/cloud.json         (default)
//	~/.envault/work.json    → ~/.envault/work.cloud.json
//	~/.envault/private.json → ~/.envault/private.cloud.json
//
// This allows multiple independent vaults, each with their own cloud sync config.
func CloudConfigPath(vaultPath string) string {
	dir := filepath.Dir(vaultPath)
	base := filepath.Base(vaultPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "vault" {
		return filepath.Join(dir, "cloud.json")
	}
	return filepath.Join(dir, name+".cloud.json")
}

// LoadCloudConfig reads the cloud config for the given vault; returns nil if it does not exist.
func LoadCloudConfig(vaultPath string) (*CloudConfig, error) {
	path := CloudConfigPath(vaultPath)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cloud config: %w", err)
	}
	var c CloudConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse cloud config: %w", err)
	}
	if c.URL == "" || c.StoreID == "" || c.Token == "" {
		return nil, nil
	}
	return &c, nil
}

// SaveCloudConfig writes the cloud config atomically.
func SaveCloudConfig(vaultPath string, c *CloudConfig) error {
	path := CloudConfigPath(vaultPath)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DeleteCloudConfig removes the cloud config file for the given vault.
func DeleteCloudConfig(vaultPath string) error {
	path := CloudConfigPath(vaultPath)
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *CloudConfig) baseURL() string {
	return strings.TrimRight(c.URL, "/") + "/api/v1/envault/sync/" + c.StoreID
}

func (c *CloudConfig) newClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *CloudConfig) newReq(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// ─── Sync operations ──────────────────────────────────────────────────────────

// GetSyncInfo retrieves version + hash from the server without downloading the blob.
func (c *CloudConfig) GetSyncInfo() (*SyncInfo, error) {
	req, err := c.newReq(http.MethodGet, c.baseURL()+"/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.newClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud: server returned %d", resp.StatusCode)
	}
	var info struct {
		SyncInfo
		Data string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("cloud: decode response: %w", err)
	}
	return &info.SyncInfo, nil
}

// Push uploads the local vault blob to the cloud.
// It reads the raw vault file from vaultPath and sends it as-is.
func (c *CloudConfig) Push(vaultPath string) error {
	raw, err := os.ReadFile(vaultPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil // nothing to push if vault doesn't exist yet
	}
	if err != nil {
		return fmt.Errorf("cloud push: read vault: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	body, _ := json.Marshal(map[string]string{"data": encoded})

	req, err := c.newReq(http.MethodPut, c.baseURL()+"/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cloud push: %w", err)
	}
	resp, err := c.newClient().Do(req)
	if err != nil {
		return fmt.Errorf("cloud push: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cloud push: server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return nil
}

// Pull downloads the cloud blob and writes it to vaultPath if the cloud has
// a newer or different version.  Returns true if the local file was updated.
func (c *CloudConfig) Pull(vaultPath string) (updated bool, err error) {
	req, err := c.newReq(http.MethodGet, c.baseURL()+"/", nil)
	if err != nil {
		return false, fmt.Errorf("cloud pull: %w", err)
	}
	resp, err := c.newClient().Do(req)
	if err != nil {
		return false, fmt.Errorf("cloud pull: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cloud pull: server returned %d", resp.StatusCode)
	}

	var result struct {
		Version   int64  `json:"version"`
		Hash      string `json:"hash"`
		Data      string `json:"data"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("cloud pull: decode: %w", err)
	}
	if result.Data == "" {
		// Cloud store is empty
		return false, nil
	}

	cloudBytes, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return false, fmt.Errorf("cloud pull: decode base64: %w", err)
	}

	// Compare with local file
	local, err := os.ReadFile(vaultPath)
	if err == nil {
		localHash := LocalHash(local)
		if localHash == result.Hash {
			return false, nil // already in sync
		}
	}

	// Write atomically
	if err := os.MkdirAll(filepath.Dir(vaultPath), 0700); err != nil {
		return false, fmt.Errorf("cloud pull: mkdir: %w", err)
	}
	tmp := vaultPath + ".cloud.tmp"
	if err := os.WriteFile(tmp, cloudBytes, 0600); err != nil {
		return false, fmt.Errorf("cloud pull: write: %w", err)
	}
	if err := os.Rename(tmp, vaultPath); err != nil {
		os.Remove(tmp)
		return false, fmt.Errorf("cloud pull: rename: %w", err)
	}
	return true, nil
}

// ─── Auto-sync helpers ────────────────────────────────────────────────────────

// AutoPull checks the cloud for changes and updates the local vault if needed.
// Errors are printed to stderr but do not abort the command.
func AutoPull(vaultPath string) {
	cfg, err := LoadCloudConfig(vaultPath)
	if err != nil || cfg == nil {
		return
	}
	updated, err := cfg.Pull(vaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cloud sync: pull failed: %v\n", err)
		return
	}
	if updated {
		fmt.Fprintln(os.Stderr, "cloud sync: vault updated from cloud")
	}
}

// AutoPush uploads the local vault to the cloud after a write operation.
// Errors are printed to stderr but do not abort the command.
func AutoPush(vaultPath string) {
	cfg, err := LoadCloudConfig(vaultPath)
	if err != nil || cfg == nil {
		return
	}
	if err := cfg.Push(vaultPath); err != nil {
		fmt.Fprintf(os.Stderr, "cloud sync: push failed: %v\n", err)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// LocalHash returns the SHA-256 hex digest of raw bytes (used for sync comparison).
func LocalHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
