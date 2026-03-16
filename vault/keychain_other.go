//go:build !darwin

package vault

import "errors"

func KeychainGet() (string, error) {
	return "", errors.New("keychain is only supported on macOS")
}

func KeychainSet(_ string) error {
	return errors.New("keychain is only supported on macOS")
}

func KeychainDelete() error {
	return errors.New("keychain is only supported on macOS")
}
