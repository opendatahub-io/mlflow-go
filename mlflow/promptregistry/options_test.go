package promptregistry

import (
	"testing"
)

func TestWithVersion(t *testing.T) {
	opts := &loadOptions{}
	WithVersion(5)(opts)

	if opts.version != 5 {
		t.Errorf("version = %d, want %d", opts.version, 5)
	}
}

func TestWithDescription(t *testing.T) {
	opts := &registerOptions{}
	WithDescription("Initial version")(opts)

	if opts.description != "Initial version" {
		t.Errorf("description = %q, want %q", opts.description, "Initial version")
	}
}

func TestWithTags(t *testing.T) {
	opts := &registerOptions{}
	tags := map[string]string{"team": "ml", "env": "prod"}
	WithTags(tags)(opts)

	if len(opts.tags) != 2 {
		t.Errorf("tags length = %d, want %d", len(opts.tags), 2)
	}
	if opts.tags["team"] != "ml" {
		t.Errorf("tags[team] = %q, want %q", opts.tags["team"], "ml")
	}
}

func TestWithMaxResults(t *testing.T) {
	opts := &listPromptsOptions{}
	WithMaxResults(50)(opts)

	if opts.maxResults != 50 {
		t.Errorf("maxResults = %d, want %d", opts.maxResults, 50)
	}
}

func TestWithPageToken(t *testing.T) {
	opts := &listPromptsOptions{}
	WithPageToken("abc123")(opts)

	if opts.pageToken != "abc123" {
		t.Errorf("pageToken = %q, want %q", opts.pageToken, "abc123")
	}
}

func TestWithNameFilter(t *testing.T) {
	opts := &listPromptsOptions{}
	WithNameFilter("greeting%")(opts)

	if opts.nameFilter != "greeting%" {
		t.Errorf("nameFilter = %q, want %q", opts.nameFilter, "greeting%")
	}
}

func TestWithTagFilter(t *testing.T) {
	opts := &listPromptsOptions{}
	tags := map[string]string{"team": "ml"}
	WithTagFilter(tags)(opts)

	if opts.tagFilter["team"] != "ml" {
		t.Errorf("tagFilter[team] = %q, want %q", opts.tagFilter["team"], "ml")
	}
}

func TestWithOrderBy(t *testing.T) {
	opts := &listPromptsOptions{}
	WithOrderBy("name ASC", "timestamp DESC")(opts)

	if len(opts.orderBy) != 2 {
		t.Errorf("orderBy length = %d, want %d", len(opts.orderBy), 2)
	}
	if opts.orderBy[0] != "name ASC" {
		t.Errorf("orderBy[0] = %q, want %q", opts.orderBy[0], "name ASC")
	}
}

func TestWithVersionsMaxResults(t *testing.T) {
	opts := &listVersionsOptions{}
	WithVersionsMaxResults(25)(opts)

	if opts.maxResults != 25 {
		t.Errorf("maxResults = %d, want %d", opts.maxResults, 25)
	}
}

func TestWithVersionsPageToken(t *testing.T) {
	opts := &listVersionsOptions{}
	WithVersionsPageToken("xyz789")(opts)

	if opts.pageToken != "xyz789" {
		t.Errorf("pageToken = %q, want %q", opts.pageToken, "xyz789")
	}
}

func TestWithVersionsTagFilter(t *testing.T) {
	opts := &listVersionsOptions{}
	tags := map[string]string{"status": "prod"}
	WithVersionsTagFilter(tags)(opts)

	if opts.tagFilter["status"] != "prod" {
		t.Errorf("tagFilter[status] = %q, want %q", opts.tagFilter["status"], "prod")
	}
}

func TestWithVersionsOrderBy(t *testing.T) {
	opts := &listVersionsOptions{}
	WithVersionsOrderBy("version DESC")(opts)

	if len(opts.orderBy) != 1 {
		t.Errorf("orderBy length = %d, want %d", len(opts.orderBy), 1)
	}
	if opts.orderBy[0] != "version DESC" {
		t.Errorf("orderBy[0] = %q, want %q", opts.orderBy[0], "version DESC")
	}
}
