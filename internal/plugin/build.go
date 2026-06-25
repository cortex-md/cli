package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

type BuildOptions struct {
	Watch bool
}

func Build(dir string, opts BuildOptions) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	m, err := manifest.LoadPlugin(absDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	ux.Info("Building plugin: %s v%s", m.Name, m.Version)

	if err := runBundler(absDir, opts.Watch); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	distPath := filepath.Join(absDir, "dist")
	if !fileExists(distPath) {
		return fmt.Errorf("build output directory not found: %s", distPath)
	}

	mainFile := filepath.Join(absDir, m.Main)
	if !fileExists(mainFile) {
		return fmt.Errorf("main entry file not found: %s", m.Main)
	}

	ux.Success("Build complete!")
	ux.Info("Output: %s", m.Main)

	return nil
}

func runBundler(dir string, watch bool) error {
	packageJSONPath := filepath.Join(dir, "package.json")
	if !fileExists(packageJSONPath) {
		return fmt.Errorf("package.json not found")
	}

	bundler := detectBundler(dir)
	var cmd *exec.Cmd

	switch bundler {
	case "bun":
		if watch {
			cmd = exec.Command("bun", "run", "dev")
		} else {
			cmd = exec.Command("bun", "run", "build")
		}
	case "npm":
		if watch {
			cmd = exec.Command("npm", "run", "dev")
		} else {
			cmd = exec.Command("npm", "run", "build")
		}
	case "pnpm":
		if watch {
			cmd = exec.Command("pnpm", "run", "dev")
		} else {
			cmd = exec.Command("pnpm", "run", "build")
		}
	case "yarn":
		if watch {
			cmd = exec.Command("yarn", "dev")
		} else {
			cmd = exec.Command("yarn", "build")
		}
	default:
		return fmt.Errorf("no package manager found (bun, npm, pnpm, or yarn required)")
	}

	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ux.Info("Running: %s", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bundler command failed: %w", err)
	}

	return nil
}

func detectBundler(dir string) string {
	bundlers := []struct {
		name string
		file string
	}{
		{"bun", "bun.lockb"},
		{"pnpm", "pnpm-lock.yaml"},
		{"yarn", "yarn.lock"},
		{"npm", "package-lock.json"},
	}

	for _, b := range bundlers {
		if fileExists(filepath.Join(dir, b.file)) {
			return b.name
		}
	}

	if execExists("bun") {
		return "bun"
	}
	if execExists("pnpm") {
		return "pnpm"
	}
	if execExists("yarn") {
		return "yarn"
	}
	if execExists("npm") {
		return "npm"
	}

	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func execExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
