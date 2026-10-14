package dev

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cortex/cli/internal/build"
)

func TestResolveVaultPathFromAncestor(t *testing.T) {
	root := t.TempDir()
	vault := filepath.Join(root, "vault")
	nested := filepath.Join(vault, "notes", "daily")
	if err := os.MkdirAll(filepath.Join(vault, ".cortex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, ".cortex", "vault-id.json"), []byte(`{"uuid":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveVaultPath(nested, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != vault {
		t.Fatalf("resolved vault = %s, want %s", resolved, vault)
	}
}

func TestInstallCopiesFilesAndKeepsPluginData(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "vault", ".cortex", "plugins", "test-plugin")
	bundle := filepath.Join(root, "index.js")
	if err := os.WriteFile(bundle, []byte("module.exports = {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(installDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "data", "cache.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(installDir, []build.File{
		{Path: "manifest.json", Content: []byte(`{"id":"test-plugin"}`)},
		{Path: "dist/index.js", Source: bundle},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"manifest.json", "dist/index.js", DevMarker, "data/cache.json"} {
		if _, err := os.Stat(filepath.Join(installDir, filepath.FromSlash(path))); err != nil {
			t.Fatalf("%s missing after install: %v", path, err)
		}
	}
	if IsForeignInstall(installDir) {
		t.Fatal("dev install was reported as foreign")
	}
}

func TestUninstallRefusesMarketplaceInstalls(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "plugins", "test-plugin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Uninstall(installDir); !errors.Is(err, ErrNotDevInstall) {
		t.Fatalf("Uninstall error = %v, want ErrNotDevInstall", err)
	}
	if err := os.WriteFile(filepath.Join(installDir, DevMarker), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(installDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installDir); !os.IsNotExist(err) {
		t.Fatalf("install dir still exists: %v", err)
	}
}
