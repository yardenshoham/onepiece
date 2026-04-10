package crunchyroll

import "time"

// OnePieceSeriesID is the Crunchyroll series ID for One Piece.
const OnePieceSeriesID = "GRMG8ZQZR"

// Profile represents the user's Crunchyroll profile.
type Profile struct {
	ProfileName string `json:"profile_name"`
}

// authResponse represents the response from the auth endpoint.
type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	AccountID    string `json:"account_id"`
}

// WatchHistoryResponse represents the paginated watch history response.
type WatchHistoryResponse struct {
	Total int                 `json:"total"`
	Data  []WatchHistoryEntry `json:"data"`
}

// WatchHistoryEntry represents a single entry in the watch history.
type WatchHistoryEntry struct {
	ID           string    `json:"id"`
	DatePlayed   time.Time `json:"date_played"`
	FullyWatched bool      `json:"fully_watched"`
	Panel        Panel     `json:"panel"`
}

// Panel holds the display metadata for a watch history entry.
type Panel struct {
	Title           string          `json:"title"`
	EpisodeMetadata EpisodeMetadata `json:"episode_metadata"`
}

// EpisodeMetadata holds episode-level information.
type EpisodeMetadata struct {
	EpisodeNumber  int       `json:"episode_number"`
	SeasonNumber   int       `json:"season_number"`
	SeasonTitle    string    `json:"season_title"`
	SeriesID       string    `json:"series_id"`
	SeriesTitle    string    `json:"series_title"`
	EpisodeAirDate time.Time `json:"episode_air_date"`
	DurationMS     int       `json:"duration_ms"`
}

// Season represents a season of a series.
type Season struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	SeasonNumber     int    `json:"season_number"`
	NumberOfEpisodes int    `json:"number_of_episodes"`
	SlugTitle        string `json:"slug_title"`
}

// seasonsResponse wraps the seasons API response.
type seasonsResponse struct {
	Data []Season `json:"data"`
}

// Series represents series metadata.
type Series struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// seriesResponse wraps the series API response.
type seriesResponse struct {
	Data []Series `json:"data"`
}
