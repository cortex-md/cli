package registry

import (
	"context"
	"fmt"

	"github.com/cortex/cli/internal/auth"
	gh "github.com/cortex/cli/internal/github"
	"github.com/cortex/cli/internal/publish"
)

func Submit(ctx context.Context, publishJSONPath string) (string, error) {
	opts, err := LoadSubmitOptions(publishJSONPath)
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

func LoadSubmitOptions(publishJSONPath string) (PublishIndexOptions, error) {
	publishManifest, err := publish.LoadManifest(publishJSONPath)
	if err != nil {
		return PublishIndexOptions{}, err
	}

	return PublishIndexOptions{
		RegistryOwner: firstNonEmpty(publishManifest.Registry.Owner, publish.DefaultRegistryOwner),
		RegistryRepo:  firstNonEmpty(publishManifest.Registry.Repo, publish.DefaultRegistryRepo),
		BaseBranch:    firstNonEmpty(publishManifest.Registry.BaseBranch, publish.DefaultRegistryBase),
		IndexFile:     publishManifest.Registry.IndexFile,
		Kind:          string(publishManifest.Kind),
		Entry: IndexEntry{
			ID:            publishManifest.Registry.Entry.ID,
			Name:          publishManifest.Registry.Entry.Name,
			Author:        publishManifest.Registry.Entry.Author,
			AuthorURL:     publishManifest.Registry.Entry.AuthorURL,
			Description:   publishManifest.Registry.Entry.Description,
			CoverImageURL: publishManifest.Registry.Entry.CoverImageURL,
			Repo:          publishManifest.Registry.Entry.Repo,
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
