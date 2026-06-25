package publish

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMetadataPrefersInputOverPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"description": "Package description",
		"repository": "package/repo",
		"author": {"name": "Package Author", "url": "https://example.com/package"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	metadata, err := ResolveMetadata(dir, MetadataInput{
		Kind:        KindPlugin,
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.2.3",
		Author:      "Manifest Author",
		Description: "Manifest description",
		Repository:  "manifest/repo",
	})
	if err != nil {
		t.Fatal(err)
	}

	if metadata.Author != "Manifest Author" {
		t.Fatalf("author = %s", metadata.Author)
	}
	if metadata.Description != "Manifest description" {
		t.Fatalf("description = %s", metadata.Description)
	}
	if metadata.Repository != "manifest/repo" {
		t.Fatalf("repository = %s", metadata.Repository)
	}
}

func TestResolveMetadataFallsBackToGitOrigin(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "remote", "add", "origin", "git@github.com:furqas/origin-plugin.git")

	metadata, err := ResolveMetadata(dir, MetadataInput{
		Kind:        KindPlugin,
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.2.3",
		Author:      "furqas",
		Description: "A useful plugin",
	})
	if err != nil {
		t.Fatal(err)
	}

	if metadata.Repository != "furqas/origin-plugin" {
		t.Fatalf("repository = %s, want furqas/origin-plugin", metadata.Repository)
	}
}

func TestResolveMetadataReportsMissingMarketplaceFields(t *testing.T) {
	_, err := ResolveMetadata(t.TempDir(), MetadataInput{
		Kind:    KindTheme,
		ID:      "test-theme",
		Name:    "Test Theme",
		Version: "1.2.3",
	})
	if err == nil {
		t.Fatal("expected missing metadata error")
	}
	for _, expected := range []string{"author", "description", "repository", "owner/repo"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to mention %q, got %q", expected, err.Error())
		}
	}
}

func TestResolveMetadataExplainsMissingRepository(t *testing.T) {
	_, err := ResolveMetadata(t.TempDir(), MetadataInput{
		Kind:        KindPlugin,
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.2.3",
		Author:      "furqas",
		Description: "A useful plugin",
	})
	if err == nil {
		t.Fatal("expected missing repository error")
	}

	for _, expected := range []string{
		"missing marketplace repository",
		"manifest.json",
		"package.json repository",
		"git remote origin",
		"owner/repo",
		"host this plugin's releases",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to mention %q, got %q", expected, err.Error())
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s", args, output)
	}
}
