package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cortex/cli/pkg/manifest"
)

type DoctorIssue struct {
	Severity string
	Message  string
	Fix      string
}

type DoctorResult struct {
	Passed bool
	Issues []DoctorIssue
}

func Doctor(dir string) (*DoctorResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	result := &DoctorResult{
		Passed: true,
		Issues: []DoctorIssue{},
	}

	if !fileExists(filepath.Join(absDir, "manifest.json")) {
		result.addFail("manifest.json missing", "run `cortex plugin create` or add a valid manifest")
		return result, nil
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		result.addFail("invalid manifest", "fix manifest.json required fields and schema")
		return result, nil
	}

	if !fileExists(filepath.Join(absDir, "package.json")) {
		result.addFail("package.json missing", "create package.json with build/dev scripts")
	}

	if !fileExists(filepath.Join(absDir, "src", "index.ts")) {
		result.addFail("src/index.ts missing", "create src/index.ts as plugin entry")
	}

	if !execExists("git") {
		result.addWarn("git not found in PATH", "install git for publish and release workflows")
	}

	if !execExists("bun") && !execExists("npm") && !execExists("pnpm") && !execExists("yarn") {
		result.addFail("no package manager found", "install bun, npm, pnpm, or yarn")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		result.addWarn("GITHUB_TOKEN is not set", "set GITHUB_TOKEN before running publish")
	}

	if m.Repository == "" {
		result.addWarn("manifest.repository is empty", "set repository for better registry metadata")
	}

	validateResult, validateErr := Validate(absDir, ValidateOptions{Strict: false})
	if validateErr != nil {
		result.addFail("validate command failed", "run `cortex plugin validate` and fix errors")
	} else {
		if len(validateResult.Errors) > 0 {
			result.addFail("plugin has validation errors", "run `cortex plugin validate` and fix all errors")
		}
		if len(validateResult.Warnings) > 0 {
			result.addWarn("plugin has validation warnings", "run `cortex plugin validate --strict` for release readiness")
		}
	}

	if fileExists(filepath.Join(absDir, "dist")) {
		mainPath := filepath.Join(absDir, m.Main)
		if fileExists(mainPath) {
			stat, statErr := os.Stat(mainPath)
			if statErr == nil {
				sizeKB := float64(stat.Size()) / 1024
				if stat.Size() > maxBundleSizeWarning {
					result.addWarn(
						fmt.Sprintf("bundle is large (%.2f KB)", sizeKB),
						"reduce dependencies and tree-shake output",
					)
				}
			}
		}
	} else {
		result.addInfo("dist/ not found", "run `cortex plugin build` before publish")
	}

	if len(result.Issues) > 0 {
		for _, issue := range result.Issues {
			if issue.Severity == "fail" {
				result.Passed = false
				break
			}
		}
	}

	return result, nil
}

func (r *DoctorResult) addFail(message, fix string) {
	r.Issues = append(r.Issues, DoctorIssue{Severity: "fail", Message: message, Fix: fix})
	r.Passed = false
}

func (r *DoctorResult) addWarn(message, fix string) {
	r.Issues = append(r.Issues, DoctorIssue{Severity: "warn", Message: message, Fix: fix})
}

func (r *DoctorResult) addInfo(message, fix string) {
	r.Issues = append(r.Issues, DoctorIssue{Severity: "info", Message: message, Fix: fix})
}
