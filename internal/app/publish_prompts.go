package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/pkg/manifest"
)

type packageJSON struct {
	Description string      `json:"description"`
	Repository  interface{} `json:"repository"`
	Author      interface{} `json:"author"`
}

func promptPluginPublishMetadata(dir string, opts *plugin.PublishOptions) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	pluginManifest, _ := manifest.LoadPlugin(absDir)

	packageData := loadPackageJSON(absDir)

	defaultAuthor := firstNonEmpty(opts.Author, manifestPluginAuthor(pluginManifest), packageAuthor(packageData))
	defaultDescription := firstNonEmpty(opts.Description, manifestPluginDescription(pluginManifest), packageDescription(packageData))
	defaultRepository := firstNonEmpty(opts.Repository, manifestPluginRepository(pluginManifest), packageRepository(packageData))

	if err := survey.AskOne(&survey.Input{Message: "Author:", Default: defaultAuthor}, &opts.Author, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Description:", Default: defaultDescription}, &opts.Description, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Cover image URL (optional):", Default: opts.CoverImageURL}, &opts.CoverImageURL); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Repository (owner/repo or URL, optional):", Default: defaultRepository}, &opts.Repository); err != nil {
		return err
	}

	return nil
}

func promptThemePublishMetadata(dir string, opts *theme.PublishOptions) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	themeManifest, _ := manifest.LoadTheme(absDir)

	packageData := loadPackageJSON(absDir)

	defaultAuthor := firstNonEmpty(opts.Author, manifestThemeAuthor(themeManifest), packageAuthor(packageData))
	defaultDescription := firstNonEmpty(opts.Description, manifestThemeDescription(themeManifest), packageDescription(packageData))
	defaultRepository := firstNonEmpty(opts.Repository, manifestThemeRepository(themeManifest), packageRepository(packageData))

	if err := survey.AskOne(&survey.Input{Message: "Author:", Default: defaultAuthor}, &opts.Author, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Description:", Default: defaultDescription}, &opts.Description, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Cover image URL (optional):", Default: opts.CoverImageURL}, &opts.CoverImageURL); err != nil {
		return err
	}

	if err := survey.AskOne(&survey.Input{Message: "Repository (owner/repo or URL, optional):", Default: defaultRepository}, &opts.Repository); err != nil {
		return err
	}

	return nil
}

func loadPackageJSON(dir string) *packageJSON {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	parsed := &packageJSON{}
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil
	}

	return parsed
}

func packageDescription(pkg *packageJSON) string {
	if pkg == nil {
		return ""
	}
	return strings.TrimSpace(pkg.Description)
}

func packageRepository(pkg *packageJSON) string {
	if pkg == nil || pkg.Repository == nil {
		return ""
	}

	if value, ok := pkg.Repository.(string); ok {
		return strings.TrimSpace(value)
	}

	if value, ok := pkg.Repository.(map[string]interface{}); ok {
		if url, ok := value["url"].(string); ok {
			return strings.TrimSpace(url)
		}
	}

	return ""
}

func packageAuthor(pkg *packageJSON) string {
	if pkg == nil || pkg.Author == nil {
		return ""
	}

	if value, ok := pkg.Author.(string); ok {
		return strings.TrimSpace(value)
	}

	if value, ok := pkg.Author.(map[string]interface{}); ok {
		if name, ok := value["name"].(string); ok {
			return strings.TrimSpace(name)
		}
	}

	return ""
}

func manifestPluginAuthor(value *manifest.PluginManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Author)
}

func manifestPluginDescription(value *manifest.PluginManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Description)
}

func manifestPluginRepository(value *manifest.PluginManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Repository)
}

func manifestThemeAuthor(value *manifest.ThemeManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Author)
}

func manifestThemeDescription(value *manifest.ThemeManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Description)
}

func manifestThemeRepository(value *manifest.ThemeManifest) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Repository)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func printInteractivePublishTip(kind string) {
	fmt.Printf("\nInteractive %s publish metadata\n\n", kind)
}
