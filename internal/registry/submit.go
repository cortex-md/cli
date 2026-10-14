package registry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/ux"
)

type Action string

const (
	ActionAdd    Action = "add"
	ActionUpdate Action = "update"
	ActionNone   Action = "none"
)

type Plan struct {
	Submission *Submission
	Action     Action
	Existing   *IndexEntry
	Next       IndexEntry
	Content    string
}

type Result struct {
	URL     string
	Updated bool
}

func PlanSubmission(ctx context.Context, client *gh.Client, s *Submission) (*Plan, error) {
	content, _, err := client.FileContent(ctx, Owner, Repo, s.IndexFile, BaseBranch)
	if err != nil && !errors.Is(err, gh.ErrNotFound) {
		return nil, err
	}

	entries, err := ParseEntries(content)
	if err != nil {
		return nil, err
	}

	existing := FindEntry(entries, s.Entry.ID)
	plan := &Plan{Submission: s, Existing: existing, Action: ActionAdd}
	if existing != nil {
		plan.Action = ActionUpdate
	}

	entries = upsertEntry(entries, s.Entry)
	plan.Next = *FindEntry(entries, s.Entry.ID)
	if existing != nil && *existing == plan.Next {
		plan.Action = ActionNone
		return plan, nil
	}

	plan.Content, err = CanonicalJSON(entries)
	return plan, err
}

func (p *Plan) Title() string {
	verb := "add"
	if p.Action == ActionUpdate {
		verb = "update"
	}
	return fmt.Sprintf("feat(registry): %s %s %s", verb, p.Submission.Kind, p.Submission.Entry.ID)
}

func Open(ctx context.Context, client *gh.Client, plan *Plan, login string, releaseURL string) (*Result, error) {
	if err := ensureFork(ctx, client, login); err != nil {
		return nil, err
	}

	if err := client.SyncFork(ctx, login, Repo, BaseBranch); err != nil {
		ux.Warning("Could not sync your fork with upstream; continuing from the upstream commit")
	}

	upstreamSHA, err := client.BranchSHA(ctx, Owner, Repo, BaseBranch)
	if err != nil {
		return nil, err
	}

	branch := plan.Submission.BranchName()
	if err := client.ResetBranch(ctx, login, Repo, branch, upstreamSHA); err != nil {
		return nil, err
	}

	_, fileSHA, err := client.FileContent(ctx, login, Repo, plan.Submission.IndexFile, branch)
	if err != nil && !errors.Is(err, gh.ErrNotFound) {
		return nil, err
	}
	if err := client.PutFile(ctx, login, Repo, plan.Submission.IndexFile, branch, plan.Title(), plan.Content, fileSHA); err != nil {
		return nil, err
	}

	head := login + ":" + branch
	body := plan.Body(releaseURL)
	existing, err := client.FindOpenPullRequest(ctx, Owner, Repo, head)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		pull, err := client.UpdatePullRequest(ctx, Owner, Repo, existing.GetNumber(), plan.Title(), body)
		if err != nil {
			return nil, err
		}
		return &Result{URL: pull.GetHTMLURL(), Updated: true}, nil
	}

	pull, err := client.CreatePullRequest(ctx, Owner, Repo, gh.PullRequestOptions{
		Title: plan.Title(),
		Body:  body,
		Head:  head,
		Base:  BaseBranch,
	})
	if err != nil {
		return nil, err
	}
	return &Result{URL: pull.GetHTMLURL()}, nil
}

func ensureFork(ctx context.Context, client *gh.Client, login string) error {
	fork, err := client.Repository(ctx, login, Repo)
	if err == nil {
		parent := fork.GetParent()
		if !fork.GetFork() || parent == nil || parent.GetFullName() != Owner+"/"+Repo {
			return ux.NewProblem(
				fmt.Sprintf("github.com/%s/%s exists but is not a fork of %s/%s", login, Repo, Owner, Repo),
				"rename that repository, then run `cortex submit` again",
				nil,
			)
		}
		return nil
	}
	if !errors.Is(err, gh.ErrNotFound) {
		return err
	}

	if err := client.Fork(ctx, Owner, Repo); err != nil {
		return err
	}

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := client.Repository(ctx, login, Repo); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
	return errors.New("timed out waiting for GitHub to create your registry fork")
}

func (p *Plan) Body(releaseURL string) string {
	s := p.Submission
	pluginBox, themeBox := "x", " "
	if s.Kind == "theme" {
		pluginBox, themeBox = " ", "x"
	}

	var builder strings.Builder
	builder.WriteString("## Submission Type\n\n")
	builder.WriteString(fmt.Sprintf("- [%s] Plugin registry entry\n- [%s] Theme registry entry\n- [ ] Registry infrastructure only\n\n", pluginBox, themeBox))
	builder.WriteString("## Release Checklist\n\n")
	builder.WriteString("Verified by `cortex submit` before opening this pull request:\n\n")
	builder.WriteString("- [x] The repository in `repo` is public and uses `owner/repo` format.\n")
	builder.WriteString("- [x] The repository has a public latest GitHub release.\n")
	builder.WriteString("- [x] The latest release includes `manifest.json`.\n")
	builder.WriteString("- [x] The latest release `manifest.id` matches the registry entry `id`.\n")
	builder.WriteString("- [x] The release tag matches `manifest.version` (plain `MAJOR.MINOR.PATCH`).\n")
	if s.Kind == "theme" {
		builder.WriteString("- [x] Theme releases include both dark and light stylesheet assets, and `manifest.name` equals the `id`.\n")
	} else {
		builder.WriteString("- [x] Plugin releases include the bundle asset named by `manifest.main`, or an asset with the same basename.\n")
	}
	builder.WriteString("- [x] README.md exists on `main` or `master`.\n\n")
	builder.WriteString("## Review Notes\n\n")
	builder.WriteString(fmt.Sprintf("Release URL: %s\n\n", releaseURL))
	builder.WriteString(fmt.Sprintf("Repository: https://github.com/%s\n\n", s.Entry.Repo))
	builder.WriteString("Registry entry:\n\n```diff\n")
	builder.WriteString(p.Diff())
	builder.WriteString("```\n")
	return builder.String()
}

func (p *Plan) Diff() string {
	var builder strings.Builder
	before := entryLines(p.Existing)
	after := entryLines(&p.Next)
	for index := range after {
		if index < len(before) && before[index] == after[index] {
			builder.WriteString("  " + after[index] + "\n")
			continue
		}
		if index < len(before) && before[index] != "" {
			builder.WriteString("- " + before[index] + "\n")
		}
		builder.WriteString("+ " + after[index] + "\n")
	}
	return builder.String()
}

func entryLines(entry *IndexEntry) []string {
	if entry == nil {
		return nil
	}
	return []string{
		fmt.Sprintf("id: %s", entry.ID),
		fmt.Sprintf("name: %s", entry.Name),
		fmt.Sprintf("author: %s", entry.Author),
		fmt.Sprintf("authorUrl: %s", entry.AuthorURL),
		fmt.Sprintf("description: %s", entry.Description),
		fmt.Sprintf("coverImageUrl: %s", entry.CoverImageURL),
		fmt.Sprintf("repo: %s", entry.Repo),
	}
}

func DefaultCoverURL(ctx context.Context, client *gh.Client, repository string) string {
	owner, repo, _ := strings.Cut(repository, "/")
	info, err := client.Repository(ctx, owner, repo)
	if err != nil {
		return ""
	}
	branch := info.GetDefaultBranch()
	for _, name := range []string{"cover.png", "cover.jpg", "cover.webp"} {
		if client.FileExists(ctx, owner, repo, name, branch) {
			return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repository, branch, name)
		}
	}
	return ""
}
