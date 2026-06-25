package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSubmitOptionsUsesPublishManifestRegistryData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "publish.json")
	if err := os.WriteFile(path, []byte(`{
		"schemaVersion": 1,
		"kind": "theme",
		"id": "paper-theme",
		"name": "Paper Theme",
		"version": "1.2.3",
		"tagName": "v1.2.3",
		"releaseName": "Paper Theme v1.2.3",
		"releaseNotes": "release-notes.md",
		"repository": "furqas/paper-theme",
		"assets": [],
		"registry": {
			"owner": "cortex-md",
			"repo": "registry",
			"baseBranch": "main",
			"indexFile": "themes.json",
			"entry": {
				"id": "paper-theme",
				"name": "Paper Theme",
				"author": "furqas",
				"description": "A paper-like theme",
				"coverImageUrl": "",
				"repo": "furqas/paper-theme"
			}
		}
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := LoadSubmitOptions(path)
	if err != nil {
		t.Fatal(err)
	}

	if opts.IndexFile != "themes.json" {
		t.Fatalf("indexFile = %s, want themes.json", opts.IndexFile)
	}
	if opts.Kind != "theme" {
		t.Fatalf("kind = %s, want theme", opts.Kind)
	}
	if opts.Entry.Repo != "furqas/paper-theme" {
		t.Fatalf("repo = %s", opts.Entry.Repo)
	}
}
