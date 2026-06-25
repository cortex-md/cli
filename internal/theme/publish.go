package theme

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/cortex/cli/internal/publish"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

type PublishOptions struct{}

type PublishResult struct {
	ThemeID             string
	Version             string
	OutputDir           string
	PublishManifestPath string
	ReleaseNotesPath    string
	ArchivePath         string
	Assets              []publish.Asset
}

func Publish(dir string, _ PublishOptions) (*PublishResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	metadata := themeMetadata(m)
	if _, err := publish.ResolveMetadata(absDir, metadata); err != nil {
		return nil, err
	}

	ux.Info("Preparing theme release for %s v%s", m.Name, m.Version)

	if err := publish.CleanOutput(absDir); err != nil {
		return nil, err
	}

	ux.Step("Running strict validation...")
	result, err := Validate(absDir, ValidateOptions{Strict: true})
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if !result.Passed {
		return nil, fmt.Errorf("validation failed, fix errors before publishing")
	}
	ux.Success("Validation passed")

	assets, err := themeReleaseAssets(absDir, m)
	if err != nil {
		return nil, err
	}

	archiveFiles, err := themeArchiveFiles(absDir, m)
	if err != nil {
		return nil, err
	}

	ux.Step("Writing publish artifacts...")
	prepared, err := publish.Prepare(publish.PrepareOptions{
		Directory:    absDir,
		Metadata:     metadata,
		Assets:       assets,
		ArchiveFiles: archiveFiles,
	})
	if err != nil {
		return nil, err
	}

	printPreparedThemeRelease(prepared)

	return &PublishResult{
		ThemeID:             m.ID,
		Version:             m.Version,
		OutputDir:           prepared.OutputDir,
		PublishManifestPath: prepared.PublishManifestPath,
		ReleaseNotesPath:    prepared.ReleaseNotesPath,
		ArchivePath:         prepared.ArchivePath,
		Assets:              prepared.Manifest.Assets,
	}, nil
}

func themeMetadata(m *manifest.ThemeManifest) publish.MetadataInput {
	return publish.MetadataInput{
		Kind:           publish.KindTheme,
		ID:             m.ID,
		Name:           m.Name,
		Version:        m.Version,
		Author:         m.Author,
		AuthorURL:      m.AuthorURL,
		Description:    m.Description,
		Repository:     m.Repository,
		InstallCommand: fmt.Sprintf("cortex theme install %s", m.ID),
	}
}

func themeReleaseAssets(dir string, m *manifest.ThemeManifest) ([]publish.SourceAsset, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	if !fileExists(manifestPath) {
		return nil, fmt.Errorf("manifest.json not found")
	}

	assets := []publish.SourceAsset{{Name: "manifest.json", Path: manifestPath}}
	seen := map[string]bool{"manifest.json": true}
	for _, schemeName := range sortedColorschemeNames(m.Colorschemes) {
		cleanPath := filepath.Clean(m.Colorschemes[schemeName])
		fullPath := filepath.Join(dir, cleanPath)
		if !fileExists(fullPath) {
			return nil, fmt.Errorf("colorscheme file not found: %s", m.Colorschemes[schemeName])
		}

		assetName := filepath.Base(cleanPath)
		if seen[assetName] {
			continue
		}
		seen[assetName] = true
		assets = append(assets, publish.SourceAsset{Name: assetName, Path: fullPath})
	}

	return assets, nil
}

func themeArchiveFiles(dir string, m *manifest.ThemeManifest) ([]publish.ArchiveFile, error) {
	archiveFiles := []publish.ArchiveFile{}
	for _, name := range []string{"manifest.json", "package.json", "README.md", "LICENSE"} {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			archiveFiles = append(archiveFiles, publish.ArchiveFile{Name: name, Path: path})
		}
	}

	seen := map[string]bool{}
	for _, schemeName := range sortedColorschemeNames(m.Colorschemes) {
		cleanPath := filepath.Clean(m.Colorschemes[schemeName])
		fullPath := filepath.Join(dir, cleanPath)
		if !fileExists(fullPath) {
			return nil, fmt.Errorf("colorscheme file not found: %s", m.Colorschemes[schemeName])
		}
		zipPath := filepath.ToSlash(cleanPath)
		if seen[zipPath] {
			continue
		}
		seen[zipPath] = true
		archiveFiles = append(archiveFiles, publish.ArchiveFile{Name: zipPath, Path: fullPath})
	}

	return archiveFiles, nil
}

func sortedColorschemeNames(colorschemes map[string]string) []string {
	names := make([]string, 0, len(colorschemes))
	for name := range colorschemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func printPreparedThemeRelease(result *publish.Result) {
	ux.Success("Publish artifacts ready: %s", ux.Path(result.OutputDir))
	ux.Info("Publish manifest: %s", ux.Path(result.PublishManifestPath))
	ux.Info("Release notes: %s", ux.Path(result.ReleaseNotesPath))
	ux.Info("Release assets:")
	for _, asset := range result.Manifest.Assets {
		ux.Info("  - %s", asset.Name)
	}
}
