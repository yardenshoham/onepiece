package crunchyroll

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	baseURL        = "https://www.crunchyroll.com"
	authEndpoint   = baseURL + "/auth/v1/token"
	profileURL     = baseURL + "/accounts/v1/me/profile"
	basicAuthToken = "eTJhcnZqYjBoMHJndnRpemxvdnk6SlZMdndkSXBYdnhVLXFJQnZUMU04b1FUcjFxbFFKWDI="
	userAgent      = "Crunchyroll/ANDROIDTV/3.59.0_22338 (Android 13.0; en-US; TCL-S5400AF Build/TP1A.220624.014)"
	pageSize       = 100

	// maxResponseSize caps how much data we read from API responses.
	// Measured largest response (watch-history page) was ~211 KB;
	// 20× that ≈ 4.2 MB. We use 10 MB as a generous ceiling.
	maxResponseSize = 10 * 1024 * 1024

	// maxPages caps pagination to prevent infinite loops if the API
	// returns inconsistent data. 300 × 100 = 30,000 entries.
	maxPages = 300
)

// Client is a Crunchyroll API client with automatic token refresh.
type Client struct {
	logger       *slog.Logger
	httpClient   *http.Client
	email        string
	password     string
	deviceID     string
	accessToken  string
	refreshToken string
	accountID    string
	tokenExpiry  time.Time
	mu           sync.Mutex
}

// NewClient authenticates with Crunchyroll and returns a ready-to-use client.
func NewClient(ctx context.Context, logger *slog.Logger, email, password string) (*Client, error) {
	c := &Client{
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		email:      email,
		password:   password,
		deviceID:   DeriveDeviceID(email),
	}

	if err := c.authenticate(ctx); err != nil {
		return nil, fmt.Errorf("authenticating: %w", err)
	}

	// Clear credentials from memory after initial auth
	c.email = ""
	c.password = ""

	return c, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	data := url.Values{}
	data.Set("username", c.email)
	data.Set("password", c.password)
	data.Set("grant_type", "password")
	data.Set("scope", "offline_access")
	data.Set("device_id", c.deviceID)
	data.Set("device_type", "ANDROIDTV")

	return c.doAuth(ctx, data)
}

func (c *Client) refreshAccessToken(ctx context.Context) error {
	c.logger.Info("refreshing access token")

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)
	data.Set("scope", "offline_access")
	data.Set("device_id", c.deviceID)
	data.Set("device_type", "ANDROIDTV")

	return c.doAuth(ctx, data)
}

func (c *Client) doAuth(ctx context.Context, data url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authEndpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+basicAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing auth request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("reading auth response: %w", err)
	}
	if len(body) > maxResponseSize {
		return fmt.Errorf("auth response too large: %d bytes exceeds %d byte limit", len(body), maxResponseSize)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed with status %d: %s", resp.StatusCode, body)
	}

	var authResp authResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	c.accessToken = authResp.AccessToken
	c.refreshToken = authResp.RefreshToken
	c.accountID = authResp.AccountID
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	c.logger.Info("authenticated successfully")
	return nil
}

func (c *Client) ensureValidToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.tokenExpiry.Add(-30 * time.Second)) {
		return c.accessToken, nil
	}
	if err := c.refreshAccessToken(ctx); err != nil {
		return "", fmt.Errorf("failed to refresh access token: %w", err)
	}
	return c.accessToken, nil
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, error) {
	token, err := c.ensureValidToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensuring valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("response too large: %d bytes exceeds %d byte limit", len(body), maxResponseSize)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error status %d: %s", resp.StatusCode, body)
	}

	return body, nil
}

// GetProfile returns the user's Crunchyroll profile.
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	body, err := c.doGet(ctx, profileURL)
	if err != nil {
		return nil, fmt.Errorf("getting profile: %w", err)
	}

	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decoding profile: %w", err)
	}
	return &profile, nil
}

// GetWatchHistory returns a single page of watch history.
func (c *Client) GetWatchHistory(ctx context.Context, page, ps int) (*WatchHistoryResponse, error) {
	u := fmt.Sprintf("%s/content/v2/%s/watch-history?page=%d&page_size=%d&locale=en-US",
		baseURL, c.accountID, page, ps)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting watch history page %d: %w", page, err)
	}

	var resp WatchHistoryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding watch history: %w", err)
	}
	return &resp, nil
}

// GetAllWatchHistory fetches all pages of watch history.
func (c *Client) GetAllWatchHistory(ctx context.Context) ([]WatchHistoryEntry, error) {
	var all []WatchHistoryEntry

	for page := 1; page <= maxPages; page++ {
		c.logger.Debug("fetching watch history", "page", page)
		resp, err := c.GetWatchHistory(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}

		all = append(all, resp.Data...)

		if len(resp.Data) < pageSize || len(all) >= resp.Total {
			c.logger.Info("fetched watch history", "total_entries", len(all))
			return all, nil
		}
	}

	return nil, fmt.Errorf("watch history exceeded %d pages, aborting", maxPages)
}

// GetSeasons returns all seasons for a series.
func (c *Client) GetSeasons(ctx context.Context, seriesID string) ([]Season, error) {
	u := fmt.Sprintf("%s/content/v2/cms/series/%s/seasons?locale=en-US", baseURL, seriesID)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting seasons: %w", err)
	}

	var resp seasonsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding seasons: %w", err)
	}
	return resp.Data, nil
}

// GetSeries returns series metadata.
func (c *Client) GetSeries(ctx context.Context, seriesID string) (*Series, error) {
	u := fmt.Sprintf("%s/content/v2/cms/series/%s?locale=en-US", baseURL, seriesID)

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("getting series: %w", err)
	}

	var resp seriesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding series: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no series data returned for %s", seriesID)
	}
	return &resp.Data[0], nil
}

// DeriveDeviceID returns a deterministic UUID-v4-shaped device ID derived from
// the user's email so the same device ID is reused across restarts.
func DeriveDeviceID(email string) string {
	h := sha256.Sum256([]byte(email))
	// Set version 4 and variant 2 bits on the hash bytes.
	h[6] = (h[6] & 0x0f) | 0x40
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}
