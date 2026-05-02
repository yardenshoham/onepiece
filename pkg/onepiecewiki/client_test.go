package onepiecewiki

import (
	"log/slog"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestExtractParagraphs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantSubs []string
		wantNone bool
	}{
		{
			name:     "single paragraph",
			html:     "<p>Hello world</p>",
			wantSubs: []string{"Hello world"},
		},
		{
			name:     "multiple paragraphs",
			html:     "<p>First</p><p>Second</p>",
			wantSubs: []string{"First", "Second"},
		},
		{
			name:     "nested elements inside paragraph",
			html:     "<p>Hello <b>bold</b> world</p>",
			wantSubs: []string{"Hello bold world"},
		},
		{
			name:     "whitespace-only paragraph skipped",
			html:     "<p>   </p><p>Real content</p>",
			wantSubs: []string{"Real content"},
		},
		{
			name:     "non-paragraph elements ignored",
			html:     "<div>ignored</div><p>kept</p>",
			wantSubs: []string{"kept"},
		},
		{
			name:     "empty string",
			html:     "",
			wantNone: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractParagraphs(tc.html)
			if tc.wantNone {
				if got != "" {
					t.Errorf("expected empty string, got %q", got)
				}
				return
			}
			for _, sub := range tc.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("expected output to contain %q, got %q", sub, got)
				}
			}
		})
	}
}

func TestNodeText(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(strings.NewReader("<p>Hello <b>world</b>!</p>"))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	// Find the <p> node
	var pNode *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			pNode = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)

	if pNode == nil {
		t.Fatal("could not find <p> node")
	}

	got := nodeText(pNode)
	want := "Hello world!"
	if got != want {
		t.Errorf("nodeText = %q, want %q", got, want)
	}
}

// Integration tests — call the real One Piece wiki.

func TestFetchLongDescription_KnownEpisode(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	c := NewClient(slog.Default())
	desc, err := c.FetchLongDescription(t.Context(), 1)
	if err != nil {
		t.Fatalf("FetchLongDescription(1): %v", err)
	}
	if desc == "" {
		t.Fatal("expected non-empty description for episode 1")
	}
	// Episode 1 is "I'm Luffy! The Man Who Will Become the Pirate King!" — the summary
	// should mention Luffy.
	if !strings.Contains(desc, "Luffy") {
		t.Errorf("expected description to mention Luffy, got: %q", desc[:min(200, len(desc))])
	}
}

func TestFetchLongDescription_RecentEpisode(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Episode 1000 aired on November 20, 2021 — a stable page.
	c := NewClient(slog.Default())
	desc, err := c.FetchLongDescription(t.Context(), 1000)
	if err != nil {
		t.Fatalf("FetchLongDescription(1000): %v", err)
	}
	if desc == "" {
		t.Fatal("expected non-empty description for episode 1000")
	}
}

func TestFetchLongDescription_NonExistentEpisode(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	c := NewClient(slog.Default())
	// Use an episode number far beyond the current run that won't exist.
	desc, err := c.FetchLongDescription(t.Context(), 999999)
	// The wiki returns an error page for missing articles; the client should
	// either return an error or an empty description — not panic.
	if err == nil && desc != "" {
		t.Logf("got non-empty description for episode 999999 (unexpected but not fatal): %q", desc[:min(100, len(desc))])
	}
}
