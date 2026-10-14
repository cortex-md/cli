package theme

import (
	"embed"
	"errors"
	"fmt"

	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/internal/scaffold"
)

var (
	ErrInvalidID   = errors.New("invalid theme id")
	ErrInvalidName = errors.New("invalid theme name")
)

//go:embed templates/theme/*
//go:embed templates/theme/github/workflows/*
var templatesFS embed.FS

var templateFiles = []scaffold.File{
	{Template: "templates/theme/manifest.json", Output: "manifest.json"},
	{Template: "templates/theme/package.json", Output: "package.json"},
	{Template: "templates/theme/theme-dark.css", Output: "theme-dark.css"},
	{Template: "templates/theme/theme-light.css", Output: "theme-light.css"},
	{Template: "templates/theme/gitignore", Output: ".gitignore"},
	{Template: "templates/theme/README.md", Output: "README.md"},
	{Template: "templates/theme/github/workflows/ci-theme.yml", Output: ".github/workflows/ci-theme.yml"},
	{Template: "templates/theme/github/workflows/cd-theme.yml", Output: ".github/workflows/cd-theme.yml"},
}

type CreateOptions struct {
	Name        string
	ID          string
	Description string
	Author      string
	Directory   string
}

func Create(opts CreateOptions) (string, error) {
	if err := normalizeCreateOptions(&opts); err != nil {
		return "", err
	}

	targetDir := opts.Directory
	if targetDir == "" {
		targetDir = opts.ID
	}

	absDir, err := scaffold.PrepareTarget(targetDir)
	if err != nil {
		return "", err
	}

	return absDir, scaffold.Write(templatesFS, absDir, templateFiles, map[string]string{
		"ID":           opts.ID,
		"DISPLAY_NAME": opts.Name,
		"DESCRIPTION":  opts.Description,
		"AUTHOR":       opts.Author,
	})
}

func UpdateWorkflows(dir string) error {
	return scaffold.Write(templatesFS, dir, scaffold.WorkflowFiles(templateFiles), nil)
}

func normalizeCreateOptions(opts *CreateOptions) error {
	if opts.Name == "" {
		return ErrInvalidName
	}
	if opts.ID == "" {
		opts.ID = rules.NormalizeID(opts.Name)
	}
	if !rules.IsValidID(opts.ID) {
		return fmt.Errorf("%w: %q (use lowercase letters, numbers and hyphens)", ErrInvalidID, opts.ID)
	}
	if opts.Description == "" {
		opts.Description = fmt.Sprintf("A Cortex theme for %s", opts.Name)
	}
	if opts.Author == "" {
		opts.Author = "Unknown"
	}
	return nil
}
