package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newInfoCmd())
}

func newInfoCmd() *cobra.Command {
	var useKeychain bool

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show current project, vault, and session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --- Project ---
			project := resolveProject()
			projectSource := resolveProjectSource()

			fmt.Fprintf(os.Stderr, "\n  ev · local secret manager\n")
			fmt.Fprintf(os.Stderr, "  ─────────────────────────────────────────\n")
			fmt.Fprintf(os.Stderr, "  Project   : %s  (%s)\n", project, projectSource)

			// --- Vault ---
			vaultPath, _ := resolveVaultPath()
			if vault.Exists(vaultPath) {
				fi, _ := os.Stat(vaultPath)
				fmt.Fprintf(os.Stderr, "  Vault     : %s  (%s)\n", vaultPath, fmtSize(fi.Size()))
			} else {
				fmt.Fprintf(os.Stderr, "  Vault     : %s  (not created yet)\n", vaultPath)
			}

			// --- Session ---
			if expires, ok := vault.SessionInfo(project); ok {
				remaining := time.Until(expires).Round(time.Minute)
				fmt.Fprintf(os.Stderr, "  Session   : open · expires %s  (%s remaining)\n",
					expires.Local().Format("15:04:05"), fmtDuration(remaining))
			} else {
				fmt.Fprintf(os.Stderr, "  Session   : none  (run: ev open)\n")
			}

			// --- Keychain ---
			if _, err := vault.KeychainGet(); err == nil {
				fmt.Fprintf(os.Stderr, "  Keychain  : password saved\n")
			} else {
				fmt.Fprintf(os.Stderr, "  Keychain  : not set\n")
			}

			// --- Secrets count (session, keychain, or hint) ---
			if sv := trySession(); sv != nil {
				fmt.Fprintf(os.Stderr, "  Secrets   : %d (from session)\n", len(sv))
			} else if useKeychain && vault.Exists(vaultPath) {
				v, _, _, err := openVaultKeychain()
				if err == nil {
					fmt.Fprintf(os.Stderr, "  Secrets   : %d (from keychain)\n", len(v.ListKeys(project)))
				}
			} else if vault.Exists(vaultPath) {
				fmt.Fprintf(os.Stderr, "  Secrets   : run  ev list  to see keys\n")
			}

			// --- .envault file ---
			cwd, _ := os.Getwd()
			envaultFile := filepath.Join(cwd, ".envault")
			if _, err := os.Stat(envaultFile); err == nil {
				fmt.Fprintf(os.Stderr, "  .envault  : %s\n", envaultFile)
			} else {
				fmt.Fprintf(os.Stderr, "  .envault  : not found  (run: ev init)\n")
			}

			fmt.Fprintln(os.Stderr)

			// --- Project-type hint ---
			printProjectHint(project, "")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&useKeychain, "keychain", "k", false, "read master password from macOS Keychain (no prompt)")
	return cmd
}

// resolveProjectSource returns a human-readable description of where the project name came from.
func resolveProjectSource() string {
	if flagProject != "" {
		return "--project flag"
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "fallback"
	}
	if _, err := findProjectFile(cwd); err == nil {
		return ".envault file"
	}
	return "directory name"
}

func fmtSize(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
