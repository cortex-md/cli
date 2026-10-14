package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const FileName = "manifest.json"

type Kind string

const (
	KindPlugin Kind = "plugin"
	KindTheme  Kind = "theme"
)

type PluginManifest struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	MinAppVersion string   `json:"minAppVersion"`
	Author        string   `json:"author"`
	AuthorURL     string   `json:"authorUrl,omitempty"`
	Description   string   `json:"description"`
	Icon          string   `json:"icon"`
	Main          string   `json:"main"`
	Permissions   []string `json:"permissions,omitempty"`
	Repository    string   `json:"repository,omitempty"`
	CoverImageURL string   `json:"coverImageUrl,omitempty"`
	Screenshots   []string `json:"screenshots,omitempty"`
}

type ThemeManifest struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	DisplayName   string            `json:"displayName"`
	Author        string            `json:"author"`
	AuthorURL     string            `json:"authorUrl,omitempty"`
	Version       string            `json:"version"`
	MinAppVersion string            `json:"minAppVersion,omitempty"`
	Description   string            `json:"description,omitempty"`
	Colorschemes  map[string]string `json:"colorschemes"`
	Repository    string            `json:"repository,omitempty"`
	CoverImageURL string            `json:"coverImageUrl,omitempty"`
	Screenshots   []string          `json:"screenshots,omitempty"`
}

var (
	ErrManifestNotFound = errors.New("manifest.json not found")
	ErrInvalidManifest  = errors.New("invalid manifest format")
)

func Path(dir string) string {
	return filepath.Join(dir, FileName)
}

func DetectKind(dir string) (Kind, error) {
	raw, err := readRaw(dir)
	if err != nil {
		return "", err
	}
	if _, ok := raw["colorschemes"]; ok {
		return KindTheme, nil
	}
	return KindPlugin, nil
}

func ReadPlugin(dir string) (*PluginManifest, error) {
	var manifest PluginManifest
	if err := readInto(dir, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func ReadTheme(dir string) (*ThemeManifest, error) {
	var manifest ThemeManifest
	if err := readInto(dir, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func LoadPlugin(dir string) (*PluginManifest, error) {
	manifest, err := ReadPlugin(dir)
	if err != nil {
		return nil, err
	}
	if missing := missingFields(map[string]string{
		"id":          manifest.ID,
		"name":        manifest.Name,
		"version":     manifest.Version,
		"author":      manifest.Author,
		"description": manifest.Description,
		"icon":        manifest.Icon,
		"main":        manifest.Main,
	}); len(missing) > 0 {
		return nil, fmt.Errorf("manifest.json is missing %s", strings.Join(missing, ", "))
	}
	return manifest, nil
}

func LoadTheme(dir string) (*ThemeManifest, error) {
	manifest, err := ReadTheme(dir)
	if err != nil {
		return nil, err
	}
	if missing := missingFields(map[string]string{
		"id":          manifest.ID,
		"name":        manifest.Name,
		"displayName": manifest.DisplayName,
		"author":      manifest.Author,
		"version":     manifest.Version,
	}); len(missing) > 0 {
		return nil, fmt.Errorf("manifest.json is missing %s", strings.Join(missing, ", "))
	}
	if len(manifest.Colorschemes) == 0 {
		return nil, errors.New("manifest.json is missing colorschemes")
	}
	return manifest, nil
}

func ReleasePluginManifest(dir string) ([]byte, error) {
	raw, err := readRaw(dir)
	if err != nil {
		return nil, err
	}
	if permissions, ok := raw["permissions"]; ok {
		raw["capabilities"] = permissions
	} else {
		raw["capabilities"] = json.RawMessage("[]")
	}
	content, err := json.MarshalIndent(raw, "", "\t")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func readRaw(dir string) (map[string]json.RawMessage, error) {
	raw := map[string]json.RawMessage{}
	if err := readInto(dir, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func readInto(dir string, target interface{}) error {
	data, err := os.ReadFile(Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return ErrManifestNotFound
		}
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	return nil
}

func missingFields(fields map[string]string) []string {
	order := []string{"id", "name", "displayName", "version", "author", "description", "icon", "main"}
	missing := []string{}
	for _, name := range order {
		value, ok := fields[name]
		if ok && strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}
