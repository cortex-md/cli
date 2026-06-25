package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cortex/cli/pkg/manifest"
)

func TestCreateWritesWorkflowsGitignoreAndDescription(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test-theme")
	if err := Create(CreateOptions{
		Name:        "Test Theme",
		ID:          "test-theme",
		DisplayName: "Test Theme",
		Description: "A useful test theme",
		Author:      "furqas",
		Directory:   dir,
	}); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"manifest.json",
		"package.json",
		".gitignore",
		".github/workflows/ci-theme.yml",
		".github/workflows/cd-theme.yml",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}

	themeManifest, err := manifest.LoadTheme(dir)
	if err != nil {
		t.Fatal(err)
	}
	if themeManifest.Description != "A useful test theme" {
		t.Fatalf("description = %q", themeManifest.Description)
	}
}
