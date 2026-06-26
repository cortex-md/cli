package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsCurrentThemeVariables(t *testing.T) {
	dir := writeValidationTheme(t, "current-theme", func(themeName string, schemeName string) string {
		return fmt.Sprintf(`body.theme-%s-%s {
	--bg-primary: #ffffff;
	--bg-secondary: #f6f6f6;
	--text-primary: #111111;
	--text-secondary: #222222;
	--accent: #0066cc;
	--border: #cccccc;
}`, themeName, schemeName)
	})

	result, err := Validate(dir, ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("validation should pass, errors = %v", result.Errors)
	}
	if hasValidationMessage(result.Warnings, "No theme root selector") {
		t.Fatalf("theme selector should satisfy root scope warning, warnings = %v", result.Warnings)
	}
}

func TestValidateAcceptsClassOnlyThemeScope(t *testing.T) {
	dir := writeValidationTheme(t, "class-theme", func(themeName string, schemeName string) string {
		return fmt.Sprintf(`.theme-%s-%s {
	--bg-primary: #ffffff;
	--bg-secondary: #f6f6f6;
	--text-primary: #111111;
	--text-secondary: #222222;
	--accent: #0066cc;
	--border: #cccccc;
}`, themeName, schemeName)
	})

	result, err := Validate(dir, ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("validation should pass, errors = %v", result.Errors)
	}
	if hasValidationMessage(result.Warnings, "No theme root selector") {
		t.Fatalf("class selector should satisfy root scope warning, warnings = %v", result.Warnings)
	}
}

func TestValidateAcceptsLegacyAccentAndBorderAliases(t *testing.T) {
	dir := writeValidationTheme(t, "legacy-theme", func(themeName string, schemeName string) string {
		return `:root {
	--bg-primary: #ffffff;
	--bg-secondary: #f6f6f6;
	--text-primary: #111111;
	--text-secondary: #222222;
	--accent-default: #0066cc;
	--border-default: #cccccc;
}`
	})

	result, err := Validate(dir, ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("legacy aliases should pass, errors = %v", result.Errors)
	}
}

func TestValidateRejectsMissingAccentAndBorderFamilies(t *testing.T) {
	dir := writeValidationTheme(t, "broken-theme", func(themeName string, schemeName string) string {
		return `:root {
	--bg-primary: #ffffff;
	--bg-secondary: #f6f6f6;
	--text-primary: #111111;
	--text-secondary: #222222;
}`
	})

	result, err := Validate(dir, ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Fatal("validation should fail without accent and border families")
	}
	if !hasValidationMessage(result.Errors, "--accent") {
		t.Fatalf("missing accent error not found, errors = %v", result.Errors)
	}
	if !hasValidationMessage(result.Errors, "--border") {
		t.Fatalf("missing border error not found, errors = %v", result.Errors)
	}
}

func writeValidationTheme(
	t *testing.T,
	themeName string,
	cssForScheme func(themeName string, schemeName string) string,
) string {
	t.Helper()
	dir := t.TempDir()
	manifest := fmt.Sprintf(`{
	"id": "%s",
	"name": "%s",
	"displayName": "Validation Theme",
	"version": "1.0.0",
	"author": "Cortex",
	"description": "A validation fixture theme",
	"colorschemes": {
		"dark": "dark.css",
		"light": "light.css"
	}
}`, themeName, themeName)

	writeValidationFile(t, filepath.Join(dir, "manifest.json"), manifest)
	writeValidationFile(t, filepath.Join(dir, "dark.css"), cssForScheme(themeName, "dark"))
	writeValidationFile(t, filepath.Join(dir, "light.css"), cssForScheme(themeName, "light"))

	return dir
}

func writeValidationFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasValidationMessage(messages []string, needle string) bool {
	for _, message := range messages {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
