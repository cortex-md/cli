package registry

import (
	"strings"
	"testing"
)

func TestUpsertEntryPreservesExistingOptionalFields(t *testing.T) {
	entries := []IndexEntry{{
		ID:            "test-plugin",
		Name:          "Old Name",
		Author:        "Old Author",
		AuthorURL:     "https://example.com/author",
		Description:   "Old description",
		CoverImageURL: "https://example.com/cover.png",
		Screenshots:   []string{"https://example.com/editor.png"},
		Repo:          "owner/old",
	}}

	updated := upsertEntry(entries, IndexEntry{
		ID:          "test-plugin",
		Name:        "New Name",
		Author:      "New Author",
		Description: "New description",
		Repo:        "owner/new",
	})

	entry := updated[0]
	if entry.AuthorURL != "https://example.com/author" {
		t.Fatalf("authorUrl = %s", entry.AuthorURL)
	}
	if entry.CoverImageURL != "https://example.com/cover.png" {
		t.Fatalf("coverImageUrl = %s", entry.CoverImageURL)
	}
	if len(entry.Screenshots) != 1 || entry.Screenshots[0] != "https://example.com/editor.png" {
		t.Fatalf("screenshots = %v", entry.Screenshots)
	}
	content, err := CanonicalJSON(updated)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "\"coverImageUrl\": \"https://example.com/cover.png\",\n\t\t\"screenshots\": [") {
		t.Fatalf("screenshots must follow coverImageUrl:\n%s", content)
	}
	if entry.Name != "New Name" || entry.Repo != "owner/new" {
		t.Fatalf("entry was not updated: %+v", entry)
	}
}
