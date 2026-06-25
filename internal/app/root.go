package app

import (
	"github.com/cortex/cli/internal/buildinfo"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:   "cortex",
		Short: "Cortex CLI for plugin and theme development",
		Long:  "Official CLI tool for creating, developing, validating, and publishing Cortex plugins and themes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				ux.PrintBannerCompact(buildinfo.Version)
				return nil
			}
			ux.PrintBanner(buildinfo.Version)
			return cmd.Help()
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	cmd.AddCommand(
		NewInitCommand(),
		NewLoginCommand(),
		NewLogoutCommand(),
		NewPluginCommand(),
		NewThemeCommand(),
		NewRegistryCommand(),
	)

	return cmd
}
