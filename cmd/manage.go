package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"envault/web"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newManageCmd())
}

func newManageCmd() *cobra.Command {
	var port int
	var bind string

	cmd := &cobra.Command{
		Use:   "manage",
		Short: "Start the web UI for managing secrets",
		Long:  `Opens a local web interface at http://localhost:7777 for managing projects and secrets.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate bind address
			if bind != "127.0.0.1" && bind != "::1" {
				return fmt.Errorf("bind address must be 127.0.0.1 or ::1 for security")
			}

			v, path, password, err := openVault()
			if err != nil {
				return err
			}

			srv := &web.Server{
				Vault:     v,
				VaultPath: path,
				Password:  password,
				Port:      port,
				Bind:      bind,
			}

			url := fmt.Sprintf("http://localhost:%d", port)
			fmt.Fprintf(os.Stderr, "envault manage  →  %s\n", url)
			fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop.\n")

			// Open browser after a short delay
			go func() {
				time.Sleep(300 * time.Millisecond)
				openBrowser(url)
			}()

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				srv.Shutdown(shutCtx)
			}()

			return srv.Start()
		},
	}

	cmd.Flags().IntVar(&port, "port", 7777, "port to listen on")
	cmd.Flags().StringVar(&bind, "bind", "127.0.0.1", "address to bind (127.0.0.1 or ::1 only)")
	return cmd
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		err = exec.Command("open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
	}
}
