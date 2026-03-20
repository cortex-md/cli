package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	gh "github.com/cortex/cli/internal/github"
	"github.com/google/go-github/v60/github"
)

type IndexEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Author        string `json:"author"`
	Description   string `json:"description"`
	CoverImageURL string `json:"coverImageUrl"`
	Repo          string `json:"repo"`
}

type PublishIndexOptions struct {
	RegistryOwner string
	RegistryRepo  string
	BaseBranch    string
	IndexFile     string
	Kind          string
	Entry         IndexEntry
}

func PublishToRegistry(ctx context.Context, client *gh.Client, opts PublishIndexOptions) (string, error) {
	if opts.RegistryOwner == "" || opts.RegistryRepo == "" {
		return "", fmt.Errorf("registry owner and repo are required")
	}

	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}

	if opts.IndexFile == "" {
		return "", fmt.Errorf("index file is required")
	}

	baseOwner := opts.RegistryOwner
	baseRepo := opts.RegistryRepo

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve authenticated GitHub user: %w", err)
	}

	forkOwner := user.GetLogin()
	if forkOwner == "" {
		return "", fmt.Errorf("could not resolve authenticated GitHub username")
	}

	if _, forkErr := client.GetRepository(ctx, forkOwner, baseRepo); forkErr != nil {
		if _, createForkErr := client.ForkRepository(ctx, baseOwner, baseRepo, gh.ForkOptions{}); createForkErr != nil {
			if !strings.Contains(strings.ToLower(createForkErr.Error()), "job scheduled") {
				return "", fmt.Errorf("failed to create registry fork: %w", createForkErr)
			}
		}

		if waitErr := waitForFork(ctx, client, forkOwner, baseRepo); waitErr != nil {
			return "", fmt.Errorf("failed waiting for registry fork readiness: %w", waitErr)
		}
	}

	client.SetRepo(forkOwner, baseRepo)

	baseBranch, err := waitForBranch(ctx, client, opts.BaseBranch)
	if err != nil {
		return "", fmt.Errorf("failed to load fork branch %s: %w", opts.BaseBranch, err)
	}

	branchName := buildBranchName(opts.Kind, opts.Entry.ID)
	if err := client.CreateBranch(ctx, branchName, baseBranch.GetCommit().GetSHA()); err != nil {
		return "", fmt.Errorf("failed to create registry branch in fork: %w", err)
	}

	entries, sha, err := loadEntries(ctx, client, opts.IndexFile, opts.BaseBranch)
	if err != nil {
		return "", err
	}

	entries = upsertEntry(entries, opts.Entry)
	content, err := marshalEntries(entries)
	if err != nil {
		return "", err
	}

	commitMessage := fmt.Sprintf("chore(registry): update %s %s", opts.Kind, opts.Entry.ID)
	if err := client.CreateOrUpdateFile(ctx, opts.IndexFile, commitMessage, content, branchName, sha); err != nil {
		return "", fmt.Errorf("failed to update registry file: %w", err)
	}

	title := fmt.Sprintf("chore(registry): update %s %s", opts.Kind, opts.Entry.ID)
	body := buildPRBody(opts, fmt.Sprintf("%s:%s", forkOwner, branchName))

	client.SetRepo(baseOwner, baseRepo)
	pr, err := client.CreatePullRequest(ctx, gh.PullRequestOptions{
		Title:               title,
		Body:                body,
		Head:                fmt.Sprintf("%s:%s", forkOwner, branchName),
		Base:                opts.BaseBranch,
		MaintainerCanModify: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create registry pull request: %w", err)
	}

	return pr.GetHTMLURL(), nil
}

func waitForBranch(ctx context.Context, client *gh.Client, branch string) (*github.Branch, error) {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		currentBranch, err := client.GetBranch(ctx, branch)
		if err == nil {
			return currentBranch, nil
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}

	return nil, lastErr
}

func waitForFork(ctx context.Context, client *gh.Client, owner string, repo string) error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		_, err := client.GetRepository(ctx, owner, repo)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(3 * time.Second)
	}

	return lastErr
}

func loadEntries(ctx context.Context, client *gh.Client, filePath, branch string) ([]IndexEntry, *string, error) {
	content, sha, err := client.GetFileContent(ctx, filePath, branch)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []IndexEntry{}, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to load registry file: %w", err)
	}

	if strings.TrimSpace(content) == "" {
		return []IndexEntry{}, stringPtr(sha), nil
	}

	entries := []IndexEntry{}
	if err := json.Unmarshal([]byte(content), &entries); err != nil {
		return nil, nil, fmt.Errorf("failed to parse registry JSON: %w", err)
	}

	return entries, stringPtr(sha), nil
}

func upsertEntry(entries []IndexEntry, next IndexEntry) []IndexEntry {
	updated := false
	for index := range entries {
		if entries[index].ID == next.ID {
			entries[index] = next
			updated = true
			break
		}
	}

	if !updated {
		entries = append(entries, next)
	}

	for left := 0; left < len(entries)-1; left++ {
		for right := left + 1; right < len(entries); right++ {
			if entries[right].ID < entries[left].ID {
				entries[left], entries[right] = entries[right], entries[left]
			}
		}
	}

	return entries
}

func marshalEntries(entries []IndexEntry) (string, error) {
	content, err := json.MarshalIndent(entries, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal registry JSON: %w", err)
	}

	return string(content) + "\n", nil
}

func buildBranchName(kind, id string) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("publish/%s-%s-%s", kind, id, timestamp)
}

func buildPRBody(opts PublishIndexOptions, branchName string) string {
	return fmt.Sprintf("## Summary\n\n- Update %s `%s`\n- Update `%s` with registry metadata\n- Source repository: `%s`\n\n## Links\n\n- Latest release: https://github.com/%s/releases/latest\n- Branch: %s\n", opts.Kind, opts.Entry.ID, opts.IndexFile, opts.Entry.Repo, strings.TrimPrefix(strings.TrimPrefix(opts.Entry.Repo, "https://github.com/"), "git@github.com:"), branchName)
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
