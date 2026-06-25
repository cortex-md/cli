package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cortex/cli/internal/dev"
	"github.com/cortex/cli/internal/registry"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

type registrySubmitRunner func(context.Context, string) (string, error)

var defaultBuildManifestPath = filepath.Join("dist", "cortex-build", "build.json")

func NewThemeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage themes",
	}

	cmd.AddCommand(
		NewThemeCreateCommand(),
		NewThemeDevCommand(),
		NewThemeLinkCommand(),
		NewThemeUnlinkCommand(),
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

			if opts.DisplayName == "" {
				var displayName string
				prompt := &survey.Input{
					Message: "Display name:",
					Default: opts.Name,
				}
				if err := survey.AskOne(prompt, &displayName); err != nil {
					return err
				}
				opts.DisplayName = displayName
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

			ux.Step("Creating theme '%s'...", opts.Name)

			if err := theme.Create(opts); err != nil {
				ux.Error("Failed to create theme: %v", err)
				return err
			}

			ux.Success("Theme '%s' created successfully!", opts.Name)
			ux.Info("Directory: %s", opts.ID)
			fmt.Println()
			ux.Info("Next steps:")
			fmt.Println("  " + ux.Command("cd "+opts.ID))
			fmt.Println("  Edit theme-dark.css and theme-light.css to customize your theme")
			fmt.Println("  " + ux.Command("cortex theme validate"))

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Theme ID")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Theme description")
	cmd.Flags().StringVar(&opts.Author, "author", "", "Theme author")
	cmd.Flags().StringVar(&opts.Directory, "dir", "", "Target directory")

	return cmd
}

func NewThemeDevCommand() *cobra.Command {
	var skipLink bool
	var vaultPath string

	cmd := &cobra.Command{
		Use:   "dev [directory]",
		Short: "Start theme development mode with hot reload",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			return dev.StartTheme(dir, dev.ThemeDevOptions{
				SkipLink:  skipLink,
				VaultPath: vaultPath,
			})
		},
	}

	cmd.Flags().BoolVar(&skipLink, "skip-link", false, "Skip linking theme")
	cmd.Flags().StringVar(&vaultPath, "vault", "", "Cortex vault path")

	return cmd
}

func NewThemeLinkCommand() *cobra.Command {
	var vaultPath string

	cmd := &cobra.Command{
		Use:   "link [directory]",
		Short: "Link theme for development",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			_, err := dev.LinkTheme(dir, dev.LinkOptions{VaultPath: vaultPath})
			return err
		},
	}

	cmd.Flags().StringVar(&vaultPath, "vault", "", "Cortex vault path")

	return cmd
}

func NewThemeUnlinkCommand() *cobra.Command {
	var vaultPath string

	cmd := &cobra.Command{
		Use:   "unlink [directory|theme-id]",
		Short: "Unlink development theme",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return dev.UnlinkTheme(".", dev.LinkOptions{VaultPath: vaultPath})
			}

			arg := args[0]
			if _, err := os.Stat(arg); err == nil {
				return dev.UnlinkTheme(arg, dev.LinkOptions{VaultPath: vaultPath})
			}

			return dev.UnlinkThemeByID(arg, dev.LinkOptions{VaultPath: vaultPath})
		},
	}

	cmd.Flags().StringVar(&vaultPath, "vault", "", "Cortex vault path")

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
	return &cobra.Command{
		Use:   "publish [directory]",
		Short: "Prepare theme release artifacts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			result, err := theme.Publish(dir, theme.PublishOptions{})
			if err != nil {
				ux.Error("Publish failed: %v", err)
				return err
			}

			fmt.Println()
			ux.Info("Next steps:")
			fmt.Println("  " + ux.Command("git tag v"+result.Version))
			fmt.Println("  " + ux.Command("git push origin v"+result.Version))
			fmt.Println("  " + ux.Command("cortex registry submit") + "  # after the release is live")

			return nil
		},
	}
}

func NewRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage the plugin and theme registry",
	}

	cmd.AddCommand(
		NewRegistrySyncCommand(),
		NewRegistrySubmitCommand(),
	)

	return cmd
}

func NewRegistrySubmitCommand() *cobra.Command {
	return newRegistrySubmitCommand(registry.Submit)
}

func newRegistrySubmitCommand(run registrySubmitRunner) *cobra.Command {
	return &cobra.Command{
		Use:   "submit [build-json]",
		Short: "Submit prepared release build metadata to the registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			buildJSONPath := defaultBuildManifestPath
			if len(args) > 0 {
				buildJSONPath = args[0]
			}

			ux.Step("Submitting registry metadata...")
			prURL, err := run(cmd.Context(), buildJSONPath)
			if err != nil {
				ux.Error("Registry submit failed: %v", err)
				return err
			}

			ux.Success("Registry PR created: %s", prURL)
			return nil
		},
	}
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
