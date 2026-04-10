package poller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
	"github.com/yardenshoham/onepiece/pkg/tracker"
)

// Poller periodically fetches data from Crunchyroll and recomputes the dashboard.
type Poller struct {
	logger   *slog.Logger
	client   *crunchyroll.Client
	tracker  *tracker.Tracker
	interval time.Duration

	mu        sync.RWMutex
	dashboard *tracker.Dashboard
}

// NewPoller creates a poller that fetches data and recomputes the dashboard.
func NewPoller(logger *slog.Logger, client *crunchyroll.Client, tracker *tracker.Tracker, interval time.Duration) *Poller {
	return &Poller{
		logger:   logger,
		client:   client,
		tracker:  tracker,
		interval: interval,
	}
}

// Start begins polling. It performs an immediate fetch, then repeats on interval.
// It blocks until ctx is cancelled. Returns the first fetch error (if any).
func (p *Poller) Start(ctx context.Context) error {
	p.logger.Info("polling started", "interval", p.interval)

	// Initial fetch is blocking and fatal on error
	if err := p.fetch(ctx); err != nil {
		return fmt.Errorf("initial fetch failed: %w", err)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("polling stopped")
			return nil
		case <-ticker.C:
			if err := p.fetch(ctx); err != nil {
				p.logger.Error("polling failed", "error", err)
			}
		}
	}
}

func (p *Poller) fetch(ctx context.Context) error {
	start := time.Now()

	profile, err := p.client.GetProfile(ctx)
	if err != nil {
		return fmt.Errorf("getting profile: %w", err)
	}

	history, err := p.client.GetAllWatchHistory(ctx)
	if err != nil {
		return fmt.Errorf("getting watch history: %w", err)
	}

	seasons, err := p.client.GetSeasons(ctx, crunchyroll.OnePieceSeriesID)
	if err != nil {
		return fmt.Errorf("getting seasons: %w", err)
	}

	dashboard := p.tracker.Compute(*profile, history, seasons)

	p.mu.Lock()
	p.dashboard = dashboard
	p.mu.Unlock()

	p.logger.Info("polling completed",
		"episodes", dashboard.EpisodesWatched,
		"took", time.Since(start).Round(time.Millisecond),
	)
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
