package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cortex/cli/internal/report"
	"github.com/cortex/cli/pkg/manifest"
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

	result := checkTheme(t, dir)
	if !result.Passed(false) {
		t.Fatalf("validation should pass, errors = %v", messages(result, report.LevelFail))
	}
	if hasValidationMessage(messages(result, report.LevelWarn), "has no body.theme-") {
		t.Fatalf("theme selector should satisfy root scope warning, warnings = %v", messages(result, report.LevelWarn))
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

	result := checkTheme(t, dir)
	if !result.Passed(false) {
		t.Fatalf("validation should pass, errors = %v", messages(result, report.LevelFail))
	}
	if hasValidationMessage(messages(result, report.LevelWarn), "has no body.theme-") {
		t.Fatalf("class selector should satisfy root scope warning, warnings = %v", messages(result, report.LevelWarn))
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

	result := checkTheme(t, dir)
	if !result.Passed(false) {
		t.Fatalf("legacy aliases should pass, errors = %v", messages(result, report.LevelFail))
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

	result := checkTheme(t, dir)
	if result.Passed(false) {
		t.Fatal("validation should fail without accent and border families")
	}
	if !hasValidationMessage(messages(result, report.LevelFail), "--accent") {
		t.Fatalf("missing accent error not found, errors = %v", messages(result, report.LevelFail))
	}
	if !hasValidationMessage(messages(result, report.LevelFail), "--border") {
		t.Fatalf("missing border error not found, errors = %v", messages(result, report.LevelFail))
	}
}

func writeValidationTheme(
	t *testing.T,
	themeName string,
	cssForScheme func(themeName string, schemeName string) string,
) string {
	t.Helper()
	dir := t.TempDir()
	manifestJSON := fmt.Sprintf(`{
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

	writeValidationFile(t, filepath.Join(dir, "manifest.json"), manifestJSON)
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

func checkTheme(t *testing.T, dir string) *report.Report {
	t.Helper()
	themeManifest, err := manifest.LoadTheme(dir)
	if err != nil {
		t.Fatal(err)
	}
	return Check(dir, themeManifest)
}

func messages(result *report.Report, level report.Level) []string {
	found := []string{}
	for _, section := range result.Sections {
		for _, check := range section.Checks {
			if check.Level == level {
				found = append(found, check.Message)
			}
		}
	}
	return found
}
