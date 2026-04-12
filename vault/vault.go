package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/crypto/argon2"
	"golang.org/x/term"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 32
	nonceLen     = 12
	vaultVersion = 1
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)

// VaultFile is the on-disk JSON structure.
type VaultFile struct {
	Version    int    `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// Data is the decrypted in-memory structure.
type Data struct {
	Projects map[string]*Project `json:"projects"`
}

// Project holds secrets for a single project.
type Project struct {
	Vars map[string]string `json:"vars"`
}

// Vault holds the decrypted data in memory.
type Vault struct {
	data Data
	mu   sync.RWMutex
}

// DefaultVaultPath returns ~/.envault/vault.json.
func DefaultVaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".envault", "vault.json"), nil
}

// Exists reports whether the vault file exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ValidateName checks that a project or key name is safe.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	if len(name) > 128 {
		return errors.New("name must be 128 characters or fewer")
	}
	if !validName.MatchString(name) {
		return errors.New("name may only contain letters, digits, underscores, hyphens, and dots")
	}
	return nil
}

// Open decrypts and loads the vault. Returns an empty vault if the file does not exist.
func Open(path, password string) (*Vault, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Vault{data: Data{Projects: make(map[string]*Project)}}, nil
		}
		if errors.Is(err, syscall.ENOTDIR) {
			dir := filepath.Dir(path)
			return nil, fmt.Errorf("%s exists as a file instead of a directory\n\nIf this is a leftover .envault project-config file, remove it and re-run:\n  rm %s", dir, dir)
		}
		return nil, fmt.Errorf("reading vault: %w", err)
	}

	var vf VaultFile
	if err := json.Unmarshal(raw, &vf); err != nil {
		return nil, fmt.Errorf("parsing vault file: %w", err)
	}

	key := deriveKey([]byte(password), vf.Salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, vf.Nonce, vf.Ciphertext, nil)
	if err != nil {
		return nil, errors.New("could not decrypt vault: invalid password or corrupted file")
	}

	var data Data
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("parsing vault data: %w", err)
	}
	if data.Projects == nil {
		data.Projects = make(map[string]*Project)
	}

	return &Vault{data: data}, nil
}

// Save encrypts and writes the vault atomically.
func (v *Vault) Save(path, password string) error {
	v.mu.RLock()
	plaintext, err := json.Marshal(v.data)
	v.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshaling vault data: %w", err)
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generating salt: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generating nonce: %w", err)
	}

	key := deriveKey([]byte(password), salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("creating GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	vf := VaultFile{
		Version:    vaultVersion,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}
	out, err := json.Marshal(vf)
	if err != nil {
		return fmt.Errorf("marshaling vault file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating vault directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return fmt.Errorf("writing temp vault: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("saving vault: %w", err)
	}

	return nil
}

func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// --- CRUD ---

func (v *Vault) ensureProject(project string) *Project {
	if _, ok := v.data.Projects[project]; !ok {
		v.data.Projects[project] = &Project{Vars: make(map[string]string)}
	}
	return v.data.Projects[project]
}

func (v *Vault) Set(project, key, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ensureProject(project).Vars[key] = value
}

func (v *Vault) Get(project, key string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	p, ok := v.data.Projects[project]
	if !ok {
		return "", false
	}
	val, ok := p.Vars[key]
	return val, ok
}

func (v *Vault) GetAll(project string) map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	p, ok := v.data.Projects[project]
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(p.Vars))
	for k, val := range p.Vars {
		out[k] = val
	}
	return out
}

func (v *Vault) ListKeys(project string) []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	p, ok := v.data.Projects[project]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(p.Vars))
	for k := range p.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (v *Vault) Delete(project, key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	p, ok := v.data.Projects[project]
	if !ok {
		return fmt.Errorf("project %q not found", project)
	}
	if _, ok := p.Vars[key]; !ok {
		return fmt.Errorf("key %q not found in project %q", key, project)
	}
	delete(p.Vars, key)
	return nil
}

func (v *Vault) ListProjects() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]string, 0, len(v.data.Projects))
	for name := range v.data.Projects {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (v *Vault) CreateProject(name string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.data.Projects[name]; ok {
		return fmt.Errorf("project %q already exists", name)
	}
	v.data.Projects[name] = &Project{Vars: make(map[string]string)}
	return nil
}

func (v *Vault) DeleteProject(name string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, ok := v.data.Projects[name]; !ok {
		return fmt.Errorf("project %q not found", name)
	}
	delete(v.data.Projects, name)
	return nil
}

// --- Terminal helpers ---

// PromptPassword reads a password without echo. Prompt goes to stderr.
func PromptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

// PromptSecret reads a secret value without echo. Prompt goes to stderr.
func PromptSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
