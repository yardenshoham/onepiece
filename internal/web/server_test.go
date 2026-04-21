package web

import (
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
	tr := tracker.NewTracker(logger)
	_ = tr
	p := poller.NewPoller(logger, nil, nil, time.Hour, "")

	if d != nil {
		p.SetDashboard(d)
	}

	return NewServer(logger, p, config)
}

func TestHealthEndpointNotReady(t *testing.T) {
	t.Parallel()

	s := newTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("got status %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(w.Body.String(), "not ready") {
		t.Errorf("got body %q, want 'not ready'", w.Body.String())
	}
}

func TestHealthEndpointReady(t *testing.T) {
	t.Parallel()

	d := &tracker.Dashboard{
		ProfileName:     "Test",
		EpisodesWatched: 37,
		TotalEpisodes:   1178,
	}
	s := newTestServer(d)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDashboardPageLoading(t *testing.T) {
	t.Parallel()

	s := newTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Loading") {
		t.Error("expected loading page content")
	}
}

func TestDashboardPageWithData(t *testing.T) {
	t.Parallel()

	d := &tracker.Dashboard{
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

	s := newTestServer(d)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	for _, want := range []string{"Nakama", "37 / 1178", "1.7 episodes/day", "Luffy Rises!", "East Blue"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q", want)
		}
	}

	if !strings.Contains(body, `<relative-time datetime="2026-04-10T10:30:00Z" format="relative">`) {
		t.Error("expected dashboard to render relative-time element")
	}

	if !strings.Contains(body, "https://unpkg.com/@github/relative-time-element@5.0.0/dist/index.js") {
		t.Error("expected dashboard to include relative-time-element script")
	}
}

func TestAboutPage(t *testing.T) {
	t.Parallel()

	s := newTestServer(nil)
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	w := httptest.NewRecorder()

	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "What is this?") {
		t.Error("expected about page content")
	}
	if !strings.Contains(body, "gomponents") {
		t.Error("expected gomponents credit")
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
