package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	rootCmd.AddCommand(newLoadCmd())
}

func newLoadCmd() *cobra.Command {
	var shell string
	var reveal bool

	cmd := &cobra.Command{
		Use:   "load",
		Short: "Output export statements for eval in the current shell",
		Long: `Outputs shell export statements so secrets are available in the current session.

  eval "$(envault load)"              # bash / zsh — values never printed on screen
  envault load --shell fish | source  # fish

Uses an active session (envault open) if available — no password prompt needed.
When run directly in a terminal, values are masked for safety.
Use --reveal to print actual values to the terminal.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project := resolveProject()
			var vars map[string]string

			// Priority 1: active session
			if sv := trySession(); sv != nil {
				vars = sv
			} else {
				// Priority 2: full vault (prompts for password)
				v, _, _, err := openVault()
				if err != nil {
					return err
				}
				vars = v.GetAll(project)
			}

			if len(vars) == 0 {
				fmt.Fprintf(os.Stderr, "No secrets in project %q\n", project)
				return nil
			}

			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			// When stdout is a terminal the user ran "envault load" directly.
			// Show a masked preview so values don't appear on screen.
			// When stdout is a pipe (eval "$(envault load)") output real values.
			isTTY := term.IsTerminal(int(os.Stdout.Fd()))

			if isTTY && !reveal {
				shellName := strings.ToLower(shell)
				fmt.Fprintf(os.Stderr, "Project: %s  (%d secret(s))\n\n", project, len(keys))
				for _, k := range keys {
					switch shellName {
					case "fish":
						fmt.Printf("set -x %s ••••••••\n", k)
					default:
						fmt.Printf("export %s=••••••••\n", k)
					}
				}
				fmt.Fprintf(os.Stderr, "\nRun:  eval \"$(envault load)\"  to load into your shell.\n")
				return nil
			}

			// Real output — for eval or --reveal
			switch strings.ToLower(shell) {
			case "fish":
				for _, k := range keys {
					fmt.Printf("set -x %s %s;\n", k, fishQuote(vars[k]))
				}
			default:
				for _, k := range keys {
					fmt.Printf("export %s=%s\n", k, shQuote(vars[k]))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shell, "shell", "sh", "shell format: sh, bash, zsh, fish")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "print actual values to the terminal (use with care)")
	return cmd
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fishQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}
