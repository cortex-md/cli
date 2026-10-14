package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cortex/cli/internal/report"
)

func CheckReadiness(r *report.Report, dir string, kind Kind, explicitRepository string) {
	if repository := ResolveRepository(dir, explicitRepository); repository != "" {
		r.Pass("Releases are published from %s", repository)
	} else {
		r.Warn(
			"No GitHub repository found for releases",
			`push the project to GitHub (git remote origin) or set "repository": "owner/repo" in manifest.json`,
		)
	}

	if fileExists(filepath.Join(dir, "README.md")) {
		r.Pass("README.md is present (shown on the Marketplace page)")
	} else {
		r.Warn("README.md is missing", "add a README.md; the registry requires it and Marketplace displays it")
	}

	if !fileExists(filepath.Join(dir, "LICENSE")) && !fileExists(filepath.Join(dir, "LICENSE.md")) {
		r.Info("No LICENSE file", "add a LICENSE so users know how they can use your "+string(kind))
	}

	checkReleaseWorkflow(r, dir, kind)
}

func checkReleaseWorkflow(r *report.Report, dir string, kind Kind) {
	workflowPath := filepath.Join(dir, ".github", "workflows", "cd-"+string(kind)+".yml")
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		r.Warn("No release workflow found", "run `cortex create --update-ci` to add the GitHub Actions release workflow")
		return
	}

	if !strings.Contains(string(content), "cortex pack") {
		r.Warn("Release workflow is outdated", "run `cortex create --update-ci`; releases are now triggered by `cortex release` tags")
		return
	}

	r.Pass("Release workflow publishes on version tags")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
