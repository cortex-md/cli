package publish

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cortex/cli/pkg/zipx"
)

const (
	SchemaVersion        = 1
	OutputDirName        = "cortex-publish"
	DefaultRegistryOwner = "cortex-md"
	DefaultRegistryRepo  = "registry"
	DefaultRegistryBase  = "main"
)

type Kind string

const (
	KindPlugin Kind = "plugin"
	KindTheme  Kind = "theme"
)

type MetadataInput struct {
	Kind           Kind
	ID             string
	Name           string
	Version        string
	Author         string
	AuthorURL      string
	Description    string
	CoverImageURL  string
	Repository     string
	InstallCommand string
}

type Metadata struct {
	Kind           Kind
	ID             string
	Name           string
	Version        string
	Author         string
	AuthorURL      string
	Description    string
	CoverImageURL  string
	Repository     string
	InstallCommand string
}

type SourceAsset struct {
	Name string
	Path string
}

type ArchiveFile struct {
	Name string
	Path string
}

type Asset struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type RegistryEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Author        string `json:"author"`
	AuthorURL     string `json:"authorUrl,omitempty"`
	Description   string `json:"description"`
	CoverImageURL string `json:"coverImageUrl"`
	Repo          string `json:"repo"`
}

type RegistrySpec struct {
	Owner      string        `json:"owner"`
	Repo       string        `json:"repo"`
	BaseBranch string        `json:"baseBranch"`
	IndexFile  string        `json:"indexFile"`
	Entry      RegistryEntry `json:"entry"`
}

type Manifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	Kind          Kind         `json:"kind"`
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Version       string       `json:"version"`
	TagName       string       `json:"tagName"`
	ReleaseName   string       `json:"releaseName"`
	ReleaseNotes  string       `json:"releaseNotes"`
	Repository    string       `json:"repository"`
	Assets        []Asset      `json:"assets"`
	Registry      RegistrySpec `json:"registry"`
}

type PrepareOptions struct {
	Directory    string
	Metadata     MetadataInput
	Assets       []SourceAsset
	ArchiveFiles []ArchiveFile
}

type Result struct {
	OutputDir           string
	AssetsDir           string
	PublishManifestPath string
	ReleaseNotesPath    string
	ArchivePath         string
	Manifest            Manifest
}

type packageJSON struct {
	Description string      `json:"description"`
	Repository  interface{} `json:"repository"`
	Author      interface{} `json:"author"`
}

func CleanOutput(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	return os.RemoveAll(OutputDir(absDir))
}

func OutputDir(dir string) string {
	return filepath.Join(dir, "dist", OutputDirName)
}

func Prepare(opts PrepareOptions) (*Result, error) {
	absDir, err := filepath.Abs(opts.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	metadata, err := ResolveMetadata(absDir, opts.Metadata)
	if err != nil {
		return nil, err
	}

	if len(opts.Assets) == 0 {
		return nil, fmt.Errorf("release assets are required")
	}
	if len(opts.ArchiveFiles) == 0 {
		return nil, fmt.Errorf("archive files are required")
	}

	outputDir := OutputDir(absDir)
	assetsDir := filepath.Join(outputDir, "assets")
	if err := os.RemoveAll(outputDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return nil, err
	}

	assets, err := copyAssets(assetsDir, opts.Assets)
	if err != nil {
		return nil, err
	}

	archiveName := fmt.Sprintf("%s-%s.zip", metadata.ID, metadata.Version)
	archivePath := filepath.Join(assetsDir, archiveName)
	if err := zipx.Create(archivePath, archiveFileMap(opts.ArchiveFiles)); err != nil {
		return nil, err
	}
	assets = append(assets, Asset{
		Name: archiveName,
		Path: filepath.ToSlash(filepath.Join("assets", archiveName)),
	})

	releaseNotesPath := filepath.Join(outputDir, "release-notes.md")
	if err := os.WriteFile(releaseNotesPath, []byte(buildReleaseNotes(metadata)), 0o644); err != nil {
		return nil, err
	}

	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		Kind:          metadata.Kind,
		ID:            metadata.ID,
		Name:          metadata.Name,
		Version:       metadata.Version,
		TagName:       fmt.Sprintf("v%s", metadata.Version),
		ReleaseName:   fmt.Sprintf("%s v%s", metadata.Name, metadata.Version),
		ReleaseNotes:  "release-notes.md",
		Repository:    metadata.Repository,
		Assets:        assets,
		Registry: RegistrySpec{
			Owner:      DefaultRegistryOwner,
			Repo:       DefaultRegistryRepo,
			BaseBranch: DefaultRegistryBase,
			IndexFile:  indexFileForKind(metadata.Kind),
			Entry: RegistryEntry{
				ID:            metadata.ID,
				Name:          metadata.Name,
				Author:        metadata.Author,
				AuthorURL:     metadata.AuthorURL,
				Description:   metadata.Description,
				CoverImageURL: metadata.CoverImageURL,
				Repo:          metadata.Repository,
			},
		},
	}

	publishManifestPath := filepath.Join(outputDir, "publish.json")
	content, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(publishManifestPath, append(content, '\n'), 0o644); err != nil {
		return nil, err
	}

	return &Result{
		OutputDir:           outputDir,
		AssetsDir:           assetsDir,
		PublishManifestPath: publishManifestPath,
		ReleaseNotesPath:    releaseNotesPath,
		ArchivePath:         archivePath,
		Manifest:            manifest,
	}, nil
}

func ResolveMetadata(dir string, input MetadataInput) (Metadata, error) {
	packageData := loadPackageJSON(dir)

	metadata := Metadata{
		Kind:           input.Kind,
		ID:             strings.TrimSpace(input.ID),
		Name:           strings.TrimSpace(input.Name),
		Version:        strings.TrimSpace(input.Version),
		Author:         firstNonEmpty(input.Author, packageAuthor(packageData)),
		AuthorURL:      firstNonEmpty(input.AuthorURL, packageAuthorURL(packageData)),
		Description:    firstNonEmpty(input.Description, packageDescription(packageData)),
		CoverImageURL:  strings.TrimSpace(input.CoverImageURL),
		Repository:     firstNonEmpty(input.Repository, packageRepository(packageData), gitOriginRepository(dir)),
		InstallCommand: strings.TrimSpace(input.InstallCommand),
	}

	if metadata.InstallCommand == "" {
		metadata.InstallCommand = fmt.Sprintf("cortex %s install %s", metadata.Kind, metadata.ID)
	}

	if metadata.Repository != "" {
		repository, err := NormalizeGitHubRepository(metadata.Repository)
		if err != nil {
			return metadata, err
		}
		metadata.Repository = repository
	}

	if missing := missingMetadata(metadata); len(missing) > 0 {
		return metadata, missingMetadataError(metadata.Kind, missing)
	}

	return metadata, nil
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	if manifest.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported publish schema version: %d", manifest.SchemaVersion)
	}
	if manifest.Kind != KindPlugin && manifest.Kind != KindTheme {
		return nil, fmt.Errorf("unsupported publish kind: %s", manifest.Kind)
	}

	return &manifest, nil
}

func NormalizeGitHubRepository(value string) (string, error) {
	owner, repo := parseGitHubURL(value)
	if owner == "" || repo == "" {
		return "", fmt.Errorf("repository must point to a GitHub repository, for example owner/repo or https://github.com/owner/repo (got %q)", value)
	}
	return owner + "/" + repo, nil
}

func copyAssets(assetsDir string, sources []SourceAsset) ([]Asset, error) {
	assets := []Asset{}
	seen := map[string]bool{}
	for _, source := range sources {
		name := strings.TrimSpace(source.Name)
		if name == "" {
			name = filepath.Base(source.Path)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate release asset name: %s", name)
		}
		seen[name] = true

		targetPath := filepath.Join(assetsDir, name)
		if err := copyFile(source.Path, targetPath); err != nil {
			return nil, err
		}
		assets = append(assets, Asset{
			Name: name,
			Path: filepath.ToSlash(filepath.Join("assets", name)),
		})
	}
	return assets, nil
}

func archiveFileMap(files []ArchiveFile) map[string]string {
	fileMap := map[string]string{}
	for _, file := range files {
		fileMap[filepath.ToSlash(file.Name)] = file.Path
	}
	return fileMap
}

func copyFile(sourcePath string, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", sourcePath, err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", targetPath, err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}

	return nil
}

func buildReleaseNotes(metadata Metadata) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("## %s\n\n", metadata.Name))
	builder.WriteString(metadata.Description)
	builder.WriteString("\n\n")
	builder.WriteString("### Installation\n\n")
	builder.WriteString("```bash\n")
	builder.WriteString(metadata.InstallCommand)
	builder.WriteString("\n")
	builder.WriteString("```\n")
	return builder.String()
}

func missingMetadata(metadata Metadata) []string {
	missing := []string{}
	if metadata.ID == "" {
		missing = append(missing, "id")
	}
	if metadata.Name == "" {
		missing = append(missing, "name")
	}
	if metadata.Version == "" {
		missing = append(missing, "version")
	}
	if metadata.Author == "" {
		missing = append(missing, "author")
	}
	if metadata.Description == "" {
		missing = append(missing, "description")
	}
	if metadata.Repository == "" {
		missing = append(missing, "repository")
	}
	return missing
}

func missingMetadataError(kind Kind, missing []string) error {
	if len(missing) == 1 && missing[0] == "repository" {
		return fmt.Errorf("missing marketplace repository. Add %q to manifest.json, set package.json repository, or configure git remote origin. Use the GitHub repository that will host this %s's releases", "repository: \"owner/repo\"", kind)
	}

	if containsString(missing, "repository") {
		return fmt.Errorf("missing marketplace metadata: %s. Add the missing manifest/package fields before publishing. For repository, use the GitHub repo that will host releases, for example %q, or configure git remote origin", strings.Join(missing, ", "), "repository: \"owner/repo\"")
	}

	return fmt.Errorf("missing marketplace metadata: %s. Set the missing manifest.json fields or package.json metadata before publishing", strings.Join(missing, ", "))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func indexFileForKind(kind Kind) string {
	if kind == KindTheme {
		return "themes.json"
	}
	return "plugins.json"
}

func loadPackageJSON(dir string) *packageJSON {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}

	parsed := &packageJSON{}
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil
	}

	return parsed
}

func packageDescription(pkg *packageJSON) string {
	if pkg == nil {
		return ""
	}
	return strings.TrimSpace(pkg.Description)
}

func packageRepository(pkg *packageJSON) string {
	if pkg == nil || pkg.Repository == nil {
		return ""
	}

	if value, ok := pkg.Repository.(string); ok {
		return strings.TrimSpace(value)
	}

	if value, ok := pkg.Repository.(map[string]interface{}); ok {
		if url, ok := value["url"].(string); ok {
			return strings.TrimSpace(url)
		}
	}

	return ""
}

func packageAuthor(pkg *packageJSON) string {
	if pkg == nil || pkg.Author == nil {
		return ""
	}

	if value, ok := pkg.Author.(string); ok {
		return strings.TrimSpace(value)
	}

	if value, ok := pkg.Author.(map[string]interface{}); ok {
		if name, ok := value["name"].(string); ok {
			return strings.TrimSpace(name)
		}
	}

	return ""
}

func packageAuthorURL(pkg *packageJSON) string {
	if pkg == nil || pkg.Author == nil {
		return ""
	}

	if value, ok := pkg.Author.(map[string]interface{}); ok {
		if url, ok := value["url"].(string); ok {
			return strings.TrimSpace(url)
		}
	}

	return ""
}

func gitOriginRepository(dir string) string {
	command := exec.Command("git", "remote", "get-url", "origin")
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func parseGitHubURL(value string) (owner string, repo string) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "git+")
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.Split(trimmed, "?")[0]
	trimmed = strings.Split(trimmed, "#")[0]

	if !strings.Contains(trimmed, "://") && !strings.Contains(trimmed, "@") && strings.Count(trimmed, "/") == 1 {
		segments := strings.Split(trimmed, "/")
		if isValidGitHubPathSegment(segments[0]) && isValidGitHubPathSegment(segments[1]) {
			return segments[0], segments[1]
		}
	}

	if strings.HasPrefix(trimmed, "git@github.com:") {
		parts := strings.TrimPrefix(trimmed, "git@github.com:")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], strings.TrimSuffix(segments[1], ".git")
		}
	}

	if strings.Contains(trimmed, "github.com-") && strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) == 2 {
			segments := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
			if len(segments) >= 2 {
				return segments[0], strings.TrimSuffix(segments[1], ".git")
			}
		}
	}

	if strings.Contains(trimmed, "github.com/") {
		index := strings.Index(trimmed, "github.com/")
		parts := trimmed[index+len("github.com/"):]
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], strings.TrimSuffix(segments[1], ".git")
		}
	}

	return "", ""
}

func isValidGitHubPathSegment(value string) bool {
	if value == "" || strings.ContainsAny(value, " \\:\t\n\r") {
		return false
	}
	return value != "." && value != ".."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
