package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cortex/cli/internal/publish"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

type PublishOptions struct{}

type PublishResult struct {
	PluginID            string
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

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	metadata := pluginMetadata(m)
	if _, err := publish.ResolveMetadata(absDir, metadata); err != nil {
		return nil, err
	}

	ux.Info("Preparing plugin release for %s v%s", m.Name, m.Version)

	if err := publish.CleanOutput(absDir); err != nil {
		return nil, err
	}

	ux.Step("Building plugin...")
	if err := Build(absDir, BuildOptions{}); err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
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

	assets, err := pluginReleaseAssets(absDir, m)
	if err != nil {
		return nil, err
	}

	archiveFiles, err := pluginArchiveFiles(absDir)
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

	printPreparedRelease(prepared)

	return &PublishResult{
		PluginID:            m.ID,
		Version:             m.Version,
		OutputDir:           prepared.OutputDir,
		PublishManifestPath: prepared.PublishManifestPath,
		ReleaseNotesPath:    prepared.ReleaseNotesPath,
		ArchivePath:         prepared.ArchivePath,
		Assets:              prepared.Manifest.Assets,
	}, nil
}

func pluginMetadata(m *manifest.PluginManifest) publish.MetadataInput {
	return publish.MetadataInput{
		Kind:           publish.KindPlugin,
		ID:             m.ID,
		Name:           m.Name,
		Version:        m.Version,
		Author:         m.Author,
		AuthorURL:      m.AuthorURL,
		Description:    m.Description,
		Repository:     m.Repository,
		InstallCommand: fmt.Sprintf("cortex plugin install %s", m.ID),
	}
}

func pluginReleaseAssets(dir string, m *manifest.PluginManifest) ([]publish.SourceAsset, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	mainPath := filepath.Join(dir, m.Main)
	if !fileExists(manifestPath) {
		return nil, fmt.Errorf("manifest.json not found")
	}
	if !fileExists(mainPath) {
		return nil, fmt.Errorf("main entry file not found: %s", m.Main)
	}

	assets := []publish.SourceAsset{
		{Name: "manifest.json", Path: manifestPath},
		{Name: filepath.Base(m.Main), Path: mainPath},
	}

	stylesPath := filepath.Join(dir, "styles.css")
	if fileExists(stylesPath) {
		assets = append(assets, publish.SourceAsset{Name: "styles.css", Path: stylesPath})
	}

	return assets, nil
}

func pluginArchiveFiles(dir string) ([]publish.ArchiveFile, error) {
	archiveFiles := []publish.ArchiveFile{}
	for _, name := range []string{"manifest.json", "package.json", "README.md", "LICENSE"} {
		path := filepath.Join(dir, name)
		if fileExists(path) {
			archiveFiles = append(archiveFiles, publish.ArchiveFile{Name: name, Path: path})
		}
	}

	distDir := filepath.Join(dir, "dist")
	outputDir := publish.OutputDir(dir)
	if err := filepath.WalkDir(distDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == outputDir {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		archiveFiles = append(archiveFiles, publish.ArchiveFile{
			Name: filepath.ToSlash(relPath),
			Path: path,
		})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to read dist directory: %w", err)
	}

	return archiveFiles, nil
}

func printPreparedRelease(result *publish.Result) {
	ux.Success("Publish artifacts ready: %s", ux.Path(result.OutputDir))
	ux.Info("Publish manifest: %s", ux.Path(result.PublishManifestPath))
	ux.Info("Release notes: %s", ux.Path(result.ReleaseNotesPath))
	ux.Info("Release assets:")
	for _, asset := range result.Manifest.Assets {
		ux.Info("  - %s", asset.Name)
	}
}
