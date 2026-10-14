package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/internal/ux"
)

var versionFieldPattern = regexp.MustCompile(`("version"\s*:\s*")([^"]*)(")`)

func NextVersion(current string, bump string) (string, error) {
	if rules.IsPlainSemver(bump) {
		return bump, nil
	}
	if !rules.IsPlainSemver(current) {
		return "", fmt.Errorf("current version %q is not MAJOR.MINOR.PATCH", current)
	}

	parts := strings.Split(current, ".")
	numbers := make([]int, 3)
	for index, part := range parts {
		numbers[index], _ = strconv.Atoi(part)
	}

	switch bump {
	case "major":
		return fmt.Sprintf("%d.0.0", numbers[0]+1), nil
	case "minor":
		return fmt.Sprintf("%d.%d.0", numbers[0], numbers[1]+1), nil
	case "patch":
		return fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2]+1), nil
	default:
		return "", ux.NewProblem(fmt.Sprintf("Unknown version bump %q", bump), "use patch, minor, major or an explicit version like 1.4.0", nil)
	}
}

func WriteVersion(dir string, version string) ([]string, error) {
	changed := []string{}
	for _, name := range []string{"manifest.json", "package.json"} {
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		location := versionFieldPattern.FindIndex(content)
		if location == nil {
			if name == "manifest.json" {
				return nil, errors.New(`manifest.json has no "version" field`)
			}
			continue
		}

		updated := append([]byte{}, content[:location[0]]...)
		updated = append(updated, versionFieldPattern.ReplaceAll(content[location[0]:location[1]], []byte("${1}"+version+"${3}"))...)
		updated = append(updated, content[location[1]:]...)
		if bytes.Equal(updated, content) {
			continue
		}
		if err := os.WriteFile(path, updated, 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, name)
	}
	return changed, nil
}

type Git struct {
	Dir string
}

func (g Git) run(args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = g.Dir
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(output)), nil
}

func (g Git) EnsureRepository() error {
	if _, err := exec.LookPath("git"); err != nil {
		return ux.NewProblem("git is not installed", "install git to release", err)
	}
	if _, err := g.run("rev-parse", "--is-inside-work-tree"); err != nil {
		return ux.NewProblem("This project is not a git repository", "run `git init`, commit, and push it to GitHub", nil)
	}
	return nil
}

func (g Git) DirtyFiles() ([]string, error) {
	output, err := g.run("status", "--porcelain")
	if err != nil || output == "" {
		return nil, err
	}
	return strings.Split(output, "\n"), nil
}

func (g Git) Branch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

func (g Git) Upstream() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
}

func (g Git) Fetch() error {
	_, err := g.run("fetch", "--quiet", "--tags", "origin")
	return err
}

func (g Git) BehindUpstream() (int, error) {
	output, err := g.run("rev-list", "--count", "HEAD..@{u}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(output)
}

func (g Git) TagExists(tag string) bool {
	_, err := g.run("rev-parse", "-q", "--verify", "refs/tags/"+tag)
	return err == nil
}

func (g Git) CommitAndTag(files []string, message string, tag string) error {
	if len(files) > 0 {
		if _, err := g.run(append([]string{"add", "--"}, files...)...); err != nil {
			return err
		}
		if _, err := g.run("commit", "-m", message); err != nil {
			return err
		}
	}
	_, err := g.run("tag", "-a", tag, "-m", message)
	return err
}

func (g Git) Restore(files []string) error {
	if len(files) == 0 {
		return nil
	}
	_, err := g.run(append([]string{"checkout", "--"}, files...)...)
	return err
}

func (g Git) Push(branch string, tag string) error {
	if _, err := g.run("push", "origin", branch); err != nil {
		return err
	}
	_, err := g.run("push", "origin", "refs/tags/"+tag)
	return err
}

func WaitForRelease(ctx context.Context, client *gh.Client, repository string, tag string, timeout time.Duration) (*gh.Release, error) {
	owner, repo, _ := strings.Cut(repository, "/")
	deadline := time.Now().Add(timeout)

	for {
		release, err := client.ReleaseByTag(ctx, owner, repo, tag)
		if err == nil && hasReleaseAssets(release) {
			return release, nil
		}
		if err != nil && !errors.Is(err, gh.ErrNotFound) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ux.NewProblem(
				"The release did not appear in time",
				fmt.Sprintf("check the Actions tab: https://github.com/%s/actions", repository),
				nil,
			)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}

func hasReleaseAssets(release *gh.Release) bool {
	hasManifest := false
	for _, asset := range release.Assets {
		if asset.GetName() == "manifest.json" {
			hasManifest = true
		}
	}
	return !release.GetDraft() && hasManifest && len(release.Assets) >= 2
}
