package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cortex/cli/internal/project"
)

func TestNewSubmissionUsesManifestRegistryData(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
		"id": "paper-theme",
		"name": "paper-theme",
		"displayName": "Paper Theme",
		"author": "furqas",
		"version": "1.2.3",
		"description": "A paper-like theme",
		"repository": "https://github.com/furqas/paper-theme",
		"screenshots": [" https://example.com/paper.png "],
		"colorschemes": {"dark": "dark.css", "light": "light.css"}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	submission, err := NewSubmission(loaded)
	if err != nil {
		t.Fatal(err)
	}

	if submission.IndexFile != "themes.json" {
		t.Fatalf("indexFile = %s, want themes.json", submission.IndexFile)
	}
	if submission.Kind != "theme" {
		t.Fatalf("kind = %s, want theme", submission.Kind)
	}
	if submission.Entry.Repo != "furqas/paper-theme" {
		t.Fatalf("repo = %s", submission.Entry.Repo)
	}
	if submission.Entry.Name != "Paper Theme" {
		t.Fatalf("name = %s, want the display name", submission.Entry.Name)
	}
	if len(submission.Entry.Screenshots) != 1 || submission.Entry.Screenshots[0] != "https://example.com/paper.png" {
		t.Fatalf("screenshots = %v", submission.Entry.Screenshots)
	}
	if submission.BranchName() != "registry/theme-paper-theme" {
		t.Fatalf("branch = %s", submission.BranchName())
	}
}
