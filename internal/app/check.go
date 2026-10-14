package app

import (
	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

func newCheckCommand() *cobra.Command {
	var strict bool
	var skipBuild bool

	cmd := &cobra.Command{
		Use:   "check [directory]",
		Short: "Build and run every Marketplace check",
		Long: `Builds the plugin, loads the bundle the same way Cortex does, and runs the checks
the Marketplace registry applies. Themes are checked without a build.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := project.Read(projectDir(args))
			if err != nil {
				return err
			}
			ux.Step("Checking %s %s", kindLabel(p), ux.Command(p.DisplayName()))
			return runChecks(p, strict, skipBuild)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Fail on warnings too (used in CI)")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Check the existing build output")
	return cmd
}

func newPackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pack [directory]",
		Short: "Write release assets to dist/cortex-build (used by the release workflow)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := project.Load(projectDir(args))
			if err != nil {
				return err
			}
			if err := runChecks(p, false, false); err != nil {
				return err
			}
			return packProject(p)
		},
	}
}

func packProject(p *project.Project) error {
	assets, metadata, err := releaseAssets(p)
	if err != nil {
		return err
	}

	result, err := build.Prepare(build.PrepareOptions{Directory: p.Dir, Metadata: metadata, Assets: assets})
	if err != nil {
		return ux.NewProblem("Could not write release assets", "run `cortex check` and fix the reported issues", err)
	}

	ux.Blank()
	ux.Success("Release %s ready in %s", result.Manifest.TagName, ux.Path(relativeToWorkingDir(result.OutputDir)))
	for _, asset := range result.Manifest.Assets {
		ux.Info("%s", asset.Name)
	}
	return nil
}
