package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newDeleteCmd())
}

func newDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete KEY",
		Short: "Remove a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeKeyInput(args[0])
			project := resolveProject()

			if !yes {
				fmt.Fprintf(os.Stderr, "Delete %s from project %q? [y/N] ", key, project)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			v, path, password, err := openVault()
			if err != nil {
				return err
			}

			if err := v.Delete(project, key); err != nil {
				return err
			}

			if err := v.Save(path, password); err != nil {
				return err
			}

			_ = vault.RefreshSession(project, v.GetAll(project))
			vault.Audit("delete-secret", fmt.Sprintf("project=%s key=%s", project, key))
			fmt.Fprintf(os.Stderr, "Deleted %s from project %q\n", key, project)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}
