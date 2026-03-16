package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetCmd())
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Print a secret value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeKeyInput(args[0])
			project := resolveProject()

			// Priority 1: active session
			if sv := trySession(); sv != nil {
				val, ok := sv[key]
				if !ok {
					return fmt.Errorf("key %q not found in project %q", key, project)
				}
				fmt.Print(val)
				return nil
			}

			// Priority 2: full vault
			v, _, _, err := openVault()
			if err != nil {
				return err
			}

			val, ok := v.Get(project, key)
			if !ok {
				return fmt.Errorf("key %q not found in project %q", key, project)
			}
			fmt.Print(val)
			return nil
		},
	}
}
