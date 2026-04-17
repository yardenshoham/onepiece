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

// Poller periodically fetches data from Crunchyroll and recomputes the dashboard.
type Poller struct {
	logger      *slog.Logger
	client      *crunchyroll.Client
	tracker     *tracker.Tracker
	interval    time.Duration
	healthcheck *healthchecks.Client

	mu        sync.RWMutex
	dashboard *tracker.Dashboard
}

// NewPoller creates a poller that fetches data and recomputes the dashboard.
// If healthcheckUUID is non-empty, the poller will send start/success/fail
// signals to healthchecks.io for each poll cycle.
func NewPoller(logger *slog.Logger, client *crunchyroll.Client, tracker *tracker.Tracker, interval time.Duration, healthcheckUUID string) *Poller {
	p := &Poller{
		logger:   logger,
		client:   client,
		tracker:  tracker,
		interval: interval,
	}
	if healthcheckUUID != "" {
		p.healthcheck = healthchecks.NewClient(logger, healthcheckUUID)
		logger.Info("healthchecks.io monitoring enabled", "uuid", healthcheckUUID)
	}
	return p
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

	dashboard := p.tracker.Compute(*profile, history, seasons)

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

// newUUID generates a random UUID v4 string.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
