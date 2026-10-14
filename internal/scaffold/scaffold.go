package scaffold

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	InitialVersion = "0.1.0"
	MinAppVersion  = "0.1.1"
	APIVersion     = "^0.2.0"
	CLIVersion     = "^0.3.0"
)

var ErrDirExists = errors.New("directory already exists")

type File struct {
	Template string
	Output   string
}

func Write(templates embed.FS, dir string, files []File, values map[string]string) error {
	replacements := map[string]string{
		"{{VERSION}}":         InitialVersion,
		"{{MIN_APP_VERSION}}": MinAppVersion,
		"{{API_VERSION}}":     APIVersion,
		"{{CLI_VERSION}}":     CLIVersion,
	}
	for key, value := range values {
		replacements["{{"+key+"}}"] = value
	}

	for _, file := range files {
		content, err := templates.ReadFile(file.Template)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", file.Template, err)
		}

		rendered := string(content)
		for key, value := range replacements {
			rendered = strings.ReplaceAll(rendered, key, value)
		}

		target := filepath.Join(dir, file.Output)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", file.Output, err)
		}
		if err := os.WriteFile(target, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file.Output, err)
		}
	}

	return nil
}

func PrepareTarget(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(absDir); err == nil {
		return "", fmt.Errorf("%w: %s", ErrDirExists, absDir)
	}
	return absDir, os.MkdirAll(absDir, 0o755)
}

func WorkflowFiles(files []File) []File {
	workflows := []File{}
	for _, file := range files {
		if strings.HasPrefix(file.Output, ".github/") {
			workflows = append(workflows, file)
		}
	}
	return workflows
}
