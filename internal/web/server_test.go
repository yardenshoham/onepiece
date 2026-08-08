package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yardenshoham/onepiece/pkg/poller"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

func newTestServer(d *tracker.Dashboard) *Server {
	return newTestServerWithConfig(d, Config{})
}

func newTestServerWithConfig(d *tracker.Dashboard, config Config) *Server {
	logger := slog.Default()
	p := poller.NewPoller(logger, nil, nil, time.Hour, "")

	if d != nil {
		p.SetDashboard(d)
	}

	return NewServer(logger, p, config)
}

func TestRoutes(t *testing.T) {
	t.Parallel()

	populated := &tracker.Dashboard{
		ProfileName:     "Nakama",
		EpisodesWatched: 37,
		TotalEpisodes:   1178,
		ProgressPercent: 3.1,
		LastEpisode: tracker.EpisodeInfo{
			Number:      37,
			Title:       "Luffy Rises!",
			SeasonTitle: "East Blue (1-61)",
			WatchedAt:   time.Date(2026, 4, 10, 9, 58, 34, 0, time.UTC),
		},
		CurrentSeason:        "East Blue (1-61)",
		FirstWatchDate:       time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		DaysSinceFirst:       22,
		AvgEpisodesPerDay:    1.7,
		CurrentStreak:        4,
		LongestStreak:        4,
		EpisodesRemaining:    1141,
		EstimatedCatchUpDate: time.Date(2028, 3, 1, 0, 0, 0, 0, time.UTC),
		RecentEpisodes: []tracker.EpisodeInfo{
			{Number: 37, Title: "Luffy Rises!", SeasonTitle: "East Blue", WatchedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
		},
		DailyEpisodes: []tracker.DailyCount{
			{Date: "2026-04-09", Count: 2},
			{Date: "2026-04-10", Count: 1},
		},
		LastUpdated: time.Date(2026, 4, 10, 10, 30, 0, 0, time.UTC),
	}

	tests := []struct {
		name       string
		dashboard  *tracker.Dashboard
		path       string
		wantStatus int
		wantBody   []string
	}{
		{
			name:       "health before the first fetch",
			path:       "/health",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   []string{"not ready"},
		},
		{
			name:       "health once a dashboard exists",
			dashboard:  &tracker.Dashboard{ProfileName: "Test", EpisodesWatched: 37, TotalEpisodes: 1178},
			path:       "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "dashboard before the first fetch",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   []string{"Loading"},
		},
		{
			name:       "dashboard with data",
			dashboard:  populated,
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody: []string{
				"Nakama", "37 / 1178", "1.7 episodes/day", "Luffy Rises!", "East Blue",
				`/static/app.css`,
				`<relative-time datetime="2026-04-10T10:30:00Z" format="relative">`, // last updated
				`<relative-time datetime="2026-03-19T00:00:00Z" format="datetime">`, // first watch date
				`<relative-time datetime="2026-04-10T09:58:34Z" format="relative">`, // last episode watched
				`<relative-time datetime="2028-03-01T00:00:00Z" format="datetime">`, // estimated catch-up
				"https://unpkg.com/@github/relative-time-element@5.3.1/dist/index.js",
			},
		},
		{
			name:       "about",
			path:       "/about",
			wantStatus: http.StatusOK,
			wantBody:   []string{"What is this?", "gomponents", `/static/app.css`},
		},
		{
			name:       "stylesheet",
			path:       "/static/app.css",
			wantStatus: http.StatusOK,
			wantBody:   []string{"--dashboard-shell-width"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestServer(tt.dashboard)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			s.mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
			body := w.Body.String()
			for _, want := range tt.wantBody {
				if !strings.Contains(body, want) {
					t.Errorf("expected body to contain %q", want)
				}
			}
		})
	}
}

func TestAboutPageAnalyticsConfigIsPerServer(t *testing.T) {
	t.Parallel()

	serverA := newTestServerWithConfig(nil, Config{PostHogAPIKey: "alpha"})
	serverB := newTestServerWithConfig(nil, Config{PostHogAPIKey: "beta", PostHogHost: "https://us.i.posthog.com"})

	requestA := httptest.NewRequest(http.MethodGet, "/about", nil)
	responseA := httptest.NewRecorder()
	serverA.mux.ServeHTTP(responseA, requestA)

	requestB := httptest.NewRequest(http.MethodGet, "/about", nil)
	responseB := httptest.NewRecorder()
	serverB.mux.ServeHTTP(responseB, requestB)

	bodyA := responseA.Body.String()
	bodyB := responseB.Body.String()

	if !strings.Contains(bodyA, `posthog.init("alpha",{api_host:"https://eu.i.posthog.com",person_profiles:'always'})`) {
		t.Error("expected first server to render its own analytics config")
	}
	if strings.Contains(bodyA, `posthog.init("beta",{api_host:"https://us.i.posthog.com",person_profiles:'always'})`) {
		t.Error("expected first server response to exclude second server analytics config")
	}
	if !strings.Contains(bodyB, `posthog.init("beta",{api_host:"https://us.i.posthog.com",person_profiles:'always'})`) {
		t.Error("expected second server to render its own analytics config")
	}
	if strings.Contains(bodyB, `posthog.init("alpha",{api_host:"https://eu.i.posthog.com",person_profiles:'always'})`) {
		t.Error("expected second server response to exclude first server analytics config")
	}
}

func TestUnknownPathsReturn404(t *testing.T) {
	t.Parallel()

	s := newTestServer(nil)

	paths := []string{
		"/.git/config",
		"/.env",
		"/.env.production",
		"/.aws/credentials",
		"/@fs/etc/passwd",
		"/@fs/app/.git/config",
		"/admin",
		"/swagger-ui.html",
		"/actuator/env",
		"/wp-login.php",
		"/config.json",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			s.mux.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("GET %s: got status %d, want %d", path, w.Code, http.StatusNotFound)
			}
		})
	}
}

func TestLoggingMiddlewareRecordsUserAgent(t *testing.T) {
	t.Parallel()

	// JSON rather than the production text handler so the assertions read
	// fields instead of matching serialized substrings.
	var buf bytes.Buffer
	s := newTestServer(nil)
	s.logger = slog.New(slog.NewJSONHandler(&buf, nil))

	const ua = "Mozilla/5.0 (compatible; TestBot/1.0)"
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	req.Header.Set("User-Agent", ua)
	w := httptest.NewRecorder()

	s.loggingMiddleware(s.mux).ServeHTTP(w, req)

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode log line %q: %v", buf.String(), err)
	}

	want := map[string]any{
		"msg":    "request",
		"method": http.MethodGet,
		"path":   "/about",
		"status": float64(http.StatusOK),
		"ua":     ua,
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("%s = %v, want %v", key, got[key], wantValue)
		}
	}
}
