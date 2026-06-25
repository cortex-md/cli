package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cortex/cli/internal/publish"
	"github.com/cortex/cli/pkg/manifest"
)

func TestPluginReleaseAssetsMatchMarketplaceFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"manifest.json": `{"id":"test-plugin"}`,
		"dist/index.js": "export default class TestPlugin {}",
		"styles.css":    ".test {}",
	}
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	assets, err := pluginReleaseAssets(dir, &manifest.PluginManifest{
		ID:      "test-plugin",
		Version: "1.2.3",
		Main:    "dist/index.js",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"manifest.json", "index.js", "styles.css"}
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
	writeFile(t, filepath.Join(dir, "manifest.json"), `{
		"id":"local-plugin",
		"name":"Local Plugin",
		"version":"1.2.3",
		"minAppVersion":"0.1.0",
		"author":"furqas",
		"description":"A useful local plugin",
		"icon":"folder-kanban",
		"main":"dist/index.js",
		"repository":"furqas/local-plugin"
	}`)
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"build":"bun build"}}`)
	writeFile(t, filepath.Join(dir, "src", "index.ts"), "export default {}")
	writeFile(t, filepath.Join(dir, "styles.css"), ".local-plugin {}")
	installFakeBun(t)
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
		result.PublishManifestPath,
		result.ReleaseNotesPath,
		result.ArchivePath,
		filepath.Join(result.OutputDir, "assets", "manifest.json"),
		filepath.Join(result.OutputDir, "assets", "index.js"),
		filepath.Join(result.OutputDir, "assets", "styles.css"),
		filepath.Join(result.OutputDir, "assets", "local-plugin-1.2.3.zip"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected publish artifact %s: %v", path, err)
		}
	}

	publishManifest, err := publish.LoadManifest(result.PublishManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if publishManifest.Kind != publish.KindPlugin {
		t.Fatalf("kind = %s, want plugin", publishManifest.Kind)
	}
	if publishManifest.TagName != "v1.2.3" {
		t.Fatalf("tagName = %s, want v1.2.3", publishManifest.TagName)
	}
	if publishManifest.Registry.IndexFile != "plugins.json" {
		t.Fatalf("indexFile = %s, want plugins.json", publishManifest.Registry.IndexFile)
	}
	if publishManifest.Registry.Entry.Repo != "furqas/local-plugin" {
		t.Fatalf("repo = %s, want furqas/local-plugin", publishManifest.Registry.Entry.Repo)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func installFakeBun(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bun")
	content := "#!/bin/sh\n/bin/mkdir -p dist\nprintf 'export default {}\\n' > dist/index.js\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}
