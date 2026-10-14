package registry

import "testing"

func TestUpsertEntryPreservesExistingOptionalFields(t *testing.T) {
	entries := []IndexEntry{{
		ID:            "test-plugin",
		Name:          "Old Name",
		Author:        "Old Author",
		AuthorURL:     "https://example.com/author",
		Description:   "Old description",
		CoverImageURL: "https://example.com/cover.png",
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
	if entry.Name != "New Name" || entry.Repo != "owner/new" {
		t.Fatalf("entry was not updated: %+v", entry)
	}
}
