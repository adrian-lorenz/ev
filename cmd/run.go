package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newRunCmd())
}

func newRunCmd() *cobra.Command {
	var useKeychain bool
	var saveKeychain bool

	cmd := &cobra.Command{
		Use:   "run -- <command> [args...]",
		Short: "Run a command with secrets injected as environment variables",
		Long: `Injects secrets into the environment and replaces the current process
with the given command. Works seamlessly with PyCharm, VS Code, and other IDEs.

Authentication priority:
  1. Active session  (envault open)        — no prompt, recommended
  2. macOS Keychain  (--keychain flag)     — no prompt
  3. Password prompt (fallback)            — always works

PyCharm setup:
  Terminal:   envault open          (once, valid 8h)
  Run Config: envault run -- python -m uvicorn main:app --reload`,
		Example: `  envault run -- uvicorn main:app --reload
  envault run -- python main.py
  envault run --save-keychain -- echo "keychain saved"
  envault run --keychain -- python -m uvicorn main:app --reload`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var vars map[string]string

			// Priority 1: active session (envault open)
			if sv := trySession(); sv != nil {
				vars = sv
			} else if useKeychain || saveKeychain {
				// Priority 2: macOS Keychain
				password, err := vault.KeychainGet()
				if err != nil {
					fmt.Fprintln(os.Stderr, "No password in Keychain — prompting once.")
					password, err = promptAndOpen()
					if err != nil {
						return err
					}
					if err := vault.KeychainSet(password); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not save to Keychain: %v\n", err)
					} else {
						fmt.Fprintln(os.Stderr, "Password saved to macOS Keychain.")
					}
				}
				path, err := resolveVaultPath()
				if err != nil {
					return err
				}
				v, err := vault.Open(path, password)
				if err != nil {
					return err
				}
				vars = v.GetAll(resolveProject())
			} else {
				// Priority 3: password prompt
				v, _, _, err := openVault()
				if err != nil {
					return err
				}
				vars = v.GetAll(resolveProject())
			}

			// Inject secrets into environment
			for k, val := range vars {
				os.Setenv(k, val)
			}

			// For terraform: also inject as TF_VAR_<key> (hyphens → underscores)
			if len(args) > 0 && (args[0] == "terraform" || strings.HasSuffix(args[0], "/terraform")) {
				for k, val := range vars {
					tfKey := "TF_VAR_" + strings.ReplaceAll(strings.ToLower(k), "-", "_")
					if os.Getenv(tfKey) == "" {
						os.Setenv(tfKey, val)
					}
				}
			}

			// Find and exec the command (replaces current process on Unix)
			bin, err := exec.LookPath(args[0])
			if err != nil {
				return fmt.Errorf("command not found: %s", args[0])
			}

			return replaceProcess(bin, args)
		},
	}

	cmd.Flags().BoolVar(&useKeychain, "keychain", false, "read master password from macOS Keychain (no prompt)")
	cmd.Flags().BoolVar(&saveKeychain, "save-keychain", false, "prompt once, save to macOS Keychain, then run")
	// Stop flag parsing at the first non-flag argument so flags like --reload
	// are passed through to the child command, not interpreted by ev.
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func promptAndOpen() (string, error) {
	path, err := resolveVaultPath()
	if err != nil {
		return "", err
	}

	if !vault.Exists(path) {
		fmt.Fprintln(os.Stderr, "No vault found – creating a new one.")
		p1, err := vault.PromptPassword("Set master password: ")
		if err != nil {
			return "", err
		}
		p2, err := vault.PromptPassword("Confirm master password: ")
		if err != nil {
			return "", err
		}
		if p1 != p2 {
			return "", fmt.Errorf("passwords do not match")
		}
		return p1, nil
	}

	return vault.PromptPassword("Master password: ")
}
