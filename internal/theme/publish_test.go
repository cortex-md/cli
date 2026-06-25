package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/pkg/manifest"
)

func TestThemeReleaseAssetsUseBasenamesForColorschemes(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"manifest.json":   `{"id":"test-theme"}`,
		"theme-dark.css":  ".dark {}",
		"theme-light.css": ".light {}",
	}
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	assets, err := themeReleaseAssets(dir, &manifest.ThemeManifest{
		ID:      "test-theme",
		Version: "1.2.3",
		Colorschemes: map[string]string{
			"light": "./theme-light.css",
			"dark":  "./theme-dark.css",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"manifest.json", "theme-dark.css", "theme-light.css"}
	if len(assets) != len(wantNames) {
		t.Fatalf("asset count = %d, want %d", len(assets), len(wantNames))
	}
	for index, wantName := range wantNames {
		if assets[index].Name != wantName {
			t.Fatalf("asset[%d] = %s, want %s", index, assets[index].Name, wantName)
		}
	}
}

func TestPublishPreparesLocalArtifactsWithoutGithubToken(t *testing.T) {
	dir := t.TempDir()
	writeThemeFile(t, filepath.Join(dir, "manifest.json"), `{
		"id":"local-theme",
		"name":"Local Theme",
		"displayName":"Local Theme",
		"version":"1.2.3",
		"author":"furqas",
		"description":"A useful local theme",
		"repository":"furqas/local-theme",
		"colorschemes":{
			"light":"theme-light.css",
			"dark":"theme-dark.css"
		}
	}`)
	writeThemeFile(t, filepath.Join(dir, "package.json"), `{"description":"Package theme"}`)
	writeThemeFile(t, filepath.Join(dir, "theme-dark.css"), validThemeCSS())
	writeThemeFile(t, filepath.Join(dir, "theme-light.css"), validThemeCSS())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("CORTEX_TOKEN", "")

	result, err := Publish(dir, PublishOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("publish should not initialize git repository, stat err = %v", err)
	}

	for _, path := range []string{
		result.OutputDir,
		result.BuildManifestPath,
		result.ReleaseNotesPath,
		result.ArchivePath,
		filepath.Join(result.OutputDir, "assets", "manifest.json"),
		filepath.Join(result.OutputDir, "assets", "theme-dark.css"),
		filepath.Join(result.OutputDir, "assets", "theme-light.css"),
		filepath.Join(result.OutputDir, "assets", "local-theme-1.2.3.zip"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected publish artifact %s: %v", path, err)
		}
	}

	buildManifest, err := build.LoadManifest(result.BuildManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if buildManifest.Kind != build.KindTheme {
		t.Fatalf("kind = %s, want theme", buildManifest.Kind)
	}
	if buildManifest.TagName != "v1.2.3" {
		t.Fatalf("tagName = %s, want v1.2.3", buildManifest.TagName)
	}
	if buildManifest.Registry.IndexFile != "themes.json" {
		t.Fatalf("indexFile = %s, want themes.json", buildManifest.Registry.IndexFile)
	}
	if buildManifest.Registry.Entry.Repo != "furqas/local-theme" {
		t.Fatalf("repo = %s, want furqas/local-theme", buildManifest.Registry.Entry.Repo)
	}
}

func writeThemeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validThemeCSS() string {
	return `:root {
	--bg-primary: #ffffff;
	--bg-secondary: #f6f6f6;
	--bg-tertiary: #eeeeee;
	--bg-elevated: #ffffff;
	--bg-hover: #eeeeee;
	--bg-active: #dddddd;
	--text-primary: #111111;
	--text-secondary: #222222;
	--text-muted: #555555;
	--text-disabled: #777777;
	--accent-default: #0066cc;
	--accent-hover: #0059b3;
	--accent-active: #004c99;
	--border-default: #cccccc;
	--border-subtle: #eeeeee;
	--border-strong: #999999;
	--syntax-keyword: #0066cc;
	--syntax-string: #22863a;
	--syntax-comment: #6a737d;
	--syntax-number: #005cc5;
	--syntax-function: #6f42c1;
}`
}
