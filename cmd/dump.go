package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDumpCmd())
}

func newDumpCmd() *cobra.Command {
	var useKeychain bool

	cmd := &cobra.Command{
		Use:   "dump [file]",
		Short: "Dump secrets as .env file (always requires master password)",
		Long: `Write all secrets for the current project to stdout or a file in KEY=value format.
Always decrypts with the master password — never uses an active session.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := resolveProject()

			var v *vault.Vault
			var err error
			if useKeychain {
				v, _, _, err = openVaultKeychain()
			} else {
				v, _, _, err = openVault()
			}
			if err != nil {
				return err
			}

			secrets := v.GetAll(project)
			if len(secrets) == 0 {
				fmt.Fprintf(os.Stderr, "No secrets in project %q\n", project)
				return nil
			}

			keys := make([]string, 0, len(secrets))
			for k := range secrets {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			out := os.Stdout
			if len(args) == 1 {
				f, err := os.OpenFile(args[0], os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
				if err != nil {
					return fmt.Errorf("opening output file: %w", err)
				}
				defer f.Close()
				out = f
				fmt.Fprintf(os.Stderr, "Secrets dumped to %s\n", args[0])
			}

			for _, k := range keys {
				fmt.Fprintf(out, "%s=%s\n", k, quoteEnvValue(secrets[k]))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&useKeychain, "keychain", "k", false, "read master password from macOS Keychain (no prompt)")
	return cmd
}

// quoteEnvValue wraps a value in double quotes if it contains characters that
// would be misinterpreted by .env parsers (spaces, quotes, newlines, etc.).
// Embedded double quotes and backslashes are always escaped.
func quoteEnvValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	if strings.ContainsAny(v, " \t\n\r#$`!") || v == "" {
		return `"` + v + `"`
	}
	return v
}
