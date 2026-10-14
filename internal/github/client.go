package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

var ErrNotFound = errors.New("github resource not found")

type Client struct {
	gh   *github.Client
	http *http.Client
}

type Repository = github.Repository
type Release = github.RepositoryRelease
type ReleaseAsset = github.ReleaseAsset
type PullRequest = github.PullRequest

type User struct {
	Login  string
	Scopes []string
}

type PullRequestOptions struct {
	Title string
	Body  string
	Head  string
	Base  string
}

func NewClient(token string) *Client {
	httpClient := &http.Client{}
	if token != "" {
		httpClient = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	}
	httpClient.Timeout = 30 * time.Second
	return &Client{gh: github.NewClient(httpClient), http: httpClient}
}

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	user, response, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to read the authenticated GitHub user: %w", err)
	}

	scopes := []string{}
	for _, scope := range strings.Split(response.Header.Get("X-OAuth-Scopes"), ",") {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			scopes = append(scopes, trimmed)
		}
	}
	return &User{Login: user.GetLogin(), Scopes: scopes}, nil
}

func (c *Client) Repository(ctx context.Context, owner, repo string) (*Repository, error) {
	repository, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	return repository, wrap(err, "failed to read repository %s/%s", owner, repo)
}

func (c *Client) IsPublicOrgMember(ctx context.Context, org, user string) bool {
	member, _, err := c.gh.Organizations.IsPublicMember(ctx, org, user)
	return err == nil && member
}

func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	release, _, err := c.gh.Repositories.GetLatestRelease(ctx, owner, repo)
	return release, wrap(err, "failed to read the latest release of %s/%s", owner, repo)
}

func (c *Client) ReleaseByTag(ctx context.Context, owner, repo, tag string) (*Release, error) {
	release, _, err := c.gh.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	return release, wrap(err, "failed to read release %s of %s/%s", tag, owner, repo)
}

func (c *Client) DownloadText(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", ErrNotFound
	}
	if response.StatusCode >= 300 {
		return "", fmt.Errorf("download %s returned %s", url, response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return string(body), err
}

func (c *Client) FileContent(ctx context.Context, owner, repo, path, ref string) (string, string, error) {
	file, _, _, err := c.gh.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: ref})
	if err != nil {
		return "", "", wrap(err, "failed to read %s from %s/%s", path, owner, repo)
	}
	if file == nil {
		return "", "", ErrNotFound
	}
	content, err := file.GetContent()
	if err != nil {
		return "", "", fmt.Errorf("failed to decode %s: %w", path, err)
	}
	return content, file.GetSHA(), nil
}

func (c *Client) FileExists(ctx context.Context, owner, repo, path, ref string) bool {
	_, _, err := c.FileContent(ctx, owner, repo, path, ref)
	return err == nil
}

func (c *Client) Fork(ctx context.Context, owner, repo string) error {
	_, _, err := c.gh.Repositories.CreateFork(ctx, owner, repo, &github.RepositoryCreateForkOptions{DefaultBranchOnly: true})
	var accepted *github.AcceptedError
	if errors.As(err, &accepted) {
		return nil
	}
	return wrap(err, "failed to fork %s/%s", owner, repo)
}

func (c *Client) SyncFork(ctx context.Context, owner, repo, branch string) error {
	_, _, err := c.gh.Repositories.MergeUpstream(ctx, owner, repo, &github.RepoMergeUpstreamRequest{Branch: github.String(branch)})
	return wrap(err, "failed to sync fork %s/%s with upstream", owner, repo)
}

func (c *Client) BranchSHA(ctx context.Context, owner, repo, branch string) (string, error) {
	ref, _, err := c.gh.Git.GetRef(ctx, owner, repo, "heads/"+branch)
	if err != nil {
		return "", wrap(err, "failed to read branch %s of %s/%s", branch, owner, repo)
	}
	return ref.GetObject().GetSHA(), nil
}

func (c *Client) ResetBranch(ctx context.Context, owner, repo, branch, sha string) error {
	ref := &github.Reference{
		Ref:    github.String("refs/heads/" + branch),
		Object: &github.GitObject{SHA: github.String(sha)},
	}
	if _, err := c.BranchSHA(ctx, owner, repo, branch); err == nil {
		_, _, err := c.gh.Git.UpdateRef(ctx, owner, repo, ref, true)
		return wrap(err, "failed to reset branch %s", branch)
	}
	_, _, err := c.gh.Git.CreateRef(ctx, owner, repo, ref)
	return wrap(err, "failed to create branch %s", branch)
}

func (c *Client) PutFile(ctx context.Context, owner, repo, path, branch, message, content, sha string) error {
	options := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: []byte(content),
		Branch:  github.String(branch),
	}
	if sha != "" {
		options.SHA = github.String(sha)
	}
	_, _, err := c.gh.Repositories.UpdateFile(ctx, owner, repo, path, options)
	return wrap(err, "failed to write %s", path)
}

func (c *Client) FindOpenPullRequest(ctx context.Context, owner, repo, head string) (*PullRequest, error) {
	pulls, _, err := c.gh.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{State: "open", Head: head})
	if err != nil {
		return nil, wrap(err, "failed to list pull requests of %s/%s", owner, repo)
	}
	if len(pulls) == 0 {
		return nil, nil
	}
	return pulls[0], nil
}

func (c *Client) CreatePullRequest(ctx context.Context, owner, repo string, opts PullRequestOptions) (*PullRequest, error) {
	pull, _, err := c.gh.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title:               github.String(opts.Title),
		Body:                github.String(opts.Body),
		Head:                github.String(opts.Head),
		Base:                github.String(opts.Base),
		MaintainerCanModify: github.Bool(true),
	})
	return pull, wrap(err, "failed to open the pull request")
}

func (c *Client) UpdatePullRequest(ctx context.Context, owner, repo string, number int, title, body string) (*PullRequest, error) {
	pull, _, err := c.gh.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		Title: github.String(title),
		Body:  github.String(body),
	})
	return pull, wrap(err, "failed to update pull request #%d", number)
}

func wrap(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	var githubError *github.ErrorResponse
	if errors.As(err, &githubError) && githubError.Response != nil && githubError.Response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
