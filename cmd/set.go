package cmd

import (
	"fmt"
	"os"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSetCmd())
}

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set KEY [VALUE]",
		Short: "Add or update a secret",
		Example: `  envault set DB_PASSWORD
  envault set DB_PASSWORD mysecret
  envault set DB_PASSWORD mysecret --project payment-service`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeKeyInput(args[0])
			if err := vault.ValidateName(key); err != nil {
				return fmt.Errorf("invalid key: %w", err)
			}

			var value string
			if len(args) == 2 {
				value = args[1]
			} else {
				var err error
				value, err = vault.PromptSecret(fmt.Sprintf("Value for %s: ", key))
				if err != nil {
					return err
				}
			}

			v, path, password, err := openVault()
			if err != nil {
				return err
			}

			project := resolveProject()
			v.Set(project, key, value)

			if err := v.Save(path, password); err != nil {
				return err
			}

			vault.AutoPush(path)
			_ = vault.RefreshSession(project, v.GetAll(project))
			fmt.Fprintf(os.Stderr, "Set %s in project %q\n", key, project)
			return nil
		},
	}
}
