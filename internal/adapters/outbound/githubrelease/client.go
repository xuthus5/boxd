// Package githubrelease checks stable desktop releases published on GitHub.
package githubrelease

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xuthus5/boxd/internal/updatecheck"
)

const (
	defaultAPIBase  = "https://api.github.com"
	defaultTimeout  = 12 * time.Second
	releasePageSize = 30
	maxResponseSize = 4 << 20
	userAgent       = "BOXD desktop updater"
)

// Client reads the latest stable release from the public GitHub API.
type Client struct {
	repository string
	apiBase    string
	httpClient *http.Client
}

var _ updatecheck.Checker = (*Client)(nil)

// New constructs a client for repository, such as "xuthus5/boxd".
func New(repository string) *Client {
	return NewWithHTTPClient(repository, defaultAPIBase, nil)
}

// NewWithHTTPClient allows composition roots and tests to provide a custom API
// origin and HTTP client.
func NewWithHTTPClient(repository, apiBase string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	if strings.TrimSpace(apiBase) == "" {
		apiBase = defaultAPIBase
	}
	return &Client{
		repository: strings.Trim(strings.TrimSpace(repository), "/"),
		apiBase:    strings.TrimSuffix(strings.TrimSpace(apiBase), "/"),
		httpClient: client,
	}
}

// LatestRelease returns the highest stable semantic version among the newest
// published releases. Drafts and prereleases are ignored.
func (c *Client) LatestRelease(ctx context.Context) (updatecheck.Release, error) {
	if c == nil || c.repository == "" {
		return updatecheck.Release{}, fmt.Errorf("GitHub repository is not configured")
	}
	requestCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	endpoint := c.apiBase + "/repos/" + escapeRepository(c.repository) + "/releases"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return updatecheck.Release{}, fmt.Errorf("build GitHub releases URL: %w", err)
	}
	query := parsed.Query()
	query.Set("per_page", fmt.Sprintf("%d", releasePageSize))
	parsed.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return updatecheck.Release{}, fmt.Errorf("create GitHub releases request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)

	response, err := c.client().Do(request)
	if err != nil {
		return updatecheck.Release{}, fmt.Errorf("request GitHub releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
		return updatecheck.Release{}, fmt.Errorf("GitHub releases API returned HTTP %d: %s", response.StatusCode, apiError(message))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return updatecheck.Release{}, fmt.Errorf("read GitHub releases response: %w", err)
	}
	if len(body) > maxResponseSize {
		return updatecheck.Release{}, fmt.Errorf("GitHub releases response exceeds %d bytes", maxResponseSize)
	}

	var payload []releasePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return updatecheck.Release{}, fmt.Errorf("decode GitHub releases response: %w", err)
	}
	var selected updatecheck.Release
	found := false
	for _, item := range payload {
		if item.Draft || item.Prerelease || strings.TrimSpace(item.TagName) == "" {
			continue
		}
		candidate := updatecheck.Release{
			Tag:         strings.TrimSpace(item.TagName),
			Name:        strings.TrimSpace(item.Name),
			URL:         strings.TrimSpace(item.HTMLURL),
			PublishedAt: item.PublishedAt,
		}
		if !found {
			selected, found = candidate, true
			continue
		}
		comparison, compareErr := updatecheck.CompareVersions(selected.Tag, candidate.Tag)
		if compareErr == nil && comparison > 0 {
			selected = candidate
		}
	}
	if !found {
		return updatecheck.Release{}, updatecheck.ErrNoRelease
	}
	return selected, nil
}

func (c *Client) client() *http.Client {
	if c == nil || c.httpClient == nil {
		return &http.Client{Timeout: defaultTimeout}
	}
	return c.httpClient
}

func escapeRepository(repository string) string {
	parts := strings.SplitN(repository, "/", 2)
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}

func apiError(body []byte) string {
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Message) != "" {
		return payload.Message
	}
	return strings.TrimSpace(string(body))
}

type releasePayload struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}
