package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/cortex/cli/internal/buildinfo"
	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/spf13/cobra"
)

type createOptions struct {
	kind        string
	name        string
	id          string
	description string
	author      string
	directory   string
	asPlugin    bool
	asTheme     bool
	noInstall   bool
	updateCI    bool
}

func newCreateCommand() *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Start a new plugin or theme",
		Example: `  cortex create
  cortex create "Word Count" --plugin
  cortex create sunset --theme --author "Ada"
  cortex create --update-ci`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.updateCI {
				return updateWorkflows(projectDir(args))
			}
			if len(args) > 0 {
				opts.name = args[0]
			}
			return runCreate(&opts)
		},
	}

	cmd.Flags().BoolVar(&opts.asPlugin, "plugin", false, "Create a plugin")
	cmd.Flags().BoolVar(&opts.asTheme, "theme", false, "Create a theme")
	cmd.Flags().StringVar(&opts.id, "id", "", "Identifier (defaults to the name as a slug)")
	cmd.Flags().StringVar(&opts.description, "description", "", "One-sentence description")
	cmd.Flags().StringVar(&opts.author, "author", "", "Author name (defaults to git user.name)")
	cmd.Flags().StringVar(&opts.directory, "dir", "", "Target directory (defaults to the id)")
	cmd.Flags().BoolVar(&opts.noInstall, "no-install", false, "Skip installing dependencies")
	cmd.Flags().BoolVar(&opts.updateCI, "update-ci", false, "Rewrite the GitHub workflows of an existing project")
	cmd.MarkFlagsMutuallyExclusive("plugin", "theme")

	return cmd
}

func runCreate(opts *createOptions) error {
	switch {
	case opts.asPlugin:
		opts.kind = "plugin"
	case opts.asTheme:
		opts.kind = "theme"
	}
	if opts.author == "" {
		opts.author = gitUserName()
	}

	if ux.Interactive() {
		ux.PrintBanner(buildinfo.Version)
		ux.Blank()
		if err := promptCreateOptions(opts); err != nil {
			return err
		}
	} else if opts.kind == "" || opts.name == "" {
		return ux.RequireInteractive("--plugin/--theme and a name")
	}

	if opts.id == "" {
		opts.id = rules.NormalizeID(opts.name)
	}

	var dir string
	var err error
	if opts.kind == "theme" {
		dir, err = theme.Create(theme.CreateOptions{Name: opts.name, ID: opts.id, Description: opts.description, Author: opts.author, Directory: opts.directory})
	} else {
		dir, err = plugin.Create(plugin.CreateOptions{Name: opts.name, ID: opts.id, Description: opts.description, Author: opts.author, Directory: opts.directory})
	}
	if err != nil {
		return ux.NewProblem("Could not create the "+opts.kind, "pick another --id or --dir", err)
	}

	relativeDir := relativeToWorkingDir(dir)
	ux.Success("Created %s %s in %s", opts.kind, ux.Command(opts.name), ux.Path(relativeDir))

	if initGitRepository(dir) {
		ux.Success("Initialized a git repository")
	}

	if opts.kind == "plugin" && !opts.noInstall {
		if err := ux.Task("Installing dependencies", func() error { return plugin.InstallDependencies(dir) }); err != nil {
			ux.PrintError(err)
			ux.Warning("Run the install again inside %s before `cortex dev`", relativeDir)
		}
	}

	ux.NextSteps(
		ux.NextStep{Command: "cd " + relativeDir},
		ux.NextStep{Command: "cortex dev --vault <path>", Hint: "install into a vault and reload on every change"},
		ux.NextStep{Command: "cortex check", Hint: "run every Marketplace check"},
		ux.NextStep{Command: "cortex release", Hint: "publish a version through GitHub Actions"},
	)
	return nil
}

func promptCreateOptions(opts *createOptions) error {
	fields := []huh.Field{}
	if opts.kind == "" {
		opts.kind = "plugin"
		fields = append(fields, huh.NewSelect[string]().
			Title("What do you want to build?").
			Options(
				huh.NewOption("Plugin  – commands, views and editor features", "plugin"),
				huh.NewOption("Theme   – dark and light color schemes", "theme"),
			).
			Value(&opts.kind))
	}
	if opts.name == "" {
		fields = append(fields, huh.NewInput().
			Title("Name").
			Description("Shown in Cortex and on the Marketplace").
			Value(&opts.name).
			Validate(func(value string) error {
				if rules.NormalizeID(value) == "" {
					return errors.New("use at least one letter or number")
				}
				return nil
			}))
	}
	if opts.description == "" {
		fields = append(fields, huh.NewInput().Title("Description").Description("One sentence about what it does").Value(&opts.description))
	}
	if opts.author == "" {
		fields = append(fields, huh.NewInput().Title("Author").Value(&opts.author))
	}
	if len(fields) == 0 {
		return nil
	}

	if err := ux.RunForm(huh.NewGroup(fields...)); err != nil {
		return err
	}

	if opts.id == "" {
		opts.id = rules.NormalizeID(opts.name)
		return ux.RunForm(huh.NewGroup(huh.NewInput().
			Title("Identifier").
			Description("Permanent: used for the folder, the registry and updates").
			Value(&opts.id).
			Validate(func(value string) error {
				if !rules.IsValidID(value) {
					return errors.New("use lowercase letters, numbers and hyphens")
				}
				return nil
			})))
	}
	return nil
}

func updateWorkflows(dir string) error {
	p, err := project.Read(dir)
	if err != nil {
		return err
	}
	if p.Kind == manifest.KindTheme {
		err = theme.UpdateWorkflows(p.Dir)
	} else {
		err = plugin.UpdateWorkflows(p.Dir)
	}
	if err != nil {
		return err
	}
	ux.Success("Updated .github/workflows for this %s", kindLabel(p))
	ux.Info("Releases now publish when `cortex release` pushes a version tag")
	return nil
}

func gitUserName() string {
	output, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func initGitRepository(dir string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	inside := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	inside.Dir = dir
	if inside.Run() == nil {
		return false
	}
	initialize := exec.Command("git", "init", "-q", "-b", "main")
	initialize.Dir = dir
	return initialize.Run() == nil
}

func relativeToWorkingDir(dir string) string {
	workingDir, err := os.Getwd()
	if err != nil {
		return dir
	}
	relative, err := filepath.Rel(workingDir, dir)
	if err != nil || strings.HasPrefix(relative, "..") {
		return dir
	}
	return relative
}
