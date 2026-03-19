package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/cortex/cli/pkg/zipx"
)

type PublishOptions struct {
	DryRun       bool
	SkipBuild    bool
	SkipValidate bool
	Draft        bool
	Prerelease   bool
}

type PublishResult struct {
	PluginID   string
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

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	ux.Info("Publishing %s v%s", m.Name, m.Version)

	if !opts.SkipValidate {
		ux.Step("Validating plugin...")
		result, err := Validate(absDir, ValidateOptions{Strict: true})
		if err != nil {
			return nil, fmt.Errorf("validation error: %w", err)
		}
		if !result.Passed {
			return nil, fmt.Errorf("validation failed, fix errors before publishing")
		}
		ux.Success("Validation passed")
	}

	if !opts.SkipBuild {
		ux.Step("Building plugin...")
		if err := Build(absDir, BuildOptions{}); err != nil {
			return nil, fmt.Errorf("build failed: %w", err)
		}
	}

	ux.Step("Creating release archive...")
	zipPath, err := createReleaseArchive(absDir, m)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}
	defer os.Remove(zipPath)

	zipInfo, _ := os.Stat(zipPath)
	ux.Info("Archive size: %.2f KB", float64(zipInfo.Size())/1024)

	if opts.DryRun {
		ux.Success("Dry run complete - no release created")
		return &PublishResult{
			PluginID: m.ID,
			Version:  m.Version,
		}, nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	repoURL, err := getGitRemoteURL(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get git remote: %w", err)
	}

	owner, repo := parseGitHubURL(repoURL)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("could not parse GitHub repository from remote URL: %s", repoURL)
	}

	ctx := context.Background()
	client := gh.NewClient(token)
	client.SetRepo(owner, repo)

	ux.Step("Creating GitHub release...")
	tagName := fmt.Sprintf("v%s", m.Version)
	releaseName := fmt.Sprintf("%s v%s", m.Name, m.Version)
	releaseBody := generateReleaseBody(m)

	release, err := client.CreateRelease(ctx, gh.ReleaseOptions{
		TagName:    tagName,
		Name:       releaseName,
		Body:       releaseBody,
		Draft:      opts.Draft,
		Prerelease: opts.Prerelease,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	ux.Step("Uploading release asset...")
	assetName := fmt.Sprintf("%s-%s.zip", m.ID, m.Version)
	asset, err := client.UploadReleaseAsset(ctx, release.GetID(), gh.ReleaseAsset{
		Name: assetName,
		Path: zipPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload asset: %w", err)
	}

	ux.Success("Release created: %s", release.GetHTMLURL())

	return &PublishResult{
		PluginID:   m.ID,
		Version:    m.Version,
		ReleaseURL: release.GetHTMLURL(),
		AssetURL:   asset.GetBrowserDownloadURL(),
	}, nil
}

func createReleaseArchive(dir string, m *manifest.PluginManifest) (string, error) {
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

	distDir := filepath.Join(dir, "dist")
	if err := filepath.WalkDir(distDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fileMap[relPath] = data
		return nil
	}); err != nil {
		return "", fmt.Errorf("failed to read dist directory: %w", err)
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

func generateReleaseBody(m *manifest.PluginManifest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", m.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", m.Description))
	sb.WriteString("### Installation\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cortex plugin install %s\n", m.ID))
	sb.WriteString("```\n\n")
	sb.WriteString("### Requirements\n\n")
	sb.WriteString(fmt.Sprintf("- Cortex %s or later\n", m.MinAppVersion))

	return sb.String()
}
