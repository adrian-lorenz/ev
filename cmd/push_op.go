package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSyncOPCmd())
}

type opVaultInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// newSyncOPCmd creates the `ev sync 1password` command.
func newSyncOPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync secrets to external services",
	}
	cmd.AddCommand(newSync1PasswordCmd())
	return cmd
}

func newSync1PasswordCmd() *cobra.Command {
	var opVaultName string
	var projectName string

	cmd := &cobra.Command{
		Use:   "1password",
		Short: "Push project secrets to 1Password as Secure Notes",
		Long: `Creates or updates a 1Password Secure Note for one or all projects.

Each note is titled "ev: <project-name>" and contains all secrets in .env format.
If --op-vault is not set, the command lists available vaults interactively.

Requires the 1Password CLI (op) to be installed and signed in.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("op"); err != nil {
				return fmt.Errorf("1Password CLI (op) not found in PATH\nInstall from: https://developer.1password.com/docs/cli")
			}

			// Resolve 1Password vault
			if opVaultName == "" {
				chosen, err := selectOPVault()
				if err != nil {
					return err
				}
				opVaultName = chosen
			}

			// Open ev vault
			v, _, _, err := openVault()
			if err != nil {
				return err
			}

			// Determine which projects to sync
			var projects []string
			if projectName != "" {
				projects = []string{projectName}
			} else {
				projects = v.ListProjects()
			}

			if len(projects) == 0 {
				return fmt.Errorf("no projects found in vault")
			}

			for _, proj := range projects {
				if err := pushProjectToOP(proj, opVaultName, v); err != nil {
					fmt.Fprintf(os.Stderr, "  x %s: %v\n", proj, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opVaultName, "op-vault", "", "1Password vault name or ID (default: interactive selection)")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "sync only this project (default: all projects)")
	return cmd
}

// pushProjectToOP creates or updates a 1Password Secure Note for the given project.
// The note content is the project's secrets in .env format.
func pushProjectToOP(project, opVaultName string, v *vault.Vault) error {
	vars := v.GetAll(project)
	if len(vars) == 0 {
		fmt.Fprintf(os.Stderr, "  - %s: no secrets, skipping\n", project)
		return nil
	}

	content := buildEnvContent(vars)
	title := "ev: " + project
	notesField := "notesPlain=" + content

	if opItemExists(title, opVaultName) {
		out, err := exec.Command("op", "item", "edit", title,
			"--vault", opVaultName,
			notesField,
		).CombinedOutput()
		if err != nil {
			return fmt.Errorf("op item edit failed: %s", strings.TrimSpace(string(out)))
		}
		fmt.Fprintf(os.Stderr, "  up %s: updated (%d secrets)\n", project, len(vars))
	} else {
		out, err := exec.Command("op", "item", "create",
			"--category", "Secure Note",
			"--title", title,
			"--vault", opVaultName,
			notesField,
		).CombinedOutput()
		if err != nil {
			return fmt.Errorf("op item create failed: %s", strings.TrimSpace(string(out)))
		}
		fmt.Fprintf(os.Stderr, "  + %s: created (%d secrets)\n", project, len(vars))
	}
	return nil
}

func opItemExists(title, vaultName string) bool {
	cmd := exec.Command("op", "item", "get", title, "--vault", vaultName)
	return cmd.Run() == nil
}

// selectOPVault lists available 1Password vaults and lets the user choose.
// Returns immediately if only one vault is available.
func selectOPVault() (string, error) {
	out, err := exec.Command("op", "vault", "list", "--format", "json").Output()
	if err != nil {
		return "", fmt.Errorf("listing 1Password vaults (are you signed in? run `op signin`): %w", err)
	}

	var vaults []opVaultInfo
	if err := json.Unmarshal(out, &vaults); err != nil {
		return "", fmt.Errorf("parsing vault list: %w", err)
	}

	if len(vaults) == 0 {
		return "", fmt.Errorf("no 1Password vaults found")
	}
	if len(vaults) == 1 {
		fmt.Fprintf(os.Stderr, "Using vault: %s\n", vaults[0].Name)
		return vaults[0].Name, nil
	}

	fmt.Fprintln(os.Stderr, "Available 1Password vaults:")
	for i, v := range vaults {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, v.Name)
	}
	fmt.Fprint(os.Stderr, "Select vault: ")

	var choice int
	if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(vaults) {
		return "", fmt.Errorf("invalid selection")
	}
	return vaults[choice-1].Name, nil
}

// buildEnvContent formats a vars map as a .env file string.
func buildEnvContent(vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		v := vars[k]
		if needsQuoting(v) {
			v = `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(v)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func needsQuoting(s string) bool {
	return strings.ContainsAny(s, " \t\n\"'\\#$`!")
}
