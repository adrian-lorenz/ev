package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	cmd := newListCmd()
	cmd.AddCommand(newListProjectsCmd())
	rootCmd.AddCommand(cmd)
}

func newListCmd() *cobra.Command {
	var asJSON bool
	var useKeychain bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List secret keys for the current project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			project := resolveProject()
			var keys []string

			// Priority 1: active session
			if sv := trySession(); sv != nil {
				for k := range sv {
					keys = append(keys, k)
				}
				sort.Strings(keys)
			} else {
				// Priority 2: Keychain or password prompt
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
				keys = v.ListKeys(project)
			}

			if len(keys) == 0 {
				fmt.Fprintf(os.Stderr, "No secrets in project %q\n", project)
				return nil
			}

			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(keys)
			}
			for _, k := range keys {
				fmt.Println(k)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON array")
	cmd.Flags().BoolVarP(&useKeychain, "keychain", "k", false, "read master password from macOS Keychain (no prompt)")
	return cmd
}

func newListProjectsCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List all projects in the vault",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v, _, _, err := openVault()
			if err != nil {
				return err
			}
			projects := v.ListProjects()
			if len(projects) == 0 {
				fmt.Fprintln(os.Stderr, "No projects found in vault")
				return nil
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(projects)
			}
			for _, p := range projects {
				fmt.Println(p)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON array")
	return cmd
}
