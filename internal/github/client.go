package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh    *github.Client
	owner string
	repo  string
}

var ErrNotFound = errors.New("github resource not found")

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

func isGitHubNotFound(err error) bool {
	var githubError *github.ErrorResponse
	return errors.As(err, &githubError) && githubError.Response != nil && githubError.Response.StatusCode == http.StatusNotFound
}
