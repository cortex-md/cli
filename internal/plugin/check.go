package plugin

import (
	"errors"
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

const (
	bundleSizeWarning = 1 * 1024 * 1024
	bundleSizeError   = 5 * 1024 * 1024
)

var (
	requirePattern = regexp.MustCompile(`require\(\s*["']([^"']+)["']\s*\)`)
	esmPattern     = regexp.MustCompile(`(?m)^\s*(import\s.+from\s|export\s)`)
)

var bundlePatterns = []struct {
	pattern *regexp.Regexp
	fail    bool
	message string
	fix     string
}{
	{regexp.MustCompile(`\beval\s*\(`), true, "Bundle calls eval()", "remove eval(); Marketplace rejects dynamic code execution"},
	{regexp.MustCompile(`new\s+Function\s*\(`), true, "Bundle uses the Function constructor", "remove new Function(); Marketplace rejects dynamic code execution"},
	{regexp.MustCompile(`dangerouslySetInnerHTML`), false, "Bundle uses dangerouslySetInnerHTML", "render content through view descriptors instead of raw HTML"},
	{regexp.MustCompile(`<script[^>]*>`), false, "Bundle contains an inline <script> tag", "avoid injecting scripts into the Cortex window"},
}

func Check(dir string, m *manifest.PluginManifest) *report.Report {
	r := report.New()

	r.Section("Manifest")
	checkManifest(r, m)

	r.Section("Bundle")
	checkBundle(r, dir, m)

	r.Section("Publishing")
	build.CheckReadiness(r, dir, build.KindPlugin, m.Repository)

	return r
}

func checkManifest(r *report.Report, m *manifest.PluginManifest) {
	failures := r.Count(report.LevelFail)

	switch {
	case m.ID == "":
		r.Fail("id is missing", `add "id" to manifest.json (lowercase letters, numbers and hyphens)`)
	case !rules.IsValidID(m.ID):
		r.Fail(fmt.Sprintf("id %q is not a valid slug", m.ID), "use lowercase letters, numbers and hyphens, starting and ending with a letter or number")
	}

	requireField(r, "name", m.Name)
	requireField(r, "author", m.Author)
	requireField(r, "icon", m.Icon)

	checkVersionField(r, "version", m.Version)
	checkVersionField(r, "minAppVersion", m.MinAppVersion)

	if m.Description == "" {
		r.Fail("description is missing", `add a one-sentence "description" to manifest.json`)
	} else if len(m.Description) < 10 {
		r.Warn("description is very short", "describe what the plugin does in one sentence; Marketplace shows it on the card")
	}

	if m.AuthorURL != "" && !rules.IsHTTPSURL(m.AuthorURL) {
		r.Fail("authorUrl must be an https URL", `use a full https:// URL or remove "authorUrl"`)
	}

	for _, problem := range rules.ScreenshotProblems(m.Screenshots) {
		r.Warn(problem, fmt.Sprintf(`list up to %d distinct https image URLs (.png, .jpg, .webp or .gif) in "screenshots"; cortex submit rejects invalid ones`, rules.MaxScreenshots))
	}

	if m.Main == "" {
		r.Fail("main is missing", `set "main": "dist/index.js" in manifest.json`)
	} else if _, ok := rules.SafeRelativePath(m.Main); !ok {
		r.Fail(fmt.Sprintf("main %q is not a safe relative path", m.Main), "use a path inside the project, e.g. dist/index.js")
	}

	invalid := []string{}
	for _, permission := range m.Permissions {
		if !rules.IsPluginPermission(permission) {
			invalid = append(invalid, permission)
		}
	}
	if len(invalid) > 0 {
		r.Fail(
			fmt.Sprintf("Unknown permissions: %s", strings.Join(invalid, ", ")),
			"valid permissions: "+strings.Join(rules.PluginPermissions, ", "),
		)
	}

	if r.Count(report.LevelFail) == failures {
		r.Pass("%s v%s (%s)", m.Name, m.Version, describePermissions(m.Permissions))
	}
}

func requireField(r *report.Report, field string, value string) {
	if strings.TrimSpace(value) == "" {
		r.Fail(field+" is missing", fmt.Sprintf("add %q to manifest.json", field))
	}
}

func checkVersionField(r *report.Report, field string, value string) {
	switch {
	case value == "":
		r.Fail(field+" is missing", fmt.Sprintf(`add "%s": "1.0.0" to manifest.json`, field))
	case !rules.IsPlainSemver(value):
		r.Fail(fmt.Sprintf("%s %q must be MAJOR.MINOR.PATCH", field, value), "pre-release and build suffixes break Marketplace update checks")
	}
}

func describePermissions(permissions []string) string {
	if len(permissions) == 0 {
		return "no permissions"
	}
	return "permissions: " + strings.Join(permissions, ", ")
}

func checkBundle(r *report.Report, dir string, m *manifest.PluginManifest) {
	mainPath, ok := rules.SafeRelativePath(m.Main)
	if !ok {
		return
	}

	bundlePath := filepath.Join(dir, filepath.FromSlash(mainPath))
	info, err := os.Stat(bundlePath)
	if err != nil {
		r.Fail(fmt.Sprintf("Bundle %s was not found", mainPath), "make sure the build script writes to the path in manifest.main")
		return
	}

	switch size := info.Size(); {
	case size > bundleSizeError:
		r.Fail(fmt.Sprintf("Bundle is %.1f MB (limit 5 MB)", float64(size)/1024/1024), "remove heavy dependencies or load them lazily")
	case size > bundleSizeWarning:
		r.Warn(fmt.Sprintf("Bundle is %.1f MB", float64(size)/1024/1024), "keep bundles under 1 MB so Cortex starts quickly")
	}

	content, err := os.ReadFile(bundlePath)
	if err != nil {
		r.Fail("Bundle could not be read", err.Error())
		return
	}
	scanBundle(r, string(content))
	inspectBundle(r, dir, bundlePath, string(content), m)
}

func scanBundle(r *report.Report, content string) {
	for _, check := range bundlePatterns {
		if !check.pattern.MatchString(content) {
			continue
		}
		if check.fail {
			r.Fail(check.message, check.fix)
		} else {
			r.Warn(check.message, check.fix)
		}
	}

	unsupported := map[string]bool{}
	for _, match := range requirePattern.FindAllStringSubmatch(content, -1) {
		if !rules.IsRuntimeExternal(match[1]) {
			unsupported[match[1]] = true
		}
	}
	if len(unsupported) > 0 {
		modules := make([]string, 0, len(unsupported))
		for module := range unsupported {
			modules = append(modules, module)
		}
		sort.Strings(modules)
		r.Fail(
			fmt.Sprintf("Bundle requires modules Cortex does not provide: %s", strings.Join(modules, ", ")),
			"bundle them into the plugin; only "+strings.Join(rules.RuntimeExternals, ", ")+" may stay external",
		)
	}
}

func inspectBundle(r *report.Report, dir string, bundlePath string, content string, m *manifest.PluginManifest) {
	inspection, err := InspectBundle(bundlePath)
	if errors.Is(err, ErrNoJavaScriptRuntime) {
		r.Warn("Bundle was not loaded: node or bun is not installed", "install node or bun so `cortex check` can load the bundle like Cortex does")
		return
	}
	if err != nil {
		r.Fail("Bundle could not be inspected", err.Error())
		return
	}

	if inspection.Error != "" {
		fix := "export the plugin with `export default definePlugin({ ... })`"
		if esmPattern.MatchString(content) {
			fix = "build with `--format cjs --target browser`; Cortex loads plugins as CommonJS"
		} else if strings.HasPrefix(inspection.Error, "Cannot require") {
			fix = "bundle the dependency; only " + strings.Join(rules.RuntimeExternals, ", ") + " may stay external"
		}
		r.Fail("Cortex cannot load the bundle: "+inspection.Error, fix)
		return
	}

	bundleManifest := inspection.Manifest
	failures := r.Count(report.LevelFail)

	if bundleManifest.ID != m.ID {
		r.Fail(
			fmt.Sprintf("Bundle declares id %q but manifest.json has %q", bundleManifest.ID, m.ID),
			`import manifest.json in src/index.ts: definePlugin({ manifest: manifest as PluginManifest, ... })`,
		)
	}
	if bundleManifest.Version != m.Version {
		r.Fail(
			fmt.Sprintf("Bundle version %q differs from manifest.json %q", bundleManifest.Version, m.Version),
			"rebuild after bumping the version, or import manifest.json in src/index.ts",
		)
	}
	if bundleManifest.Name != m.Name {
		r.Warn(fmt.Sprintf("Bundle name %q differs from manifest.json %q", bundleManifest.Name, m.Name), "import manifest.json in src/index.ts")
	}
	if !samePermissions(bundleManifest.Permissions, m.Permissions) {
		r.Fail(
			fmt.Sprintf("Bundle permissions [%s] differ from manifest.json [%s]", strings.Join(bundleManifest.Permissions, ", "), strings.Join(m.Permissions, ", ")),
			"Marketplace shows manifest.json permissions but Cortex enforces the bundle's; import manifest.json in src/index.ts",
		)
	}

	if fileExists(filepath.Join(dir, "styles.css")) && !inspection.HasMarkdownContributions {
		r.Fail("styles.css is present but the plugin has no markdown contributions", "Cortex only loads styles.css for plugins with contributes.markdown; remove it or add markdown contributions")
	}

	if r.Count(report.LevelFail) == failures {
		r.Pass("Bundle loads the way Cortex loads it, and matches manifest.json")
	}
}

func samePermissions(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := map[string]int{}
	for _, permission := range left {
		seen[permission]++
	}
	for _, permission := range right {
		if seen[permission] == 0 {
			return false
		}
		seen[permission]--
	}
	return true
}
