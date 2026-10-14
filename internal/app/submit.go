package app

import (
	"context"
	"fmt"

	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/registry"
	"github.com/cortex/cli/internal/ux"
	"github.com/spf13/cobra"
)

type submitOptions struct {
	dryRun bool
	cover  string
}

func newSubmitCommand() *cobra.Command {
	var opts submitOptions

	cmd := &cobra.Command{
		Use:   "submit [directory]",
		Short: "List the plugin or theme on the Marketplace (registry pull request)",
		Long: `Checks the latest GitHub release, then opens or updates a pull request to
cortex-md/registry from your fork. Only needed for the first release and when the
name, description, author or cover image change: new versions reach users automatically.`,
		Example: `  cortex submit --dry-run
  cortex submit --cover https://example.com/cover.png`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := project.Load(projectDir(args))
			if err != nil {
				return err
			}
			return runSubmit(cmd.Context(), p, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Show the checks and the pull request without opening it")
	cmd.Flags().StringVar(&opts.cover, "cover", "", "https URL of the Marketplace cover image")
	return cmd
}

func runSubmit(ctx context.Context, p *project.Project, opts submitOptions) error {
	submission, err := registry.NewSubmission(p)
	if err != nil {
		return err
	}

	session, err := requireGitHub(ctx)
	if err != nil && !opts.dryRun {
		return err
	}
	client := optionalGitHub()
	if session != nil {
		client = session.Client
	}

	var preflight *registry.Preflight
	_ = ux.Task("Checking the latest release of "+submission.Entry.Repo, func() error {
		login := ""
		if session != nil {
			login = session.User.Login
		}
		preflight = registry.RunPreflight(ctx, client, submission, login)
		if !preflight.Report.Passed(false) {
			return ux.ErrReported
		}
		return nil
	})
	preflight.Report.Print()
	if !preflight.Report.Passed(false) {
		preflight.Report.PrintSummary(false)
		return ux.ErrReported
	}

	if err := resolveCover(ctx, client, submission, opts.cover); err != nil {
		return err
	}

	var plan *registry.Plan
	if err := ux.Task("Comparing with the Marketplace registry", func() error {
		plan, err = registry.PlanSubmission(ctx, client, submission)
		return err
	}); err != nil {
		return err
	}

	if plan.Action == registry.ActionNone {
		ux.Success("%s is already listed and up to date", p.DisplayName())
		ux.Info("New releases reach Cortex users automatically; submit again only when the listing changes")
		return nil
	}

	ux.Heading("Registry entry (" + submission.IndexFile + ")")
	fmt.Fprint(ux.Stdout, indent(plan.Diff()))

	if opts.dryRun {
		ux.Heading("Pull request: " + plan.Title())
		fmt.Fprint(ux.Stdout, indent(plan.Body(preflight.ReleaseURL)))
		ux.Blank()
		ux.Info("Dry run: nothing was sent to GitHub")
		return nil
	}

	ux.Blank()
	confirmed, err := ux.Confirm(fmt.Sprintf("Open a pull request to %s/%s as @%s?", registry.Owner, registry.Repo, session.User.Login), true)
	if err != nil {
		return err
	}
	if !confirmed {
		return ux.ErrReported
	}

	var result *registry.Result
	if err := ux.Task("Opening the pull request", func() error {
		result, err = registry.Open(ctx, client, plan, session.User.Login, preflight.ReleaseURL)
		return err
	}); err != nil {
		return err
	}

	if result.Updated {
		ux.Success("Updated your open pull request: %s", ux.Path(result.URL))
	} else {
		ux.Success("Pull request opened: %s", ux.Path(result.URL))
	}
	ux.Info("The registry CI validates the release; a maintainer merges it and %s shows up in Marketplace", p.DisplayName())
	return nil
}

func resolveCover(ctx context.Context, client *gh.Client, submission *registry.Submission, explicit string) error {
	if explicit != "" {
		submission.Entry.CoverImageURL = explicit
	}
	if submission.Entry.CoverImageURL != "" {
		if !isHTTPS(submission.Entry.CoverImageURL) {
			return ux.NewProblem("The cover image must be an https URL", "pass --cover https://... or set coverImageUrl in manifest.json", nil)
		}
		return nil
	}

	suggested := registry.DefaultCoverURL(ctx, client, submission.Entry.Repo)
	if suggested == "" {
		ux.Info("No cover image. Add cover.png to the repository root or pass --cover to show one in Marketplace")
		return nil
	}

	use, err := ux.Confirm("Use "+suggested+" as the Marketplace cover?", true)
	if err != nil {
		return err
	}
	if use {
		submission.Entry.CoverImageURL = suggested
	}
	return nil
}
