package project

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

type Project struct {
	Dir    string
	Kind   manifest.Kind
	Plugin *manifest.PluginManifest
	Theme  *manifest.ThemeManifest
}

func Load(dir string) (*Project, error) {
	project, err := read(dir)
	if err != nil {
		return nil, err
	}

	if project.Kind == manifest.KindTheme {
		project.Theme, err = manifest.LoadTheme(project.Dir)
	} else {
		project.Plugin, err = manifest.LoadPlugin(project.Dir)
	}
	if err != nil {
		return nil, ux.NewProblem("Invalid manifest.json", "run `cortex check` to see every issue", err)
	}

	return project, nil
}

func Read(dir string) (*Project, error) {
	project, err := read(dir)
	if err != nil {
		return nil, err
	}

	if project.Kind == manifest.KindTheme {
		project.Theme, err = manifest.ReadTheme(project.Dir)
	} else {
		project.Plugin, err = manifest.ReadPlugin(project.Dir)
	}
	if err != nil {
		return nil, ux.NewProblem("Could not parse manifest.json", "fix the JSON syntax in manifest.json", err)
	}

	return project, nil
}

func read(dir string) (*Project, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	kind, err := manifest.DetectKind(absDir)
	if errors.Is(err, manifest.ErrManifestNotFound) {
		return nil, ux.NewProblem(
			"No Cortex plugin or theme in "+absDir,
			"run this inside a project folder, pass its path, or start one with `cortex create`",
			nil,
		)
	}
	if err != nil {
		return nil, ux.NewProblem("Could not parse manifest.json", "fix the JSON syntax in manifest.json", err)
	}

	return &Project{Dir: absDir, Kind: kind}, nil
}

func (p *Project) ID() string {
	if p.Theme != nil {
		return p.Theme.ID
	}
	return p.Plugin.ID
}

func (p *Project) Version() string {
	if p.Theme != nil {
		return p.Theme.Version
	}
	return p.Plugin.Version
}

func (p *Project) DisplayName() string {
	if p.Theme != nil {
		return p.Theme.DisplayName
	}
	return p.Plugin.Name
}

func (p *Project) InstallDir(vaultPath string) string {
	folder := "plugins"
	if p.Kind == manifest.KindTheme {
		folder = "themes"
	}
	return filepath.Join(vaultPath, ".cortex", folder, p.ID())
}
