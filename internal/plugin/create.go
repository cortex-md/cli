package plugin

import (
	"embed"
	"errors"
	"fmt"

	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/internal/scaffold"
)

var (
	ErrInvalidID   = errors.New("invalid plugin id")
	ErrInvalidName = errors.New("invalid plugin name")
)

//go:embed templates/plugin/*
//go:embed templates/plugin/src/*
//go:embed templates/plugin/github/workflows/*
var templatesFS embed.FS

var templateFiles = []scaffold.File{
	{Template: "templates/plugin/manifest.json", Output: "manifest.json"},
	{Template: "templates/plugin/package.json", Output: "package.json"},
	{Template: "templates/plugin/tsconfig.json", Output: "tsconfig.json"},
	{Template: "templates/plugin/src/index.ts", Output: "src/index.ts"},
	{Template: "templates/plugin/gitignore", Output: ".gitignore"},
	{Template: "templates/plugin/README.md", Output: "README.md"},
	{Template: "templates/plugin/github/workflows/ci-plugin.yml", Output: ".github/workflows/ci-plugin.yml"},
	{Template: "templates/plugin/github/workflows/cd-plugin.yml", Output: ".github/workflows/cd-plugin.yml"},
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
		"ID":          opts.ID,
		"NAME":        opts.Name,
		"DESCRIPTION": opts.Description,
		"AUTHOR":      opts.Author,
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
		opts.Description = fmt.Sprintf("A Cortex plugin for %s", opts.Name)
	}
	if opts.Author == "" {
		opts.Author = "Unknown"
	}
	return nil
}
