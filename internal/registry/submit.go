package registry

import (
	"context"
	"fmt"

	"github.com/cortex/cli/internal/auth"
	"github.com/cortex/cli/internal/build"
	gh "github.com/cortex/cli/internal/github"
)

func Submit(ctx context.Context, buildJSONPath string) (string, error) {
	opts, err := LoadSubmitOptions(buildJSONPath)
	if err != nil {
		return "", err
	}

	token, err := auth.ResolvePublishToken()
	if err != nil {
		return "", fmt.Errorf("github token unavailable: set CORTEX_TOKEN, set GITHUB_TOKEN, or run `cortex login`")
	}

	client := gh.NewClient(token)
	return PublishToRegistry(ctx, client, opts)
}

func LoadSubmitOptions(buildJSONPath string) (PublishIndexOptions, error) {
	buildManifest, err := build.LoadManifest(buildJSONPath)
	if err != nil {
		return PublishIndexOptions{}, err
	}

	return PublishIndexOptions{
		RegistryOwner: firstNonEmpty(buildManifest.Registry.Owner, build.DefaultRegistryOwner),
		RegistryRepo:  firstNonEmpty(buildManifest.Registry.Repo, build.DefaultRegistryRepo),
		BaseBranch:    firstNonEmpty(buildManifest.Registry.BaseBranch, build.DefaultRegistryBase),
		IndexFile:     buildManifest.Registry.IndexFile,
		Kind:          string(buildManifest.Kind),
		Entry: IndexEntry{
			ID:            buildManifest.Registry.Entry.ID,
			Name:          buildManifest.Registry.Entry.Name,
			Author:        buildManifest.Registry.Entry.Author,
			AuthorURL:     buildManifest.Registry.Entry.AuthorURL,
			Description:   buildManifest.Registry.Entry.Description,
			CoverImageURL: buildManifest.Registry.Entry.CoverImageURL,
			Repo:          buildManifest.Registry.Entry.Repo,
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
