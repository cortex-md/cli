package app

import (
	"context"
	"errors"

	"github.com/cortex/cli/internal/auth"
	"github.com/cortex/cli/internal/build"
	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/report"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

func projectDir(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

func kindLabel(p *project.Project) string {
	if p.Kind == manifest.KindTheme {
		return "theme"
	}
	return "plugin"
}

func buildProject(p *project.Project) error {
	if p.Kind != manifest.KindPlugin {
		return nil
	}
	return ux.Task("Building "+p.DisplayName(), func() error {
		return plugin.Build(p.Dir)
	})
}

func checkProject(p *project.Project) *report.Report {
	if p.Kind == manifest.KindTheme {
		return theme.Check(p.Dir, p.Theme)
	}
	return plugin.Check(p.Dir, p.Plugin)
}

func runChecks(p *project.Project, strict bool, skipBuild bool) error {
	if !skipBuild {
		if err := buildProject(p); err != nil {
			return err
		}
	}

	result := checkProject(p)
	result.Print()
	result.PrintSummary(strict)
	if !result.Passed(strict) {
		return ux.ErrReported
	}
	return nil
}

func installFiles(p *project.Project) ([]build.File, error) {
	if p.Kind == manifest.KindTheme {
		return theme.InstallFiles(p.Dir, p.Theme)
	}
	return plugin.InstallFiles(p.Dir, p.Plugin)
}

func releaseAssets(p *project.Project) ([]build.SourceAsset, build.MetadataInput, error) {
	if p.Kind == manifest.KindTheme {
		assets, err := theme.ReleaseAssets(p.Dir, p.Theme)
		return assets, theme.Metadata(p.Theme), err
	}
	assets, err := plugin.ReleaseAssets(p.Dir, p.Plugin)
	return assets, plugin.Metadata(p.Plugin), err
}

type githubSession struct {
	Client *gh.Client
	User   *gh.User
	Source auth.TokenSource
}

func requireGitHub(ctx context.Context) (*githubSession, error) {
	token, source, err := auth.ResolveToken()
	if errors.Is(err, auth.ErrNoToken) {
		return nil, ux.NewProblem("You are not logged in to GitHub", "run `cortex login` (or `gh auth login`, or set CORTEX_TOKEN)", nil)
	}
	if err != nil {
		return nil, err
	}

	client := gh.NewClient(token)
	user, err := client.CurrentUser(ctx)
	if err != nil {
		return nil, ux.NewProblem("GitHub rejected the token from "+string(source), "run `cortex login` again", err)
	}
	return &githubSession{Client: client, User: user, Source: source}, nil
}

func optionalGitHub() *gh.Client {
	token, _, err := auth.ResolveToken()
	if err != nil {
		return gh.NewClient("")
	}
	return gh.NewClient(token)
}
