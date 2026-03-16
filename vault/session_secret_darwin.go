//go:build darwin

package vault

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

const sessionKeychainService = "envault-session"

func sessionKeychainAccount(project string) string {
	return fmt.Sprintf("%s:%s", keychainAccount(), project)
}

func storeSessionSecret(project string, key []byte) error {
	value := base64.StdEncoding.EncodeToString(key)
	return exec.Command(
		"security", "add-generic-password",
		"-U",
		"-a", sessionKeychainAccount(project),
		"-s", sessionKeychainService,
		"-w", value,
	).Run()
}

func loadSessionSecret(project string) ([]byte, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-a", sessionKeychainAccount(project),
		"-s", sessionKeychainService,
		"-w",
	).Output()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(string(out)))
}

func removeSessionSecret(project string) error {
	return exec.Command(
		"security", "delete-generic-password",
		"-a", sessionKeychainAccount(project),
		"-s", sessionKeychainService,
	).Run()
}
