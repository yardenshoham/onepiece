package tracker

import (
	"log/slog"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/yardenshoham/onepiece/pkg/crunchyroll"
)

// Tracker computes dashboard metrics from raw API data.
type Tracker struct {
	logger *slog.Logger
}

// NewTracker returns a new Tracker.
func NewTracker(logger *slog.Logger) *Tracker {
	return &Tracker{logger: logger}
}

// Compute takes raw Crunchyroll data and returns a Dashboard snapshot.
// now is the current time and is used for all time-relative calculations.
func (t *Tracker) Compute(now time.Time, profile crunchyroll.Profile, history []crunchyroll.WatchHistoryEntry, seasons []crunchyroll.Season) *Dashboard {
	d := &Dashboard{
		ProfileName: profile.ProfileName,
		LastUpdated: now,
	}

	// Calculate total episodes from non-remastered seasons
	for _, s := range seasons {
		if strings.Contains(strings.ToLower(s.SlugTitle), "remastered") {
			continue
		}
		d.TotalEpisodes += s.NumberOfEpisodes
	}

	// Filter: only One Piece episodes that are in the watch history.
	// Include episodes even if not fully_watched — being in the history with a
	// later episode watched means the user progressed past it.
	var filtered []crunchyroll.WatchHistoryEntry
	for _, e := range history {
		if e.Panel.EpisodeMetadata.SeriesID == crunchyroll.OnePieceSeriesID {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return d
	}

	// Sort by date_played ascending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].DatePlayed.Before(filtered[j].DatePlayed)
	})

	// Deduplicate: same season+episode should count only once (keep earliest watch)
	type episodeKey struct {
		SeasonNumber  int
		EpisodeNumber int
	}
	seen := make(map[episodeKey]bool)
	var deduped []crunchyroll.WatchHistoryEntry
	for _, e := range filtered {
		key := episodeKey{
			SeasonNumber:  e.Panel.EpisodeMetadata.SeasonNumber,
			EpisodeNumber: e.Panel.EpisodeMetadata.EpisodeNumber,
		}
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, e)
		}
	}

	d.EpisodesWatched = len(deduped)
	d.EpisodesRemaining = max(d.TotalEpisodes-d.EpisodesWatched, 0)

	if d.TotalEpisodes > 0 {
		d.ProgressPercent = math.Round(float64(d.EpisodesWatched)/float64(d.TotalEpisodes)*1000) / 10
	}

	// First and last watch dates
	d.FirstWatchDate = deduped[0].DatePlayed.UTC()
	lastEntry := deduped[len(deduped)-1]

	d.LastEpisode = EpisodeInfo{
		Number:      lastEntry.Panel.EpisodeMetadata.EpisodeNumber,
		Title:       lastEntry.Panel.Title,
		SeasonTitle: lastEntry.Panel.EpisodeMetadata.SeasonTitle,
		WatchedAt:   lastEntry.DatePlayed.UTC(),
	}
	d.CurrentSeason = lastEntry.Panel.EpisodeMetadata.SeasonTitle

	// Days since first watch
	today := now.UTC().Truncate(24 * time.Hour)
	firstDate := d.FirstWatchDate.Truncate(24 * time.Hour)
	d.DaysSinceFirst = int(today.Sub(firstDate).Hours() / 24)

	// Average episodes per day
	if d.DaysSinceFirst == 0 {
		d.AvgEpisodesPerDay = float64(d.EpisodesWatched)
	} else {
		d.AvgEpisodesPerDay = math.Round(float64(d.EpisodesWatched)/float64(d.DaysSinceFirst)*10) / 10
	}

	// Estimated catch-up date
	if d.AvgEpisodesPerDay > 0 {
		daysNeeded := math.Ceil(float64(d.EpisodesRemaining) / d.AvgEpisodesPerDay)
		d.EstimatedCatchUpDate = today.AddDate(0, 0, int(daysNeeded))
	}

	// Build daily episode counts
	dailyMap := make(map[string]int)
	for _, e := range deduped {
		day := e.DatePlayed.UTC().Truncate(24 * time.Hour).Format("2006-01-02")
		dailyMap[day]++
	}

	lastDate := lastEntry.DatePlayed.UTC().Truncate(24 * time.Hour)
	for day := firstDate; !day.After(lastDate); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		count := dailyMap[key]
		dailyMap[key] = count // ensure key exists
	}

	// Sort daily counts
	var dailyKeys []string
	for k := range dailyMap {
		dailyKeys = append(dailyKeys, k)
	}
	slices.Sort(dailyKeys)

	d.DailyEpisodes = make([]DailyCount, 0, len(dailyKeys))
	for _, k := range dailyKeys {
		d.DailyEpisodes = append(d.DailyEpisodes, DailyCount{Date: k, Count: dailyMap[k]})
	}

	// Calculate streaks
	d.CurrentStreak, d.LongestStreak = calculateStreaks(d.DailyEpisodes, today)

	// Recent episodes (last 10, most recent first)
	d.RecentEpisodes = make([]EpisodeInfo, 0, 10)
	for i := len(deduped) - 1; i >= 0 && len(d.RecentEpisodes) < 10; i-- {
		e := deduped[i]
		d.RecentEpisodes = append(d.RecentEpisodes, EpisodeInfo{
			Number:      e.Panel.EpisodeMetadata.EpisodeNumber,
			Title:       e.Panel.Title,
			SeasonTitle: e.Panel.EpisodeMetadata.SeasonTitle,
			WatchedAt:   e.DatePlayed.UTC(),
		})
	}

	t.logger.Info("computed dashboard",
		"episodes_watched", d.EpisodesWatched,
		"total_episodes", d.TotalEpisodes,
		"progress", d.ProgressPercent,
	)

	return d
}

func calculateStreaks(daily []DailyCount, now time.Time) (current, longest int) {
	if len(daily) == 0 {
		return 0, 0
	}

	// Build a set of dates with watches
	watchDays := make(map[string]bool)
	for _, dc := range daily {
		if dc.Count > 0 {
			watchDays[dc.Date] = true
		}
	}

	// Current streak: count back from today (or yesterday if no watch today)
	start := now
	if !watchDays[now.Format("2006-01-02")] {
		start = now.AddDate(0, 0, -1)
	}
	if watchDays[start.Format("2006-01-02")] {
		current = 1
		d := start.AddDate(0, 0, -1)
		for watchDays[d.Format("2006-01-02")] {
			current++
			d = d.AddDate(0, 0, -1)
		}
	}

	// Longest streak: scan all days
	streak := 0
	for _, dc := range daily {
		if dc.Count > 0 {
			streak++
			if streak > longest {
				longest = streak
			}
		} else {
			streak = 0
		}
	}

	return current, longest
}
