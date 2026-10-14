package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gh "github.com/cortex/cli/internal/github"
)

type BlockedEntry struct {
	Kind     string   `json:"kind"`
	ID       string   `json:"id"`
	Repo     string   `json:"repo"`
	Reason   string   `json:"reason"`
	Versions []string `json:"versions"`
	Since    string   `json:"since"`
}

type ItemAudit struct {
	Status     string   `json:"status"`
	CheckedTag string   `json:"checkedTag"`
	Issues     []string `json:"issues"`
}

type ItemDiscussion struct {
	URL string `json:"url"`
}

type ItemStats struct {
	Downloads  int             `json:"downloads"`
	Stars      int             `json:"stars"`
	Votes      int             `json:"votes"`
	Latest     string          `json:"latest"`
	ReleasedAt string          `json:"releasedAt"`
	Discussion *ItemDiscussion `json:"discussion"`
	Audit      *ItemAudit      `json:"audit"`
}

type Stats struct {
	GeneratedAt *time.Time           `json:"generatedAt"`
	Plugins     map[string]ItemStats `json:"plugins"`
	Themes      map[string]ItemStats `json:"themes"`
}

func rawURL(branch string, fileName string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", Owner, Repo, branch, fileName)
}

func FetchBlocked(ctx context.Context, client *gh.Client) ([]BlockedEntry, error) {
	source, err := client.DownloadText(ctx, rawURL(BaseBranch, "blocked.json"))
	if errors.Is(err, gh.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var blocked []BlockedEntry
	if err := json.Unmarshal([]byte(source), &blocked); err != nil {
		return nil, fmt.Errorf("blocked.json is invalid: %w", err)
	}
	return blocked, nil
}

func FetchStats(ctx context.Context, client *gh.Client) (*Stats, error) {
	source, err := client.DownloadText(ctx, rawURL("data", "stats.json"))
	if errors.Is(err, gh.ErrNotFound) {
		return &Stats{}, nil
	}
	if err != nil {
		return nil, err
	}
	var stats Stats
	if err := json.Unmarshal([]byte(source), &stats); err != nil {
		return nil, fmt.Errorf("stats.json is invalid: %w", err)
	}
	return &stats, nil
}

func (s *Stats) Item(kind string, id string) (ItemStats, bool) {
	items := s.Plugins
	if kind == "theme" {
		items = s.Themes
	}
	item, ok := items[id]
	return item, ok
}

func FindBlock(blocked []BlockedEntry, kind string, id string) *BlockedEntry {
	for index := range blocked {
		if blocked[index].Kind == kind && blocked[index].ID == id {
			return &blocked[index]
		}
	}
	return nil
}

func (b *BlockedEntry) BlocksVersion(version string) bool {
	if b == nil {
		return false
	}
	if b.Versions == nil {
		return true
	}
	for _, candidate := range b.Versions {
		if candidate == strings.TrimPrefix(version, "v") {
			return true
		}
	}
	return false
}
