package app

import (
	"github.com/cortex/cli/internal/buildinfo"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

const (
	groupDevelop = "develop"
	groupPublish = "publish"
	groupAccount = "account"
)

func NewRootCommand() *cobra.Command {
	cobra.EnableCommandSorting = false

	cmd := &cobra.Command{
		Use:           "cortex",
		Short:         "Build, test and publish Cortex plugins and themes",
		Version:       buildinfo.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ux.PrintHeader(buildinfo.Version)
			ux.Blank()
			_ = cmd.Help()
			ux.NextSteps(ux.NextStep{Command: "cortex create", Hint: "start a new plugin or theme"})
		},
	}

	cmd.SetVersionTemplate("Cortex CLI v{{.Version}}\n")
	cmd.PersistentFlags().BoolVarP(&ux.AssumeYes, "yes", "y", false, "Answer yes to confirmations (required without a terminal)")
	cmd.CompletionOptions.HiddenDefaultCmd = true

	cmd.AddGroup(
		&cobra.Group{ID: groupDevelop, Title: "Develop:"},
		&cobra.Group{ID: groupPublish, Title: "Publish:"},
		&cobra.Group{ID: groupAccount, Title: "Account:"},
	)

	cmd.AddCommand(
		inGroup(groupDevelop, newCreateCommand()),
		inGroup(groupDevelop, newDevCommand()),
		inGroup(groupDevelop, newCheckCommand()),
		inGroup(groupPublish, newReleaseCommand()),
		inGroup(groupPublish, newSubmitCommand()),
		inGroup(groupPublish, newStatsCommand()),
		inGroup(groupPublish, newPackCommand()),
		inGroup(groupAccount, newLoginCommand()),
		inGroup(groupAccount, newLogoutCommand()),
		inGroup(groupAccount, newWhoamiCommand()),
	)
	cmd.AddCommand(newLegacyCommands()...)
	cmd.SetHelpCommandGroupID(groupAccount)

	return cmd
}

func inGroup(group string, cmd *cobra.Command) *cobra.Command {
	cmd.GroupID = group
	return cmd
}
