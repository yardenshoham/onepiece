package tracker

import (
	"log/slog"
	"testing"
	"time"

	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
)

func makeEntry(episodeNum int, title, seasonTitle string, datePlayed time.Time, fullyWatched bool) crunchyroll.WatchHistoryEntry {
	return crunchyroll.WatchHistoryEntry{
		ID:           "test-id",
		DatePlayed:   datePlayed,
		FullyWatched: fullyWatched,
		Panel: crunchyroll.Panel{
			Title: title,
			EpisodeMetadata: crunchyroll.EpisodeMetadata{
				EpisodeNumber: episodeNum,
				SeasonNumber:  1,
				SeasonTitle:   seasonTitle,
				SeriesID:      crunchyroll.OnePieceSeriesID,
				SeriesTitle:   "One Piece",
			},
		},
	}
}

func TestComputeZeroEpisodes(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 61, SlugTitle: "east-blue-1-61"}}

	d := tr.Compute(profile, nil, seasons)

	if d.EpisodesWatched != 0 {
		t.Errorf("got EpisodesWatched %d, want 0", d.EpisodesWatched)
	}
	if d.TotalEpisodes != 61 {
		t.Errorf("got TotalEpisodes %d, want 61", d.TotalEpisodes)
	}
	if d.ProgressPercent != 0 {
		t.Errorf("got ProgressPercent %f, want 0", d.ProgressPercent)
	}
}

func TestComputeFiltersNonOnePiece(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	nonOP := crunchyroll.WatchHistoryEntry{
		DatePlayed:   time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		FullyWatched: true,
		Panel: crunchyroll.Panel{
			Title: "Some Other Show",
			EpisodeMetadata: crunchyroll.EpisodeMetadata{
				EpisodeNumber: 1,
				SeriesID:      "OTHER-SERIES",
			},
		},
	}

	d := tr.Compute(profile, []crunchyroll.WatchHistoryEntry{nonOP}, seasons)
	if d.EpisodesWatched != 0 {
		t.Errorf("got EpisodesWatched %d, want 0 (non-OP filtered)", d.EpisodesWatched)
	}
}

func TestComputeCountsPartiallyWatched(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	partial := makeEntry(1, "Ep 1", "East Blue", time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), false)
	d := tr.Compute(profile, []crunchyroll.WatchHistoryEntry{partial}, seasons)
	if d.EpisodesWatched != 1 {
		t.Errorf("got EpisodesWatched %d, want 1 (partially watched still counts)", d.EpisodesWatched)
	}
}

func TestComputeExcludesRemasteredSeasons(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{
		{NumberOfEpisodes: 61, SlugTitle: "east-blue-1-61"},
		{NumberOfEpisodes: 21, SlugTitle: "one-piece-log-fish-man-island-saga-remastered--re-edited"},
	}

	d := tr.Compute(profile, nil, seasons)
	if d.TotalEpisodes != 61 {
		t.Errorf("got TotalEpisodes %d, want 61 (remastered excluded)", d.TotalEpisodes)
	}
}

func TestComputeBasicMetrics(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Nakama"}
	seasons := []crunchyroll.Season{
		{NumberOfEpisodes: 100, SlugTitle: "east-blue"},
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Episode 1", "East Blue", now.Add(-48*time.Hour), true),
		makeEntry(2, "Episode 2", "East Blue", now.Add(-48*time.Hour+time.Hour), true),
		makeEntry(3, "Episode 3", "East Blue", now.Add(-24*time.Hour), true),
		makeEntry(4, "Episode 4", "East Blue", now.Add(time.Hour), true),
	}

	d := tr.Compute(profile, entries, seasons)

	if d.EpisodesWatched != 4 {
		t.Errorf("got EpisodesWatched %d, want 4", d.EpisodesWatched)
	}
	if d.EpisodesRemaining != 96 {
		t.Errorf("got EpisodesRemaining %d, want 96", d.EpisodesRemaining)
	}
	if d.ProgressPercent != 4.0 {
		t.Errorf("got ProgressPercent %f, want 4.0", d.ProgressPercent)
	}
	if d.LastEpisode.Number != 4 {
		t.Errorf("got LastEpisode.Number %d, want 4", d.LastEpisode.Number)
	}
	if d.ProfileName != "Nakama" {
		t.Errorf("got ProfileName %q, want %q", d.ProfileName, "Nakama")
	}
}

func TestComputeStreaks(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	entries := []crunchyroll.WatchHistoryEntry{
		// 3-day streak, then gap, then 2-day streak ending today
		makeEntry(1, "Ep 1", "East Blue", now.Add(-6*24*time.Hour), true),
		makeEntry(2, "Ep 2", "East Blue", now.Add(-5*24*time.Hour), true),
		makeEntry(3, "Ep 3", "East Blue", now.Add(-4*24*time.Hour), true),
		// gap on day -3
		makeEntry(4, "Ep 4", "East Blue", now.Add(-1*24*time.Hour), true),
		makeEntry(5, "Ep 5", "East Blue", now.Add(time.Hour), true), // today
	}

	d := tr.Compute(profile, entries, seasons)

	if d.CurrentStreak != 2 {
		t.Errorf("got CurrentStreak %d, want 2", d.CurrentStreak)
	}
	if d.LongestStreak != 3 {
		t.Errorf("got LongestStreak %d, want 3", d.LongestStreak)
	}
}

func TestComputeStreakContinuesIfNoWatchToday(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Ep 1", "East Blue", now.Add(-2*24*time.Hour), true),
		makeEntry(2, "Ep 2", "East Blue", now.Add(-1*24*time.Hour), true),
		// nothing today
	}

	d := tr.Compute(profile, entries, seasons)

	if d.CurrentStreak != 2 {
		t.Errorf("got CurrentStreak %d, want 2 (streak continues even without watching today)", d.CurrentStreak)
	}
	if d.LongestStreak != 2 {
		t.Errorf("got LongestStreak %d, want 2", d.LongestStreak)
	}
}

func TestComputeDailyEpisodesIncludesGaps(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Ep 1", "East Blue", base, true),
		// gap on Apr 2
		makeEntry(2, "Ep 2", "East Blue", base.Add(48*time.Hour), true),
	}

	d := tr.Compute(profile, entries, seasons)

	if len(d.DailyEpisodes) != 3 {
		t.Fatalf("got %d daily entries, want 3 (including gap day)", len(d.DailyEpisodes))
	}
	if d.DailyEpisodes[0].Date != "2026-04-01" || d.DailyEpisodes[0].Count != 1 {
		t.Errorf("day 0: got %+v", d.DailyEpisodes[0])
	}
	if d.DailyEpisodes[1].Date != "2026-04-02" || d.DailyEpisodes[1].Count != 0 {
		t.Errorf("day 1 (gap): got %+v", d.DailyEpisodes[1])
	}
	if d.DailyEpisodes[2].Date != "2026-04-03" || d.DailyEpisodes[2].Count != 1 {
		t.Errorf("day 2: got %+v", d.DailyEpisodes[2])
	}
}

func TestComputeRecentEpisodes(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	var entries []crunchyroll.WatchHistoryEntry
	for i := 1; i <= 15; i++ {
		entries = append(entries, makeEntry(i, "Episode", "East Blue", base.Add(time.Duration(i)*time.Hour), true))
	}

	d := tr.Compute(profile, entries, seasons)

	if len(d.RecentEpisodes) != 10 {
		t.Fatalf("got %d recent episodes, want 10", len(d.RecentEpisodes))
	}
	// Most recent first
	if d.RecentEpisodes[0].Number != 15 {
		t.Errorf("got first recent episode number %d, want 15", d.RecentEpisodes[0].Number)
	}
	if d.RecentEpisodes[9].Number != 6 {
		t.Errorf("got last recent episode number %d, want 6", d.RecentEpisodes[9].Number)
	}
}

func TestComputeDeduplicatesEpisodes(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Test"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}}

	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Ep 1", "East Blue", base, true),
		makeEntry(1, "Ep 1", "East Blue", base.Add(time.Hour), true), // duplicate
		makeEntry(2, "Ep 2", "East Blue", base.Add(2*time.Hour), true),
	}

	d := tr.Compute(profile, entries, seasons)
	if d.EpisodesWatched != 2 {
		t.Errorf("got EpisodesWatched %d, want 2 (deduped)", d.EpisodesWatched)
	}
}

func TestCalculateStreaks(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	daily := []DailyCount{
		{"2026-04-05", 2},
		{"2026-04-06", 1},
		{"2026-04-07", 3},
		{"2026-04-08", 0},
		{"2026-04-09", 1},
		{"2026-04-10", 2},
	}

	current, longest := calculateStreaks(daily, now)
	if current != 2 {
		t.Errorf("got current streak %d, want 2", current)
	}
	if longest != 3 {
		t.Errorf("got longest streak %d, want 3", longest)
	}
}

func TestCalculateStreaksFiveDaysNoWatchToday(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	daily := []DailyCount{
		{"2026-04-04", 0},
		{"2026-04-05", 1},
		{"2026-04-06", 2},
		{"2026-04-07", 1},
		{"2026-04-08", 3},
		{"2026-04-09", 1},
		{"2026-04-10", 0}, // today — haven't watched yet
	}

	current, longest := calculateStreaks(daily, now)
	if current != 5 {
		t.Errorf("got current streak %d, want 5", current)
	}
	if longest != 5 {
		t.Errorf("got longest streak %d, want 5", longest)
	}
}

func TestCalculateStreaksEmpty(t *testing.T) {
	t.Parallel()

	current, longest := calculateStreaks(nil, time.Now())
	if current != 0 || longest != 0 {
		t.Errorf("got current=%d longest=%d, want 0,0", current, longest)
	}
}
