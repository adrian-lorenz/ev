package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newOpenCmd())
	rootCmd.AddCommand(newCloseCmd())
	rootCmd.AddCommand(newSessionCmd())
}

func newOpenCmd() *cobra.Command {
	var ttlStr string

	c := &cobra.Command{
		Use:   "open",
		Short: "Unlock the vault for a timed session — no password prompts for N hours",
		Long: `Prompts for the master password once, then caches the secrets in an
encrypted session so all other commands work without re-prompting.

  envault open           # valid for 8 hours (default)
  envault open --ttl 4h  # valid for 4 hours

Stores encrypted session metadata under ~/.envault/sessions.
The session key is kept outside that file in the OS credential store.

Typical PyCharm workflow:
  1. Run "envault open" once in the terminal
  2. In Run Configuration: envault run -- python -m uvicorn main:app --reload
  3. No password prompt for the next 8 hours`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ttl, err := time.ParseDuration(ttlStr)
			if err != nil {
				return fmt.Errorf("invalid --ttl %q (example: 8h, 4h30m): %w", ttlStr, err)
			}
			if ttl <= 0 || ttl > 24*time.Hour {
				return fmt.Errorf("--ttl must be between 1m and 24h")
			}

			v, _, _, err := openVault()
			if err != nil {
				return err
			}

			project := resolveProject()
			vars := v.GetAll(project)

			if len(vars) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: project %q has no secrets.\n", project)
			}

			expires, err := vault.CreateSession(project, vars, ttl)
			if err != nil {
				return fmt.Errorf("creating session: %w", err)
			}

			remaining := time.Until(expires).Round(time.Minute)
			fmt.Fprintf(os.Stderr, "\nSession opened  ·  project: %s\n", project)
			fmt.Fprintf(os.Stderr, "Valid until:     %s  (%s)\n", expires.Local().Format("15:04:05"), fmtDuration(remaining))
			fmt.Fprintf(os.Stderr, "\nAll envault commands now run without a password prompt.\n")
			fmt.Fprintf(os.Stderr, "Run  envault close  to revoke the session early.\n")
			return nil
		},
	}

	c.Flags().StringVar(&ttlStr, "ttl", "8h", "session duration (e.g. 8h, 4h30m, 1h)")
	return c
}

func newCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close",
		Short: "Revoke the current session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			vault.DeleteSession(resolveProject())
			fmt.Fprintln(os.Stderr, "Session closed.")
			return nil
		},
	}
}

func newSessionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session",
		Short: "Show current session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project := resolveProject()
			expires, ok := vault.SessionInfo(project)
			if !ok {
				fmt.Fprintln(os.Stderr, "No active session.")
				return nil
			}
			remaining := time.Until(expires).Round(time.Minute)
			fmt.Fprintf(os.Stderr, "Active session\n")
			fmt.Fprintf(os.Stderr, "  Project : %s\n", project)
			fmt.Fprintf(os.Stderr, "  Expires : %s  (%s remaining)\n", expires.Local().Format("15:04:05"), fmtDuration(remaining))
			return nil
		},
	}
}

// ensureGitignore adds entry to .gitignore if not already present.
func ensureGitignore(entry string) {
	f, err := os.OpenFile(".gitignore", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	content, _ := io.ReadAll(f)
	if strings.Contains(string(content), entry) {
		return
	}
	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString(entry + "\n")
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
