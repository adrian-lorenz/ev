package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newInitCmd())
}

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init [project-name]",
		Short: "Create a .envault file in the current directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				name = filepath.Base(cwd)
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return fmt.Errorf("project name must not be empty")
			}

			const filename = ".envault"
			if !force {
				if _, err := os.Stat(filename); err == nil {
					return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
				}
			}

			content := fmt.Sprintf("project=%s\n", name)
			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Created .envault for project %q\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing .envault file")
	return cmd
}
