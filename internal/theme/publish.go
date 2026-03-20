package theme

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cortex/cli/internal/auth"
	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/gitx"
	"github.com/cortex/cli/internal/registry"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/cortex/cli/pkg/zipx"
	"github.com/google/go-github/v60/github"
)

type PublishOptions struct {
	DryRun         bool
	SkipValidate   bool
	Draft          bool
	Prerelease     bool
	CoverImageURL  string
	Author         string
	Description    string
	Repository     string
	UpdateOnly     bool
	SkipGitSync    bool
	SkipRegistryPR bool
}

type PublishResult struct {
	ThemeID    string
	Version    string
	ReleaseURL string
	AssetURL   string
	RegistryPR string
}

func Publish(dir string, opts PublishOptions) (*PublishResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	ux.Info("Publishing %s v%s", m.Name, m.Version)

	if !opts.SkipValidate {
		ux.Step("Validating theme...")
		result, err := Validate(absDir, ValidateOptions{Strict: true})
		if err != nil {
			return nil, fmt.Errorf("validation error: %w", err)
		}
		if !result.Passed {
			return nil, fmt.Errorf("validation failed, fix errors before publishing")
		}
		ux.Success("Validation passed")
	}

	ux.Step("Creating release archive...")
	zipPath, err := createThemeArchive(absDir, m)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}
	defer os.Remove(zipPath)

	zipInfo, _ := os.Stat(zipPath)
	ux.Info("Archive size: %.2f KB", float64(zipInfo.Size())/1024)

	if opts.DryRun {
		ux.Success("Dry run complete - no release created")
		return &PublishResult{
			ThemeID: m.ID,
			Version: m.Version,
		}, nil
	}

	token, err := auth.ResolveToken()
	if err != nil {
		return nil, fmt.Errorf("github token unavailable: run `cortex login` or set GITHUB_TOKEN")
	}

	ctx := context.Background()
	client := gh.NewClient(token)

	owner, repo, repoErr := resolveThemeRepository(ctx, client, absDir, m, opts)
	if repoErr != nil {
		return nil, repoErr
	}

	client.SetRepo(owner, repo)

	if !opts.SkipGitSync {
		remoteURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		ux.Step("Syncing local changes to repository...")
		if syncErr := gitx.SyncForPublish(gitx.SyncOptions{
			Directory:     absDir,
			RemoteURL:     remoteURL,
			CommitMessage: fmt.Sprintf("chore: publish %s v%s", m.ID, m.Version),
		}); syncErr != nil {
			return nil, fmt.Errorf("failed to sync repository before release: %w", syncErr)
		}
		ux.Success("Repository synced")
	} else {
		ux.Info("Skipping git sync")
	}

	ux.Step("Creating GitHub release...")
	tagName := fmt.Sprintf("v%s", m.Version)
	releaseName := fmt.Sprintf("%s v%s", m.Name, m.Version)
	releaseBody := generateThemeReleaseBody(m)

	release, created, err := upsertThemeRelease(ctx, client, tagName, releaseName, releaseBody, opts)
	if err != nil {
		return nil, err
	}

	ux.Step("Uploading release asset...")
	assetName := fmt.Sprintf("%s-%s.zip", m.ID, m.Version)
	asset, err := client.UpsertReleaseAsset(ctx, release, gh.ReleaseAsset{
		Name: assetName,
		Path: zipPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset: %w", err)
	}

	if created {
		ux.Success("Release created: %s", release.GetHTMLURL())
	} else {
		ux.Success("Release updated: %s", release.GetHTMLURL())
	}

	author := m.Author
	if strings.TrimSpace(opts.Author) != "" {
		author = strings.TrimSpace(opts.Author)
	}

	description := m.Description
	if strings.TrimSpace(opts.Description) != "" {
		description = strings.TrimSpace(opts.Description)
	}

	repoRef := m.Repository
	if strings.TrimSpace(opts.Repository) != "" {
		repoRef = strings.TrimSpace(opts.Repository)
	}
	if repoRef == "" {
		repoRef = fmt.Sprintf("%s/%s", owner, repo)
	}
	if strings.TrimSpace(repoRef) == "" {
		return nil, fmt.Errorf("repository is required for registry publish")
	}

	if !created || opts.UpdateOnly || opts.SkipRegistryPR {
		ux.Info("Skipping registry PR")
		return &PublishResult{
			ThemeID:    m.ID,
			Version:    m.Version,
			ReleaseURL: release.GetHTMLURL(),
			AssetURL:   asset.GetBrowserDownloadURL(),
		}, nil
	}

	ux.Step("Creating registry pull request...")
	prURL, err := registry.PublishToRegistry(ctx, client, registry.PublishIndexOptions{
		RegistryOwner: "cortex-md",
		RegistryRepo:  "registry",
		BaseBranch:    "main",
		IndexFile:     "themes.json",
		Kind:          "theme",
		Entry: registry.IndexEntry{
			ID:            m.ID,
			Name:          m.Name,
			Author:        author,
			Description:   description,
			CoverImageURL: opts.CoverImageURL,
			Repo:          repoRef,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("release created but registry PR failed: %w", err)
	}

	ux.Success("Registry PR created: %s", prURL)

	return &PublishResult{
		ThemeID:    m.ID,
		Version:    m.Version,
		ReleaseURL: release.GetHTMLURL(),
		AssetURL:   asset.GetBrowserDownloadURL(),
		RegistryPR: prURL,
	}, nil
}

func upsertThemeRelease(ctx context.Context, client *gh.Client, tagName string, releaseName string, releaseBody string, opts PublishOptions) (*github.RepositoryRelease, bool, error) {
	existing, err := client.GetRelease(ctx, tagName)
	if err == nil {
		ux.Step("Release already exists, updating metadata...")
		updated, updateErr := client.UpdateRelease(ctx, existing.GetID(), gh.ReleaseOptions{
			TagName:    tagName,
			Name:       releaseName,
			Body:       releaseBody,
			Draft:      opts.Draft,
			Prerelease: opts.Prerelease,
		})
		if updateErr != nil {
			return nil, false, fmt.Errorf("failed to update release: %w", updateErr)
		}
		return updated, false, nil
	}

	if !errors.Is(err, gh.ErrNotFound) {
		return nil, false, fmt.Errorf("failed to inspect release: %w", err)
	}

	created, createErr := client.CreateRelease(ctx, gh.ReleaseOptions{
		TagName:    tagName,
		Name:       releaseName,
		Body:       releaseBody,
		Draft:      opts.Draft,
		Prerelease: opts.Prerelease,
	})
	if createErr != nil {
		return nil, false, fmt.Errorf("failed to create release: %w", createErr)
	}

	return created, true, nil
}

func createThemeArchive(dir string, m *manifest.ThemeManifest) (string, error) {
	tempDir := os.TempDir()
	zipPath := filepath.Join(tempDir, fmt.Sprintf("%s-%s.zip", m.ID, m.Version))

	filesToInclude := []string{
		"manifest.json",
		"package.json",
		"README.md",
		"LICENSE",
	}

	fileMap := make(map[string][]byte)

	for _, file := range filesToInclude {
		path := filepath.Join(dir, file)
		if data, err := os.ReadFile(path); err == nil {
			fileMap[file] = data
		}
	}

	for _, cssPath := range m.Colorschemes {
		fullPath := filepath.Join(dir, cssPath)
		if data, err := os.ReadFile(fullPath); err == nil {
			fileMap[cssPath] = data
		}
	}

	if err := zipx.CreateFromBytes(zipPath, fileMap); err != nil {
		return "", fmt.Errorf("failed to create zip: %w", err)
	}

	return zipPath, nil
}

func getGitRemoteURL(dir string) (string, error) {
	gitDir := filepath.Join(dir, ".git")
	configPath := filepath.Join(gitDir, "config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("not a git repository or no remote configured")
	}

	lines := strings.Split(string(data), "\n")
	inRemoteOrigin := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[remote \"origin\"]" {
			inRemoteOrigin = true
			continue
		}

		if inRemoteOrigin && strings.HasPrefix(line, "[") {
			break
		}

		if inRemoteOrigin && strings.HasPrefix(line, "url = ") {
			return strings.TrimPrefix(line, "url = "), nil
		}
	}

	return "", fmt.Errorf("no origin remote found")
}

func parseGitHubURL(url string) (owner, repo string) {
	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.TrimPrefix(url, "git@github.com:")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1]
		}
	}

	if strings.Contains(url, "github.com-") && strings.Contains(url, ":") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			segments := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
			if len(segments) >= 2 {
				return segments[0], strings.TrimSuffix(segments[1], ".git")
			}
		}
	}

	if strings.Contains(url, "github.com/") {
		idx := strings.Index(url, "github.com/")
		parts := url[idx+len("github.com/"):]
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1]
		}
	}

	return "", ""
}

func generateThemeReleaseBody(m *manifest.ThemeManifest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", m.Name))
	if m.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", m.Description))
	}

	sb.WriteString("### Colorschemes\n\n")
	for scheme := range m.Colorschemes {
		sb.WriteString(fmt.Sprintf("- %s\n", scheme))
	}
	sb.WriteString("\n")

	sb.WriteString("### Installation\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cortex theme install %s\n", m.ID))
	sb.WriteString("```\n")

	return sb.String()
}

func resolveThemeRepository(ctx context.Context, client *gh.Client, absDir string, m *manifest.ThemeManifest, opts PublishOptions) (string, string, error) {
	repositoryRef := strings.TrimSpace(opts.Repository)
	if repositoryRef == "" {
		repositoryRef = strings.TrimSpace(m.Repository)
	}

	if repositoryRef == "" {
		if remoteURL, remoteErr := getGitRemoteURL(absDir); remoteErr == nil {
			repositoryRef = remoteURL
		}
	}

	if repositoryRef != "" {
		owner, repo := parseGitHubURL(repositoryRef)
		if owner != "" && repo != "" {
			return owner, repo, nil
		}
		return "", "", fmt.Errorf("could not parse GitHub repository from %s", repositoryRef)
	}

	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve authenticated user for repository creation: %w", err)
	}

	owner := user.GetLogin()
	if owner == "" {
		return "", "", fmt.Errorf("could not resolve GitHub username")
	}

	repoName := sanitizeThemeRepositoryName(m.ID)
	if repoName == "" {
		repoName = sanitizeThemeRepositoryName(m.Name)
	}
	if repoName == "" {
		repoName = "cortex-theme"
	}

	existingRepository, existingErr := client.GetRepository(ctx, owner, repoName)
	if existingErr == nil {
		ux.Info("Using existing repository: %s", existingRepository.GetHTMLURL())
		return owner, repoName, nil
	}

	ux.Step("No repository found, creating public repository %s/%s...", owner, repoName)
	repository, createErr := client.CreateRepository(ctx, gh.CreateRepositoryOptions{
		Name:        repoName,
		Description: firstNonEmptyThemeString(opts.Description, m.Description),
		Private:     false,
	})
	if createErr != nil {
		return "", "", fmt.Errorf("failed to create repository automatically: %w", createErr)
	}

	ux.Success("Repository created: %s", repository.GetHTMLURL())

	return owner, repoName, nil
}

func sanitizeThemeRepositoryName(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(trimmed, "-")
	trimmed = strings.Trim(trimmed, "-._")
	if len(trimmed) > 100 {
		return trimmed[:100]
	}
	return trimmed
}

func firstNonEmptyThemeString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
