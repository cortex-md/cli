package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestPluginPublishHelpOmitsLegacyFlags(t *testing.T) {
	output := commandOutput(t, NewRootCommand(), "plugin", "publish", "--help")
	assertOmitsLegacyPublishFlags(t, output)
}

func TestThemePublishHelpOmitsLegacyFlags(t *testing.T) {
	output := commandOutput(t, NewRootCommand(), "theme", "publish", "--help")
	assertOmitsLegacyPublishFlags(t, output)
}

func TestRegistrySubmitUsesDefaultBuildManifest(t *testing.T) {
	called := false
	var gotPath string
	cmd := newRegistrySubmitCommand(func(ctx context.Context, path string) (string, error) {
		called = true
		gotPath = path
		return "https://github.com/cortex-md/registry/pull/1", nil
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected registry submit runner to be called")
	}
	if gotPath != filepath.Join("dist", "cortex-build", "build.json") {
		t.Fatalf("default build json path = %q", gotPath)
	}

	cmd = newRegistrySubmitCommand(func(ctx context.Context, path string) (string, error) {
		called = true
		gotPath = path
		return "https://github.com/cortex-md/registry/pull/1", nil
	})
	called = false
	cmd.SetArgs([]string{"dist/cortex-build/build.json"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("expected registry submit runner to be called")
	}
	if gotPath != "dist/cortex-build/build.json" {
		t.Fatalf("build json path = %q", gotPath)
	}
}

func TestScaffoldedCDWorkflowsPublishReleaseOnly(t *testing.T) {
	workflowPaths := map[string]string{
		"plugin": "../plugin/templates/plugin/github/workflows/cd-plugin.yml",
		"theme":  "../theme/templates/theme/github/workflows/cd-theme.yml",
	}

	for kind, path := range workflowPaths {
		t.Run(kind, func(t *testing.T) {
			content := readFileString(t, path)

			assertContains(t, content, "permissions:\n  contents: write")
			assertContains(t, content, "Resolve release tag")
			assertContains(t, content, "branches:\n      - main")
			assertContains(t, content, "GITHUB_REF_TYPE")
			assertContains(t, content, "GITHUB_REF_NAME")
			assertContains(t, content, "git push origin")
			assertContains(t, content, "--latest")
			assertContains(t, content, "Publish GitHub release")
			assertOmits(t, content, "Require registry token")
			assertOmits(t, content, "CORTEX_TOKEN")
			assertOmits(t, content, "cortex registry submit")
		})
	}
}

func commandOutput(t *testing.T, cmd *cobra.Command, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertContains(t *testing.T, output string, expected string) {
	t.Helper()
	if !bytes.Contains([]byte(output), []byte(expected)) {
		t.Fatalf("expected output to contain %q:\n%s", expected, output)
	}
}

func assertOmits(t *testing.T, output string, unexpected string) {
	t.Helper()
	if bytes.Contains([]byte(output), []byte(unexpected)) {
		t.Fatalf("expected output to omit %q:\n%s", unexpected, output)
	}
}

func assertOmitsLegacyPublishFlags(t *testing.T, output string) {
	t.Helper()
	for _, legacyFlag := range []string{
		"--dry-run",
		"--skip-build",
		"--skip-validate",
		"--draft",
		"--prerelease",
		"--update-only",
		"--no-interactive",
		"--skip-git-sync",
		"--skip-registry-pr",
		"--yes",
		"--create-repo",
		"--repository",
		"--author",
		"--author-url",
		"--description",
		"--cover-image-url",
	} {
		assertOmits(t, output, legacyFlag)
	}
}
