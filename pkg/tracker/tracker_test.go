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
			},
		},
	}
}

func TestComputeCounts(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		history      []crunchyroll.WatchHistoryEntry
		seasons      []crunchyroll.Season
		wantWatched  int
		wantTotal    int
		wantProgress float64
	}{
		{
			name:        "no history",
			seasons:     []crunchyroll.Season{{NumberOfEpisodes: 61, SlugTitle: "east-blue-1-61"}},
			wantWatched: 0,
			wantTotal:   61,
		},
		{
			name: "non-One Piece entries are filtered out",
			history: []crunchyroll.WatchHistoryEntry{{
				DatePlayed:   base,
				FullyWatched: true,
				Panel: crunchyroll.Panel{
					Title: "Some Other Show",
					EpisodeMetadata: crunchyroll.EpisodeMetadata{
						EpisodeNumber: 1,
						SeriesID:      "OTHER-SERIES",
					},
				},
			}},
			seasons:     []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}},
			wantWatched: 0,
			wantTotal:   100,
		},
		{
			name:         "partially watched still counts",
			history:      []crunchyroll.WatchHistoryEntry{makeEntry(1, "Ep 1", "East Blue", base, false)},
			seasons:      []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}},
			wantWatched:  1,
			wantTotal:    100,
			wantProgress: 1.0,
		},
		{
			name: "remastered seasons excluded from the total",
			seasons: []crunchyroll.Season{
				{NumberOfEpisodes: 61, SlugTitle: "east-blue-1-61"},
				{NumberOfEpisodes: 21, SlugTitle: "one-piece-log-fish-man-island-saga-remastered--re-edited"},
			},
			wantWatched: 0,
			wantTotal:   61,
		},
		{
			name: "the same episode watched twice counts once",
			history: []crunchyroll.WatchHistoryEntry{
				makeEntry(1, "Ep 1", "East Blue", base, true),
				makeEntry(1, "Ep 1", "East Blue", base.Add(time.Hour), true), // duplicate
				makeEntry(2, "Ep 2", "East Blue", base.Add(2*time.Hour), true),
			},
			seasons:      []crunchyroll.Season{{NumberOfEpisodes: 100, SlugTitle: "east-blue"}},
			wantWatched:  2,
			wantTotal:    100,
			wantProgress: 2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := NewTracker(slog.Default())

			d := tr.Compute(now, crunchyroll.Profile{ProfileName: "Test"}, tt.history, tt.seasons)

			if d.EpisodesWatched != tt.wantWatched {
				t.Errorf("got EpisodesWatched %d, want %d", d.EpisodesWatched, tt.wantWatched)
			}
			if d.TotalEpisodes != tt.wantTotal {
				t.Errorf("got TotalEpisodes %d, want %d", d.TotalEpisodes, tt.wantTotal)
			}
			if d.ProgressPercent != tt.wantProgress {
				t.Errorf("got ProgressPercent %v, want %v", d.ProgressPercent, tt.wantProgress)
			}
		})
	}
}

func TestComputeBasicMetrics(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Nakama"}
	seasons := []crunchyroll.Season{
		{NumberOfEpisodes: 100, SlugTitle: "east-blue"},
	}

	now := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Episode 1", "East Blue", now.Add(-48*time.Hour), true),
		makeEntry(2, "Episode 2", "East Blue", now.Add(-48*time.Hour+time.Hour), true),
		makeEntry(3, "Episode 3", "East Blue", now.Add(-24*time.Hour), true),
		makeEntry(4, "Episode 4", "East Blue", now.Add(time.Hour), true),
	}

	d := tr.Compute(now, profile, entries, seasons)

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

	now := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		// 3-day streak, then gap, then 2-day streak ending today
		makeEntry(1, "Ep 1", "East Blue", now.Add(-6*24*time.Hour), true),
		makeEntry(2, "Ep 2", "East Blue", now.Add(-5*24*time.Hour), true),
		makeEntry(3, "Ep 3", "East Blue", now.Add(-4*24*time.Hour), true),
		// gap on day -3
		makeEntry(4, "Ep 4", "East Blue", now.Add(-1*24*time.Hour), true),
		makeEntry(5, "Ep 5", "East Blue", now.Add(time.Hour), true), // today
	}

	d := tr.Compute(now, profile, entries, seasons)

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

	now := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Ep 1", "East Blue", now.Add(-2*24*time.Hour), true),
		makeEntry(2, "Ep 2", "East Blue", now.Add(-1*24*time.Hour), true),
		// nothing today
	}

	d := tr.Compute(now, profile, entries, seasons)

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
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	entries := []crunchyroll.WatchHistoryEntry{
		makeEntry(1, "Ep 1", "East Blue", base, true),
		// gap on Apr 2
		makeEntry(2, "Ep 2", "East Blue", base.Add(48*time.Hour), true),
	}

	d := tr.Compute(now, profile, entries, seasons)

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
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	var entries []crunchyroll.WatchHistoryEntry
	for i := 1; i <= 15; i++ {
		entries = append(entries, makeEntry(i, "Episode", "East Blue", base.Add(time.Duration(i)*time.Hour), true))
	}

	d := tr.Compute(now, profile, entries, seasons)

	if len(d.RecentEpisodes) != 12 {
		t.Fatalf("got %d recent episodes, want 12", len(d.RecentEpisodes))
	}
	// Most recent first
	if d.RecentEpisodes[0].Number != 15 {
		t.Errorf("got first recent episode number %d, want 15", d.RecentEpisodes[0].Number)
	}
	if d.RecentEpisodes[11].Number != 4 {
		t.Errorf("got last recent episode number %d, want 4", d.RecentEpisodes[11].Number)
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

func TestEpisodeInfoLabelAndIsSpecial(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		info      EpisodeInfo
		wantLabel string
		wantSpec  bool
	}{
		{"story episode", EpisodeInfo{Number: 592, EpisodeLabel: "592"}, "592", false},
		{"wano special", EpisodeInfo{Number: 3, EpisodeLabel: "SP3"}, "SP3", true},
		{"egghead recap", EpisodeInfo{Number: 40, EpisodeLabel: "Recap"}, "Recap", true},
		{"missing label falls back to number", EpisodeInfo{Number: 592}, "592", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := c.info.Label(); got != c.wantLabel {
				t.Errorf("Label() = %q, want %q", got, c.wantLabel)
			}
			if got := c.info.IsSpecial(); got != c.wantSpec {
				t.Errorf("IsSpecial() = %v, want %v", got, c.wantSpec)
			}
		})
	}
}

// TestComputeCatchUpDateStableAtSteadyRate is the reason the projection uses a
// net rate: watching at a constant pace while episodes keep airing must not
// push the estimate further away every day.
func TestComputeCatchUpDateStableAtSteadyRate(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Nakama"}
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Watch 4 episodes a day for 100 days, then simulate 70 more days at the
	// same pace with one new episode airing every 7 days.
	const perDay = 4
	var entries []crunchyroll.WatchHistoryEntry
	episode := 0
	addDay := func(day int) {
		for range perDay {
			episode++
			entries = append(entries, makeEntry(episode, "Ep", "East Blue", start.AddDate(0, 0, day), true))
		}
	}
	for day := range 100 {
		addDay(day)
	}

	totalEpisodes := 3000
	now := start.AddDate(0, 0, 99)
	want := tr.Compute(now, profile, entries, []crunchyroll.Season{{NumberOfEpisodes: totalEpisodes}}).EstimatedCatchUpDate
	if want.IsZero() {
		t.Fatal("got zero EstimatedCatchUpDate, want a date")
	}

	for day := 100; day < 170; day++ {
		addDay(day)
		if day%7 == 0 {
			totalEpisodes++
		}
		now = start.AddDate(0, 0, day)
		got := tr.Compute(now, profile, entries, []crunchyroll.Season{{NumberOfEpisodes: totalEpisodes}}).EstimatedCatchUpDate
		// Integer episode counts and day rounding leave a little jitter; what
		// matters is that the date does not drift steadily away.
		if diff := got.Sub(want); diff < -3*24*time.Hour || diff > 3*24*time.Hour {
			t.Fatalf("day %d: got EstimatedCatchUpDate %v, want within 3 days of %v", day, got.Format(dateFormat), want.Format(dateFormat))
		}
	}
}

func TestComputeNoCatchUpDateWhenSlowerThanReleases(t *testing.T) {
	t.Parallel()
	tr := NewTracker(slog.Default())

	profile := crunchyroll.Profile{ProfileName: "Nakama"}
	seasons := []crunchyroll.Season{{NumberOfEpisodes: 1200}}
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// One episode every 14 days: slower than the release rate, so the gap
	// never closes.
	var entries []crunchyroll.WatchHistoryEntry
	for i := range 10 {
		entries = append(entries, makeEntry(i+1, "Ep", "East Blue", start.AddDate(0, 0, i*14), true))
	}

	d := tr.Compute(start.AddDate(0, 0, 9*14), profile, entries, seasons)

	if !d.EstimatedCatchUpDate.IsZero() {
		t.Errorf("got EstimatedCatchUpDate %v, want zero", d.EstimatedCatchUpDate)
	}
}
