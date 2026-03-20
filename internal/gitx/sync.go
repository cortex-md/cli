package gitx

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type SyncOptions struct {
	Directory     string
	RemoteURL     string
	CommitMessage string
}

func SyncForPublish(opts SyncOptions) error {
	if strings.TrimSpace(opts.Directory) == "" {
		return fmt.Errorf("directory is required")
	}

	absDir, err := filepath.Abs(opts.Directory)
	if err != nil {
		return err
	}

	if err := ensureRepository(absDir); err != nil {
		return err
	}

	if err := ensureOriginRemote(absDir, opts.RemoteURL); err != nil {
		return err
	}

	if err := runGit(absDir, "add", "-A"); err != nil {
		return err
	}

	hasChanges, err := repositoryHasChanges(absDir)
	if err != nil {
		return err
	}

	if hasChanges {
		message := strings.TrimSpace(opts.CommitMessage)
		if message == "" {
			message = "chore: publish"
		}
		if err := runGit(absDir, "commit", "-m", message); err != nil {
			return err
		}
	}

	branch, err := currentBranch(absDir)
	if err != nil {
		return err
	}

	if err := runGit(absDir, "push", "-u", "origin", branch); err != nil {
		return err
	}

	return nil
}

func ensureRepository(dir string) error {
	if err := runGit(dir, "rev-parse", "--is-inside-work-tree"); err == nil {
		return nil
	}

	if err := runGit(dir, "init"); err != nil {
		return err
	}

	if err := runGit(dir, "checkout", "-B", "main"); err != nil {
		return err
	}

	return nil
}

func ensureOriginRemote(dir string, remoteURL string) error {
	if strings.TrimSpace(remoteURL) == "" {
		return fmt.Errorf("remote URL is required")
	}

	current, err := runGitOutput(dir, "remote", "get-url", "origin")
	if err != nil {
		return runGit(dir, "remote", "add", "origin", remoteURL)
	}

	if strings.TrimSpace(current) == strings.TrimSpace(remoteURL) {
		return nil
	}

	return runGit(dir, "remote", "set-url", "origin", remoteURL)
}

func repositoryHasChanges(dir string) (bool, error) {
	output, err := runGitOutput(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}

func currentBranch(dir string) (string, error) {
	branch, err := runGitOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		if checkoutErr := runGit(dir, "checkout", "-B", "main"); checkoutErr != nil {
			return "", checkoutErr
		}
		return "main", nil
	}

	trimmed := strings.TrimSpace(branch)
	if trimmed == "" || trimmed == "HEAD" {
		if checkoutErr := runGit(dir, "checkout", "-B", "main"); checkoutErr != nil {
			return "", checkoutErr
		}
		return "main", nil
	}

	return trimmed, nil
}

func runGit(dir string, args ...string) error {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func runGitOutput(dir string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
