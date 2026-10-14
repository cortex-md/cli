package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/registry"
	"github.com/cortex/cli/internal/release"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

const releaseTimeout = 15 * time.Minute

func newReleaseCommand() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:   "release [patch|minor|major|<version>]",
		Short: "Bump the version, tag it and let GitHub Actions publish it",
		Long: `Runs every check, bumps the version in manifest.json and package.json, commits,
creates the v<version> tag and pushes both. The release workflow then publishes the
GitHub release that Cortex installs from.`,
		Example: `  cortex release patch
  cortex release 1.0.0 --wait`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bump := ""
			if len(args) > 0 {
				bump = args[0]
			}
			return runRelease(cmd.Context(), bump, wait)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the GitHub release and offer to submit it to the Marketplace")
	return cmd
}

func runRelease(ctx context.Context, bump string, wait bool) error {
	p, err := project.Load(".")
	if err != nil {
		return err
	}

	git := release.Git{Dir: p.Dir}
	branch, err := prepareGit(git)
	if err != nil {
		return err
	}

	repository := build.ResolveRepository(p.Dir, repositoryOf(p))
	if repository == "" {
		return ux.NewProblem("No GitHub repository for this project", "add a GitHub remote: git remote add origin git@github.com:<you>/<repo>.git", nil)
	}

	if bump == "" {
		if bump, err = promptBump(p.Version()); err != nil {
			return err
		}
	}
	version, err := release.NextVersion(p.Version(), bump)
	if err != nil {
		return err
	}
	tag := build.TagName(version)
	if git.TagExists(tag) {
		return ux.NewProblem("Tag "+tag+" already exists", "pick a higher version", nil)
	}

	previous := p.Version()
	changed, err := release.WriteVersion(p.Dir, version)
	if err != nil {
		return err
	}
	if p, err = project.Load(p.Dir); err != nil {
		return err
	}

	ux.Step("Releasing %s %s → %s", p.DisplayName(), ux.Muted(build.TagName(previous)), ux.Command(tag))
	if err := runChecks(p, false, false); err != nil {
		restoreFiles(git, changed)
		return err
	}

	ux.Blank()
	confirmed, err := ux.Confirm(fmt.Sprintf("Commit, tag %s and push to origin/%s?", tag, branch), true)
	if err != nil || !confirmed {
		restoreFiles(git, changed)
		if err != nil {
			return err
		}
		return ux.ErrReported
	}

	if err := ux.Task("Committing and tagging "+tag, func() error {
		return git.CommitAndTag(changed, "release: "+tag, tag)
	}); err != nil {
		return err
	}
	if err := ux.Task("Pushing to origin", func() error { return git.Push(branch, tag) }); err != nil {
		return ux.NewProblem("Push failed", "fix the problem and run `git push origin "+branch+" "+tag+"`", err)
	}

	ux.Success("GitHub Actions is publishing %s: %s", tag, ux.Path("https://github.com/"+repository+"/actions"))
	if !wait {
		ux.NextSteps(ux.NextStep{Command: "cortex submit", Hint: "first release only: list it on the Marketplace"})
		return nil
	}

	client := optionalGitHub()
	var published string
	if err := ux.Task("Waiting for the release to be published", func() error {
		result, err := release.WaitForRelease(ctx, client, repository, tag, releaseTimeout)
		if err == nil {
			published = result.GetHTMLURL()
		}
		return err
	}); err != nil {
		return err
	}
	ux.Success("Released: %s", ux.Path(published))

	return offerSubmit(ctx, p)
}

func prepareGit(git release.Git) (string, error) {
	if err := git.EnsureRepository(); err != nil {
		return "", err
	}

	dirty, err := git.DirtyFiles()
	if err != nil {
		return "", err
	}
	if len(dirty) > 0 {
		return "", ux.NewProblem("You have uncommitted changes", "commit or stash them before releasing:\n    "+strings.Join(dirty, "\n    "), nil)
	}

	branch, err := git.Branch()
	if err != nil {
		return "", err
	}
	if _, err := git.Upstream(); err != nil {
		return "", ux.NewProblem("Branch "+branch+" is not pushed to GitHub", "run `git push -u origin "+branch+"` first", nil)
	}
	if err := ux.Task("Fetching origin", git.Fetch); err != nil {
		return "", err
	}
	if behind, err := git.BehindUpstream(); err == nil && behind > 0 {
		return "", ux.NewProblem(fmt.Sprintf("Branch %s is %d commits behind origin", branch, behind), "run `git pull` first", nil)
	}
	return branch, nil
}

func promptBump(current string) (string, error) {
	if !ux.Interactive() {
		return "", ux.RequireInteractive("version bump (patch, minor, major or x.y.z)")
	}
	options := []huh.Option[string]{}
	for _, bump := range []string{"patch", "minor", "major"} {
		next, err := release.NextVersion(current, bump)
		if err != nil {
			return "", err
		}
		options = append(options, huh.NewOption(fmt.Sprintf("%-6s %s → %s", bump, current, next), bump))
	}
	bump := "patch"
	err := ux.RunForm(huh.NewGroup(huh.NewSelect[string]().Title("Which version?").Options(options...).Value(&bump)))
	return bump, err
}

func restoreFiles(git release.Git, files []string) {
	if err := git.Restore(files); err != nil {
		ux.Warning("Could not restore %s: %v", strings.Join(files, ", "), err)
	}
}

func offerSubmit(ctx context.Context, p *project.Project) error {
	submission, err := registry.NewSubmission(p)
	if err != nil {
		return err
	}
	plan, err := registry.PlanSubmission(ctx, optionalGitHub(), submission)
	if err != nil {
		return err
	}
	if plan.Action == registry.ActionNone {
		ux.Success("%s is already on the Marketplace; Cortex users get this update automatically", p.DisplayName())
		return nil
	}

	submit, err := ux.Confirm("Submit "+p.DisplayName()+" to the Marketplace now?", true)
	if err != nil || !submit {
		ux.NextSteps(ux.NextStep{Command: "cortex submit", Hint: "list it on the Marketplace when you are ready"})
		return nil
	}
	return runSubmit(ctx, p, submitOptions{})
}

func repositoryOf(p *project.Project) string {
	if p.Theme != nil {
		return p.Theme.Repository
	}
	return p.Plugin.Repository
}
