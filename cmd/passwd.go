package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newPasswdCmd())
}

func newPasswdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "passwd",
		Short: "Change the master password",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveVaultPath()
			if err != nil {
				return err
			}
			if !vault.Exists(path) {
				return errors.New("no vault found – run any command to create one first")
			}

			current, err := vault.PromptPassword("Current master password: ")
			if err != nil {
				return err
			}

			v, err := vault.Open(path, current)
			if err != nil {
				return err
			}

			newPw, err := vault.PromptPassword("New master password: ")
			if err != nil {
				return err
			}
			if newPw == "" {
				return errors.New("new password must not be empty")
			}
			confirm, err := vault.PromptPassword("Confirm new password: ")
			if err != nil {
				return err
			}
			if newPw != confirm {
				return errors.New("passwords do not match")
			}

			// Backup before overwriting
			stamp := time.Now().Format("2006-01-02T15-04-05")
			backupDir := filepath.Join(filepath.Dir(path), "backups")
			_ = os.MkdirAll(backupDir, 0700)
			bak := filepath.Join(backupDir, "vault-"+stamp+"-passwd-change.json")
			if err := copyFile(path, bak); err != nil {
				return fmt.Errorf("creating backup before password change: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Backup saved to %s\n", bak)

			if err := v.Save(path, newPw); err != nil {
				return fmt.Errorf("saving vault: %w", err)
			}

			vault.Audit("passwd-change", "master password changed")
			fmt.Fprintln(os.Stderr, "Master password changed.")
			return nil
		},
	}
}
