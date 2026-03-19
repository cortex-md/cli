package dev

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

func GetPluginsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".cortex", "plugins"), nil
}

func Link(pluginDir string) error {
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	pluginsDir, err := GetPluginsDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	linkPath := filepath.Join(pluginsDir, m.ID)

	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing link: %w", err)
		}
	}

	if err := os.Symlink(absDir, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	ux.Success("Linked %s → %s", m.ID, absDir)

	return nil
}

func Unlink(pluginDir string) error {
	absDir, err := filepath.Abs(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	return UnlinkByID(m.ID)
}

func UnlinkByID(pluginID string) error {
	pluginsDir, err := GetPluginsDir()
	if err != nil {
		return err
	}

	linkPath := filepath.Join(pluginsDir, pluginID)

	info, err := os.Lstat(linkPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("plugin %s is not linked", pluginID)
	}
	if err != nil {
		return fmt.Errorf("failed to check link: %w", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink (installed plugin?)", pluginID)
	}

	if err := os.Remove(linkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	ux.Success("Unlinked %s", pluginID)

	return nil
}

func IsLinked(pluginID string) (bool, string, error) {
	pluginsDir, err := GetPluginsDir()
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

func ListLinked() (map[string]string, error) {
	pluginsDir, err := GetPluginsDir()
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
