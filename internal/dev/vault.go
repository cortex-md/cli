package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortex/cli/internal/config"
	"github.com/cortex/cli/internal/ux"
)

const vaultMarker = ".cortex/vault-id.json"

func ResolveVaultPath(startDir string, explicitVaultPath string) (string, error) {
	candidate := strings.TrimSpace(explicitVaultPath)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv("CORTEX_VAULT"))
	}
	if candidate != "" {
		return validateVault(candidate)
	}

	if vault := findAncestorVault(startDir); vault != "" {
		return vault, nil
	}

	if last := config.Load().LastVault; last != "" && IsVault(last) {
		return last, nil
	}

	return "", ux.NewProblem(
		"No Cortex vault selected",
		"pass --vault /path/to/vault (remembered for next time) or set CORTEX_VAULT",
		nil,
	)
}

func IsVault(path string) bool {
	_, err := os.Stat(filepath.Join(path, filepath.FromSlash(vaultMarker)))
	return err == nil
}

func validateVault(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve vault path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return "", ux.NewProblem("Vault not found: "+absPath, "pass the folder you open in Cortex", nil)
	}
	if !IsVault(absPath) {
		return "", ux.NewProblem(
			absPath+" is not a Cortex vault",
			"open the folder in Cortex once so it creates .cortex/, then try again",
			nil,
		)
	}
	return absPath, nil
}

func findAncestorVault(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		if IsVault(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
