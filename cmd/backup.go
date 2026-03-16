package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newBackupCmd())
	rootCmd.AddCommand(newRestoreCmd())
}

func newBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup [file]",
		Short: "Back up the vault to a file",
		Long: `Copy the encrypted vault to a timestamped file in ~/.envault/backups/,
or to the path you specify.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := resolveVaultPath()
			if err != nil {
				return err
			}
			if !fileExists(src) {
				return fmt.Errorf("no vault found at %s", src)
			}

			var dst string
			if len(args) == 1 {
				dst = args[0]
			} else {
				stamp := time.Now().Format("2006-01-02T15-04-05")
				backupDir := filepath.Join(filepath.Dir(src), "backups")
				if err := os.MkdirAll(backupDir, 0700); err != nil {
					return fmt.Errorf("creating backup directory: %w", err)
				}
				dst = filepath.Join(backupDir, "vault-"+stamp+".json")
			}

			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("backing up vault: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Vault backed up to %s\n", dst)
			return nil
		},
	}
}

func newRestoreCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "restore <file>",
		Short: "Restore the vault from a backup file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			// Resolve symlinks to prevent symlink-based path traversal.
			realSrc, err := filepath.EvalSymlinks(src)
			if err != nil {
				return fmt.Errorf("backup file not found: %s", src)
			}
			fi, err := os.Stat(realSrc)
			if err != nil {
				return fmt.Errorf("backup file not found: %s", src)
			}
			if !fi.Mode().IsRegular() {
				return fmt.Errorf("backup path must be a regular file: %s", src)
			}
			src = realSrc

			dst, err := resolveVaultPath()
			if err != nil {
				return err
			}

			if !yes {
				fmt.Fprintf(os.Stderr, "Restore vault from %s? This will overwrite %s. [y/N] ", src, dst)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			// Back up current vault before overwriting
			if fileExists(dst) {
				stamp := time.Now().Format("2006-01-02T15-04-05")
				backupDir := filepath.Join(filepath.Dir(dst), "backups")
				_ = os.MkdirAll(backupDir, 0700)
				pre := filepath.Join(backupDir, "vault-pre-restore-"+stamp+".json")
				if err := copyFile(dst, pre); err == nil {
					fmt.Fprintf(os.Stderr, "Current vault saved to %s\n", pre)
				}
			}

			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("restoring vault: %w", err)
			}

			vault.Audit("restore", fmt.Sprintf("src=%s dst=%s", src, dst))
			fmt.Fprintf(os.Stderr, "Vault restored from %s\n", src)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
