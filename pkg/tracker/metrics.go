package tracker

import "time"

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

	// Metadata
	LastUpdated time.Time
}

// EpisodeInfo holds information about a single watched episode.
type EpisodeInfo struct {
	Number       int
	Title        string
	Description  string
	SeasonTitle  string
	ThumbnailURL string
	SlugTitle    string
	DurationMS   int
	WatchedAt    time.Time
}

// DailyCount holds the episode count for a single calendar day.
type DailyCount struct {
	Date  string // "2006-01-02" format (UTC)
	Count int
}
