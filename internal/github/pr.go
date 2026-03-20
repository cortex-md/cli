package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v60/github"
)

type PullRequestOptions struct {
	Title               string
	Body                string
	Head                string
	Base                string
	MaintainerCanModify bool
}

type ForkOptions struct {
	Organization string
}

func (c *Client) CreatePullRequest(ctx context.Context, opts PullRequestOptions) (*github.PullRequest, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	pr := &github.NewPullRequest{
		Title:               github.String(opts.Title),
		Body:                github.String(opts.Body),
		Head:                github.String(opts.Head),
		Base:                github.String(opts.Base),
		MaintainerCanModify: github.Bool(opts.MaintainerCanModify),
	}

	created, _, err := c.gh.PullRequests.Create(ctx, c.owner, c.repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return created, nil
}

func (c *Client) GetPullRequest(ctx context.Context, number int) (*github.PullRequest, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	pr, _, err := c.gh.PullRequests.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	return pr, nil
}

func (c *Client) ListPullRequests(ctx context.Context, state string) ([]*github.PullRequest, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	opts := &github.PullRequestListOptions{
		State:       state,
		ListOptions: github.ListOptions{PerPage: 30},
	}

	prs, _, err := c.gh.PullRequests.List(ctx, c.owner, c.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	return prs, nil
}

func (c *Client) ForkRepository(ctx context.Context, owner, repo string, opts ForkOptions) (*github.Repository, error) {
	forkOpts := &github.RepositoryCreateForkOptions{}
	if opts.Organization != "" {
		forkOpts.Organization = opts.Organization
	}

	fork, _, err := c.gh.Repositories.CreateFork(ctx, owner, repo, forkOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fork repository: %w", err)
	}

	return fork, nil
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	repository, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repository, nil
}

func (c *Client) CreateBranch(ctx context.Context, branchName, baseSHA string) error {
	if c.owner == "" || c.repo == "" {
		return fmt.Errorf("repository not set, call SetRepo first")
	}

	ref := &github.Reference{
		Ref:    github.String("refs/heads/" + branchName),
		Object: &github.GitObject{SHA: github.String(baseSHA)},
	}

	_, _, err := c.gh.Git.CreateRef(ctx, c.owner, c.repo, ref)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

func (c *Client) GetBranch(ctx context.Context, branch string) (*github.Branch, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	b, _, err := c.gh.Repositories.GetBranch(ctx, c.owner, c.repo, branch, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	return b, nil
}

func (c *Client) CreateOrUpdateFile(ctx context.Context, path, message, content, branch string, sha *string) error {
	if c.owner == "" || c.repo == "" {
		return fmt.Errorf("repository not set, call SetRepo first")
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: []byte(content),
		Branch:  github.String(branch),
	}

	if sha != nil {
		opts.SHA = sha
		_, _, err := c.gh.Repositories.UpdateFile(ctx, c.owner, c.repo, path, opts)
		if err != nil {
			return fmt.Errorf("failed to update file: %w", err)
		}

		return nil
	}

	_, _, err := c.gh.Repositories.CreateFile(ctx, c.owner, c.repo, path, opts)
	if err != nil {
		return fmt.Errorf("failed to create/update file: %w", err)
	}

	return nil
}

func (c *Client) GetFileContent(ctx context.Context, path, branch string) (string, string, error) {
	if c.owner == "" || c.repo == "" {
		return "", "", fmt.Errorf("repository not set, call SetRepo first")
	}

	opts := &github.RepositoryContentGetOptions{Ref: branch}

	content, _, _, err := c.gh.Repositories.GetContents(ctx, c.owner, c.repo, path, opts)
	if err != nil {
		var githubError *github.ErrorResponse
		if errors.As(err, &githubError) && githubError.Response != nil && githubError.Response.StatusCode == http.StatusNotFound {
			return "", "", os.ErrNotExist
		}
		return "", "", fmt.Errorf("failed to get file content: %w", err)
	}

	if content == nil {
		return "", "", fmt.Errorf("file not found: %s", path)
	}

	decoded, err := content.GetContent()
	if err != nil {
		return "", "", fmt.Errorf("failed to decode content: %w", err)
	}

	return decoded, content.GetSHA(), nil
}
