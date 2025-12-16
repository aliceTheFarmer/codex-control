package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	releasesLatestURL = "https://api.github.com/repos/openai/codex/releases/latest"
	releasesListURL   = "https://api.github.com/repos/openai/codex/releases"
	userAgent         = "codex-control/1.0"
)

// Release represents a GitHub release entry.
type Release struct {
	Tag         string
	PublishedAt time.Time
	Assets      []Asset
}

// Asset represents an artifact tied to a release.
type Asset struct {
	Name string
	URL  string
	Size int64
}

// Client fetches Codex release metadata from GitHub.
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient builds a GitHub client using the provided http.Client.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	return &Client{httpClient: httpClient, token: token}
}

// Latest fetches the newest release metadata.
func (c *Client) Latest(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesLatestURL, nil)
	if err != nil {
		return Release{}, err
	}
	c.decorateHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected GitHub status: %s", resp.Status)
	}
	var body releasePayload
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Release{}, err
	}
	return body.toRelease(), nil
}

// List fetches releases up to the requested limit.
func (c *Client) List(ctx context.Context, limit int) ([]Release, error) {
	if limit <= 0 {
		limit = 1
	}
	perPage := 100
	releases := make([]Release, 0, limit)
	for page := 1; len(releases) < limit; page++ {
		url := fmt.Sprintf("%s?per_page=%d&page=%d", releasesListURL, perPage, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		c.decorateHeaders(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected GitHub status: %s", resp.Status)
		}
		var payload []releasePayload
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if len(payload) == 0 {
			break
		}
		for _, item := range payload {
			releases = append(releases, item.toRelease())
			if len(releases) >= limit {
				break
			}
		}
	}
	return releases, nil
}

// FindAsset finds an asset by name.
func (r Release) FindAsset(name string) (Asset, bool) {
	for _, asset := range r.Assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return Asset{}, false
}

func (c *Client) decorateHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
}

type releasePayload struct {
	TagName     string         `json:"tag_name"`
	PublishedAt string         `json:"published_at"`
	Assets      []assetPayload `json:"assets"`
}

type assetPayload struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

func (r releasePayload) toRelease() Release {
	assets := make([]Asset, 0, len(r.Assets))
	for _, asset := range r.Assets {
		assets = append(assets, Asset{Name: asset.Name, URL: asset.URL, Size: asset.Size})
	}
	return Release{Tag: r.TagName, PublishedAt: parseTime(r.PublishedAt), Assets: assets}
}

func parseTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	return time.Time{}
}
