package plugin

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
	ErrInvalidID   = errors.New("invalid plugin id")
	ErrDirExists   = errors.New("directory already exists")
	ErrInvalidName = errors.New("invalid plugin name")
)

var idPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

//go:embed templates/plugin/*
//go:embed templates/plugin/src/*
//go:embed templates/plugin/github/workflows/*
var templatesFS embed.FS

type CreateOptions struct {
	Name        string
	ID          string
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

	if err := createStructure(absDir); err != nil {
		return err
	}

	if err := writeTemplateFiles(absDir, opts); err != nil {
		return err
	}

	return nil
}

func validateCreateOptions(opts *CreateOptions) error {
	if opts.ID == "" {
		opts.ID = normalizeID(opts.Name)
	}

	if !idPattern.MatchString(opts.ID) {
		return ErrInvalidID
	}

	if opts.Name == "" {
		return ErrInvalidName
	}

	if opts.Version == "" {
		opts.Version = "0.1.0"
	}

	if opts.Description == "" {
		opts.Description = fmt.Sprintf("A Cortex plugin for %s", opts.Name)
	}

	if opts.Author == "" {
		opts.Author = "Unknown"
	}

	return nil
}

func NormalizeID(name string) string {
	id := strings.ToLower(name)
	id = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")
	return id
}

func normalizeID(name string) string {
	return NormalizeID(name)
}

func createStructure(dir string) error {
	dirs := []string{
		dir,
		filepath.Join(dir, "src"),
	}

	for _, d := range dirs {
		if err := fsx.EnsureDir(d); err != nil {
			return err
		}
	}

	return nil
}

func writeTemplateFiles(dir string, opts CreateOptions) error {
	className := toPascalCase(opts.ID)

	replacements := map[string]string{
		"{{ID}}":          opts.ID,
		"{{NAME}}":        opts.Name,
		"{{VERSION}}":     opts.Version,
		"{{DESCRIPTION}}": opts.Description,
		"{{AUTHOR}}":      opts.Author,
		"{{CLASS_NAME}}":  className,
	}

	templateFiles := map[string]string{
		"templates/plugin/manifest.json":                  "manifest.json",
		"templates/plugin/package.json":                   "package.json",
		"templates/plugin/tsconfig.json":                  "tsconfig.json",
		"templates/plugin/src/index.ts":                   "src/index.ts",
		"templates/plugin/gitignore":                      ".gitignore",
		"templates/plugin/README.md":                      "README.md",
		"templates/plugin/github/workflows/ci-plugin.yml": ".github/workflows/ci-plugin.yml",
		"templates/plugin/github/workflows/cd-plugin.yml": ".github/workflows/cd-plugin.yml",
	}

	for templatePath, outputPath := range templateFiles {
		content, err := templatesFS.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		processedContent := applyReplacements(string(content), replacements)
		fullPath := filepath.Join(dir, outputPath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
		}

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

func toPascalCase(s string) string {
	parts := strings.Split(s, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
