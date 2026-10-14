package theme

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/pkg/manifest"
)

func InstallFiles(dir string, m *manifest.ThemeManifest) ([]build.File, error) {
	files := []build.File{{Path: manifest.FileName, Source: manifest.Path(dir)}}
	seen := map[string]bool{manifest.FileName: true}

	for _, scheme := range sortedSchemes(m.Colorschemes) {
		relativePath, ok := rules.SafeRelativePath(m.Colorschemes[scheme])
		if !ok {
			return nil, fmt.Errorf("colorschemes.%s %q is not a safe relative path", scheme, m.Colorschemes[scheme])
		}
		if seen[relativePath] {
			continue
		}
		seen[relativePath] = true

		source := filepath.Join(dir, filepath.FromSlash(relativePath))
		if _, err := os.Stat(source); err != nil {
			return nil, fmt.Errorf("%s stylesheet %s not found", scheme, relativePath)
		}
		files = append(files, build.File{Path: relativePath, Source: source})
	}

	return files, nil
}

func ReleaseAssets(dir string, m *manifest.ThemeManifest) ([]build.SourceAsset, error) {
	files, err := InstallFiles(dir, m)
	if err != nil {
		return nil, err
	}

	assets := []build.SourceAsset{}
	usedNames := map[string]bool{}
	for _, file := range files {
		name := path.Base(file.Path)
		if usedNames[name] {
			name = schemeForPath(m, file.Path) + ".css"
		}
		usedNames[name] = true
		assets = append(assets, build.SourceAsset{Name: name, Path: file.Source})
	}
	return assets, nil
}

func schemeForPath(m *manifest.ThemeManifest, relativePath string) string {
	for scheme, candidate := range m.Colorschemes {
		if normalized, ok := rules.SafeRelativePath(candidate); ok && normalized == relativePath {
			return scheme
		}
	}
	return relativePath
}

func Metadata(m *manifest.ThemeManifest) build.MetadataInput {
	return build.MetadataInput{
		Kind:          build.KindTheme,
		ID:            m.ID,
		Name:          m.DisplayName,
		Version:       m.Version,
		Author:        m.Author,
		AuthorURL:     m.AuthorURL,
		Description:   m.Description,
		CoverImageURL: m.CoverImageURL,
		Repository:    m.Repository,
	}
}
