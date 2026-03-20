package app

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cortex/cli/internal/dev"
	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
	}

	cmd.AddCommand(
		NewPluginCreateCommand(),
		NewPluginDevCommand(),
		NewPluginReloadCommand(),
		NewPluginBuildCommand(),
		NewPluginValidateCommand(),
		NewPluginDoctorCommand(),
		NewPluginPublishCommand(),
		NewPluginLinkCommand(),
		NewPluginUnlinkCommand(),
		NewPluginSearchCommand(),
		NewPluginInstallCommand(),
		NewPluginUpdateCommand(),
	)

	return cmd
}

func NewPluginCreateCommand() *cobra.Command {
	var opts plugin.CreateOptions

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new plugin project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Name = args[0]
			}

			if opts.Name == "" {
				var name string
				prompt := &survey.Input{
					Message: "Plugin name:",
				}
				if err := survey.AskOne(prompt, &name, survey.WithValidator(survey.Required)); err != nil {
					return err
				}
				opts.Name = name
			}

			if opts.ID == "" {
				var id string
				prompt := &survey.Input{
					Message: "Plugin ID:",
					Default: plugin.NormalizeID(opts.Name),
				}
				if err := survey.AskOne(prompt, &id); err != nil {
					return err
				}
				opts.ID = id
			}

			if opts.Description == "" {
				var desc string
				prompt := &survey.Input{
					Message: "Description:",
					Default: fmt.Sprintf("A Cortex plugin for %s", opts.Name),
				}
				if err := survey.AskOne(prompt, &desc); err != nil {
					return err
				}
				opts.Description = desc
			}

			if opts.Author == "" {
				var author string
				prompt := &survey.Input{
					Message: "Author:",
				}
				if err := survey.AskOne(prompt, &author); err != nil {
					return err
				}
				opts.Author = author
			}

			ux.Step("Creating plugin '%s'...", opts.Name)

			if err := plugin.Create(opts); err != nil {
				ux.Error("Failed to create plugin: %v", err)
				return err
			}

			ux.Success("Plugin '%s' created successfully!", opts.Name)
			ux.Info("Directory: %s", opts.ID)
			fmt.Println()
			ux.Info("Next steps:")
			fmt.Println("  cd " + opts.ID)
			fmt.Println("  bun install")
			fmt.Println("  bun run dev")

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Plugin ID")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Plugin description")
	cmd.Flags().StringVar(&opts.Author, "author", "", "Plugin author")
	cmd.Flags().StringVar(&opts.Directory, "dir", "", "Target directory")

	return cmd
}

func NewPluginDevCommand() *cobra.Command {
	var skipBuild bool
	var skipLink bool

	cmd := &cobra.Command{
		Use:   "dev [directory]",
		Short: "Start development mode with hot reload",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := dev.DevOptions{
				SkipInitialBuild: skipBuild,
				SkipLink:         skipLink,
			}

			return dev.Start(dir, opts)
		},
	}

	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip initial build")
	cmd.Flags().BoolVar(&skipLink, "skip-link", false, "Skip linking plugin")

	return cmd
}

func NewPluginReloadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Manually reload the plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Warning("Plugin reload coming soon!")
			return nil
		},
	}
}

func NewPluginBuildCommand() *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "build [directory]",
		Short: "Build the plugin for production",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := plugin.BuildOptions{
				Watch: watch,
			}

			return plugin.Build(dir, opts)
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes and rebuild")

	return cmd
}

func NewPluginValidateCommand() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate [directory]",
		Short: "Validate plugin structure and security",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := plugin.ValidateOptions{
				Strict: strict,
			}

			result, err := plugin.Validate(dir, opts)
			if err != nil {
				ux.Error("Validation failed: %v", err)
				return err
			}

			result.Print()

			if !result.Passed {
				return fmt.Errorf("validation failed")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Treat warnings as errors")

	return cmd
}

func NewPluginDoctorCommand() *cobra.Command {
	var dir string

	return &cobra.Command{
		Use:   "doctor [directory]",
		Short: "Run diagnostics on the plugin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := dir
			if target == "" {
				target = "."
			}
			if len(args) > 0 {
				target = args[0]
			}

			result, err := plugin.Doctor(target)
			if err != nil {
				return err
			}

			if len(result.Issues) == 0 {
				ux.Success("No issues found. Plugin is healthy.")
				return nil
			}

			for _, issue := range result.Issues {
				switch issue.Severity {
				case "fail":
					ux.Error("%s", issue.Message)
				case "warn":
					ux.Warning("%s", issue.Message)
				default:
					ux.Info("%s", issue.Message)
				}
				if issue.Fix != "" {
					ux.Info("Fix: %s", issue.Fix)
				}
			}

			if !result.Passed {
				return fmt.Errorf("doctor found blocking issues")
			}

			return nil
		},
	}
}

func NewPluginPublishCommand() *cobra.Command {
	var dryRun bool
	var skipBuild bool
	var skipValidate bool
	var draft bool
	var prerelease bool
	var coverImageURL string
	var author string
	var description string
	var repository string
	var updateOnly bool
	var nonInteractive bool
	var skipGitSync bool
	var skipRegistryPR bool

	cmd := &cobra.Command{
		Use:   "publish [directory]",
		Short: "Publish plugin to registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := plugin.PublishOptions{
				DryRun:         dryRun,
				SkipBuild:      skipBuild,
				SkipValidate:   skipValidate,
				Draft:          draft,
				Prerelease:     prerelease,
				CoverImageURL:  coverImageURL,
				Author:         author,
				Description:    description,
				Repository:     repository,
				UpdateOnly:     updateOnly,
				SkipGitSync:    skipGitSync,
				SkipRegistryPR: skipRegistryPR,
			}

			if len(args) == 0 && !nonInteractive {
				printInteractivePublishTip("plugin")
				if err := promptPluginPublishMetadata(dir, &opts); err != nil {
					return err
				}
			}

			result, err := plugin.Publish(dir, opts)
			if err != nil {
				ux.Error("Publish failed: %v", err)
				return err
			}

			if !dryRun && result.ReleaseURL != "" {
				fmt.Println()
				ux.Info("Release URL: %s", result.ReleaseURL)
				ux.Info("Asset URL: %s", result.AssetURL)
				if result.RegistryPR != "" {
					ux.Info("Registry PR: %s", result.RegistryPR)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate and build without creating release")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip build step")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip validation step")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create release as draft")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark release as prerelease")
	cmd.Flags().BoolVar(&updateOnly, "update-only", false, "Update existing release and skip registry PR")
	cmd.Flags().BoolVar(&nonInteractive, "no-interactive", false, "Disable interactive prompts")
	cmd.Flags().BoolVar(&skipGitSync, "skip-git-sync", false, "Skip git add/commit/push before release")
	cmd.Flags().BoolVar(&skipRegistryPR, "skip-registry-pr", false, "Skip opening registry pull request")
	cmd.Flags().StringVar(&author, "author", "", "Override author for registry metadata")
	cmd.Flags().StringVar(&description, "description", "", "Override description for registry metadata")
	cmd.Flags().StringVar(&repository, "repository", "", "Override repository for registry metadata")
	cmd.Flags().StringVar(&coverImageURL, "cover-image-url", "", "Cover image URL for registry listing")

	return cmd
}

func NewPluginLinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "link [directory]",
		Short: "Link plugin for development",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			return dev.Link(dir)
		},
	}
}

func NewPluginUnlinkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unlink [directory|plugin-id]",
		Short: "Unlink development plugin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return dev.Unlink(".")
			}

			arg := args[0]

			if _, err := os.Stat(arg); err == nil {
				return dev.Unlink(arg)
			}

			return dev.UnlinkByID(arg)
		},
	}
}

func NewPluginSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search for plugins in the registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Warning("Plugin search coming soon!")
			return nil
		},
	}
}

func NewPluginInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install [plugin-id]",
		Short: "Install a plugin from the registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Warning("Plugin install coming soon!")
			return nil
		},
	}
}

func NewPluginUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update [plugin-id]",
		Short: "Update an installed plugin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Warning("Plugin update coming soon!")
			return nil
		},
	}
}
