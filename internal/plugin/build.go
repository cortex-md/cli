package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cortex/cli/internal/ux"
)

var ErrNoPackageManager = errors.New("no package manager found")

var lockfiles = []struct {
	manager string
	file    string
}{
	{"bun", "bun.lock"},
	{"bun", "bun.lockb"},
	{"pnpm", "pnpm-lock.yaml"},
	{"yarn", "yarn.lock"},
	{"npm", "package-lock.json"},
}

func DetectPackageManager(dir string) string {
	for _, lockfile := range lockfiles {
		if fileExists(filepath.Join(dir, lockfile.file)) && execExists(lockfile.manager) {
			return lockfile.manager
		}
	}
	for _, manager := range []string{"bun", "pnpm", "yarn", "npm"} {
		if execExists(manager) {
			return manager
		}
	}
	return ""
}

func Build(dir string) error {
	if !fileExists(filepath.Join(dir, "package.json")) {
		return ux.NewProblem("package.json not found", "plugins are built with the `build` script in package.json", nil)
	}
	if !fileExists(filepath.Join(dir, "node_modules")) {
		return ux.NewProblem("Dependencies are not installed", "run `"+installCommand(DetectPackageManager(dir))+"` first", nil)
	}

	manager := DetectPackageManager(dir)
	if manager == "" {
		return ux.NewProblem("No package manager found", "install bun (recommended), npm, pnpm or yarn", ErrNoPackageManager)
	}

	return runQuiet(dir, manager, "run", "build")
}

func InstallDependencies(dir string) error {
	manager := DetectPackageManager(dir)
	if manager == "" {
		return ux.NewProblem("No package manager found", "install bun (recommended), npm, pnpm or yarn", ErrNoPackageManager)
	}
	return runQuiet(dir, manager, "install")
}

func installCommand(manager string) string {
	if manager == "" {
		manager = "bun"
	}
	return manager + " install"
}

func runQuiet(dir string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = dir
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output

	if err := command.Run(); err != nil {
		return ux.NewProblem(
			fmt.Sprintf("`%s %s` failed", name, strings.Join(args, " ")),
			"fix the errors above and try again",
			errors.New(strings.TrimSpace(tail(output.String(), 40))),
		)
	}
	return nil
}

func tail(text string, lines int) string {
	parts := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(parts) <= lines {
		return text
	}
	return strings.Join(parts[len(parts)-lines:], "\n")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func execExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
