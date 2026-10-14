package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/report"
	"github.com/cortex/cli/internal/rules"
	"github.com/cortex/cli/pkg/manifest"
)

type cssVariableRequirement struct {
	Name    string
	Aliases []string
}

var requiredCSSVariables = []cssVariableRequirement{
	{Name: "--bg-primary"},
	{Name: "--bg-secondary"},
	{Name: "--text-primary"},
	{Name: "--text-secondary"},
	{Name: "--accent", Aliases: []string{"--accent-default"}},
	{Name: "--border", Aliases: []string{"--border-default"}},
}

var recommendedCSSVariables = []string{
	"--bg-tertiary",
	"--bg-elevated",
	"--bg-hover",
	"--bg-active",
	"--text-muted",
	"--text-disabled",
	"--text-on-accent",
	"--accent-hover",
	"--accent-active",
	"--accent-subtle",
	"--accent-border",
	"--accent-text",
	"--brand",
	"--brand-hover",
	"--brand-active",
	"--border-subtle",
	"--border-strong",
	"--border-focus",
	"--link",
	"--link-hover",
	"--link-broken",
	"--syntax-keyword",
	"--syntax-string",
	"--syntax-comment",
	"--syntax-number",
	"--syntax-function",
	"--syntax-heading",
}

var requiredColorschemes = []string{"dark", "light"}

const (
	cssFileSizeWarning = 50 * 1024
	cssFileSizeError   = 200 * 1024
)

var externalURLPattern = regexp.MustCompile(`url\s*\(\s*['"]?https?://`)

func Check(dir string, m *manifest.ThemeManifest) *report.Report {
	r := report.New()

	r.Section("Manifest")
	checkManifest(r, m)

	r.Section("Stylesheets")
	checkStylesheets(r, dir, m)

	r.Section("Publishing")
	build.CheckReadiness(r, dir, build.KindTheme, m.Repository)

	return r
}

func checkManifest(r *report.Report, m *manifest.ThemeManifest) {
	failures := r.Count(report.LevelFail)

	switch {
	case m.ID == "":
		r.Fail("id is missing", `add "id" to manifest.json (lowercase letters, numbers and hyphens)`)
	case !rules.IsValidID(m.ID):
		r.Fail(fmt.Sprintf("id %q is not a valid slug", m.ID), "use lowercase letters, numbers and hyphens")
	}

	switch {
	case m.Name == "":
		r.Fail("name is missing", fmt.Sprintf(`set "name": %q in manifest.json`, m.ID))
	case !rules.IsValidThemeName(m.Name):
		r.Fail(fmt.Sprintf("name %q must be a lowercase slug", m.Name), "Cortex uses name in the body.theme-<name>-<scheme> class")
	case m.ID != "" && m.Name != m.ID:
		r.Fail(fmt.Sprintf("name %q must equal id %q", m.Name, m.ID), "Marketplace matches installed themes by name; set name to the id and update the CSS selectors")
	}

	for field, value := range map[string]string{"displayName": m.DisplayName, "author": m.Author} {
		if strings.TrimSpace(value) == "" {
			r.Fail(field+" is missing", fmt.Sprintf("add %q to manifest.json", field))
		}
	}

	switch {
	case m.Version == "":
		r.Fail("version is missing", `add "version": "1.0.0" to manifest.json`)
	case !rules.IsPlainSemver(m.Version):
		r.Fail(fmt.Sprintf("version %q must be MAJOR.MINOR.PATCH", m.Version), "pre-release and build suffixes break Marketplace update checks")
	}

	if m.MinAppVersion != "" && !rules.IsPlainSemver(m.MinAppVersion) {
		r.Fail(fmt.Sprintf("minAppVersion %q must be MAJOR.MINOR.PATCH", m.MinAppVersion), "use the oldest Cortex version the theme supports, e.g. 0.1.1")
	}

	if m.AuthorURL != "" && !rules.IsHTTPSURL(m.AuthorURL) {
		r.Fail("authorUrl must be an https URL", `use a full https:// URL or remove "authorUrl"`)
	}

	if len(m.Description) < 10 {
		r.Warn("description is missing or very short", "describe the theme in one sentence; Marketplace shows it on the card")
	}

	for _, scheme := range requiredColorschemes {
		path, ok := m.Colorschemes[scheme]
		if !ok {
			r.Fail(fmt.Sprintf("colorschemes.%s is missing", scheme), "Cortex themes ship both a dark and a light stylesheet")
			continue
		}
		if _, safe := rules.SafeRelativePath(path); !safe {
			r.Fail(fmt.Sprintf("colorschemes.%s %q is not a safe relative path", scheme, path), "use a path inside the project, e.g. theme-dark.css")
		}
	}

	if r.Count(report.LevelFail) == failures {
		r.Pass("%s v%s", m.DisplayName, m.Version)
	}
}

func checkStylesheets(r *report.Report, dir string, m *manifest.ThemeManifest) {
	for _, scheme := range sortedSchemes(m.Colorschemes) {
		relativePath, ok := rules.SafeRelativePath(m.Colorschemes[scheme])
		if !ok {
			continue
		}

		fullPath := filepath.Join(dir, filepath.FromSlash(relativePath))
		content, err := os.ReadFile(fullPath)
		if err != nil {
			r.Fail(fmt.Sprintf("%s stylesheet %s was not found", scheme, relativePath), "create the file or fix colorschemes in manifest.json")
			continue
		}

		failures := r.Count(report.LevelFail)
		warnings := r.Count(report.LevelWarn)
		checkCSSContent(r, string(content), m.Name, scheme, relativePath)
		checkCSSSize(r, len(content), relativePath)
		if r.Count(report.LevelFail) == failures && r.Count(report.LevelWarn) == warnings {
			r.Pass("%s (%s)", relativePath, scheme)
		}
	}
}

func checkCSSContent(r *report.Report, content string, themeName string, scheme string, path string) {
	missing := []string{}
	for _, requirement := range requiredCSSVariables {
		if !hasCSSVariableRequirement(content, requirement) {
			missing = append(missing, requirement.Name)
		}
	}
	if len(missing) > 0 {
		r.Fail(fmt.Sprintf("%s is missing required variables: %s", path, strings.Join(missing, ", ")), "define them inside body.theme-"+themeName+"-"+scheme)
	}

	missingRecommended := 0
	for _, name := range recommendedCSSVariables {
		if !hasCSSVariable(content, name) {
			missingRecommended++
		}
	}
	if missingRecommended > 0 {
		r.Info(fmt.Sprintf("%s leaves %d recommended variables to the Cortex defaults", path, missingRecommended), "")
	}

	if !hasThemeScope(content, themeName, scheme) {
		r.Warn(
			fmt.Sprintf("%s has no body.theme-%s-%s selector", path, themeName, scheme),
			fmt.Sprintf("scope the variables with body.theme-%s-%s so they only apply when the theme is active", themeName, scheme),
		)
	}

	if strings.Contains(content, "@import") {
		r.Warn(path+" uses @import", "inline the imported CSS; imports slow down theme loading")
	}
	if externalURLPattern.MatchString(content) {
		r.Warn(path+" loads remote URLs", "remote assets do not work offline; inline them")
	}
}

func checkCSSSize(r *report.Report, size int, path string) {
	switch {
	case size > cssFileSizeError:
		r.Fail(fmt.Sprintf("%s is %.0f KB (limit 200 KB)", path, float64(size)/1024), "remove unused rules")
	case size > cssFileSizeWarning:
		r.Warn(fmt.Sprintf("%s is %.0f KB", path, float64(size)/1024), "keep stylesheets under 50 KB")
	}
}

func hasCSSVariableRequirement(content string, requirement cssVariableRequirement) bool {
	if hasCSSVariable(content, requirement.Name) {
		return true
	}
	for _, alias := range requirement.Aliases {
		if hasCSSVariable(content, alias) {
			return true
		}
	}
	return false
}

func hasCSSVariable(content string, name string) bool {
	return strings.Contains(content, name+":")
}

func hasThemeScope(content string, themeName string, scheme string) bool {
	return strings.Contains(content, ":root") || strings.Contains(content, fmt.Sprintf(".theme-%s-%s", themeName, scheme))
}

func sortedSchemes(colorschemes map[string]string) []string {
	names := make([]string, 0, len(colorschemes))
	for name := range colorschemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
