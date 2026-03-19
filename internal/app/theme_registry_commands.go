package app

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func NewThemeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage themes",
	}

	cmd.AddCommand(
		NewThemeCreateCommand(),
		NewThemeValidateCommand(),
		NewThemePublishCommand(),
	)

	return cmd
}

func NewThemeCreateCommand() *cobra.Command {
	var opts theme.CreateOptions

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new theme project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Name = args[0]
			}

			if opts.Name == "" {
				var name string
				prompt := &survey.Input{
					Message: "Theme name:",
				}
				if err := survey.AskOne(prompt, &name, survey.WithValidator(survey.Required)); err != nil {
					return err
				}
				opts.Name = name
			}

			if opts.ID == "" {
				var id string
				prompt := &survey.Input{
					Message: "Theme ID:",
					Default: normalizeThemeID(opts.Name),
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
					Default: fmt.Sprintf("A Cortex theme for %s", opts.Name),
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

			ux.Step(fmt.Sprintf("Creating theme '%s'...", opts.Name))

			if err := theme.Create(opts); err != nil {
				ux.Error(fmt.Sprintf("Failed to create theme: %v", err))
				return err
			}

			ux.Success(fmt.Sprintf("Theme '%s' created successfully!", opts.Name))
			ux.Info(fmt.Sprintf("Directory: %s", opts.ID))
			fmt.Println()
			ux.Info("Next steps:")
			fmt.Println("  cd " + opts.ID)
			fmt.Println("  Edit theme-dark.css and theme-light.css to customize your theme")
			fmt.Println("  cortex theme validate")

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Theme ID")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Theme description")
	cmd.Flags().StringVar(&opts.Author, "author", "", "Theme author")
	cmd.Flags().StringVar(&opts.Directory, "dir", "", "Target directory")

	return cmd
}

func NewThemeValidateCommand() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate [directory]",
		Short: "Validate theme structure and CSS",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := theme.ValidateOptions{
				Strict: strict,
			}

			result, err := theme.Validate(dir, opts)
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

func NewThemePublishCommand() *cobra.Command {
	var dryRun bool
	var skipValidate bool
	var draft bool
	var prerelease bool

	cmd := &cobra.Command{
		Use:   "publish [directory]",
		Short: "Publish theme to registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			opts := theme.PublishOptions{
				DryRun:       dryRun,
				SkipValidate: skipValidate,
				Draft:        draft,
				Prerelease:   prerelease,
			}

			result, err := theme.Publish(dir, opts)
			if err != nil {
				ux.Error("Publish failed: %v", err)
				return err
			}

			if !dryRun && result.ReleaseURL != "" {
				fmt.Println()
				ux.Info("Release URL: %s", result.ReleaseURL)
				ux.Info("Asset URL: %s", result.AssetURL)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without creating release")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip validation step")
	cmd.Flags().BoolVar(&draft, "draft", false, "Create release as draft")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark release as prerelease")

	return cmd
}

func NewRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage the plugin and theme registry",
	}

	cmd.AddCommand(
		NewRegistrySyncCommand(),
	)

	return cmd
}

func NewRegistrySyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync local registry cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Warning("Registry sync coming soon!")
			return nil
		},
	}
}

func normalizeThemeID(name string) string {
	return theme.NormalizeID(name)
}
