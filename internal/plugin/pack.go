package plugin

import (
	"fmt"
	"path/filepath"

	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/pkg/manifest"
)

func InstallFiles(dir string, m *manifest.PluginManifest) ([]build.File, error) {
	mainPath, ok := rules.SafeRelativePath(m.Main)
	if !ok {
		return nil, fmt.Errorf("manifest.main %q is not a safe relative path", m.Main)
	}

	releaseManifest, err := manifest.ReleasePluginManifest(dir)
	if err != nil {
		return nil, err
	}

	bundlePath := filepath.Join(dir, filepath.FromSlash(mainPath))
	if !fileExists(bundlePath) {
		return nil, fmt.Errorf("bundle %s not found; run the build first", mainPath)
	}

	files := []build.File{
		{Path: manifest.FileName, Content: releaseManifest},
		{Path: mainPath, Source: bundlePath},
	}

	stylesPath := filepath.Join(dir, "styles.css")
	if fileExists(stylesPath) {
		files = append(files, build.File{Path: "styles.css", Source: stylesPath})
	}

	return files, nil
}

func ReleaseAssets(dir string, m *manifest.PluginManifest) ([]build.SourceAsset, error) {
	files, err := InstallFiles(dir, m)
	if err != nil {
		return nil, err
	}

	assets := make([]build.SourceAsset, 0, len(files))
	for _, file := range files {
		assets = append(assets, build.SourceAsset{
			Name:    filepath.Base(filepath.FromSlash(file.Path)),
			Path:    file.Source,
			Content: file.Content,
		})
	}
	return assets, nil
}

func Metadata(m *manifest.PluginManifest) build.MetadataInput {
	return build.MetadataInput{
		Kind:          build.KindPlugin,
		ID:            m.ID,
		Name:          m.Name,
		Version:       m.Version,
		Author:        m.Author,
		AuthorURL:     m.AuthorURL,
		Description:   m.Description,
		CoverImageURL: m.CoverImageURL,
		Repository:    m.Repository,
	}
}
