// Package onepiecewiki fetches episode summaries from the One Piece fandom wiki
// using the MediaWiki api.php endpoint.
package onepiecewiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const apiEndpoint = "https://onepiece.fandom.com/api.php"

// Client fetches episode summaries from the One Piece fandom wiki.
type Client struct {
	logger     *slog.Logger
	httpClient *http.Client
}

// NewClient returns a new Client.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		logger:     logger,
		httpClient: &http.Client{},
	}
}

type sectionsResponse struct {
	Parse struct {
		Sections []struct {
			Index string `json:"index"`
			Line  string `json:"line"`
		} `json:"sections"`
	} `json:"parse"`
}

type textResponse struct {
	Parse struct {
		Text string `json:"text"`
	} `json:"parse"`
}

// FetchLongDescription fetches the Long Summary (falling back to Short Summary)
// for the given episode number. Returns empty string (without error) when the
// page exists but has no recognized summary section.
func (c *Client) FetchLongDescription(ctx context.Context, episodeNumber int) (string, error) {
	page := fmt.Sprintf("Episode_%d", episodeNumber)

	sectionIndex, err := c.findSummarySection(ctx, page)
	if err != nil {
		return "", fmt.Errorf("finding summary section for episode %d: %w", episodeNumber, err)
	}
	if sectionIndex == "" {
		c.logger.Debug("no summary section found", "episode", episodeNumber)
		return "", nil
	}

	text, err := c.fetchSectionText(ctx, page, sectionIndex)
	if err != nil {
		return "", fmt.Errorf("fetching section text for episode %d: %w", episodeNumber, err)
	}

	return text, nil
}

// findSummarySection returns the section index string for "Long Summary",
// falling back to "Short Summary". Returns "" if neither is found.
func (c *Client) findSummarySection(ctx context.Context, page string) (string, error) {
	params := url.Values{
		"action":        {"parse"},
		"page":          {page},
		"prop":          {"sections"},
		"format":        {"json"},
		"formatversion": {"2"},
	}
	body, err := c.get(ctx, params)
	if err != nil {
		return "", err
	}

	var sr sectionsResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", fmt.Errorf("parsing sections response: %w", err)
	}

	var fallback string
	for _, s := range sr.Parse.Sections {
		if s.Line == "Long Summary" {
			return s.Index, nil
		}
		if s.Line == "Short Summary" && fallback == "" {
			fallback = s.Index
		}
	}
	return fallback, nil
}

// fetchSectionText fetches and normalizes the paragraph text for the given section.
func (c *Client) fetchSectionText(ctx context.Context, page, sectionIndex string) (string, error) {
	params := url.Values{
		"action":        {"parse"},
		"page":          {page},
		"prop":          {"text"},
		"section":       {sectionIndex},
		"format":        {"json"},
		"formatversion": {"2"},
	}
	body, err := c.get(ctx, params)
	if err != nil {
		return "", err
	}

	var tr textResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parsing text response: %w", err)
	}

	return extractParagraphs(tr.Parse.Text), nil
}

// get performs an authenticated GET request to the wiki API.
func (c *Client) get(ctx context.Context, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "OnePieceTracker/1.0 (https://github.com/yardenshoham/onepiece)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	return data, nil
}

// extractParagraphs parses an HTML fragment and returns the concatenated text
// of all top-level <p> elements, separated by newlines.
func extractParagraphs(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "p" {
			text := nodeText(n)
			if text = strings.TrimSpace(text); text != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(text)
			}
			return // don't recurse into p children for nesting
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return sb.String()
}

// nodeText returns the concatenated text content of a node and its descendants.
func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
