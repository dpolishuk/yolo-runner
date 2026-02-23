package docs

import (
	"strings"
	"testing"
)

func TestTicketFrontmatterSchemaDocumentsDependenciesAndTopologies(t *testing.T) {
	testSchema := readRepoFile(t, "docs", "ticket-frontmatter-schema.md")

	required := []string{
		"## Ticket Frontmatter Schema",
		"| `deps` ",
		"deps must be a YAML array",
		"### Serial dependencies",
		"### Parallel dependencies",
		"## Core Test Cases for Dependency Graphs",
		"Linear chain",
		"Fan-out",
		"Fan-in",
		"Cycles",
	}

	for _, expected := range required {
		if !containsCaseInsensitive(testSchema, expected) {
			t.Fatalf("ticket-frontmatter-schema.md missing required content: %q", expected)
		}
	}
}

func containsCaseInsensitive(haystack string, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
