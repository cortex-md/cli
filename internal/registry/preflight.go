package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/cortex/cli/internal/build"
	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/report"
	"github.com/cortex/cli/internal/rules"
)

type releaseManifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Main         string            `json:"main"`
	Permissions  []string          `json:"permissions"`
	Capabilities []string          `json:"capabilities"`
	Colorschemes map[string]string `json:"colorschemes"`
}

type Preflight struct {
	Report     *report.Report
	ReleaseURL string
	Release    *gh.Release
}

func RunPreflight(ctx context.Context, client *gh.Client, s *Submission, login string) *Preflight {
	r := report.New()
	result := &Preflight{Report: r}
	checkMarketplaceReview(ctx, r, client, s, login)

	r.Section("Release on GitHub")

	owner, repo, _ := strings.Cut(s.Entry.Repo, "/")
	repository, err := client.Repository(ctx, owner, repo)
	if errors.Is(err, gh.ErrNotFound) {
		r.Fail("Repository "+s.Entry.Repo+" was not found", "push the project to GitHub and make it public")
		return result
	}
	if err != nil {
		r.Fail("Could not read "+s.Entry.Repo, err.Error())
		return result
	}
	if repository.GetPrivate() {
		r.Fail(s.Entry.Repo+" is private", "make the repository public so Cortex can download releases")
	} else {
		r.Pass("%s is public", s.Entry.Repo)
	}

	release, err := client.LatestRelease(ctx, owner, repo)
	if errors.Is(err, gh.ErrNotFound) {
		r.Fail("No published release yet", "run `cortex release` and wait for the GitHub Action to publish it")
		return result
	}
	if err != nil {
		r.Fail("Could not read the latest release", err.Error())
		return result
	}
	result.Release = release
	result.ReleaseURL = release.GetHTMLURL()

	manifestAsset := findAsset(release.Assets, "manifest.json")
	if manifestAsset == nil {
		r.Fail("Latest release "+release.GetTagName()+" has no manifest.json asset", "publish releases with the workflow from `cortex create --update-ci`")
		return result
	}

	source, err := client.DownloadText(ctx, manifestAsset.GetBrowserDownloadURL())
	if err != nil {
		r.Fail("Could not download manifest.json from the release", err.Error())
		return result
	}
	var published releaseManifest
	if err := json.Unmarshal([]byte(source), &published); err != nil {
		r.Fail("Release manifest.json is not valid JSON", err.Error())
		return result
	}

	checkReleaseManifest(r, s, release, &published)
	if s.Kind == build.KindTheme {
		checkThemeAssets(r, release, &published)
	} else {
		checkPluginAssets(r, release, &published)
	}

	if client.FileExists(ctx, owner, repo, "README.md", repository.GetDefaultBranch()) {
		r.Pass("README.md is on %s", repository.GetDefaultBranch())
	} else {
		r.Fail("README.md is missing on "+repository.GetDefaultBranch(), "commit a README.md; Marketplace shows it on the detail page")
	}

	return result
}

func checkMarketplaceReview(ctx context.Context, r *report.Report, client *gh.Client, s *Submission, login string) {
	r.Section("Marketplace review")

	owner, _, _ := strings.Cut(s.Entry.Repo, "/")
	switch {
	case login == "":
		r.Info("Ownership is checked when you submit while logged in", "")
	case strings.EqualFold(owner, login) || client.IsPublicOrgMember(ctx, owner, login):
		r.Pass("@%s can submit %s", login, s.Entry.Repo)
	default:
		r.Warn(
			fmt.Sprintf("@%s is not the owner of %s nor a public member of %s", login, s.Entry.Repo, owner),
			"maintainers will review ownership manually; make your organization membership public to skip it",
		)
	}

	blocked, err := FetchBlocked(ctx, client)
	if err != nil {
		r.Warn("Could not read the registry block list", err.Error())
		return
	}
	block := FindBlock(blocked, string(s.Kind), s.Entry.ID)
	switch {
	case block == nil:
		r.Pass("Not on the block list")
	case block.Versions == nil:
		r.Fail(fmt.Sprintf("%s %q is blocked: %s", s.Kind, s.Entry.ID, block.Reason), "open an issue in cortex-md/registry to appeal")
	case block.BlocksVersion(s.Version):
		r.Fail(fmt.Sprintf("Version %s was pulled: %s", s.Version, block.Reason), "release a fixed version with `cortex release`")
	default:
		r.Info(fmt.Sprintf("Versions %s were pulled earlier: %s", strings.Join(block.Versions, ", "), block.Reason), "")
	}
}

func checkReleaseManifest(r *report.Report, s *Submission, release *gh.Release, published *releaseManifest) {
	tag := release.GetTagName()
	switch {
	case published.ID != s.Entry.ID:
		r.Fail(fmt.Sprintf("Release manifest id %q differs from %q", published.ID, s.Entry.ID), "publish a release built from this project")
	case !rules.IsPlainSemver(published.Version):
		r.Fail(fmt.Sprintf("Release version %q is not MAJOR.MINOR.PATCH", published.Version), "release a plain version with `cortex release`")
	case strings.TrimPrefix(tag, "v") != published.Version:
		r.Fail(fmt.Sprintf("Release tag %s does not match manifest version %s", tag, published.Version), "Cortex compares the tag with the installed version; release with `cortex release`")
	default:
		r.Pass("Latest release %s matches the manifest", tag)
	}

	if published.Version != s.Version {
		r.Info(fmt.Sprintf("Local version %s is not released yet (latest is %s)", s.Version, published.Version), "")
	}

	if s.Kind == build.KindTheme && published.Name != published.ID {
		r.Fail(fmt.Sprintf("Theme name %q must equal id %q", published.Name, published.ID), "set name to the id in manifest.json and release again")
	}

	permissions := published.Permissions
	if len(permissions) == 0 {
		permissions = published.Capabilities
	}
	unknown := []string{}
	for _, permission := range permissions {
		if !rules.IsPluginPermission(permission) {
			unknown = append(unknown, permission)
		}
	}
	if len(unknown) > 0 {
		r.Fail(
			"Release declares unknown permissions: "+strings.Join(unknown, ", "),
			"use only "+strings.Join(rules.PluginPermissions, ", ")+"; contributions such as commands and views need no permission",
		)
	}
}

func checkPluginAssets(r *report.Report, release *gh.Release, published *releaseManifest) {
	mainPath, ok := rules.SafeRelativePath(published.Main)
	if !ok {
		r.Fail(fmt.Sprintf("Release manifest main %q is not a safe relative path", published.Main), "use a path such as dist/index.js")
		return
	}
	if findAsset(release.Assets, mainPath) == nil && findAsset(release.Assets, path.Base(mainPath)) == nil {
		r.Fail("Release is missing the bundle asset "+path.Base(mainPath), "publish releases with `cortex pack` (used by the release workflow)")
		return
	}
	r.Pass("Release ships manifest.json and %s", path.Base(mainPath))
}

func checkThemeAssets(r *report.Report, release *gh.Release, published *releaseManifest) {
	missing := []string{}
	for _, scheme := range []string{"dark", "light"} {
		cssPath, ok := rules.SafeRelativePath(published.Colorschemes[scheme])
		if !ok {
			missing = append(missing, scheme)
			continue
		}
		if findAsset(release.Assets, cssPath) == nil && findAsset(release.Assets, path.Base(cssPath)) == nil && findAsset(release.Assets, scheme+".css") == nil {
			missing = append(missing, scheme)
		}
	}
	if len(missing) > 0 {
		r.Fail("Release is missing the "+strings.Join(missing, " and ")+" stylesheet", "publish releases with `cortex pack` (used by the release workflow)")
		return
	}
	r.Pass("Release ships manifest.json and both stylesheets")
}

func findAsset(assets []*gh.ReleaseAsset, name string) *gh.ReleaseAsset {
	for _, asset := range assets {
		if asset.GetName() == name {
			return asset
		}
	}
	return nil
}
