package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorDoesNotRequireGithubTokenForLocalPublish(t *testing.T) {
	dir := t.TempDir()
	writeDoctorFile(t, filepath.Join(dir, "manifest.json"), `{
		"id":"doctor-plugin",
		"name":"Doctor Plugin",
		"version":"1.2.3",
		"minAppVersion":"0.1.0",
		"author":"furqas",
		"description":"A useful doctor plugin",
		"icon":"puzzle",
		"main":"dist/index.js"
	}`)
	writeDoctorFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"build":"bun build"}}`)
	writeDoctorFile(t, filepath.Join(dir, "src", "index.ts"), "export default {}")
	installDoctorFakeExecutable(t, "bun")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("CORTEX_TOKEN", "")

	result, err := Doctor(dir)
	if err != nil {
		t.Fatal(err)
	}

	foundRepositoryWarning := false
	for _, issue := range result.Issues {
		text := issue.Message + " " + issue.Fix
		if strings.Contains(text, "GITHUB_TOKEN") {
			t.Fatalf("doctor should not mention GITHUB_TOKEN for local publish: %#v", issue)
		}
		if strings.Contains(issue.Message, "manifest.repository is empty") {
			foundRepositoryWarning = true
			if !strings.Contains(issue.Fix, "git remote origin") {
				t.Fatalf("repository warning should mention git remote fallback: %#v", issue)
			}
		}
	}

	if !foundRepositoryWarning {
		t.Fatal("expected repository metadata warning")
	}
}

func writeDoctorFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func installDoctorFakeExecutable(t *testing.T, name string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}
