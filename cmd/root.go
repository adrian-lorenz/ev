package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

var (
	flagProject   string
	flagVaultPath string
)

var rootCmd = &cobra.Command{
	Use:   "ev",
	Short: "Local encrypted secret manager",
	Long: `ev stores secrets encrypted outside your project directory,
keeping them away from AI coding agents, git repos, and logs.`,
}

// SetVersion injects the build-time version into the root command.
func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagProject, "project", "p", "", "project name (overrides .envault file)")
	rootCmd.PersistentFlags().StringVar(&flagVaultPath, "vault", "", "vault file path (default: ~/.envault/vault.json)")
}

// resolveProject returns the active project name.
// Priority: --project flag > .envault (searched upward) > current directory name
func resolveProject() string {
	if flagProject != "" {
		return flagProject
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "default"
	}
	if name, err := findProjectFile(cwd); err == nil && name != "" {
		return name
	}
	// Warn if a .envault file exists locally but couldn't be parsed
	if _, err := os.Stat(filepath.Join(cwd, ".envault")); err == nil {
		fmt.Fprintln(os.Stderr, "Warning: .envault file found but could not be parsed — falling back to directory name")
	}
	return filepath.Base(cwd)
}

// findProjectFile walks up the directory tree from start looking for a .envault file.
// Stops at a git root (.git directory), the home directory, or filesystem root to avoid
// picking up unrelated .envault files from parent repositories or the home directory.
func findProjectFile(start string) (string, error) {
	home, _ := os.UserHomeDir()
	dir := start
	for {
		// Don't read a .envault at the home directory — it conflicts with the vault
		// directory (~/.envault/vault.json) and shouldn't act as a global default.
		if dir != home {
			name, err := parseProjectFile(filepath.Join(dir, ".envault"))
			if err == nil {
				return name, nil
			}
		}

		// Stop at git repository root — don't leak into parent repos
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // filesystem root reached
		}
		// Stop before entering the home directory
		if parent == home {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf(".envault not found")
}

// parseProjectFile reads the project name from a single .envault file.
func parseProjectFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "project" {
			return strings.TrimSpace(parts[1]), nil
		}
		return line, nil
	}
	return "", fmt.Errorf("empty .envault file")
}

// resolveVaultPath returns the active vault path.
func resolveVaultPath() (string, error) {
	if flagVaultPath != "" {
		return flagVaultPath, nil
	}
	return vault.DefaultVaultPath()
}

// trySession returns cached vars from an active session, or nil if none exists.
// Commands that only read secrets should call this first before prompting for a password.
func trySession() map[string]string {
	vars, _ := vault.LoadSession(resolveProject())
	return vars
}

// openVaultKeychain opens the vault using the password stored in macOS Keychain.
// Falls back to a password prompt if no Keychain entry exists.
func openVaultKeychain() (*vault.Vault, string, string, error) {
	path, err := resolveVaultPath()
	if err != nil {
		return nil, "", "", err
	}
	password, err := vault.KeychainGet()
	if err != nil {
		fmt.Fprintln(os.Stderr, "No password in Keychain — prompting.")
		return openVault()
	}
	v, err := vault.Open(path, password)
	if err != nil {
		return nil, "", "", err
	}
	return v, path, password, nil
}

// openVault prompts for the master password and opens (or creates) the vault.
// If the vault is new, it asks to set a password with confirmation.
// Before opening, it checks the cloud for newer data and pulls if available.
func openVault() (*vault.Vault, string, string, error) {
	path, err := resolveVaultPath()
	if err != nil {
		return nil, "", "", err
	}

	// Pull from cloud if configured (non-fatal – sync errors are printed to stderr)
	vault.AutoPull(path)

	var password string
	if !vault.Exists(path) {
		fmt.Fprintln(os.Stderr, "No vault found – creating a new one.")
		p1, err := vault.PromptPassword("Set master password: ")
		if err != nil {
			return nil, "", "", err
		}
		if p1 == "" {
			return nil, "", "", errors.New("master password must not be empty")
		}
		p2, err := vault.PromptPassword("Confirm master password: ")
		if err != nil {
			return nil, "", "", err
		}
		if p1 != p2 {
			return nil, "", "", errors.New("passwords do not match")
		}
		password = p1
	} else {
		password, err = vault.PromptPassword("Master password: ")
		if err != nil {
			return nil, "", "", err
		}
	}

	v, err := vault.Open(path, password)
	if err != nil {
		return nil, "", "", err
	}
	return v, path, password, nil
}
