package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidVersion = errors.New("invalid semver format")
	ErrEmptyVersion   = errors.New("version cannot be empty")
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
}

func Parse(v string) (*Version, error) {
	if v == "" {
		return nil, ErrEmptyVersion
	}

	v = strings.TrimPrefix(v, "v")

	buildSplit := strings.SplitN(v, "+", 2)
	versionAndPre := buildSplit[0]
	build := ""
	if len(buildSplit) > 1 {
		build = buildSplit[1]
	}

	preSplit := strings.SplitN(versionAndPre, "-", 2)
	versionPart := preSplit[0]
	prerelease := ""
	if len(preSplit) > 1 {
		prerelease = preSplit[1]
	}

	parts := strings.Split(versionPart, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected MAJOR.MINOR.PATCH format", ErrInvalidVersion)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return nil, fmt.Errorf("%w: invalid major version", ErrInvalidVersion)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return nil, fmt.Errorf("%w: invalid minor version", ErrInvalidVersion)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil || patch < 0 {
		return nil, fmt.Errorf("%w: invalid patch version", ErrInvalidVersion)
	}

	return &Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Build:      build,
	}, nil
}

func (v *Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}

	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease != other.Prerelease {
		return strings.Compare(v.Prerelease, other.Prerelease)
	}

	return 0
}

func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

func (v *Version) GreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

func (v *Version) Equal(other *Version) bool {
	return v.Compare(other) == 0
}

func IsValid(v string) bool {
	_, err := Parse(v)
	return err == nil
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
