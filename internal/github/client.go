package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh    *github.Client
	owner string
	repo  string
}

type ReleaseOptions struct {
	TagName         string
	Name            string
	Body            string
	Draft           bool
	Prerelease      bool
	GenerateNotes   bool
	TargetCommitish string
}

type ReleaseAsset struct {
	Name string
	Path string
}

func NewClient(token string) *Client {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh: github.NewClient(tc),
	}
}

func (c *Client) SetRepo(owner, repo string) {
	c.owner = owner
	c.repo = repo
}

func (c *Client) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}
	return user, nil
}

func (c *Client) CreateRelease(ctx context.Context, opts ReleaseOptions) (*github.RepositoryRelease, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	release := &github.RepositoryRelease{
		TagName:              github.String(opts.TagName),
		Name:                 github.String(opts.Name),
		Body:                 github.String(opts.Body),
		Draft:                github.Bool(opts.Draft),
		Prerelease:           github.Bool(opts.Prerelease),
		GenerateReleaseNotes: github.Bool(opts.GenerateNotes),
	}

	if opts.TargetCommitish != "" {
		release.TargetCommitish = github.String(opts.TargetCommitish)
	}

	created, _, err := c.gh.Repositories.CreateRelease(ctx, c.owner, c.repo, release)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	return created, nil
}

func (c *Client) UploadReleaseAsset(ctx context.Context, releaseID int64, asset ReleaseAsset) (*github.ReleaseAsset, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	file, err := os.Open(asset.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open asset file: %w", err)
	}
	defer file.Close()

	name := asset.Name
	if name == "" {
		name = filepath.Base(asset.Path)
	}

	uploadOpts := &github.UploadOptions{
		Name: name,
	}

	uploaded, _, err := c.gh.Repositories.UploadReleaseAsset(ctx, c.owner, c.repo, releaseID, uploadOpts, file)
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset: %w", err)
	}

	return uploaded, nil
}

func (c *Client) GetRelease(ctx context.Context, tag string) (*github.RepositoryRelease, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	release, _, err := c.gh.Repositories.GetReleaseByTag(ctx, c.owner, c.repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return release, nil
}

func (c *Client) DeleteRelease(ctx context.Context, releaseID int64) error {
	if c.owner == "" || c.repo == "" {
		return fmt.Errorf("repository not set, call SetRepo first")
	}

	_, err := c.gh.Repositories.DeleteRelease(ctx, c.owner, c.repo, releaseID)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	return nil
}

func (c *Client) ListReleases(ctx context.Context) ([]*github.RepositoryRelease, error) {
	if c.owner == "" || c.repo == "" {
		return nil, fmt.Errorf("repository not set, call SetRepo first")
	}

	opts := &github.ListOptions{PerPage: 30}
	releases, _, err := c.gh.Repositories.ListReleases(ctx, c.owner, c.repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	return releases, nil
}
