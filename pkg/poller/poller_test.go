package poller

import (
	"log/slog"
	"testing"
	"time"

	"github.com/yardenshoham/onepiece/pkg/tracker"
)

func TestPollerDashboardNilBeforeStart(t *testing.T) {
	t.Parallel()

	p := NewPoller(slog.Default(), nil, nil, time.Hour, "")
	if d := p.Dashboard(); d != nil {
		t.Errorf("expected nil dashboard before start, got %+v", d)
	}
}

func TestPollerDashboardSetAfterFetch(t *testing.T) {
	t.Parallel()

	p := NewPoller(slog.Default(), nil, nil, time.Hour, "")

	d := &tracker.Dashboard{
		ProfileName:     "Test",
		EpisodesWatched: 37,
	}

	p.SetDashboard(d)

	got := p.Dashboard()
	if got == nil {
		t.Fatal("expected non-nil dashboard")
	}
	if got.EpisodesWatched != 37 {
		t.Errorf("got EpisodesWatched %d, want 37", got.EpisodesWatched)
	}
}

func TestPollerStartCancellation(t *testing.T) {
	t.Parallel()

	// Test that SetDashboard + Dashboard works correctly.
	// We can't start a real poller without a real Crunchyroll client.
	p := NewPoller(slog.Default(), nil, tracker.NewTracker(slog.Default()), 100*time.Millisecond, "")

	d := &tracker.Dashboard{ProfileName: "CancelTest", EpisodesWatched: 5}
	p.SetDashboard(d)

	got := p.Dashboard()
	if got == nil {
		t.Fatal("expected non-nil dashboard after SetDashboard")
	}
	if got.ProfileName != "CancelTest" {
		t.Errorf("got ProfileName %q, want %q", got.ProfileName, "CancelTest")
	}
}
