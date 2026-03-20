package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
	"github.com/cortex/cli/pkg/semver"
)

type ValidationResult struct {
	Passed   bool
	Errors   []string
	Warnings []string
	Info     []string
}

type ValidateOptions struct {
	Strict bool
}

var requiredCSSVariables = []string{
	"--bg-primary",
	"--bg-secondary",
	"--text-primary",
	"--text-secondary",
	"--accent-default",
	"--border-default",
}

var recommendedCSSVariables = []string{
	"--bg-tertiary",
	"--bg-elevated",
	"--bg-hover",
	"--bg-active",
	"--text-muted",
	"--text-disabled",
	"--accent-hover",
	"--accent-active",
	"--border-subtle",
	"--border-strong",
	"--syntax-keyword",
	"--syntax-string",
	"--syntax-comment",
	"--syntax-number",
	"--syntax-function",
}

const (
	maxCSSFileSizeWarning = 50 * 1024
	maxCSSFileSizeError   = 200 * 1024
)

func Validate(dir string, opts ValidateOptions) (*ValidationResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	result := &ValidationResult{
		Passed:   true,
		Errors:   []string{},
		Warnings: []string{},
		Info:     []string{},
	}

	ux.Step("Validating theme structure...")

	if err := validateThemeStructure(absDir, result); err != nil {
		return result, err
	}

	ux.Step("Loading manifest...")

	m, err := manifest.LoadTheme(absDir)
	if err != nil {
		result.addError("Failed to load manifest: %v", err)
		result.Passed = false
		return result, nil
	}

	ux.Step("Validating manifest schema...")

	validateThemeManifest(m, result)

	ux.Step("Validating CSS files...")

	validateCSSFiles(absDir, m, result, opts.Strict)

	if opts.Strict && len(result.Warnings) > 0 {
		for _, warn := range result.Warnings {
			result.addError("%s", warn)
		}
		result.Warnings = []string{}
		result.Passed = false
	}

	return result, nil
}

func validateThemeStructure(dir string, result *ValidationResult) error {
	required := []string{
		"manifest.json",
	}

	for _, file := range required {
		path := filepath.Join(dir, file)
		if !fileExists(path) {
			result.addError("Required file missing: %s", file)
			result.Passed = false
		}
	}

	return nil
}

func validateThemeManifest(m *manifest.ThemeManifest, result *ValidationResult) {
	if !semver.IsValid(m.Version) {
		result.addError("Invalid version format: %s (must be valid semver)", m.Version)
		result.Passed = false
	}

	idPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !idPattern.MatchString(m.ID) {
		result.addError("Invalid theme ID: %s (must be lowercase alphanumeric with hyphens)", m.ID)
		result.Passed = false
	}

	if len(m.ID) < 3 {
		result.addWarning("Theme ID is very short: %s (recommended: 3+ characters)", m.ID)
	}

	if m.DisplayName == "" {
		result.addWarning("No displayName specified (using name)")
	}

	if m.Description == "" || len(m.Description) < 10 {
		result.addWarning("Description is too short (recommended: 10+ characters)")
	}

	if len(m.Colorschemes) == 0 {
		result.addError("No colorschemes defined in manifest")
		result.Passed = false
	}

	hasDark := false
	hasLight := false
	for scheme := range m.Colorschemes {
		if scheme == "dark" {
			hasDark = true
		}
		if scheme == "light" {
			hasLight = true
		}
	}

	if !hasDark && !hasLight {
		result.addWarning("Theme has no 'dark' or 'light' colorscheme - may not be selectable in app")
	}

	result.addInfo("Theme: %s v%s", m.Name, m.Version)
	result.addInfo("Author: %s", m.Author)
	result.addInfo("Colorschemes: %d", len(m.Colorschemes))
}

func validateCSSFiles(dir string, m *manifest.ThemeManifest, result *ValidationResult, strict bool) {
	for schemeName, cssPath := range m.Colorschemes {
		fullPath := filepath.Join(dir, cssPath)

		if !fileExists(fullPath) {
			result.addError("CSS file not found for colorscheme '%s': %s", schemeName, cssPath)
			result.Passed = false
			continue
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			result.addError("Failed to read CSS file %s: %v", cssPath, err)
			result.Passed = false
			continue
		}

		validateCSSContent(string(content), schemeName, cssPath, result, strict)
		validateCSSFileSize(fullPath, schemeName, result, strict)
	}
}

func validateCSSContent(content string, schemeName string, filePath string, result *ValidationResult, strict bool) {
	for _, varName := range requiredCSSVariables {
		if !strings.Contains(content, varName+":") {
			result.addError("Missing required CSS variable in %s (%s): %s", filePath, schemeName, varName)
			result.Passed = false
		}
	}

	missingRecommended := []string{}
	for _, varName := range recommendedCSSVariables {
		if !strings.Contains(content, varName+":") {
			missingRecommended = append(missingRecommended, varName)
		}
	}

	if len(missingRecommended) > 0 {
		result.addWarning("Missing recommended CSS variables in %s: %d variables", filePath, len(missingRecommended))
	}

	if strings.Contains(content, "@import") {
		result.addWarning("@import found in %s - may affect theme loading performance", filePath)
	}

	urlPattern := regexp.MustCompile(`url\s*\(\s*['"]?https?://`)
	if urlPattern.MatchString(content) {
		result.addWarning("External URL found in %s - may not work offline", filePath)
	}

	if !strings.Contains(content, ":root") {
		result.addWarning("No :root selector found in %s - variables should be defined in :root", filePath)
	}
}

func validateCSSFileSize(path string, schemeName string, result *ValidationResult, strict bool) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	size := info.Size()
	sizeKB := float64(size) / 1024

	if size > maxCSSFileSizeError {
		msg := fmt.Sprintf("CSS file too large for %s: %.2f KB (max: 200 KB)", schemeName, sizeKB)
		if strict {
			result.addError("%s", msg)
			result.Passed = false
		} else {
			result.addWarning("%s", msg)
		}
	} else if size > maxCSSFileSizeWarning {
		result.addWarning("CSS file is large for %s: %.2f KB (recommended: < 50 KB)", schemeName, sizeKB)
	} else {
		result.addInfo("CSS file size (%s): %.2f KB", schemeName, sizeKB)
	}
}

func (r *ValidationResult) addError(format string, args ...interface{}) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *ValidationResult) addWarning(format string, args ...interface{}) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *ValidationResult) addInfo(format string, args ...interface{}) {
	r.Info = append(r.Info, fmt.Sprintf(format, args...))
}

func (r *ValidationResult) Print() {
	fmt.Println()

	if len(r.Info) > 0 {
		for _, info := range r.Info {
			ux.Info("%s", info)
		}
		fmt.Println()
	}

	if len(r.Warnings) > 0 {
		ux.Warning("Found %d warning(s):", len(r.Warnings))
		for _, warn := range r.Warnings {
			fmt.Printf("  ⚠ %s\n", warn)
		}
		fmt.Println()
	}

	if len(r.Errors) > 0 {
		ux.Error("Found %d error(s):", len(r.Errors))
		for _, err := range r.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
		fmt.Println()
	}

	if r.Passed {
		ux.Success("Validation passed!")
	} else {
		ux.Error("Validation failed")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
