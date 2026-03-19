package theme

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cortex/cli/internal/fsx"
)

var (
	ErrInvalidID   = errors.New("invalid theme id")
	ErrDirExists   = errors.New("directory already exists")
	ErrInvalidName = errors.New("invalid theme name")
)

//go:embed templates/theme/*
var templatesFS embed.FS

type CreateOptions struct {
	Name        string
	ID          string
	DisplayName string
	Description string
	Author      string
	Directory   string
	Version     string
}

func Create(opts CreateOptions) error {
	if err := validateCreateOptions(&opts); err != nil {
		return err
	}

	targetDir := opts.Directory
	if targetDir == "" {
		targetDir = opts.ID
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	if fsx.Exists(absDir) {
		return ErrDirExists
	}

	if err := fsx.EnsureDir(absDir); err != nil {
		return err
	}

	if err := writeTemplateFiles(absDir, opts); err != nil {
		return err
	}

	return nil
}

func validateCreateOptions(opts *CreateOptions) error {
	if opts.Name == "" {
		return ErrInvalidName
	}

	if opts.ID == "" {
		opts.ID = normalizeID(opts.Name)
	}

	if opts.DisplayName == "" {
		opts.DisplayName = opts.Name
	}

	if opts.Version == "" {
		opts.Version = "0.1.0"
	}

	if opts.Description == "" {
		opts.Description = fmt.Sprintf("A Cortex theme for %s", opts.Name)
	}

	if opts.Author == "" {
		opts.Author = "Unknown"
	}

	return nil
}

func normalizeID(name string) string {
	id := strings.ToLower(name)
	id = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	return id
}

func writeTemplateFiles(dir string, opts CreateOptions) error {
	replacements := map[string]string{
		"{{ID}}":           opts.ID,
		"{{NAME}}":         opts.Name,
		"{{DISPLAY_NAME}}": opts.DisplayName,
		"{{VERSION}}":      opts.Version,
		"{{DESCRIPTION}}":  opts.Description,
		"{{AUTHOR}}":       opts.Author,
	}

	templateFiles := map[string]string{
		"templates/theme/manifest.json":   "manifest.json",
		"templates/theme/package.json":    "package.json",
		"templates/theme/theme-dark.css":  "theme-dark.css",
		"templates/theme/theme-light.css": "theme-light.css",
		"templates/theme/README.md":       "README.md",
	}

	for templatePath, outputPath := range templateFiles {
		content, err := templatesFS.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		processedContent := applyReplacements(string(content), replacements)
		fullPath := filepath.Join(dir, outputPath)

		if err := os.WriteFile(fullPath, []byte(processedContent), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, err)
		}
	}

	return nil
}

func applyReplacements(template string, replacements map[string]string) string {
	result := template
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}
	return result
}

func NormalizeID(name string) string {
	return normalizeID(name)
}
