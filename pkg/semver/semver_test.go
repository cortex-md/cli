package semver

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		wantErr   bool
		wantMajor int
		wantMinor int
		wantPatch int
		wantPre   string
		wantBuild string
	}{
		{"1.2.3", false, 1, 2, 3, "", ""},
		{"v1.2.3", false, 1, 2, 3, "", ""},
		{"0.1.0", false, 0, 1, 0, "", ""},
		{"1.2.3-alpha", false, 1, 2, 3, "alpha", ""},
		{"1.2.3-beta.1", false, 1, 2, 3, "beta.1", ""},
		{"1.2.3+build.123", false, 1, 2, 3, "", "build.123"},
		{"1.2.3-rc.1+build.456", false, 1, 2, 3, "rc.1", "build.456"},
		{"", true, 0, 0, 0, "", ""},
		{"1.2", true, 0, 0, 0, "", ""},
		{"1.2.x", true, 0, 0, 0, "", ""},
		{"invalid", true, 0, 0, 0, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if v.Major != tt.wantMajor || v.Minor != tt.wantMinor || v.Patch != tt.wantPatch {
				t.Errorf("Parse(%q) = %d.%d.%d, want %d.%d.%d",
					tt.input, v.Major, v.Minor, v.Patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
			}
			if v.Prerelease != tt.wantPre {
				t.Errorf("Parse(%q) prerelease = %q, want %q", tt.input, v.Prerelease, tt.wantPre)
			}
			if v.Build != tt.wantBuild {
				t.Errorf("Parse(%q) build = %q, want %q", tt.input, v.Build, tt.wantBuild)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3, "", ""}, "1.2.3"},
		{Version{0, 1, 0, "", ""}, "0.1.0"},
		{Version{1, 2, 3, "alpha", ""}, "1.2.3-alpha"},
		{Version{1, 2, 3, "", "build.123"}, "1.2.3+build.123"},
		{Version{1, 2, 3, "rc.1", "build.456"}, "1.2.3-rc.1+build.456"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("Version.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		v1   string
		v2   string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.2.4", "1.2.3", 1},
		{"1.2.3", "1.3.0", -1},
		{"1.3.0", "1.2.3", 1},
		{"1.2.3", "2.0.0", -1},
		{"2.0.0", "1.2.3", 1},
		{"1.2.3", "1.2.3-alpha", 1},
		{"1.2.3-alpha", "1.2.3", -1},
		{"1.2.3-alpha", "1.2.3-beta", -1},
		{"1.2.3-beta", "1.2.3-alpha", 1},
	}

	for _, tt := range tests {
		t.Run(tt.v1+"_vs_"+tt.v2, func(t *testing.T) {
			v1, err1 := Parse(tt.v1)
			v2, err2 := Parse(tt.v2)
			if err1 != nil || err2 != nil {
				t.Fatalf("Parse error: %v, %v", err1, err2)
			}
			if got := v1.Compare(v2); got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true},
		{"0.1.0", true},
		{"1.2.3-alpha", true},
		{"1.2.3+build", true},
		{"", false},
		{"1.2", false},
		{"1.2.x", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsValid(tt.input); got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
