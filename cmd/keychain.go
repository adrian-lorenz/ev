package cmd

import (
	"fmt"
	"os"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	keychainCmd := &cobra.Command{
		Use:   "keychain",
		Short: "Manage the macOS Keychain entry for the master password",
	}

	keychainCmd.AddCommand(
		&cobra.Command{
			Use:   "save",
			Short: "Save the master password to macOS Keychain",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				password, err := vault.PromptPassword("Master password to save: ")
				if err != nil {
					return err
				}
				if err := vault.KeychainSet(password); err != nil {
					return err
				}
				fmt.Fprintln(os.Stderr, "Password saved to macOS Keychain.")
				fmt.Fprintln(os.Stderr, "You can now use --keychain with any command.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "delete",
			Short: "Remove the master password from macOS Keychain",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := vault.KeychainDelete(); err != nil {
					return fmt.Errorf("could not delete keychain entry: %w", err)
				}
				fmt.Fprintln(os.Stderr, "Keychain entry deleted.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "check",
			Short: "Check whether a password is stored in macOS Keychain",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if _, err := vault.KeychainGet(); err != nil {
					fmt.Fprintln(os.Stderr, "No envault password in Keychain.")
					return nil
				}
				fmt.Fprintln(os.Stderr, "Password found in Keychain.")
				return nil
			},
		},
	)

	rootCmd.AddCommand(keychainCmd)
}
