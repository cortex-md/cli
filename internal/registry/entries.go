package registry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type IndexEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Author        string `json:"author"`
	AuthorURL     string `json:"authorUrl,omitempty"`
	Description   string `json:"description"`
	CoverImageURL string `json:"coverImageUrl"`
	Repo          string `json:"repo"`
}

func ParseEntries(content string) ([]IndexEntry, error) {
	if strings.TrimSpace(content) == "" {
		return []IndexEntry{}, nil
	}
	entries := []IndexEntry{}
	if err := json.Unmarshal([]byte(content), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse registry JSON: %w", err)
	}
	return entries, nil
}

func FindEntry(entries []IndexEntry, id string) *IndexEntry {
	for index := range entries {
		if entries[index].ID == id {
			entry := entries[index]
			return &entry
		}
	}
	return nil
}

func CanonicalJSON(entries []IndexEntry) (string, error) {
	content, err := json.MarshalIndent(entries, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal registry JSON: %w", err)
	}
	return string(content) + "\n", nil
}

func upsertEntry(entries []IndexEntry, next IndexEntry) []IndexEntry {
	updated := false
	for index := range entries {
		if entries[index].ID == next.ID {
			entries[index] = mergeEntry(entries[index], next)
			updated = true
			break
		}
	}
	if !updated {
		entries = append(entries, next)
	}

	sort.SliceStable(entries, func(left, right int) bool {
		return entries[left].ID < entries[right].ID
	})
	return entries
}

func mergeEntry(current IndexEntry, next IndexEntry) IndexEntry {
	if next.ID != "" {
		current.ID = next.ID
	}
	if next.Name != "" {
		current.Name = next.Name
	}
	if next.Author != "" {
		current.Author = next.Author
	}
	if next.AuthorURL != "" {
		current.AuthorURL = next.AuthorURL
	}
	if next.Description != "" {
		current.Description = next.Description
	}
	if next.CoverImageURL != "" {
		current.CoverImageURL = next.CoverImageURL
	}
	if next.Repo != "" {
		current.Repo = next.Repo
	}
	return current
}
