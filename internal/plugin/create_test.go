package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWritesCIAndCDWorkflows(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "test-plugin")
	if err := Create(CreateOptions{
		Name:        "Test Plugin",
		ID:          "test-plugin",
		Description: "A useful test plugin",
		Author:      "furqas",
		Directory:   dir,
	}); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"manifest.json",
		"package.json",
		".github/workflows/ci-plugin.yml",
		".github/workflows/cd-plugin.yml",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected scaffold file %s: %v", path, err)
		}
	}
}
