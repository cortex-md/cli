package rules

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var PluginPermissions = []string{
	"vault:read",
	"vault:write",
	"vault:delete",
	"vault:watch",
	"editor:read",
	"editor:write",
	"workspace:tabs",
	"theme:read",
	"bookmarks:read",
	"bookmarks:write",
	"data",
	"notifications",
}

var RuntimeExternals = []string{
	"@cortex.md/api",
	"@codemirror/state",
	"@codemirror/view",
}

var (
	idPattern          = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
	themeNamePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	plainSemverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	nonIDCharacters    = regexp.MustCompile(`[^a-z0-9-]+`)
	repeatedHyphens    = regexp.MustCompile(`-{2,}`)
)

func IsValidID(id string) bool {
	return idPattern.MatchString(id)
}

func IsValidThemeName(name string) bool {
	return themeNamePattern.MatchString(name)
}

func IsPlainSemver(version string) bool {
	return plainSemverPattern.MatchString(version)
}

func IsPluginPermission(permission string) bool {
	for _, candidate := range PluginPermissions {
		if candidate == permission {
			return true
		}
	}
	return false
}

func IsRuntimeExternal(module string) bool {
	for _, candidate := range RuntimeExternals {
		if candidate == module {
			return true
		}
	}
	return false
}

func IsHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func SafeRelativePath(path string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", false
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "", false
	}

	segments := []string{}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." {
			continue
		}
		if segment == ".." {
			return "", false
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 {
		return "", false
	}

	return strings.Join(segments, "/"), true
}

func NormalizeID(name string) string {
	id := nonIDCharacters.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	id = repeatedHyphens.ReplaceAllString(id, "-")
	return strings.Trim(id, "-")
}

const MaxScreenshots = 6

var screenshotExtensions = []string{".png", ".jpg", ".jpeg", ".webp", ".gif"}

func ScreenshotProblems(screenshots []string) []string {
	problems := []string{}
	if len(screenshots) > MaxScreenshots {
		problems = append(problems, fmt.Sprintf("screenshots has %d images; the Marketplace shows at most %d", len(screenshots), MaxScreenshots))
	}
	seen := map[string]bool{}
	for _, screenshot := range screenshots {
		value := strings.TrimSpace(screenshot)
		switch {
		case !IsHTTPSURL(value):
			problems = append(problems, fmt.Sprintf("screenshot %q is not an https URL", screenshot))
		case !hasScreenshotExtension(value):
			problems = append(problems, fmt.Sprintf("screenshot %q must end in %s", screenshot, strings.Join(screenshotExtensions, ", ")))
		case seen[value]:
			problems = append(problems, fmt.Sprintf("screenshot %q is listed twice", screenshot))
		}
		seen[value] = true
	}
	return problems
}

func hasScreenshotExtension(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	for _, extension := range screenshotExtensions {
		if strings.HasSuffix(path, extension) {
			return true
		}
	}
	return false
}
