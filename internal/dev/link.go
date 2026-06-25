package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

type LinkOptions struct {
	VaultPath string
}

type LinkResult struct {
	ID        string
	LinkPath  string
	ParentDir string
	Target    string
}

func ResolveVaultPath(startDir string, explicitVaultPath string) (string, error) {
	candidate := strings.TrimSpace(explicitVaultPath)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv("CORTEX_VAULT"))
	}
	if candidate != "" {
		return normalizeVaultPath(candidate)
	}

	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err == nil && !info.IsDir() {
		absDir = filepath.Dir(absDir)
	}

	for {
		identityPath := filepath.Join(absDir, ".cortex", "vault-id.json")
		if _, err := os.Stat(identityPath); err == nil {
			return absDir, nil
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}

	return "", fmt.Errorf("could not resolve Cortex vault; run this command inside a vault or pass --vault /path/to/vault")
}

func GetPluginsDir(vaultPath string) (string, error) {
	vaultRoot, err := ResolveVaultPath(vaultPath, vaultPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(vaultRoot, ".cortex", "plugins"), nil
}

func GetThemesDir(vaultPath string) (string, error) {
	vaultRoot, err := ResolveVaultPath(vaultPath, vaultPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(vaultRoot, ".cortex", "themes"), nil
}

func Link(pluginDir string, opts LinkOptions) (*LinkResult, error) {
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	pluginsDir, err := resolveKindDir(absDir, opts.VaultPath, "plugins")
	if err != nil {
		return nil, err
	}

	result, err := linkDirectory(absDir, pluginsDir, m.ID)
	if err != nil {
		return nil, err
	}

	ux.Success("Linked %s -> %s", m.ID, absDir)
	ux.Info("Vault plugin path: %s", result.LinkPath)

	return result, nil
}

func LinkTheme(themeDir string, opts LinkOptions) (*LinkResult, error) {
	absDir, err := filepath.Abs(themeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	themesDir, err := resolveKindDir(absDir, opts.VaultPath, "themes")
	if err != nil {
		return nil, err
	}

	result, err := linkDirectory(absDir, themesDir, m.ID)
	if err != nil {
		return nil, err
	}

	ux.Success("Linked %s -> %s", m.ID, absDir)
	ux.Info("Vault theme path: %s", result.LinkPath)

	return result, nil
}

func Unlink(pluginDir string, opts LinkOptions) error {
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	return UnlinkByID(m.ID, opts)
}

func UnlinkTheme(themeDir string, opts LinkOptions) error {
	absDir, err := filepath.Abs(themeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	return UnlinkThemeByID(m.ID, opts)
}

func UnlinkByID(pluginID string, opts LinkOptions) error {
	pluginsDir, err := resolveKindDir(".", opts.VaultPath, "plugins")
	if err != nil {
		return err
	}

	return unlinkByID(pluginID, pluginsDir)
}

func UnlinkThemeByID(themeID string, opts LinkOptions) error {
	themesDir, err := resolveKindDir(".", opts.VaultPath, "themes")
	if err != nil {
		return err
	}

	return unlinkByID(themeID, themesDir)
}

func IsLinked(pluginID string, opts LinkOptions) (bool, string, error) {
	pluginsDir, err := resolveKindDir(".", opts.VaultPath, "plugins")
	if err != nil {
		return false, "", err
	}

	linkPath := filepath.Join(pluginsDir, pluginID)

	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false, "", nil
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		return false, "", err
	}

	return true, target, nil
}

func ListLinked(opts LinkOptions) (map[string]string, error) {
	pluginsDir, err := resolveKindDir(".", opts.VaultPath, "plugins")
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(pluginsDir)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	links := make(map[string]string)

	for _, entry := range entries {
		linkPath := filepath.Join(pluginsDir, entry.Name())

		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		target, err := os.Readlink(linkPath)
		if err != nil {
			continue
		}

		links[entry.Name()] = target
	}

	return links, nil
}

func normalizeVaultPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve vault path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("vault path does not exist: %s", absPath)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("vault path is not a directory: %s", absPath)
	}

	return absPath, nil
}

func resolveKindDir(startDir string, vaultPath string, kind string) (string, error) {
	vaultRoot, err := ResolveVaultPath(startDir, vaultPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(vaultRoot, ".cortex", kind), nil
}

func linkDirectory(targetDir string, parentDir string, id string) (*LinkResult, error) {
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	linkPath := filepath.Join(parentDir, id)
	if info, err := os.Lstat(linkPath); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return nil, fmt.Errorf("%s already exists and is not a symlink", linkPath)
		}
		if err := os.Remove(linkPath); err != nil {
			return nil, fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	if err := os.Symlink(targetDir, linkPath); err != nil {
		return nil, fmt.Errorf("failed to create symlink: %w", err)
	}
	if err := TouchDevReloadMarker(parentDir, id); err != nil {
		return nil, err
	}

	return &LinkResult{ID: id, LinkPath: linkPath, ParentDir: parentDir, Target: targetDir}, nil
}

func TouchDevReloadMarker(parentDir string, id string) error {
	if strings.TrimSpace(parentDir) == "" || strings.TrimSpace(id) == "" {
		return nil
	}
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("failed to create reload marker directory: %w", err)
	}
	markerPath := filepath.Join(parentDir, fmt.Sprintf(".reload-%s", id))
	content := []byte(time.Now().UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(markerPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write reload marker: %w", err)
	}
	return nil
}

func RemoveDevLink(linkPath string, target string, parentDir string, id string) error {
	if strings.TrimSpace(linkPath) == "" {
		return nil
	}

	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return TouchDevReloadMarker(parentDir, id)
	}
	if err != nil {
		return fmt.Errorf("failed to inspect dev link: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("dev link path is not a symlink: %s", linkPath)
	}

	currentTarget, err := os.Readlink(linkPath)
	if err != nil {
		return fmt.Errorf("failed to read dev link: %w", err)
	}
	if currentTarget != target {
		return fmt.Errorf("dev link target changed from %s to %s", target, currentTarget)
	}

	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("failed to remove dev link: %w", err)
	}
	return TouchDevReloadMarker(parentDir, id)
}

func unlinkByID(id string, parentDir string) error {
	linkPath := filepath.Join(parentDir, id)

	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s is not linked in this vault", id)
	}
	if err != nil {
		return fmt.Errorf("failed to check link: %w", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink; installed items are not removed by unlink", id)
	}

	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	ux.Success("Unlinked %s", id)

	return nil
}
