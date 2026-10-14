package registry

import (
	"github.com/cortex/cli/internal/build"
	"github.com/cortex/cli/internal/plugin"
	"github.com/cortex/cli/internal/project"
	"github.com/cortex/cli/internal/theme"
	"github.com/cortex/cli/internal/ux"
	"github.com/cortex/cli/pkg/manifest"
)

const (
	Owner      = "cortex-md"
	Repo       = "registry"
	BaseBranch = "main"
)

type Submission struct {
	Kind      build.Kind
	IndexFile string
	Version   string
	Entry     IndexEntry
	Project   *project.Project
}

func NewSubmission(p *project.Project) (*Submission, error) {
	var input build.MetadataInput
	if p.Kind == manifest.KindTheme {
		input = theme.Metadata(p.Theme)
	} else {
		input = plugin.Metadata(p.Plugin)
	}

	metadata, err := build.ResolveMetadata(p.Dir, input)
	if err != nil {
		return nil, ux.NewProblem("Registry metadata is incomplete", "run `cortex check` and fix the reported fields", err)
	}

	indexFile := "plugins.json"
	if metadata.Kind == build.KindTheme {
		indexFile = "themes.json"
	}

	return &Submission{
		Kind:      metadata.Kind,
		IndexFile: indexFile,
		Version:   metadata.Version,
		Project:   p,
		Entry: IndexEntry{
			ID:            metadata.ID,
			Name:          metadata.Name,
			Author:        metadata.Author,
			AuthorURL:     metadata.AuthorURL,
			Description:   metadata.Description,
			CoverImageURL: metadata.CoverImageURL,
			Repo:          metadata.Repository,
		},
	}, nil
}

func (s *Submission) BranchName() string {
	return "registry/" + string(s.Kind) + "-" + s.Entry.ID
}
