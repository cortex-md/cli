package app

import (
	"fmt"
	"time"

	"github.com/cortex/cli/internal/auth"
	"github.com/cortex/cli/internal/buildinfo"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func NewLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with GitHub using device flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID := buildinfo.GitHubClientID
			if clientID == "" {
				ux.Error("GitHub client ID not configured in this build")
				ux.Info("This is a development build. Use a release build or set GITHUB_CLIENT_ID at build time.")
				return fmt.Errorf("missing client ID")
			}

			client := auth.NewClient(clientID)

			ux.Step("Initiating GitHub device flow...")

			deviceResp, err := client.StartDeviceFlow(cmd.Context())
			if err != nil {
				ux.Error("Failed to start device flow: %v", err)
				return err
			}

			if err := auth.CopyToClipboard(deviceResp.UserCode); err == nil {
				ux.Success("Code copied to clipboard: %s", deviceResp.UserCode)
			} else {
				ux.Info("Enter code: %s", deviceResp.UserCode)
			}

			fmt.Println()
			ux.Step("Opening browser...")

			if err := auth.OpenBrowser(deviceResp.VerificationURI); err != nil {
				ux.Warning("Could not open browser automatically")
				ux.Info("Please visit: %s", deviceResp.VerificationURI)
			}

			fmt.Println()
			ux.Step("Waiting for authorization...")

			ctx := cmd.Context()
			token, err := client.PollForToken(ctx, deviceResp.DeviceCode, deviceResp.Interval)
			if err != nil {
				ux.Error("Authorization failed: %v", err)
				return err
			}

			if err := auth.SaveToken(token); err != nil {
				ux.Error("Failed to save token: %v", err)
				return err
			}

			ux.Success("Successfully authenticated with GitHub!")
			return nil
		},
	}
}

func NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored GitHub credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !auth.IsAuthenticated() {
				ux.Warning("Not currently authenticated")
				return nil
			}

			if err := auth.DeleteToken(); err != nil {
				ux.Error("Failed to remove credentials: %v", err)
				return err
			}

			ux.Success("Successfully logged out")
			return nil
		},
	}
}

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new plugin or theme project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Info("Interactive project initialization coming soon!")
			time.Sleep(500 * time.Millisecond)
			ux.Warning("Use 'cortex plugin create' or 'cortex theme create' for now")
			return nil
		},
	}
}
