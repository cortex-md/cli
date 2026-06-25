package dev

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLinkPluginUsesVaultScopedPluginDirectory(t *testing.T) {
	root := t.TempDir()
	vault := filepath.Join(root, "vault")
	pluginDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(filepath.Join(vault, ".cortex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{
		"id":"test-plugin",
		"name":"Test Plugin",
		"version":"1.0.0",
		"minAppVersion":"0.1.0",
		"author":"Tester",
		"description":"A test plugin",
		"icon":"puzzle",
		"main":"dist/index.js"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Link(pluginDir, LinkOptions{VaultPath: vault})
	if err != nil {
		t.Fatal(err)
	}

	wantLink := filepath.Join(vault, ".cortex", "plugins", "test-plugin")
	if result.LinkPath != wantLink {
		t.Fatalf("link path = %s, want %s", result.LinkPath, wantLink)
	}
	target, err := os.Readlink(wantLink)
	if err != nil {
		t.Fatal(err)
	}
	if target != pluginDir {
		t.Fatalf("symlink target = %s, want %s", target, pluginDir)
	}
	markerPath := filepath.Join(vault, ".cortex", "plugins", ".reload-test-plugin")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("reload marker was not created: %v", err)
	}
}

func TestRemoveDevLinkRemovesMatchingSymlinkAndTouchesMarker(t *testing.T) {
	root := t.TempDir()
	parentDir := filepath.Join(root, "vault", ".cortex", "plugins")
	targetDir := filepath.Join(root, "plugin")
	linkPath := filepath.Join(parentDir, "test-plugin")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatal(err)
	}

	if err := RemoveDevLink(linkPath, targetDir, parentDir, "test-plugin"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("dev link still exists or returned unexpected error: %v", err)
	}
	markerPath := filepath.Join(parentDir, ".reload-test-plugin")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("reload marker was not created: %v", err)
	}
}
