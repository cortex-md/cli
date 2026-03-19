package manifest

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PluginManifest struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	MinAppVersion string   `json:"minAppVersion"`
	Author        string   `json:"author"`
	Description   string   `json:"description"`
	Icon          string   `json:"icon"`
	Main          string   `json:"main"`
	Repository    string   `json:"repository,omitempty"`
	License       string   `json:"license,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
}

type ThemeManifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	DisplayName  string            `json:"displayName"`
	Author       string            `json:"author"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Colorschemes map[string]string `json:"colorschemes"`
	Repository   string            `json:"repository,omitempty"`
	License      string            `json:"license,omitempty"`
}

var (
	ErrManifestNotFound = errors.New("manifest.json not found")
	ErrInvalidManifest  = errors.New("invalid manifest format")
	ErrMissingField     = errors.New("required field missing")
)

func LoadPlugin(dir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrManifestNotFound
		}
		return nil, err
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, ErrInvalidManifest
	}

	if err := validatePlugin(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func LoadTheme(dir string) (*ThemeManifest, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrManifestNotFound
		}
		return nil, err
	}

	var manifest ThemeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, ErrInvalidManifest
	}

	if err := validateTheme(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func SavePlugin(dir string, manifest *PluginManifest) error {
	if err := validatePlugin(manifest); err != nil {
		return err
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, data, 0o644)
}

func SaveTheme(dir string, manifest *ThemeManifest) error {
	if err := validateTheme(manifest); err != nil {
		return err
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, data, 0o644)
}

func validatePlugin(m *PluginManifest) error {
	if m.ID == "" {
		return errors.New("id is required")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.Version == "" {
		return errors.New("version is required")
	}
	if m.Author == "" {
		return errors.New("author is required")
	}
	if m.Description == "" {
		return errors.New("description is required")
	}
	if m.Main == "" {
		return errors.New("main is required")
	}
	if m.Icon == "" {
		return errors.New("icon is required")
	}
	return nil
}

func validateTheme(m *ThemeManifest) error {
	if m.ID == "" {
		return errors.New("id is required")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.DisplayName == "" {
		return errors.New("displayName is required")
	}
	if m.Author == "" {
		return errors.New("author is required")
	}
	if m.Version == "" {
		return errors.New("version is required")
	}
	if m.Colorschemes == nil || len(m.Colorschemes) == 0 {
		return errors.New("colorschemes is required")
	}
	return nil
}

func NormalizeID(name string) string {
	id := strings.ToLower(name)
	id = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	return id
}
