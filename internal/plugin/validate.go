package plugin

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

var securityPatterns = []struct {
	Pattern  *regexp.Regexp
	Severity string
	Message  string
}{
	{regexp.MustCompile(`\beval\s*\(`), "critical", "eval() usage detected - potential security risk"},
	{regexp.MustCompile(`new\s+Function\s*\(`), "critical", "Function constructor detected - potential security risk"},
	{regexp.MustCompile(`child_process`), "critical", "child_process usage detected - not allowed in plugins"},
	{regexp.MustCompile(`\bexec\s*\(`), "critical", "exec() usage detected - not allowed in plugins"},
	{regexp.MustCompile(`\bspawn\s*\(`), "critical", "spawn() usage detected - not allowed in plugins"},

	{regexp.MustCompile(`@tauri-apps/api`), "critical", "Direct Tauri API usage detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`@tauri-apps/plugin`), "critical", "Direct Tauri plugin usage detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`tauri\.invoke`), "critical", "Direct Tauri invoke detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`__TAURI__`), "critical", "Direct Tauri global detected - use @cortex/plugin-api instead"},

	{regexp.MustCompile(`from\s+['"]react-native['""]`), "critical", "Direct React Native import detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`from\s+['"]expo-`), "critical", "Direct Expo import detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`import\s*\{[^}]*\}\s*from\s+['"]react-native`), "critical", "Direct React Native import detected - use @cortex/plugin-api instead"},

	{regexp.MustCompile(`from\s+['"]fs['""]`), "critical", "Direct Node.js fs import detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`from\s+['"]node:fs['""]`), "critical", "Direct Node.js fs import detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`require\s*\(\s*['"]fs['"]\s*\)`), "critical", "Direct Node.js fs require detected - use @cortex/plugin-api instead"},
	{regexp.MustCompile(`from\s+['"]path['""]`), "warning", "Direct Node.js path import detected - use @cortex/plugin-api utilities"},
	{regexp.MustCompile(`from\s+['"]node:path['""]`), "warning", "Direct Node.js path import detected - use @cortex/plugin-api utilities"},

	{regexp.MustCompile(`<script[^>]*>`), "warning", "Inline script tag detected"},
	{regexp.MustCompile(`dangerouslySetInnerHTML`), "warning", "dangerouslySetInnerHTML detected - potential XSS risk"},
}

const (
	maxBundleSizeWarning = 1 * 1024 * 1024
	maxBundleSizeError   = 5 * 1024 * 1024
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

	ux.Step("Validating plugin structure...")

	if err := validateStructure(absDir, result); err != nil {
		return result, err
	}

	ux.Step("Loading manifest...")

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		result.addError("Failed to load manifest: %v", err)
		result.Passed = false
		return result, nil
	}

	ux.Step("Validating manifest schema...")

	validateManifest(m, result)

	ux.Step("Checking for security issues...")

	validateSecurity(absDir, result)

	ux.Step("Checking bundle size...")

	validateBundleSize(absDir, m, result, opts.Strict)

	if opts.Strict && len(result.Warnings) > 0 {
		for _, warn := range result.Warnings {
			result.addError("%s", warn)
		}
		result.Warnings = []string{}
		result.Passed = false
	}

	return result, nil
}

func validateStructure(dir string, result *ValidationResult) error {
	required := []string{
		"manifest.json",
		"package.json",
		"src",
	}

	for _, file := range required {
		path := filepath.Join(dir, file)
		if !fileExists(path) {
			result.addError("Required file/directory missing: %s", file)
			result.Passed = false
		}
	}

	return nil
}

func validateManifest(m *manifest.PluginManifest, result *ValidationResult) {
	if !semver.IsValid(m.Version) {
		result.addError("Invalid version format: %s (must be valid semver)", m.Version)
		result.Passed = false
	}

	if m.MinAppVersion != "" && !semver.IsValid(m.MinAppVersion) {
		result.addError("Invalid minAppVersion format: %s (must be valid semver)", m.MinAppVersion)
		result.Passed = false
	}

	idPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !idPattern.MatchString(m.ID) {
		result.addError("Invalid plugin ID: %s (must be lowercase alphanumeric with hyphens)", m.ID)
		result.Passed = false
	}

	if len(m.ID) < 3 {
		result.addWarning("Plugin ID is very short: %s (recommended: 3+ characters)", m.ID)
	}

	if m.Description == "" || len(m.Description) < 10 {
		result.addWarning("Description is too short (recommended: 10+ characters)")
	}

	if m.Icon == "" {
		result.addWarning("No icon specified (using default)")
	}

	result.addInfo("Plugin: %s v%s", m.Name, m.Version)
	result.addInfo("Author: %s", m.Author)
}

func validateSecurity(dir string, result *ValidationResult) {
	distPath := filepath.Join(dir, "dist")
	if !fileExists(distPath) {
		result.addWarning("dist/ directory not found (run 'cortex plugin build' first)")
		return
	}

	err := filepath.WalkDir(distPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".js") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(dir, path)
		scanContent(string(content), relPath, result)

		return nil
	})

	if err != nil {
		result.addWarning("Failed to scan files for security issues: %v", err)
	}
}

func scanContent(content string, filePath string, result *ValidationResult) {
	for _, pattern := range securityPatterns {
		if pattern.Pattern.MatchString(content) {
			msg := fmt.Sprintf("%s in %s", pattern.Message, filePath)
			if pattern.Severity == "critical" {
				result.addError("%s", msg)
				result.Passed = false
			} else {
				result.addWarning("%s", msg)
			}
		}
	}
}

func validateBundleSize(dir string, m *manifest.PluginManifest, result *ValidationResult, strict bool) {
	mainPath := filepath.Join(dir, m.Main)
	if !fileExists(mainPath) {
		result.addError("Main entry file not found: %s", m.Main)
		result.Passed = false
		return
	}

	info, err := os.Stat(mainPath)
	if err != nil {
		result.addWarning("Failed to check bundle size: %v", err)
		return
	}

	size := info.Size()
	sizeKB := float64(size) / 1024
	sizeMB := sizeKB / 1024

	if size > maxBundleSizeError {
		msg := fmt.Sprintf("Bundle size too large: %.2f MB (max: 5 MB)", sizeMB)
		if strict {
			result.addError("%s", msg)
			result.Passed = false
		} else {
			result.addWarning("%s", msg)
		}
	} else if size > maxBundleSizeWarning {
		result.addWarning("Bundle size is large: %.2f MB (recommended: < 1 MB)", sizeMB)
	} else {
		result.addInfo("Bundle size: %.2f KB", sizeKB)
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
