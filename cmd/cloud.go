package cmd

// cloud.go – ev cloud subcommands for GitWall sync.
//
//   ev cloud setup   – interactive setup (URL, GitWall token, store name)
//   ev cloud push    – force-upload local vault to cloud
//   ev cloud pull    – force-download cloud vault to local
//   ev cloud status  – print version, hash, last-updated
//   ev cloud reset   – remove cloud config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"envault/vault"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCloudCmd())
}

func newCloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "GitWall cloud sync – back up and restore your vault",
	}
	cmd.AddCommand(
		newCloudSetupCmd(),
		newCloudPushCmd(),
		newCloudPullCmd(),
		newCloudStatusCmd(),
		newCloudResetCmd(),
	)
	return cmd
}

// ─── setup ────────────────────────────────────────────────────────────────────

func newCloudSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure GitWall cloud sync",
		Long: `Connects ev to a GitWall instance for cloud backup.

You will need:
  1. The URL of your GitWall instance (e.g. https://git.example.com)
  2. A GitWall access token with 'repo' scope (create at /settings/tokens)
  3. A name for the remote secret store (default: "default")

ev will create the store on the server and save the store token locally.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, err := resolveVaultPath()
			if err != nil {
				return err
			}

			// Check if already configured
			existing, _ := vault.LoadCloudConfig(vaultPath)
			if existing != nil {
				fmt.Fprintln(os.Stderr, "Cloud sync is already configured.")
				fmt.Fprintf(os.Stderr, "Config: %s\n", vault.CloudConfigPath(vaultPath))
				fmt.Fprintln(os.Stderr, "Run  ev cloud reset  to remove the current config first.")
				return nil
			}

			fmt.Fprintln(os.Stderr, "\nev cloud sync setup")
			fmt.Fprintln(os.Stderr, "───────────────────")
			fmt.Fprintf(os.Stderr, "Vault: %s\n\n", vaultPath)

			gwURL, err := promptLine("GitWall URL (e.g. https://git.example.com): ")
			if err != nil {
				return err
			}
			gwURL = strings.TrimRight(strings.TrimSpace(gwURL), "/")
			if gwURL == "" {
				return fmt.Errorf("URL must not be empty")
			}

			gwToken, err := vault.PromptSecret("GitWall access token (gw_...): ")
			if err != nil {
				return err
			}
			gwToken = strings.TrimSpace(gwToken)
			if gwToken == "" {
				return fmt.Errorf("access token must not be empty")
			}

			storeName, err := promptLine("Store name [default]: ")
			if err != nil {
				return err
			}
			storeName = strings.TrimSpace(storeName)
			if storeName == "" {
				storeName = "default"
			}

			// Create the store on the server
			fmt.Fprintln(os.Stderr, "\nCreating store on GitWall...")
			storeID, storeToken, err := createRemoteStore(gwURL, gwToken, storeName)
			if err != nil {
				return fmt.Errorf("failed to create store: %w", err)
			}

			cfg := &vault.CloudConfig{
				URL:     gwURL,
				StoreID: storeID,
				Token:   storeToken,
			}
			if err := vault.SaveCloudConfig(vaultPath, cfg); err != nil {
				return fmt.Errorf("save cloud config: %w", err)
			}

			fmt.Fprintln(os.Stderr, "Store created and config saved.")
			fmt.Fprintf(os.Stderr, "Store ID: %s\n\n", storeID)
			fmt.Fprintln(os.Stderr, "Pushing current vault to cloud...")

			if vault.Exists(vaultPath) {
				if err := cfg.Push(vaultPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: initial push failed: %v\n", err)
				} else {
					fmt.Fprintln(os.Stderr, "Vault pushed successfully.")
				}
			} else {
				fmt.Fprintln(os.Stderr, "No local vault found – nothing to push yet.")
			}
			fmt.Fprintln(os.Stderr, "\nCloud sync is now active. Changes are synced automatically.")
			return nil
		},
	}
}

// ─── push ─────────────────────────────────────────────────────────────────────

func newCloudPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Upload local vault to the cloud (overwrites cloud version)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireCloudConfig()
			if err != nil {
				return err
			}
			vaultPath, err := resolveVaultPath()
			if err != nil {
				return err
			}
			if !vault.Exists(vaultPath) {
				return fmt.Errorf("no local vault found at %s", vaultPath)
			}
			fmt.Fprintln(os.Stderr, "Pushing vault to cloud...")
			if err := cfg.Push(vaultPath); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Done.")
			return nil
		},
	}
}

// ─── pull ─────────────────────────────────────────────────────────────────────

func newCloudPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Download vault from the cloud (overwrites local version)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := requireCloudConfig()
			if err != nil {
				return err
			}
			vaultPath, err := resolveVaultPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Pulling vault from cloud...")
			updated, err := cfg.Pull(vaultPath)
			if err != nil {
				return err
			}
			if updated {
				fmt.Fprintln(os.Stderr, "Vault updated from cloud.")
			} else {
				fmt.Fprintln(os.Stderr, "Already up to date.")
			}
			return nil
		},
	}
}

// ─── status ───────────────────────────────────────────────────────────────────

func newCloudStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show cloud sync status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, _ := resolveVaultPath()
			cfg, err := vault.LoadCloudConfig(vaultPath)
			if err != nil {
				return err
			}
			if cfg == nil {
				fmt.Fprintln(os.Stderr, "Cloud sync not configured.  Run: ev cloud setup")
				return nil
			}

			fmt.Fprintf(os.Stderr, "\n  ev cloud sync status\n")
			fmt.Fprintf(os.Stderr, "  ──────────────────────────────────────\n")
			fmt.Fprintf(os.Stderr, "  Vault   : %s\n", vaultPath)
			fmt.Fprintf(os.Stderr, "  Server  : %s\n", cfg.URL)
			fmt.Fprintf(os.Stderr, "  Store   : %s\n", cfg.StoreID)

			info, err := cfg.GetSyncInfo()
			if err != nil {
				fmt.Fprintf(os.Stderr, "  Cloud   : unreachable (%v)\n", err)
			} else if info.Version == 0 {
				fmt.Fprintf(os.Stderr, "  Cloud   : empty (no data pushed yet)\n")
			} else {
				updatedAt, _ := time.Parse(time.RFC3339, info.UpdatedAt)
				fmt.Fprintf(os.Stderr, "  Version : %d\n", info.Version)
				fmt.Fprintf(os.Stderr, "  Hash    : %s\n", info.Hash[:16]+"...")
				fmt.Fprintf(os.Stderr, "  Updated : %s\n", updatedAt.Local().Format("2006-01-02 15:04:05"))
			}

			// Compare with local
			if vault.Exists(vaultPath) {
				raw, err := os.ReadFile(vaultPath)
				if err == nil {
					localHash := vault.LocalHash(raw)
					if info != nil && localHash == info.Hash {
						fmt.Fprintf(os.Stderr, "  Sync    : up to date\n")
					} else if info != nil && info.Version > 0 {
						fmt.Fprintf(os.Stderr, "  Sync    : local differs from cloud\n")
					}
				}
			}
			fmt.Fprintln(os.Stderr)
			return nil
		},
	}
}

// ─── reset ────────────────────────────────────────────────────────────────────

func newCloudResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Remove cloud sync configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			vaultPath, _ := resolveVaultPath()
			if err := vault.DeleteCloudConfig(vaultPath); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Cloud sync configuration removed.")
			return nil
		},
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func requireCloudConfig() (*vault.CloudConfig, error) {
	vaultPath, err := resolveVaultPath()
	if err != nil {
		return nil, err
	}
	cfg, err := vault.LoadCloudConfig(vaultPath)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("cloud sync not configured — run: ev cloud setup")
	}
	return cfg, nil
}

func promptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	var line string
	_, err := fmt.Scanln(&line)
	if err != nil && err.Error() != "unexpected newline" {
		return "", err
	}
	return line, nil
}

// createRemoteStore calls POST /api/v1/envault/stores using a GitWall user token.
func createRemoteStore(gwURL, gwToken, name string) (storeID, storeToken string, err error) {
	bodyBytes, _ := json.Marshal(map[string]string{"name": name})
	req, err := http.NewRequest(http.MethodPost,
		strings.TrimRight(gwURL, "/")+"/api/v1/envault/stores",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+gwToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("connect to %s: %w", gwURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("envault sync is disabled on this GitWall instance")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", fmt.Errorf("invalid GitWall token")
	}
	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		if msg := errBody["error"]; msg != "" {
			return "", "", fmt.Errorf("server: %s", msg)
		}
		return "", "", fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var result struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("parse response: %w", err)
	}
	return result.ID, result.Token, nil
}
