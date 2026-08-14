package tracker

import (
	"strconv"
	"time"
)

// Dashboard holds all computed metrics for the web UI.
type Dashboard struct {
	ProfileName string

	// Progress
	EpisodesWatched int
	TotalEpisodes   int
	ProgressPercent float64

	// Current position
	LastEpisode   EpisodeInfo
	CurrentSeason string

	// Watch rate
	FirstWatchDate    time.Time
	DaysSinceFirst    int
	AvgEpisodesPerDay float64

	// Streaks
	CurrentStreak int
	LongestStreak int

	// Prediction
	EpisodesRemaining    int
	EstimatedCatchUpDate time.Time

	// Recent activity
	RecentEpisodes []EpisodeInfo

	// Per-day breakdown for chart
	DailyEpisodes []DailyCount

	// Total watch time
	TotalWatchTimeMS int

	// Metadata
	LastUpdated time.Time
}

// EpisodeInfo holds information about a single watched episode.
type EpisodeInfo struct {
	Number          int
	EpisodeLabel    string // Crunchyroll's display label ("592", "SP3", "Recap"); may be empty
	Title           string
	Description     string
	LongDescription string // enriched summary from the One Piece Wiki (populated by poller when quiz is enabled)
	SeasonTitle     string
	ThumbnailURL    string
	DurationMS      int
	WatchedAt       time.Time
}

// Label returns the episode's display label, falling back to its number.
func (e EpisodeInfo) Label() string {
	if e.EpisodeLabel != "" {
		return e.EpisodeLabel
	}
	return strconv.Itoa(e.Number)
}

// IsSpecial reports whether this is a special or recap rather than a numbered
// story episode. Crunchyroll labels those "SP3" or "Recap" while reusing a low
// episode number (Wano's specials are numbered 2-11), so anything whose label
// disagrees with its number must not be treated as that episode.
func (e EpisodeInfo) IsSpecial() bool {
	return e.EpisodeLabel != "" && e.EpisodeLabel != strconv.Itoa(e.Number)
}

// DailyCount holds the episode count for a single calendar day.
type DailyCount struct {
	Date  string // "2006-01-02" format (UTC)
	Count int
}
