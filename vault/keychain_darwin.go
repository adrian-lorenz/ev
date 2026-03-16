//go:build darwin

package vault

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const keychainService = "envault"

func keychainAccount() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "envault"
}

// KeychainGet retrieves the master password from macOS Keychain.
func KeychainGet() (string, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-a", keychainAccount(),
		"-s", keychainService,
		"-w",
	).Output()
	if err != nil {
		return "", errors.New("no password in keychain")
	}
	return strings.TrimSpace(string(out)), nil
}

// KeychainSet stores the master password in macOS Keychain.
func KeychainSet(password string) error {
	err := exec.Command(
		"security", "add-generic-password",
		"-U",
		"-a", keychainAccount(),
		"-s", keychainService,
		"-w", password,
	).Run()
	if err != nil {
		return fmt.Errorf("saving to keychain: %w", err)
	}
	return nil
}

// KeychainDelete removes the stored password from macOS Keychain.
func KeychainDelete() error {
	return exec.Command(
		"security", "delete-generic-password",
		"-a", keychainAccount(),
		"-s", keychainService,
	).Run()
}
