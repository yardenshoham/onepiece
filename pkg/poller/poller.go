package poller

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
	"github.com/yardenshoham/onepiece/pkg/healthchecks"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// WikiEnricher fetches long descriptions for episodes from an external wiki.
// Implementations must be safe to call concurrently.
type WikiEnricher interface {
	FetchLongDescription(ctx context.Context, episodeNumber int) (string, error)
}

// Poller periodically fetches data from Crunchyroll and recomputes the dashboard.
type Poller struct {
	logger      *slog.Logger
	client      *crunchyroll.Client
	tracker     *tracker.Tracker
	interval    time.Duration
	healthcheck *healthchecks.Client
	wiki        WikiEnricher

	mu        sync.RWMutex
	dashboard *tracker.Dashboard

	descMu    sync.Mutex
	descCache map[int]string // episode number → long description
}

// NewPoller creates a poller that fetches data and recomputes the dashboard.
// If healthcheckUUID is non-empty, the poller will send start/success/fail
// signals to healthchecks.io for each poll cycle.
func NewPoller(logger *slog.Logger, client *crunchyroll.Client, tracker *tracker.Tracker, interval time.Duration, healthcheckUUID string) *Poller {
	p := &Poller{
		logger:    logger,
		client:    client,
		tracker:   tracker,
		interval:  interval,
		descCache: make(map[int]string),
	}
	if healthcheckUUID != "" {
		p.healthcheck = healthchecks.NewClient(logger, healthcheckUUID)
		logger.Info("healthchecks.io monitoring enabled", "uuid", healthcheckUUID)
	}
	return p
}

// SetWikiEnricher registers a WikiEnricher that will populate LongDescription
// on recent episodes during each poll cycle. Call before the first Fetch.
func (p *Poller) SetWikiEnricher(w WikiEnricher) {
	p.wiki = w
}

// Start begins the polling loop. It blocks until ctx is cancelled.
func (p *Poller) Start(ctx context.Context) error {
	p.logger.Info("polling started", "interval", p.interval)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("polling stopped")
			return nil
		case <-ticker.C:
			if err := p.Fetch(ctx); err != nil {
				p.logger.Error("polling failed", "error", err)
			}
		}
	}
}

// Fetch performs a single data fetch and updates the dashboard.
func (p *Poller) Fetch(ctx context.Context) error {
	start := time.Now()

	// Generate a run ID so healthchecks.io can correlate start/completion signals
	rid := newUUID()

	if p.healthcheck != nil {
		p.healthcheck.Start(ctx, rid)
	}

	profile, err := p.client.GetProfile(ctx)
	if err != nil {
		err = fmt.Errorf("getting profile: %w", err)
		if p.healthcheck != nil {
			p.healthcheck.Fail(ctx, rid, err.Error())
		}
		return err
	}

	history, err := p.client.GetAllWatchHistory(ctx)
	if err != nil {
		err = fmt.Errorf("getting watch history: %w", err)
		if p.healthcheck != nil {
			p.healthcheck.Fail(ctx, rid, err.Error())
		}
		return err
	}

	seasons, err := p.client.GetSeasons(ctx, crunchyroll.OnePieceSeriesID)
	if err != nil {
		err = fmt.Errorf("getting seasons: %w", err)
		if p.healthcheck != nil {
			p.healthcheck.Fail(ctx, rid, err.Error())
		}
		return err
	}

	dashboard := p.tracker.Compute(time.Now().UTC(), *profile, history, seasons)

	if p.wiki != nil {
		p.enrichDashboard(ctx, dashboard)
	}

	p.mu.Lock()
	p.dashboard = dashboard
	p.mu.Unlock()

	took := time.Since(start).Round(time.Millisecond)

	p.logger.Info("polling completed",
		"episodes", dashboard.EpisodesWatched,
		"took", took,
	)

	if p.healthcheck != nil {
		body := fmt.Sprintf("profile=%s episodes=%d took=%s",
			dashboard.ProfileName, dashboard.EpisodesWatched, took)
		p.healthcheck.Success(ctx, rid, body)
	}

	return nil
}

// Dashboard returns the latest computed dashboard, or nil if not yet fetched.
func (p *Poller) Dashboard() *tracker.Dashboard {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dashboard
}

// SetDashboard sets the dashboard directly. This is intended for testing.
func (p *Poller) SetDashboard(d *tracker.Dashboard) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dashboard = d
}

// enrichDashboard fetches long descriptions for the first five recent episodes
// that are not yet in the cache, and writes them into the dashboard in-place.
// Fetches are concurrent; failures are logged and skipped.
func (p *Poller) enrichDashboard(ctx context.Context, d *tracker.Dashboard) {
	limit := min(5, len(d.RecentEpisodes))
	if limit == 0 {
		return
	}

	episodes := d.RecentEpisodes[:limit]

	// Determine which episode numbers need a wiki fetch.
	p.descMu.Lock()
	var toFetch []int
	for _, ep := range episodes {
		if _, ok := p.descCache[ep.Number]; !ok {
			toFetch = append(toFetch, ep.Number)
		}
	}
	p.descMu.Unlock()

	if len(toFetch) > 0 {
		var wg sync.WaitGroup
		for _, num := range toFetch {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				desc, err := p.wiki.FetchLongDescription(ctx, n)
				if err != nil {
					p.logger.Warn("wiki fetch failed", "episode", n, "error", err)
					return
				}
				p.descMu.Lock()
				p.descCache[n] = desc
				p.descMu.Unlock()
			}(num)
		}
		wg.Wait()
	}

	// Apply cached descriptions to the dashboard slice.
	p.descMu.Lock()
	defer p.descMu.Unlock()
	for i := range d.RecentEpisodes {
		if desc, ok := p.descCache[d.RecentEpisodes[i].Number]; ok {
			d.RecentEpisodes[i].LongDescription = desc
		}
	}
}

// newUUID generates a random UUID v4 string.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
